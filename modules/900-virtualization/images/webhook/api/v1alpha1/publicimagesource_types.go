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
	cdiv1 "kubevirt.io/containerized-data-importer-api/pkg/apis/core/v1beta1"
)

// PublicImageSourceStatus defines the observed state of PublicImageSource
type PublicImageSourceStatus struct {
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+kubebuilder:resource:scope=Cluster

// PublicImageSource is the Schema for the publicimagesources API
type PublicImageSource struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   cdiv1.DataVolumeSource  `json:"spec,omitempty"`
	Status PublicImageSourceStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// PublicImageSourceList contains a list of PublicImageSource
type PublicImageSourceList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []PublicImageSource `json:"items"`
}

func init() {
	SchemeBuilder.Register(&PublicImageSource{}, &PublicImageSourceList{})
}
