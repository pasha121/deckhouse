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
	"strings"

	"github.com/flant/addon-operator/pkg/module_manager/go_hook"
	"github.com/flant/addon-operator/sdk"
	"github.com/flant/shell-operator/pkg/kube_events_manager/types"
	corev1 "k8s.io/api/core/v1"
	storagev1 "k8s.io/api/storage/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	ktypes "k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/pointer"
	cdiv1 "kubevirt.io/containerized-data-importer-api/pkg/apis/core/v1beta1"

	"github.com/deckhouse/deckhouse/modules/900-virtualization/api/v1alpha1"
)

const (
	storageClassesSnapshot   = "diskHandlerStorageClass"
	disksSnapshot            = "diskHandlerVirtualMachineDisk"
	clusterImagesSnapshot    = "diskHandlerClusterVirtualMachineImage"
	dataVolumesSnapshot      = "diskHandlerDataVolume"
	cdiDataVolumeCRDSnapshot = "diskHandlerCDIDataVolumeCRD"
)

var diskHandlerHookConfig = &go_hook.HookConfig{
	Queue: "/modules/virtualization/disk-handler",
	Kubernetes: []go_hook.KubernetesConfig{
		// A binding with dynamic kind has index 0 for simplicity.
		{
			Name:       dataVolumesSnapshot,
			ApiVersion: "",
			Kind:       "",
			FilterFunc: applyDataVolumeFilter,
		},
		{
			Name:       storageClassesSnapshot,
			ApiVersion: "storage.k8s.io/v1",
			Kind:       "StorageClass",
			FilterFunc: applyStorageClassFilter,
		},
		{
			Name:       disksSnapshot,
			ApiVersion: gv,
			Kind:       "VirtualMachineDisk",
			FilterFunc: applyVirtualMachineDiskFilter,
		},
		{
			Name:       clusterImagesSnapshot,
			ApiVersion: gv,
			Kind:       "ClusterVirtualMachineImage",
			FilterFunc: applyClusterVirtualMachineImageFilter,
		},
		{
			Name:       cdiDataVolumeCRDSnapshot,
			ApiVersion: "apiextensions.k8s.io/v1",
			Kind:       "CustomResourceDefinition",
			NameSelector: &types.NameSelector{
				MatchNames: []string{"datavolumes.cdi.kubevirt.io"},
			},
			FilterFunc: applyCRDExistenseFilter,
		},
	},
}

var _ = sdk.RegisterFunc(diskHandlerHookConfig, handleVirtualMachineDisks)

type StorageClassSnapshot struct {
	Name                string
	Namespace           string
	AccessModes         []corev1.PersistentVolumeAccessMode
	VolumeMode          *corev1.PersistentVolumeMode
	DefaultStorageClass bool
}

type VirtualMachineDiskSnapshot struct {
	Name             string
	Namespace        string
	UID              ktypes.UID
	StorageClassName string
	Size             resource.Quantity
	Source           *corev1.TypedLocalObjectReference
}

type ClusterVirtualMachineImageSnapshot struct {
	Name             string
	Namespace        string
	UID              ktypes.UID
	StorageClassName string
	Size             resource.Quantity
	Remote           *cdiv1.DataVolumeSource
	Source           *v1alpha1.TypedObjectReference
}

type DataVolumeSnapshot struct {
	Name      string
	Namespace string
}

func applyStorageClassFilter(obj *unstructured.Unstructured) (go_hook.FilterResult, error) {
	storageClass := &storagev1.StorageClass{}
	err := sdk.FromUnstructured(obj, storageClass)
	if err != nil {
		return nil, fmt.Errorf("cannot convert object to StorageClass: %v", err)
	}
	sc := &StorageClassSnapshot{
		Name: storageClass.Name,
	}

	if storageClass.Annotations["storageclass.kubernetes.io/is-default-class"] == "true" {
		sc.DefaultStorageClass = true
	}

	volumeMode := storageClass.Parameters["virtualization.deckhouse.io/volumeMode"]
	if volumeMode != "" {
		a := corev1.PersistentVolumeMode(volumeMode)
		sc.VolumeMode = &a
	}

	accessModes := storageClass.Parameters["virtualization.deckhouse.io/accessModes"]
	if accessModes != "" {
		sc.AccessModes = []corev1.PersistentVolumeAccessMode{}
		for _, am := range strings.Split(accessModes, ",") {
			sc.AccessModes = append(sc.AccessModes, corev1.PersistentVolumeAccessMode(am))
		}
	}

	return sc, nil
}

func applyVirtualMachineDiskFilter(obj *unstructured.Unstructured) (go_hook.FilterResult, error) {
	disk := &v1alpha1.VirtualMachineDisk{}
	err := sdk.FromUnstructured(obj, disk)
	if err != nil {
		return nil, fmt.Errorf("cannot convert object to VirtualMachineDisk: %v", err)
	}

	return &VirtualMachineDiskSnapshot{
		Name:             disk.Name,
		Namespace:        disk.Namespace,
		UID:              disk.UID,
		StorageClassName: disk.Spec.StorageClassName,
		Size:             disk.Spec.Size,
		Source:           disk.Spec.Source,
	}, nil
}

func applyClusterVirtualMachineImageFilter(obj *unstructured.Unstructured) (go_hook.FilterResult, error) {
	clusterImage := &v1alpha1.ClusterVirtualMachineImage{}
	err := sdk.FromUnstructured(obj, clusterImage)
	if err != nil {
		return nil, fmt.Errorf("cannot convert object to DataVolume: %v", err)
	}

	return &ClusterVirtualMachineImageSnapshot{
		Name:      clusterImage.Name,
		Namespace: clusterImage.Namespace,
		UID:       clusterImage.UID,
		Source:    clusterImage.Spec.Source,
		Remote:    clusterImage.Spec.Remote,
	}, nil
}

func applyDataVolumeFilter(obj *unstructured.Unstructured) (go_hook.FilterResult, error) {
	volume := &cdiv1.DataVolume{}
	err := sdk.FromUnstructured(obj, volume)
	if err != nil {
		return nil, fmt.Errorf("cannot convert object to DataVolume: %v", err)
	}

	return &DataVolumeSnapshot{
		Name:      volume.Name,
		Namespace: volume.Namespace,
	}, nil
}

// handleVirtualMachineDisks
//
// synopsis:
//   TODO
func handleVirtualMachineDisks(input *go_hook.HookInput) error {
	// CDI manages it's own CRDs, so we need to wait for them before starting the watch
	if diskHandlerHookConfig.Kubernetes[0].Kind == "" {
		if len(input.Snapshots[cdiDataVolumeCRDSnapshot]) > 0 {
			// CDI installed
			input.LogEntry.Infof("CDI DataVolume CRD installed, update kind for binding datavolumes.cdi.kubevirt.io")
			*input.BindingActions = append(*input.BindingActions, go_hook.BindingAction{
				Name:       dataVolumesSnapshot,
				Action:     "UpdateKind",
				ApiVersion: "cdi.kubevirt.io/v1beta1",
				Kind:       "DataVolume",
			})
			// Save new kind as current kind.
			diskHandlerHookConfig.Kubernetes[0].Kind = "DataVolume"
			diskHandlerHookConfig.Kubernetes[0].ApiVersion = "cdi.kubevirt.io/v1beta1"
			// Binding changed, hook will be restarted with new objects in snapshot.
			return nil
		}
		// CDI is not yet installed, do nothing
		return nil
	}

	// Start main hook logic
	storageClassSnap := input.Snapshots[storageClassesSnapshot]
	diskSnap := input.Snapshots[disksSnapshot]
	clusterImageSnap := input.Snapshots[clusterImagesSnapshot]
	dataVolumeSnap := input.Snapshots[dataVolumesSnapshot]

	if len(diskSnap) == 0 && len(storageClassSnap) == 0 {
		input.LogEntry.Warnln("VirtualMachineDisk and StorageClass not found. Skip")
		return nil
	}

	for _, sRaw := range diskSnap {
		disk := sRaw.(*VirtualMachineDiskSnapshot)
		if getDataVolume(&dataVolumeSnap, disk.Namespace, "disk-"+disk.Name) != nil {
			// DataVolume found, noting to do
			continue
		}

		// DataVolume not found, needs to create a new one

		// Lookup for storageClass
		storageClass := getStorageClass(&storageClassSnap, disk.StorageClassName)
		if storageClass == nil {
			input.LogEntry.Warnln("StorageClass not found. Skip")
			continue
		}

		source := &v1alpha1.TypedObjectReference{}
		source.APIGroup = disk.Source.APIGroup
		source.Kind = disk.Source.Kind
		source.Name = disk.Source.Name

		dataVolumeSource, err := resolveDataVolumeSource(&diskSnap, &clusterImageSnap, &dataVolumeSnap, source)
		if err != nil {
			input.LogEntry.Warnf("%s. Skip", err)
		}

		dataVolume := &cdiv1.DataVolume{
			TypeMeta: metav1.TypeMeta{
				Kind:       "DataVolume",
				APIVersion: "cdi.kubevirt.io/v1beta1",
			},
			ObjectMeta: v1.ObjectMeta{
				Name:      "disk-" + disk.Name,
				Namespace: disk.Namespace,
				OwnerReferences: []v1.OwnerReference{{
					APIVersion:         gv,
					BlockOwnerDeletion: pointer.Bool(true),
					Controller:         pointer.Bool(true),
					Kind:               "VirtualMachineDisk",
					Name:               disk.Name,
					UID:                disk.UID,
				}},
			},
			Spec: cdiv1.DataVolumeSpec{
				Source: dataVolumeSource,
				PVC: &corev1.PersistentVolumeClaimSpec{
					AccessModes:      storageClass.AccessModes,
					StorageClassName: &storageClass.Name,
					VolumeMode:       storageClass.VolumeMode,
					Resources: corev1.ResourceRequirements{
						Requests: corev1.ResourceList{
							corev1.ResourceStorage: disk.Size, // TODO nil check?
						},
					},
				},
			},
		}
		input.PatchCollector.Create(dataVolume)
	}

	return nil
}

func getStorageClass(snapshot *[]go_hook.FilterResult, name string) *StorageClassSnapshot {
	for _, dRaw := range *snapshot {
		storageClass := dRaw.(*StorageClassSnapshot)
		if name != "" {
			if storageClass.Name == name {
				return storageClass
			}
		} else {
			if storageClass.DefaultStorageClass {
				return storageClass
			}
		}
	}
	return nil
}

func getClusterImage(snapshot *[]go_hook.FilterResult, name string) *ClusterVirtualMachineImageSnapshot {
	for _, dRaw := range *snapshot {
		clusterImage := dRaw.(*ClusterVirtualMachineImageSnapshot)
		if clusterImage.Name == name {
			return clusterImage
		}
	}
	return nil
}

func getDisk(snapshot *[]go_hook.FilterResult, namespace, name string) *VirtualMachineDiskSnapshot {
	for _, dRaw := range *snapshot {
		disk := dRaw.(*VirtualMachineDiskSnapshot)
		if disk.Namespace == namespace && disk.Name == name {
			return disk
		}
	}
	return nil
}

func getDataVolume(snapshot *[]go_hook.FilterResult, namespace, name string) *DataVolumeSnapshot {
	for _, dRaw := range *snapshot {
		dataVolume := dRaw.(*DataVolumeSnapshot)
		if dataVolume.Namespace == namespace && dataVolume.Name == name {
			return dataVolume
		}
	}
	return nil
}

func resolveDataVolumeSource(diskSnap, clusterImageSnap, dataVolumeSnap *[]go_hook.FilterResult, source *v1alpha1.TypedObjectReference) (*cdiv1.DataVolumeSource, error) {
	if source == nil {
		return &cdiv1.DataVolumeSource{Blank: &cdiv1.DataVolumeBlankImage{}}, nil
	}
	switch source.Kind {
	case "VirtualMachineDisk":
		disk := getDisk(diskSnap, source.Namespace, source.Name)
		if disk == nil {
			return nil, fmt.Errorf("VirtualMachineDisk not found")
		}
		return &cdiv1.DataVolumeSource{PVC: &cdiv1.DataVolumeSourcePVC{Namespace: disk.Namespace, Name: "disk-" + disk.Name}}, nil
	case "ClusterVirtualMachineImage":
		clusterImage := getClusterImage(clusterImageSnap, source.Name)
		if clusterImage == nil {
			return nil, fmt.Errorf("ClusterVirtualMachineImage not found")
		}
		if clusterImage.Remote != nil {
			return clusterImage.Remote, nil
		}
		if clusterImage.Source != nil {
			return resolveDataVolumeSource(diskSnap, clusterImageSnap, dataVolumeSnap, clusterImage.Source)
		}
		return nil, fmt.Errorf("Neither source and remote specified")
	case "VirtualMachineImage":
		// TODO handle namespaced VirtualMachineImage
		return nil, fmt.Errorf("Neither source and remote specified")
	}
	return nil, fmt.Errorf("Unknown type of source")
}
