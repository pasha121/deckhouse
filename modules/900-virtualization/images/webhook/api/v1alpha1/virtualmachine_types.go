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
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k6tv1 "kubevirt.io/api/core/v1"
)

// VirtualMachineSpec defines the desired state of VirtualMachine
type VirtualMachineSpec struct {
	Running                  *bool                         `json:"running,omitempty" optional:"true"`
	IPAddressReservationName *string                       `json:"IPAddressReservationName,omitempty"`
	CloudInit                *k6tv1.CloudInitNoCloudSource `json:"cloudInit,omitempty"`
	Resources                *k6tv1.ResourceRequirements   `json:"resources,omitempty"`
	Disks                    *[]VolumeSource               `json:"disks,omitempty"`
}

// VirtualMachineStatus defines the observed state of VirtualMachine
type VirtualMachineStatus struct {
	// Phase is a human readable, high-level representation of the status of the virtual machine
	Phase k6tv1.VirtualMachinePrintableStatus `json:"phase,omitempty"`
	// NodeName is the name where the VirtualMachineInstance is currently running.
	NodeName string `json:"nodeName,omitempty"`
	// IP address of Virtual Machine
	VMIP string `json:"vmIP,omitempty"`
}

type ClusterImageSourceSource struct {
	// Name represents the name of the ClusterImageSource in the same namespace
	Name string `json:"name"`
	// Hotpluggable indicates whether the volume can be hotplugged and hotunplugged.
	// +optional
	Hotpluggable bool `json:"hotpluggable,omitempty"`
	// Will be used to create a stand-alone PVC to provision the volume.
	VolumeClaimTemplate v1.PersistentVolumeClaim `json:"volumeClaimTemplate,omitempty"`
}

type ImageSourceSource struct {
	// Name represents the name of the ImageSource in the same namespace
	Name string `json:"name"`
	// Hotpluggable indicates whether the volume can be hotplugged and hotunplugged.
	// +optional
	Hotpluggable bool `json:"hotpluggable,omitempty"`
	// Will be used to create a stand-alone PVC to provision the volume.
	VolumeClaimTemplate v1.PersistentVolumeClaim `json:"volumeClaimTemplate,omitempty"`
}

// Represents the source of a volume to mount.
// Only one of its members may be specified.
type VolumeSource struct {
	// Bus indicates the type of disk device to emulate.
	// supported values: virtio, sata, scsi, usb.
	Bus k6tv1.DiskBus `json:"bus,omitempty"`
	// ImageSourceSource represents a reference to a ImageSource in the same namespace.
	// Directly attached to the vmi via qemu.
	// +optional
	ImageSource ClusterImageSourceSource `json:"imageSource,omitempty"`
	// ClusterImageSourceSource represents a reference to a ClusterImageSource.
	// Directly attached to the vmi via qemu.
	// +optional
	ClusterImageSource ClusterImageSourceSource `json:"clusterImageSource,omitempty"`
	// PersistentVolumeClaimVolumeSource represents a reference to a PersistentVolumeClaim in the same namespace.
	// Directly attached to the vmi via qemu.
	// More info: https://kubernetes.io/docs/concepts/storage/persistent-volumes#persistentvolumeclaims
	// +optional
	PersistentVolumeClaim *k6tv1.PersistentVolumeClaimVolumeSource `json:"persistentVolumeClaim,omitempty"`
	// ContainerDisk references a docker image, embedding a qcow or raw disk.
	// More info: https://kubevirt.gitbooks.io/user-guide/registry-disk.html
	// +optional
	ContainerDisk *k6tv1.ContainerDiskSource `json:"containerDisk,omitempty"`
	// Ephemeral is a special volume source that "wraps" specified source and provides copy-on-write image on top of it.
	// +optional
	Ephemeral *k6tv1.EphemeralVolumeSource `json:"ephemeral,omitempty"`
	// EmptyDisk represents a temporary disk which shares the vmis lifecycle.
	// More info: https://kubevirt.gitbooks.io/user-guide/disks-and-volumes.html
	// +optional
	EmptyDisk *k6tv1.EmptyDiskSource `json:"emptyDisk,omitempty"`
	// DataVolume represents the dynamic creation a PVC for this volume as well as
	// the process of populating that PVC with a disk image.
	// +optional
	DataVolume *k6tv1.DataVolumeSource `json:"dataVolume,omitempty"`
	// ConfigMapSource represents a reference to a ConfigMap in the same namespace.
	// More info: https://kubernetes.io/docs/tasks/configure-pod-container/configure-pod-configmap/
	// +optional
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// VirtualMachine is the Schema for the virtualmachines API
type VirtualMachine struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   VirtualMachineSpec   `json:"spec,omitempty"`
	Status VirtualMachineStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// VirtualMachineList contains a list of VirtualMachine
type VirtualMachineList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []VirtualMachine `json:"items"`
}

func init() {
	SchemeBuilder.Register(&VirtualMachine{}, &VirtualMachineList{})
}
