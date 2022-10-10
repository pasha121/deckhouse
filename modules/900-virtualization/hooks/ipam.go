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
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

const (
	leasesSnapshot = "iplease"
	claimsSnapshot = "ipclaim"
)

var _ = sdk.RegisterFunc(&go_hook.HookConfig{
	Queue: "/modules/virtualization/disks-handler",
	Kubernetes: []go_hook.KubernetesConfig{
		{
			Name:       leasesSnapshot,
			ApiVersion: "deckhouse.io/v1alpha1",
			Kind:       "IPAdressLease",
			FilterFunc: applyIPAMFilter,
		},
		{
			Name:       claimsSnapshot,
			ApiVersion: "deckhouse.io/v1alpha1",
			Kind:       "IPAdressLease",
			FilterFunc: applyIPAMFilter,
		},
	},
}, handleIPAM)

func applyIPAMFilter(obj *unstructured.Unstructured) (go_hook.FilterResult, error) {
	return obj, nil
}

// handleDisks
//
// synopsis:
//   TODO
func handleIPAM(input *go_hook.HookInput) error {
	leaseSnap := input.Snapshots[leasesSnapshot]
	if len(leaseSnap) == 0 {
		input.LogEntry.Warnln("IPAdressLease not found. Skip")
		return nil
	}

	claimSnap := input.Snapshots[claimsSnapshot]
	if len(claimSnap) == 0 {
		input.LogEntry.Warnln("IPAdressClaim not found. Skip")
		return nil
	}

	allocatedIPs := make(map[string]struct{})
	releasedIPs := make(map[string]struct{})

LEASES_LOOP:
	for _, sRaw := range input.Snapshots[leasesSnapshot] {
		lease := sRaw.(v1alpha1.IPAddressLease)
		for _, dRaw := range input.Snapshots[claimsSnapshot] {
			claim := dRaw.(v1alpha1.IPAddressClaim)
			if claim.GetName() == lease.GetName() { // TODO: add namespace matching
				allocatedIPs[lease.GetName()] = struct{}{}
				continue LEASES_LOOP
			}
		}
		delete(allocatedIPs, lease.GetName())
	}

	// Release free leases
	for name := range releasedIPs {
		// TODO: append patch for delete Lease
		delete(allocatedIPs, name)
	}

	// Allocate new leases
	for _, sRaw := range input.Snapshots[claimsSnapshot] {
		claim := sRaw.(v1alpha1.IPAddressClaim)
		if _, ok := allocatedIPs[claim.GetName()]; !ok {
			// TODO: append patch for creating lease
			allocatedIPs[claim.GetName()] = struct{}{}
		}
	}

	return nil
}
