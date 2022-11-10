//go:build !ignore_autogenerated
// +build !ignore_autogenerated

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

// Code generated by controller-gen. DO NOT EDIT.

package v1alpha1

import (
	"k8s.io/api/core/v1"
	runtime "k8s.io/apimachinery/pkg/runtime"
	corev1 "kubevirt.io/api/core/v1"
	"kubevirt.io/containerized-data-importer-api/pkg/apis/core/v1beta1"
)

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *BootDisk) DeepCopyInto(out *BootDisk) {
	*out = *in
	if in.Source != nil {
		in, out := &in.Source, &out.Source
		*out = new(v1.TypedLocalObjectReference)
		(*in).DeepCopyInto(*out)
	}
	out.Size = in.Size.DeepCopy()
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new BootDisk.
func (in *BootDisk) DeepCopy() *BootDisk {
	if in == nil {
		return nil
	}
	out := new(BootDisk)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ClusterVirtualMachineImage) DeepCopyInto(out *ClusterVirtualMachineImage) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	in.Spec.DeepCopyInto(&out.Spec)
	out.Status = in.Status
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ClusterVirtualMachineImage.
func (in *ClusterVirtualMachineImage) DeepCopy() *ClusterVirtualMachineImage {
	if in == nil {
		return nil
	}
	out := new(ClusterVirtualMachineImage)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *ClusterVirtualMachineImage) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ClusterVirtualMachineImageList) DeepCopyInto(out *ClusterVirtualMachineImageList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ListMeta.DeepCopyInto(&out.ListMeta)
	if in.Items != nil {
		in, out := &in.Items, &out.Items
		*out = make([]ClusterVirtualMachineImage, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ClusterVirtualMachineImageList.
func (in *ClusterVirtualMachineImageList) DeepCopy() *ClusterVirtualMachineImageList {
	if in == nil {
		return nil
	}
	out := new(ClusterVirtualMachineImageList)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *ClusterVirtualMachineImageList) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ClusterVirtualMachineImageSpec) DeepCopyInto(out *ClusterVirtualMachineImageSpec) {
	*out = *in
	if in.Remote != nil {
		in, out := &in.Remote, &out.Remote
		*out = new(v1beta1.DataVolumeSource)
		(*in).DeepCopyInto(*out)
	}
	if in.Source != nil {
		in, out := &in.Source, &out.Source
		*out = new(TypedObjectReference)
		(*in).DeepCopyInto(*out)
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ClusterVirtualMachineImageSpec.
func (in *ClusterVirtualMachineImageSpec) DeepCopy() *ClusterVirtualMachineImageSpec {
	if in == nil {
		return nil
	}
	out := new(ClusterVirtualMachineImageSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ClusterVirtualMachineImageStatus) DeepCopyInto(out *ClusterVirtualMachineImageStatus) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ClusterVirtualMachineImageStatus.
func (in *ClusterVirtualMachineImageStatus) DeepCopy() *ClusterVirtualMachineImageStatus {
	if in == nil {
		return nil
	}
	out := new(ClusterVirtualMachineImageStatus)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *DiskSource) DeepCopyInto(out *DiskSource) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new DiskSource.
func (in *DiskSource) DeepCopy() *DiskSource {
	if in == nil {
		return nil
	}
	out := new(DiskSource)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *TypedObjectReference) DeepCopyInto(out *TypedObjectReference) {
	*out = *in
	in.TypedLocalObjectReference.DeepCopyInto(&out.TypedLocalObjectReference)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new TypedObjectReference.
func (in *TypedObjectReference) DeepCopy() *TypedObjectReference {
	if in == nil {
		return nil
	}
	out := new(TypedObjectReference)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *VirtualMachine) DeepCopyInto(out *VirtualMachine) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	in.Spec.DeepCopyInto(&out.Spec)
	out.Status = in.Status
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new VirtualMachine.
func (in *VirtualMachine) DeepCopy() *VirtualMachine {
	if in == nil {
		return nil
	}
	out := new(VirtualMachine)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *VirtualMachine) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *VirtualMachineDisk) DeepCopyInto(out *VirtualMachineDisk) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	in.Spec.DeepCopyInto(&out.Spec)
	out.Status = in.Status
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new VirtualMachineDisk.
func (in *VirtualMachineDisk) DeepCopy() *VirtualMachineDisk {
	if in == nil {
		return nil
	}
	out := new(VirtualMachineDisk)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *VirtualMachineDisk) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *VirtualMachineDiskList) DeepCopyInto(out *VirtualMachineDiskList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ListMeta.DeepCopyInto(&out.ListMeta)
	if in.Items != nil {
		in, out := &in.Items, &out.Items
		*out = make([]VirtualMachineDisk, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new VirtualMachineDiskList.
func (in *VirtualMachineDiskList) DeepCopy() *VirtualMachineDiskList {
	if in == nil {
		return nil
	}
	out := new(VirtualMachineDiskList)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *VirtualMachineDiskList) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *VirtualMachineDiskSpec) DeepCopyInto(out *VirtualMachineDiskSpec) {
	*out = *in
	out.Size = in.Size.DeepCopy()
	if in.Source != nil {
		in, out := &in.Source, &out.Source
		*out = new(v1.TypedLocalObjectReference)
		(*in).DeepCopyInto(*out)
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new VirtualMachineDiskSpec.
func (in *VirtualMachineDiskSpec) DeepCopy() *VirtualMachineDiskSpec {
	if in == nil {
		return nil
	}
	out := new(VirtualMachineDiskSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *VirtualMachineDiskStatus) DeepCopyInto(out *VirtualMachineDiskStatus) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new VirtualMachineDiskStatus.
func (in *VirtualMachineDiskStatus) DeepCopy() *VirtualMachineDiskStatus {
	if in == nil {
		return nil
	}
	out := new(VirtualMachineDiskStatus)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *VirtualMachineIPAddressLease) DeepCopyInto(out *VirtualMachineIPAddressLease) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	out.Spec = in.Spec
	out.Status = in.Status
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new VirtualMachineIPAddressLease.
func (in *VirtualMachineIPAddressLease) DeepCopy() *VirtualMachineIPAddressLease {
	if in == nil {
		return nil
	}
	out := new(VirtualMachineIPAddressLease)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *VirtualMachineIPAddressLease) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *VirtualMachineIPAddressLeaseList) DeepCopyInto(out *VirtualMachineIPAddressLeaseList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ListMeta.DeepCopyInto(&out.ListMeta)
	if in.Items != nil {
		in, out := &in.Items, &out.Items
		*out = make([]VirtualMachineIPAddressLease, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new VirtualMachineIPAddressLeaseList.
func (in *VirtualMachineIPAddressLeaseList) DeepCopy() *VirtualMachineIPAddressLeaseList {
	if in == nil {
		return nil
	}
	out := new(VirtualMachineIPAddressLeaseList)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *VirtualMachineIPAddressLeaseList) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *VirtualMachineIPAddressLeaseSpec) DeepCopyInto(out *VirtualMachineIPAddressLeaseSpec) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new VirtualMachineIPAddressLeaseSpec.
func (in *VirtualMachineIPAddressLeaseSpec) DeepCopy() *VirtualMachineIPAddressLeaseSpec {
	if in == nil {
		return nil
	}
	out := new(VirtualMachineIPAddressLeaseSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *VirtualMachineIPAddressLeaseStatus) DeepCopyInto(out *VirtualMachineIPAddressLeaseStatus) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new VirtualMachineIPAddressLeaseStatus.
func (in *VirtualMachineIPAddressLeaseStatus) DeepCopy() *VirtualMachineIPAddressLeaseStatus {
	if in == nil {
		return nil
	}
	out := new(VirtualMachineIPAddressLeaseStatus)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *VirtualMachineList) DeepCopyInto(out *VirtualMachineList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ListMeta.DeepCopyInto(&out.ListMeta)
	if in.Items != nil {
		in, out := &in.Items, &out.Items
		*out = make([]VirtualMachine, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new VirtualMachineList.
func (in *VirtualMachineList) DeepCopy() *VirtualMachineList {
	if in == nil {
		return nil
	}
	out := new(VirtualMachineList)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *VirtualMachineList) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *VirtualMachineSpec) DeepCopyInto(out *VirtualMachineSpec) {
	*out = *in
	if in.Running != nil {
		in, out := &in.Running, &out.Running
		*out = new(bool)
		**out = **in
	}
	if in.Resources != nil {
		in, out := &in.Resources, &out.Resources
		*out = make(v1.ResourceList, len(*in))
		for key, val := range *in {
			(*out)[key] = val.DeepCopy()
		}
	}
	in.BootDisk.DeepCopyInto(&out.BootDisk)
	if in.CloudInit != nil {
		in, out := &in.CloudInit, &out.CloudInit
		*out = new(corev1.CloudInitNoCloudSource)
		(*in).DeepCopyInto(*out)
	}
	if in.DiskAttachments != nil {
		in, out := &in.DiskAttachments, &out.DiskAttachments
		*out = new([]DiskSource)
		if **in != nil {
			in, out := *in, *out
			*out = make([]DiskSource, len(*in))
			copy(*out, *in)
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new VirtualMachineSpec.
func (in *VirtualMachineSpec) DeepCopy() *VirtualMachineSpec {
	if in == nil {
		return nil
	}
	out := new(VirtualMachineSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *VirtualMachineStatus) DeepCopyInto(out *VirtualMachineStatus) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new VirtualMachineStatus.
func (in *VirtualMachineStatus) DeepCopy() *VirtualMachineStatus {
	if in == nil {
		return nil
	}
	out := new(VirtualMachineStatus)
	in.DeepCopyInto(out)
	return out
}
