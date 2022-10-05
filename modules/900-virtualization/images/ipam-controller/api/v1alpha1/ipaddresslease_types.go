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

// IPAddressLeaseSpec defines the desired state of IPAddressLease
type IPAddressLeaseSpec struct {
	// IP-address for reservation
	Address string `json:"address,omitempty"`
	// Static represents the static lease
	Static bool `json:"static,omitempty"`
}

// IPAddressLeaseStatus defines the observed state of IPAddressLease
type IPAddressLeaseStatus struct {
	Allocated bool `json:"allocated,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// IPAddressLease is the Schema for the ipaddressleases API
type IPAddressLease struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   IPAddressLeaseSpec   `json:"spec,omitempty"`
	Status IPAddressLeaseStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// IPAddressLeaseList contains a list of IPAddressLease
type IPAddressLeaseList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []IPAddressLease `json:"items"`
}

func init() {
	SchemeBuilder.Register(&IPAddressLease{}, &IPAddressLeaseList{})
}
