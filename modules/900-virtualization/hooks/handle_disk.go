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
	"github.com/deckhouse/deckhouse/modules/900-virtualization/api/v1alpha1"
	"github.com/flant/addon-operator/pkg/module_manager/go_hook"
	"github.com/flant/addon-operator/sdk"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	cdiv1 "kubevirt.io/containerized-data-importer-api/pkg/apis/core/v1beta1"
)

const (
	d8VMSnapshot = "d8vm"
)

var _ = sdk.RegisterFunc(&go_hook.HookConfig{
	Queue: "/modules/virtualization/disks-handler",
	Kubernetes: []go_hook.KubernetesConfig{
		{
			Name:       vmisSnapshot,
			ApiVersion: "deckhouse.io/v1alpha1",
			Kind:       "Disk",
			FilterFunc: applyDisksFilter,
		},
	},
}, handleDisks)

func applyDisksFilter(obj *unstructured.Unstructured) (go_hook.FilterResult, error) {
	return obj, nil
}

// handleDisks
//
// synopsis:
//   TODO
func handleDisks(input *go_hook.HookInput) error {
	d8VMSnap := input.Snapshots[d8VMSnapshot]
	if len(d8VMSnap) == 0 {
		input.LogEntry.Warnln("VirtualMachine not found. Skip")
		return nil
	}

	for _, sRaw := range input.Snapshots[d8VMSnapshot] {
		disk := sRaw.(v1alpha1.Disk)
		datavolume := cdiv1.DataVolume{
			ObjectMeta: disk.ObjectMeta,
		}

		datavolume.Spec.PVC.Resources.Requests[v1.ResourceStorage] = disk.Spec.Size

		if disk.Spec.Source.Name != "" {
			// TODO: read ImageSource
			var imageSource v1alpha1.PublicImageSource
			datavolume.Spec.Source = &imageSource.Spec
		}

		if disk.Spec.Type != "" {
			// TODO: read DiskType
			var diskType v1alpha1.DiskType
			datavolume.Spec.PVC.AccessModes = diskType.Spec.AccessModes
			datavolume.Spec.PVC.VolumeMode = diskType.Spec.VolumeMode
			datavolume.Spec.Storage.StorageClassName = diskType.Spec.StorageClassName
		}

	}
	return nil
}
