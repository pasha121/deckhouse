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

	"github.com/flant/addon-operator/pkg/module_manager/go_hook"
	"github.com/flant/addon-operator/sdk"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/pointer"
	cdiv1 "kubevirt.io/containerized-data-importer-api/pkg/apis/core/v1beta1"

	"github.com/deckhouse/deckhouse/modules/900-virtualization/api/v1alpha1"
)

const (
	diskTypesSnapshot          = "diskHandlerDisktype"
	disksSnapshot              = "diskHandlerDisk"
	publicImageSourcesSnapshot = "diskHandlerPublicImageSource"
	dataVolumeSnapshot         = "diskHandlerDataVolume"
)

var _ = sdk.RegisterFunc(&go_hook.HookConfig{
	Queue: "/modules/virtualization/disk-handler",
	Kubernetes: []go_hook.KubernetesConfig{
		{
			Name:       diskTypesSnapshot,
			ApiVersion: gv,
			Kind:       "DiskType",
			FilterFunc: applyDiskTypeFilter,
		},
		{
			Name:       disksSnapshot,
			ApiVersion: gv,
			Kind:       "Disk",
			FilterFunc: applyDiskFilter,
		},
		{
			Name:       publicImageSourcesSnapshot,
			ApiVersion: gv,
			Kind:       "PublicImageSource",
			FilterFunc: applyPublicImageSourceFilter,
		},
		{
			Name:       dataVolumeSnapshot,
			ApiVersion: "cdi.kubevirt.io/v1beta1",
			Kind:       "DataVolume",
			FilterFunc: applyDataVolumeFilter,
		},
	},
}, handleDisks)

type DiskTypeSnapshot struct {
	Name      string
	Namespace string
	Spec      v1alpha1.DiskTypeSpec
}

type DiskSnapshot struct {
	Name      string
	Namespace string
	UID       types.UID
	Type      string
	Size      resource.Quantity
	Source    v1alpha1.ImageSourceRef
}

type PublicImageSourceSnapshot struct {
	Name      string
	Namespace string
	Source    cdiv1.DataVolumeSource
}

type DataVolumeSnapshot struct {
	Name      string
	Namespace string
}

func applyDiskTypeFilter(obj *unstructured.Unstructured) (go_hook.FilterResult, error) {
	diskType := &v1alpha1.DiskType{}
	err := sdk.FromUnstructured(obj, diskType)
	if err != nil {
		return nil, fmt.Errorf("cannot convert object to DiskType: %v", err)
	}

	return &DiskTypeSnapshot{
		Name:      diskType.Name,
		Namespace: diskType.Namespace,
		Spec:      diskType.Spec,
	}, nil
}

func applyDiskFilter(obj *unstructured.Unstructured) (go_hook.FilterResult, error) {
	disk := &v1alpha1.Disk{}
	err := sdk.FromUnstructured(obj, disk)
	if err != nil {
		return nil, fmt.Errorf("cannot convert object to Disk: %v", err)
	}

	return &DiskSnapshot{
		Name:      disk.Name,
		Namespace: disk.Namespace,
		UID:       disk.UID,
		Type:      disk.Spec.Type,
		Size:      disk.Spec.Size,
		Source:    disk.Spec.Source,
	}, nil
}

func applyPublicImageSourceFilter(obj *unstructured.Unstructured) (go_hook.FilterResult, error) {
	publicImageSource := &v1alpha1.PublicImageSource{}
	err := sdk.FromUnstructured(obj, publicImageSource)
	if err != nil {
		return nil, fmt.Errorf("cannot convert object to DataVolume: %v", err)
	}

	return &PublicImageSourceSnapshot{
		Name:      publicImageSource.Name,
		Namespace: publicImageSource.Namespace,
		Source:    publicImageSource.Spec,
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

// handleDisks
//
// synopsis:
//   TODO
func handleDisks(input *go_hook.HookInput) error {
	diskTypeSnap := input.Snapshots[diskTypesSnapshot]
	diskSnap := input.Snapshots[disksSnapshot]
	publicImageSourceSnap := input.Snapshots[publicImageSourcesSnapshot]
	dataVolumeSnap := input.Snapshots[dataVolumeSnapshot]

	if len(diskSnap) == 0 && len(diskTypeSnap) == 0 {
		input.LogEntry.Warnln("Disk and DiskType not found. Skip")
		return nil
	}

	for _, sRaw := range diskSnap {
		disk := sRaw.(*DiskSnapshot)
		for _, dRaw := range dataVolumeSnap {
			dataVolume := dRaw.(*DataVolumeSnapshot)
			if dataVolume.Namespace != disk.Namespace {
				continue
			}
			if dataVolume.Name != disk.Name {
				continue
			}
			// DataVolume found
			continue
		}
		// DataVolume not found, needs to create a new one

		var diskTypeSpec v1alpha1.DiskTypeSpec
		var diskTypeFound bool
		for _, dRaw := range diskTypeSnap {
			diskType := dRaw.(*DiskTypeSnapshot)
			if diskType.Name == disk.Type {
				diskTypeSpec = diskType.Spec
				diskTypeFound = true
			}
		}
		if !diskTypeFound {
			input.LogEntry.Warnln("DiskType not found. Skip")
			continue
		}

		var imageSource cdiv1.DataVolumeSource
		if disk.Source.Name != "" {
			var imageSourceFound bool
			if disk.Source.Scope == v1alpha1.ImageSourceScopePublic || disk.Source.Scope == "" {
				for _, dRaw := range publicImageSourceSnap {
					publicImageSource := dRaw.(*PublicImageSourceSnapshot)
					if publicImageSource.Name == disk.Source.Name {
						imageSource = publicImageSource.Source
						imageSourceFound = true
					}
				}
			}
			if disk.Source.Scope == v1alpha1.ImageSourceScopePrivate || disk.Source.Scope == "" && !imageSourceFound {
				// TODO handle privateImageSource
			}
			if !imageSourceFound {
				input.LogEntry.Warnln("ImageSource not found. Skip")
				continue
			}
		} else {
			imageSource = cdiv1.DataVolumeSource{
				Blank: &cdiv1.DataVolumeBlankImage{},
			}
		}

		dataVolume := &cdiv1.DataVolume{
			TypeMeta: metav1.TypeMeta{
				Kind:       "DataVolume",
				APIVersion: "cdi.kubevirt.io/v1beta1",
			},
			ObjectMeta: v1.ObjectMeta{
				Name:      disk.Name,
				Namespace: disk.Namespace,
				OwnerReferences: []v1.OwnerReference{{
					APIVersion:         gv,
					BlockOwnerDeletion: pointer.Bool(true),
					Controller:         pointer.Bool(true),
					Kind:               "Disk",
					Name:               disk.Name,
					UID:                disk.UID,
				}},
			},
			Spec: cdiv1.DataVolumeSpec{
				Source: &imageSource,
				PVC: &corev1.PersistentVolumeClaimSpec{
					AccessModes:      diskTypeSpec.AccessModes,
					StorageClassName: diskTypeSpec.StorageClassName,
					VolumeMode:       diskTypeSpec.VolumeMode,
					Resources: corev1.ResourceRequirements{
						Requests: corev1.ResourceList{
							corev1.ResourceStorage: disk.Size,
						},
					},
				},
			},
		}
		input.PatchCollector.Create(dataVolume)
	}

	return nil
}
