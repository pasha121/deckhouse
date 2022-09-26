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

// ClusterImageSourceSpec defines the desired state of ClusterImageSource
type ClusterImageSourceSpec struct {
	// Source is string used to import Virtual Machine Image
	Source string `json:"source,omitempty"`
}

// ClusterImageSourceStatus defines the observed state of ClusterImageSource
type ClusterImageSourceStatus struct {
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+kubebuilder:resource:scope=Cluster

// ClusterImageSource is the Schema for the clusterimagesources API
type ClusterImageSource struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ClusterImageSourceSpec   `json:"spec,omitempty"`
	Status ClusterImageSourceStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// ClusterImageSourceList contains a list of ClusterImageSource
type ClusterImageSourceList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ClusterImageSource `json:"items"`
}

func init() {
	SchemeBuilder.Register(&ClusterImageSource{}, &ClusterImageSourceList{})
}
