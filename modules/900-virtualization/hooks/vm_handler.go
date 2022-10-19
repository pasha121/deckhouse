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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/utils/pointer"
	virtv1 "kubevirt.io/api/core/v1"

	"github.com/deckhouse/deckhouse/modules/900-virtualization/api/v1alpha1"
)

const (
	deckhouseVMSnapshot = "vmHandlerDeckhouseVM"
	kubevirtVMSnapshot  = "vmHandlerKubevirtVM"
	diskNamesSnapshot   = "disksNamesSnapshot"
)

var _ = sdk.RegisterFunc(&go_hook.HookConfig{
	Queue: "/modules/virtualization/vm-handler",
	Kubernetes: []go_hook.KubernetesConfig{
		{
			Name:       kubevirtVMSnapshot,
			ApiVersion: "kubevirt.io/v1",
			Kind:       "VirtualMachine",
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
			Kind:       "Disk",
			FilterFunc: applyDiskNamesFilter,
		},
	},
}, handleVMs)

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

func applyDiskNamesFilter(obj *unstructured.Unstructured) (go_hook.FilterResult, error) {
	disk := &v1alpha1.Disk{}
	err := sdk.FromUnstructured(obj, disk)
	if err != nil {
		return nil, fmt.Errorf("cannot convert object to Disk: %v", err)
	}

	return &DiskSnapshot{
		Name:      disk.Name,
		Namespace: disk.Namespace,
	}, nil
}

// handleVMs
//
// synopsis:
//   TODO
func handleVMs(input *go_hook.HookInput) error {
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
				setVMFields(d8vm, vm)
				return sdk.ToUnstructured(&vm)
			}
			input.PatchCollector.Filter(apply, "kubevirt.io/v1", "VirtualMachine", d8vm.Namespace, d8vm.Name)

			continue VM_LOOP
		}

		// KubeVirt VirtualMachine not found, needs to create a new one

		var bootDiskName string
		if d8vm.Spec.BootDisk.Disk != (v1alpha1.DiskSource{}) {
			bootDiskName = d8vm.Spec.BootDisk.Disk.Name
		}

		if d8vm.Spec.BootDisk.Image != (v1alpha1.ImageSource{}) {
			if bootDiskName != "" {
				input.LogEntry.Errorln("Disk source can't be specifed with image source for bootDisk")
				continue
			}
			bootDiskName = d8vm.Name + "-boot"
			if bootDiskName != "" {
				bootDiskFound := false
				for _, dRaw := range diskNameSnap {
					disk := dRaw.(*DiskSnapshot)
					if disk.Namespace != d8vm.Namespace {
						continue
					}
					if disk.Name != bootDiskName {
						continue
					}
					bootDiskFound = true
				}
				if !bootDiskFound {
					// Create a new Disk
					disk := &v1alpha1.Disk{
						TypeMeta: metav1.TypeMeta{
							Kind:       "Disk",
							APIVersion: gv,
						},
						ObjectMeta: v1.ObjectMeta{
							Name:      bootDiskName,
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
						Spec: v1alpha1.DiskSpec{
							Type: d8vm.Spec.BootDisk.Image.Type,
							Size: d8vm.Spec.BootDisk.Image.Size,
							Source: v1alpha1.ImageSourceRef{
								Name:  d8vm.Spec.BootDisk.Image.Name,
								Scope: v1alpha1.ImageSourceScopePublic,
							},
						},
					}
					input.PatchCollector.Create(disk)
				}
			}
		}

		kvvm := &virtv1.VirtualMachine{}
		setVMFields(d8vm, kvvm)
		input.PatchCollector.Create(kvvm)

	}

	return nil
}

func setVMFields(d8vm *v1alpha1.VirtualMachine, vm *virtv1.VirtualMachine) {
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
							Name:         vm.Name + "-boot",
							Hotpluggable: false,
						},
					},
				},
				{
					Name: "cloudinit",
					VolumeSource: virtv1.VolumeSource{
						CloudInitNoCloud: &virtv1.CloudInitNoCloudSource{
							// TODO ssh public key
							UserData: d8vm.Spec.CloudInit.UserData,
						},
					},
				},
			},
		},
	}

	// attach extra disks
	if d8vm.Spec.Disks != nil {
		for i, disk := range *d8vm.Spec.Disks {
			diskName := "disk-" + strconv.Itoa(i)
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
						Name:         disk.Name,
						Hotpluggable: disk.Hotpluggable,
					},
				},
			})
		}
	}
}
