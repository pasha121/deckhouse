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
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	cdiv1 "kubevirt.io/containerized-data-importer-api/pkg/apis/core/v1beta1"

	"github.com/deckhouse/deckhouse/modules/900-virtualization/api/v1alpha1"
)

const (
	diskTypesSnapshot          = "disktype"
	disksSnapshot              = "disk"
	publicImageSourcesSnapshot = "publicimagesource"
	dataVolumeSnapshot         = "datavolume"
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
}

type DiskSnapshot struct {
	Name      string
	Namespace string
}

type PublicImageSourceSnapshot struct {
	Name      string
	Namespace string
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
	}, nil
}

func applyDiskFilter(obj *unstructured.Unstructured) (go_hook.FilterResult, error) {
	disk := &v1alpha1.Disk{}
	err := sdk.FromUnstructured(obj, disk)
	if err != nil {
		return nil, fmt.Errorf("cannot convert object to Disk: %v", err)
	}

	return &VirtualMachineSnapshot{
		Name:      disk.Name,
		Namespace: disk.Namespace,
	}, nil
}

func applyPublicImageSourceFilter(obj *unstructured.Unstructured) (go_hook.FilterResult, error) {
	volume := &v1alpha1.PublicImageSource{}
	err := sdk.FromUnstructured(obj, volume)
	if err != nil {
		return nil, fmt.Errorf("cannot convert object to DataVolume: %v", err)
	}

	return &DataVolumeSnapshot{
		Name:      volume.Name,
		Namespace: volume.Namespace,
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
	return nil
}
