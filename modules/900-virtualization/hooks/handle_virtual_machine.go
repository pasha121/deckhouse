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

	v1alpha1 "github.com/deckhouse/deckhouse/modules/900-virtualization/api/v1alpha1"
)

const (
	d8VMSnapshot = "d8vm"
)

var _ = sdk.RegisterFunc(&go_hook.HookConfig{
	Queue: "/modules/virtualization/d8vm-handler",
	Kubernetes: []go_hook.KubernetesConfig{
		{
			Name:       vmisSnapshot,
			ApiVersion: "deckhouse.io/v1alpha1",
			Kind:       "VirtualMachine",
			FilterFunc: applyD8VMFilter,
		},
	},
}, handleD8VM)

func applyD8VMFilter(obj *unstructured.Unstructured) (go_hook.FilterResult, error) {
	return obj, nil
}

// handleD8VM
//
// synopsis:
//   TODO
func handleD8VM(input *go_hook.HookInput) error {
	d8VMSnap := input.Snapshots[d8VMSnapshot]
	if len(d8VMSnap) == 0 {
		input.LogEntry.Warnln("VirtualMachine not found. Skip")
		return nil
	}

	for _, sRaw := range input.Snapshots[d8VMSnapshot] {
		d8vm := sRaw.(v1alpha1.VirtualMachine)
		requestedIP := *d8vm.Spec.StaticIPAddress

		if isIPAllocated(requestedIP) {
			ip, err := ensureIPAddressLease(requestedIP)
			if err != nil {
				return err
			}
			if !isIPUnused(requestedIP) {
				return fmt.Errorf("requested ip is in use by other vm or reservation")
			}
		}
		createBootDiskForVM()
		createKubeVirtVM()
	}
	return nil
}

func allocateIP(ip string) (string, error) {
}

func releaseIP(ip string) {
}

func ensureIPAddressLease(requestedIP string) (string, error) {
	static := (requestedIP != "")
	ip, err := allocateIP(ip)
	if err != nil {
		releaseIP(ip)
	}
	return ip, err
}
