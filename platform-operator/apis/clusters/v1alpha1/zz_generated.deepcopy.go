//go:build !ignore_autogenerated
// +build !ignore_autogenerated

// Copyright (c) 2020, 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

// Code generated by controller-gen. DO NOT EDIT.

package v1alpha1

import (
	"k8s.io/apimachinery/pkg/runtime"
)

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *Condition) DeepCopyInto(out *Condition) {
	*out = *in
	if in.LastTransitionTime != nil {
		in, out := &in.LastTransitionTime, &out.LastTransitionTime
		*out = (*in).DeepCopy()
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new Condition.
func (in *Condition) DeepCopy() *Condition {
	if in == nil {
		return nil
	}
	out := new(Condition)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *RancherRegistration) DeepCopyInto(out *RancherRegistration) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new RancherRegistration.
func (in *RancherRegistration) DeepCopy() *RancherRegistration {
	if in == nil {
		return nil
	}
	out := new(RancherRegistration)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *VerrazzanoManagedCluster) DeepCopyInto(out *VerrazzanoManagedCluster) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	out.Spec = in.Spec
	in.Status.DeepCopyInto(&out.Status)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new VerrazzanoManagedCluster.
func (in *VerrazzanoManagedCluster) DeepCopy() *VerrazzanoManagedCluster {
	if in == nil {
		return nil
	}
	out := new(VerrazzanoManagedCluster)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *VerrazzanoManagedCluster) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *VerrazzanoManagedClusterList) DeepCopyInto(out *VerrazzanoManagedClusterList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ListMeta.DeepCopyInto(&out.ListMeta)
	if in.Items != nil {
		in, out := &in.Items, &out.Items
		*out = make([]VerrazzanoManagedCluster, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new VerrazzanoManagedClusterList.
func (in *VerrazzanoManagedClusterList) DeepCopy() *VerrazzanoManagedClusterList {
	if in == nil {
		return nil
	}
	out := new(VerrazzanoManagedClusterList)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *VerrazzanoManagedClusterList) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *VerrazzanoManagedClusterSpec) DeepCopyInto(out *VerrazzanoManagedClusterSpec) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new VerrazzanoManagedClusterSpec.
func (in *VerrazzanoManagedClusterSpec) DeepCopy() *VerrazzanoManagedClusterSpec {
	if in == nil {
		return nil
	}
	out := new(VerrazzanoManagedClusterSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *VerrazzanoManagedClusterStatus) DeepCopyInto(out *VerrazzanoManagedClusterStatus) {
	*out = *in
	if in.Conditions != nil {
		in, out := &in.Conditions, &out.Conditions
		*out = make([]Condition, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
	if in.LastAgentConnectTime != nil {
		in, out := &in.LastAgentConnectTime, &out.LastAgentConnectTime
		*out = (*in).DeepCopy()
	}
	out.RancherRegistration = in.RancherRegistration
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new VerrazzanoManagedClusterStatus.
func (in *VerrazzanoManagedClusterStatus) DeepCopy() *VerrazzanoManagedClusterStatus {
	if in == nil {
		return nil
	}
	out := new(VerrazzanoManagedClusterStatus)
	in.DeepCopyInto(out)
	return out
}
