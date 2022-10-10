/*
Copyright 2022.

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

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// IPAddressClaimSpec defines the desired state of IPAddressClaim
type IPAddressClaimSpec struct {
	// Static represents the static claim
	Static bool `json:"static,omitempty"`
}

// IPAddressClaimStatus defines the observed state of IPAddressClaim
type IPAddressClaimStatus struct {
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+kubebuilder:resource:scope=Cluster
//+kubebuilder:printcolumn:JSONPath=".spec.static",name=Static,type=string

// IPAddressClaim is the Schema for the ipaddressclaims API
type IPAddressClaim struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   IPAddressClaimSpec   `json:"spec,omitempty"`
	Status IPAddressClaimStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// IPAddressClaimList contains a list of IPAddressClaim
type IPAddressClaimList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []IPAddressClaim `json:"items"`
}

func init() {
	SchemeBuilder.Register(&IPAddressClaim{}, &IPAddressClaimList{})
}
