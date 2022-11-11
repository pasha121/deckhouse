/*
Copyright 2021 Flant JSC

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package hooks

import (
	"fmt"
	"strconv"

	"github.com/flant/addon-operator/pkg/module_manager/go_hook"
	"github.com/flant/addon-operator/sdk"
	"github.com/flant/shell-operator/pkg/kube_events_manager/types"
	"gopkg.in/yaml.v2"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/utils/pointer"
	virtv1 "kubevirt.io/api/core/v1"

	"github.com/deckhouse/deckhouse/modules/900-virtualization/api/v1alpha1"
)

const (
	deckhouseVMSnapshot    = "vmHandlerDeckhouseVM"
	kubevirtVMSnapshot     = "vmHandlerKubevirtVM"
	kubevirtVMsCRDSnapshot = "vmHandlerKubevirtVMCRD"
	diskNamesSnapshot      = "disksNamesSnapshot"
)

var vmHandlerHookConfig = &go_hook.HookConfig{
	Queue: "/modules/virtualization/vm-handler",
	Kubernetes: []go_hook.KubernetesConfig{
		// A binding with dynamic kind has index 0 for simplicity.
		{
			Name:       kubevirtVMSnapshot,
			ApiVersion: "",
			Kind:       "",
			FilterFunc: applyKubevirtVMFilter,
		},
		{
			Name:       deckhouseVMSnapshot,
			ApiVersion: gv,
			Kind:       "VirtualMachine",
			FilterFunc: applyDeckhouseVMFilter,
		},
		{
			Name:       diskNamesSnapshot,
			ApiVersion: gv,
			Kind:       "VirtualMachineDisk",
			FilterFunc: applyVirtualMachineDiskNamesFilter,
		},
		{
			Name:       kubevirtVMsCRDSnapshot,
			ApiVersion: "apiextensions.k8s.io/v1",
			Kind:       "CustomResourceDefinition",
			NameSelector: &types.NameSelector{
				MatchNames: []string{"virtualmachines.kubevirt.io"},
			},
			FilterFunc: applyCRDExistenseFilter,
		},
	},
}

var _ = sdk.RegisterFunc(vmHandlerHookConfig, handleVMs)

type InstanceClassCrdInfo struct {
	Name string
	Spec interface{}
}

func applyKubevirtVMFilter(obj *unstructured.Unstructured) (go_hook.FilterResult, error) {
	vm := &virtv1.VirtualMachine{}
	err := sdk.FromUnstructured(obj, vm)
	if err != nil {
		return nil, fmt.Errorf("cannot convert object to VirtualMachine: %v", err)
	}

	return vm, nil
}

func applyDeckhouseVMFilter(obj *unstructured.Unstructured) (go_hook.FilterResult, error) {
	vm := &v1alpha1.VirtualMachine{}
	err := sdk.FromUnstructured(obj, vm)
	if err != nil {
		return nil, fmt.Errorf("cannot convert object to VirtualMachine: %v", err)
	}

	return vm, nil
}

func applyVirtualMachineDiskNamesFilter(obj *unstructured.Unstructured) (go_hook.FilterResult, error) {
	disk := &v1alpha1.VirtualMachineDisk{}
	err := sdk.FromUnstructured(obj, disk)
	if err != nil {
		return nil, fmt.Errorf("cannot convert object to VirtualMachineDisk: %v", err)
	}

	return &VirtualMachineDiskSnapshot{
		Name:      disk.Name,
		Namespace: disk.Namespace,
	}, nil
}

// handleVMs
//
// synopsis:
//   TODO
func handleVMs(input *go_hook.HookInput) error {
	// KubeVirt manages it's own CRDs, so we need to wait for them before starting the watch
	if vmHandlerHookConfig.Kubernetes[0].Kind == "" {
		if len(input.Snapshots[kubevirtVMsCRDSnapshot]) > 0 {
			// KubeVirt installed
			input.LogEntry.Infof("KubeVirt VirtualMachine CRD installed, update kind for binding VirtualMachines.kubevirt.io")
			*input.BindingActions = append(*input.BindingActions, go_hook.BindingAction{
				Name:       kubevirtVMSnapshot,
				Action:     "UpdateKind",
				ApiVersion: "kubevirt.io/v1",
				Kind:       "VirtualMachine",
			})
			// Save new kind as current kind.
			vmHandlerHookConfig.Kubernetes[0].Kind = "VirtualMachine"
			vmHandlerHookConfig.Kubernetes[0].ApiVersion = "kubevirt.io/v1"
			// Binding changed, hook will be restarted with new objects in snapshot.
			return nil
		}
		// KubeVirt is not yet installed, do nothing
		return nil
	}

	// Start main hook logic
	kubevirtVMSnap := input.Snapshots[kubevirtVMSnapshot]
	deckhouseVMSnap := input.Snapshots[deckhouseVMSnapshot]
	diskNameSnap := input.Snapshots[diskNamesSnapshot]

	if len(kubevirtVMSnap) == 0 && len(deckhouseVMSnap) == 0 {
		input.LogEntry.Warnln("VirtualMachine not found. Skip")
		return nil
	}

VM_LOOP:
	for _, sRaw := range deckhouseVMSnap {
		d8vm := sRaw.(*v1alpha1.VirtualMachine)
		if d8vm.Status.IPAddress == "" {
			// IPAddress is not set by IPAM, nothing todo
			continue
		}
		for _, dRaw := range kubevirtVMSnap {
			kvvm := dRaw.(*virtv1.VirtualMachine)
			if d8vm.Namespace != kvvm.Namespace {
				continue
			}
			if d8vm.Name != kvvm.Name {
				continue
			}
			// KubeVirt VirtualMachine found
			apply := func(u *unstructured.Unstructured) (*unstructured.Unstructured, error) {
				vm := &virtv1.VirtualMachine{}
				err := sdk.FromUnstructured(u, vm)
				if err != nil {
					return nil, err
				}
				err = setVMFields(d8vm, vm)
				if err != nil {
					return nil, err
				}
				return sdk.ToUnstructured(&vm)
			}
			input.PatchCollector.Filter(apply, "kubevirt.io/v1", "VirtualMachine", d8vm.Namespace, d8vm.Name)

			continue VM_LOOP
		}

		// KubeVirt VirtualMachine not found, needs to create a new one

		var bootVirtualMachineDiskName string
		switch d8vm.Spec.BootDisk.Source.Kind {
		case "VirtualMachineDisk":
			bootVirtualMachineDiskName = d8vm.Spec.BootDisk.Source.Name
			if disk := getDisk(&diskNameSnap, d8vm.Namespace, d8vm.Spec.BootDisk.Source.Name); disk == nil {
				input.LogEntry.Warnln("Disk not found")
			}
		case "ClusterVirtualMachineImage":
			bootVirtualMachineDiskName = d8vm.Name + "-boot"
			if disk := getDisk(&diskNameSnap, d8vm.Namespace, bootVirtualMachineDiskName); disk == nil {
				// Create a new VirtualMachineDisk
				disk := &v1alpha1.VirtualMachineDisk{
					TypeMeta: metav1.TypeMeta{
						Kind:       "VirtualMachineDisk",
						APIVersion: gv,
					},
					ObjectMeta: v1.ObjectMeta{
						Name:      bootVirtualMachineDiskName,
						Namespace: d8vm.Namespace,
						OwnerReferences: []v1.OwnerReference{{
							APIVersion:         gv,
							BlockOwnerDeletion: pointer.Bool(true),
							Controller:         pointer.Bool(true),
							Kind:               "VirtualMachine",
							Name:               d8vm.Name,
							UID:                d8vm.UID,
						}},
					},
					Spec: v1alpha1.VirtualMachineDiskSpec{
						StorageClassName: d8vm.Spec.BootDisk.StorageClassName,
						Size:             d8vm.Spec.BootDisk.Size,
						Source:           d8vm.Spec.BootDisk.Source,
					},
				}
				input.PatchCollector.Create(disk)
			}
		default:
			input.LogEntry.Warnln("Unknown source kind")
		}

		kvvm := &virtv1.VirtualMachine{}
		err := setVMFields(d8vm, kvvm)
		if err != nil {
			return err
		}
		input.PatchCollector.Create(kvvm)
	}

	return nil
}

func setVMFields(d8vm *v1alpha1.VirtualMachine, vm *virtv1.VirtualMachine) error {
	vm.TypeMeta = metav1.TypeMeta{
		Kind:       "VirtualMachine",
		APIVersion: "kubevirt.io/v1",
	}
	vm.SetName(d8vm.Name)
	vm.SetNamespace(d8vm.Namespace)
	vm.SetOwnerReferences([]v1.OwnerReference{{
		APIVersion:         gv,
		BlockOwnerDeletion: pointer.Bool(true),
		Controller:         pointer.Bool(true),
		Kind:               "VirtualMachine",
		Name:               d8vm.Name,
		UID:                d8vm.UID,
	}})

	cloudInit := make(map[string]interface{})
	if d8vm.Spec.CloudInit != nil {
		err := yaml.Unmarshal([]byte(d8vm.Spec.CloudInit.UserData), &cloudInit)
		if err != nil {
			return fmt.Errorf("cannot parse cloudInit config for VirtualMachine: %v", err)
		}
	}
	if d8vm.Spec.SSHPublicKey != "" {
		cloudInit["ssh_authorized_keys"] = []string{d8vm.Spec.SSHPublicKey}
	}
	if d8vm.Spec.UserName != "" {
		cloudInit["user"] = d8vm.Spec.UserName
	}
	cloudInitRaw, _ := yaml.Marshal(cloudInit)

	vm.Spec.Running = d8vm.Spec.Running
	vm.Spec.Template = &virtv1.VirtualMachineInstanceTemplateSpec{
		ObjectMeta: v1.ObjectMeta{
			Annotations: map[string]string{
				"cni.cilium.io/ipAddrs":  d8vm.Status.IPAddress,
				"cni.cilium.io/macAddrs": "f6:e1:74:94:b8:1a",
			},
		},
		Spec: virtv1.VirtualMachineInstanceSpec{
			Domain: virtv1.DomainSpec{
				Devices: virtv1.Devices{
					Interfaces: []virtv1.Interface{{
						Name:  "default",
						Model: "virtio",
						InterfaceBindingMethod: virtv1.InterfaceBindingMethod{
							Macvtap: &virtv1.InterfaceMacvtap{},
						},
					}},
					Disks: []virtv1.Disk{
						{
							Name: "boot",
							DiskDevice: virtv1.DiskDevice{
								Disk: &virtv1.DiskTarget{
									Bus: "virtio",
								},
							},
						},
						{
							Name: "cloudinit",
							DiskDevice: virtv1.DiskDevice{
								Disk: &virtv1.DiskTarget{
									Bus: "virtio",
								},
							},
						},
					},
				},
				Resources: virtv1.ResourceRequirements{
					Requests: d8vm.Spec.Resources,
				},
			},
			Networks: []virtv1.Network{{
				Name: "default",
				NetworkSource: virtv1.NetworkSource{
					Pod: &virtv1.PodNetwork{},
				}}},
			Volumes: []virtv1.Volume{
				{
					Name: "boot",
					VolumeSource: virtv1.VolumeSource{
						DataVolume: &virtv1.DataVolumeSource{
							Name:         "disk-" + vm.Name + "-boot", // TODO kind Disk?
							Hotpluggable: false,
						},
					},
				},
				{
					Name: "cloudinit",
					VolumeSource: virtv1.VolumeSource{
						CloudInitNoCloud: &virtv1.CloudInitNoCloudSource{
							// TODO handle cloudinit from secret
							UserData: fmt.Sprintf("#cloud-config\n%s", cloudInitRaw),
						},
					},
				},
			},
		},
	}

	// attach extra disks
	if d8vm.Spec.DiskAttachments != nil {
		for i, disk := range *d8vm.Spec.DiskAttachments {
			diskName := "disk-" + strconv.Itoa(i+1)
			vm.Spec.Template.Spec.Domain.Devices.Disks = append(vm.Spec.Template.Spec.Domain.Devices.Disks, virtv1.Disk{
				Name: diskName,
				DiskDevice: virtv1.DiskDevice{
					Disk: &virtv1.DiskTarget{
						Bus: disk.Bus,
					},
				},
			})
			vm.Spec.Template.Spec.Volumes = append(vm.Spec.Template.Spec.Volumes, virtv1.Volume{
				Name: diskName,
				VolumeSource: virtv1.VolumeSource{
					DataVolume: &virtv1.DataVolumeSource{
						Name:         "disk-" + disk.Name,
						Hotpluggable: disk.Hotpluggable,
					},
				},
			})
		}
	}
	return nil
}
