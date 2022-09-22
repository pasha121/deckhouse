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

// IPAddressReservationSpec defines the desired state of IPAddressReservation
type IPAddressReservationSpec struct {
	// IP-address for reservation
	Address string `json:"address,omitempty"`
}

// IPAddressReservationStatus defines the observed state of IPAddressReservation
type IPAddressReservationStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// IPAddressReservation is the Schema for the ipaddressreservations API
type IPAddressReservation struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   IPAddressReservationSpec   `json:"spec,omitempty"`
	Status IPAddressReservationStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// IPAddressReservationList contains a list of IPAddressReservation
type IPAddressReservationList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []IPAddressReservation `json:"items"`
}

func init() {
	SchemeBuilder.Register(&IPAddressReservation{}, &IPAddressReservationList{})
}
