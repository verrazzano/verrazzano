//go:build !ignore_autogenerated
// +build !ignore_autogenerated

// Copyright (c) 2022, 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

// Code generated by controller-gen. DO NOT EDIT.

package v1alpha1

import (
	"k8s.io/apimachinery/pkg/runtime"
)

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ArgoCDRegistration) DeepCopyInto(out *ArgoCDRegistration) {
	*out = *in
	if in.Timestamp != nil {
		in, out := &in.Timestamp, &out.Timestamp
		*out = (*in).DeepCopy()
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ArgoCDRegistration.
func (in *ArgoCDRegistration) DeepCopy() *ArgoCDRegistration {
	if in == nil {
		return nil
	}
	out := new(ArgoCDRegistration)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ClusterNetwork) DeepCopyInto(out *ClusterNetwork) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ClusterNetwork.
func (in *ClusterNetwork) DeepCopy() *ClusterNetwork {
	if in == nil {
		return nil
	}
	out := new(ClusterNetwork)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ClusterReference) DeepCopyInto(out *ClusterReference) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ClusterReference.
func (in *ClusterReference) DeepCopy() *ClusterReference {
	if in == nil {
		return nil
	}
	out := new(ClusterReference)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *CommonClusterSpec) DeepCopyInto(out *CommonClusterSpec) {
	*out = *in
	out.IdentityRef = in.IdentityRef
	if in.PrivateRegistry != nil {
		in, out := &in.PrivateRegistry, &out.PrivateRegistry
		*out = new(PrivateRegistry)
		**out = **in
	}
	if in.Proxy != nil {
		in, out := &in.Proxy, &out.Proxy
		*out = new(Proxy)
		**out = **in
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new CommonClusterSpec.
func (in *CommonClusterSpec) DeepCopy() *CommonClusterSpec {
	if in == nil {
		return nil
	}
	out := new(CommonClusterSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *CommonOCI) DeepCopyInto(out *CommonOCI) {
	*out = *in
	if in.SSHPublicKey != nil {
		in, out := &in.SSHPublicKey, &out.SSHPublicKey
		*out = new(string)
		**out = **in
	}
	if in.CloudInitScript != nil {
		in, out := &in.CloudInitScript, &out.CloudInitScript
		*out = make([]string, len(*in))
		copy(*out, *in)
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new CommonOCI.
func (in *CommonOCI) DeepCopy() *CommonOCI {
	if in == nil {
		return nil
	}
	out := new(CommonOCI)
	in.DeepCopyInto(out)
	return out
}

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
func (in *Kubernetes) DeepCopyInto(out *Kubernetes) {
	*out = *in
	out.KubernetesBase = in.KubernetesBase
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new Kubernetes.
func (in *Kubernetes) DeepCopy() *Kubernetes {
	if in == nil {
		return nil
	}
	out := new(Kubernetes)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *KubernetesBase) DeepCopyInto(out *KubernetesBase) {
	*out = *in
	out.ClusterNetwork = in.ClusterNetwork
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new KubernetesBase.
func (in *KubernetesBase) DeepCopy() *KubernetesBase {
	if in == nil {
		return nil
	}
	out := new(KubernetesBase)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *KubernetesInformation) DeepCopyInto(out *KubernetesInformation) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new KubernetesInformation.
func (in *KubernetesInformation) DeepCopy() *KubernetesInformation {
	if in == nil {
		return nil
	}
	out := new(KubernetesInformation)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *NamedOCINode) DeepCopyInto(out *NamedOCINode) {
	*out = *in
	in.OCINode.DeepCopyInto(&out.OCINode)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new NamedOCINode.
func (in *NamedOCINode) DeepCopy() *NamedOCINode {
	if in == nil {
		return nil
	}
	out := new(NamedOCINode)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *NamespacedRef) DeepCopyInto(out *NamespacedRef) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new NamespacedRef.
func (in *NamespacedRef) DeepCopy() *NamespacedRef {
	if in == nil {
		return nil
	}
	out := new(NamespacedRef)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *Network) DeepCopyInto(out *Network) {
	*out = *in
	if in.Subnets != nil {
		in, out := &in.Subnets, &out.Subnets
		*out = make([]Subnet, len(*in))
		copy(*out, *in)
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new Network.
func (in *Network) DeepCopy() *Network {
	if in == nil {
		return nil
	}
	out := new(Network)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *OCI) DeepCopyInto(out *OCI) {
	*out = *in
	in.CommonOCI.DeepCopyInto(&out.CommonOCI)
	in.ControlPlane.DeepCopyInto(&out.ControlPlane)
	if in.Workers != nil {
		in, out := &in.Workers, &out.Workers
		*out = make([]NamedOCINode, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
	if in.Network != nil {
		in, out := &in.Network, &out.Network
		*out = new(Network)
		(*in).DeepCopyInto(*out)
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new OCI.
func (in *OCI) DeepCopy() *OCI {
	if in == nil {
		return nil
	}
	out := new(OCI)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *OCINode) DeepCopyInto(out *OCINode) {
	*out = *in
	if in.Shape != nil {
		in, out := &in.Shape, &out.Shape
		*out = new(string)
		**out = **in
	}
	if in.OCPUs != nil {
		in, out := &in.OCPUs, &out.OCPUs
		*out = new(int)
		**out = **in
	}
	if in.MemoryGbs != nil {
		in, out := &in.MemoryGbs, &out.MemoryGbs
		*out = new(int)
		**out = **in
	}
	if in.BootVolumeGbs != nil {
		in, out := &in.BootVolumeGbs, &out.BootVolumeGbs
		*out = new(int)
		**out = **in
	}
	if in.Replicas != nil {
		in, out := &in.Replicas, &out.Replicas
		*out = new(int)
		**out = **in
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new OCINode.
func (in *OCINode) DeepCopy() *OCINode {
	if in == nil {
		return nil
	}
	out := new(OCINode)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *OCIOCNEClusterSpec) DeepCopyInto(out *OCIOCNEClusterSpec) {
	*out = *in
	in.CommonClusterSpec.DeepCopyInto(&out.CommonClusterSpec)
	out.KubernetesBase = in.KubernetesBase
	out.OCNE = in.OCNE
	in.OCI.DeepCopyInto(&out.OCI)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new OCIOCNEClusterSpec.
func (in *OCIOCNEClusterSpec) DeepCopy() *OCIOCNEClusterSpec {
	if in == nil {
		return nil
	}
	out := new(OCIOCNEClusterSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *OCNE) DeepCopyInto(out *OCNE) {
	*out = *in
	out.Dependencies = in.Dependencies
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new OCNE.
func (in *OCNE) DeepCopy() *OCNE {
	if in == nil {
		return nil
	}
	out := new(OCNE)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *OCNEDependencies) DeepCopyInto(out *OCNEDependencies) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new OCNEDependencies.
func (in *OCNEDependencies) DeepCopy() *OCNEDependencies {
	if in == nil {
		return nil
	}
	out := new(OCNEDependencies)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *OCNEOCIQuickCreate) DeepCopyInto(out *OCNEOCIQuickCreate) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	in.Spec.DeepCopyInto(&out.Spec)
	out.Status = in.Status
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new OCNEOCIQuickCreate.
func (in *OCNEOCIQuickCreate) DeepCopy() *OCNEOCIQuickCreate {
	if in == nil {
		return nil
	}
	out := new(OCNEOCIQuickCreate)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *OCNEOCIQuickCreate) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *OCNEOCIQuickCreateList) DeepCopyInto(out *OCNEOCIQuickCreateList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ListMeta.DeepCopyInto(&out.ListMeta)
	if in.Items != nil {
		in, out := &in.Items, &out.Items
		*out = make([]OCNEOCIQuickCreate, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new OCNEOCIQuickCreateList.
func (in *OCNEOCIQuickCreateList) DeepCopy() *OCNEOCIQuickCreateList {
	if in == nil {
		return nil
	}
	out := new(OCNEOCIQuickCreateList)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *OCNEOCIQuickCreateList) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *OCNEOCIQuickCreateStatus) DeepCopyInto(out *OCNEOCIQuickCreateStatus) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new OCNEOCIQuickCreateStatus.
func (in *OCNEOCIQuickCreateStatus) DeepCopy() *OCNEOCIQuickCreateStatus {
	if in == nil {
		return nil
	}
	out := new(OCNEOCIQuickCreateStatus)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *OKE) DeepCopyInto(out *OKE) {
	*out = *in
	in.CommonOCI.DeepCopyInto(&out.CommonOCI)
	if in.NodePools != nil {
		in, out := &in.NodePools, &out.NodePools
		*out = make([]NamedOCINode, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
	if in.VirtualNodePools != nil {
		in, out := &in.VirtualNodePools, &out.VirtualNodePools
		*out = make([]VirtualNodePool, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
	if in.Network != nil {
		in, out := &in.Network, &out.Network
		*out = new(OKENetwork)
		(*in).DeepCopyInto(*out)
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new OKE.
func (in *OKE) DeepCopy() *OKE {
	if in == nil {
		return nil
	}
	out := new(OKE)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *OKENetwork) DeepCopyInto(out *OKENetwork) {
	*out = *in
	if in.Config != nil {
		in, out := &in.Config, &out.Config
		*out = new(Network)
		(*in).DeepCopyInto(*out)
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new OKENetwork.
func (in *OKENetwork) DeepCopy() *OKENetwork {
	if in == nil {
		return nil
	}
	out := new(OKENetwork)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *OKEQuickCreate) DeepCopyInto(out *OKEQuickCreate) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	in.Spec.DeepCopyInto(&out.Spec)
	out.Status = in.Status
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new OKEQuickCreate.
func (in *OKEQuickCreate) DeepCopy() *OKEQuickCreate {
	if in == nil {
		return nil
	}
	out := new(OKEQuickCreate)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *OKEQuickCreate) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *OKEQuickCreateList) DeepCopyInto(out *OKEQuickCreateList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ListMeta.DeepCopyInto(&out.ListMeta)
	if in.Items != nil {
		in, out := &in.Items, &out.Items
		*out = make([]OKEQuickCreate, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new OKEQuickCreateList.
func (in *OKEQuickCreateList) DeepCopy() *OKEQuickCreateList {
	if in == nil {
		return nil
	}
	out := new(OKEQuickCreateList)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *OKEQuickCreateList) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *OKEQuickCreateSpec) DeepCopyInto(out *OKEQuickCreateSpec) {
	*out = *in
	out.IdentityRef = in.IdentityRef
	out.Kubernetes = in.Kubernetes
	in.OKE.DeepCopyInto(&out.OKE)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new OKEQuickCreateSpec.
func (in *OKEQuickCreateSpec) DeepCopy() *OKEQuickCreateSpec {
	if in == nil {
		return nil
	}
	out := new(OKEQuickCreateSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *OKEQuickCreateStatus) DeepCopyInto(out *OKEQuickCreateStatus) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new OKEQuickCreateStatus.
func (in *OKEQuickCreateStatus) DeepCopy() *OKEQuickCreateStatus {
	if in == nil {
		return nil
	}
	out := new(OKEQuickCreateStatus)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *PrivateRegistry) DeepCopyInto(out *PrivateRegistry) {
	*out = *in
	out.CredentialsSecret = in.CredentialsSecret
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new PrivateRegistry.
func (in *PrivateRegistry) DeepCopy() *PrivateRegistry {
	if in == nil {
		return nil
	}
	out := new(PrivateRegistry)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *Proxy) DeepCopyInto(out *Proxy) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new Proxy.
func (in *Proxy) DeepCopy() *Proxy {
	if in == nil {
		return nil
	}
	out := new(Proxy)
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
func (in *Subnet) DeepCopyInto(out *Subnet) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new Subnet.
func (in *Subnet) DeepCopy() *Subnet {
	if in == nil {
		return nil
	}
	out := new(Subnet)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *VerrazzanoInformation) DeepCopyInto(out *VerrazzanoInformation) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new VerrazzanoInformation.
func (in *VerrazzanoInformation) DeepCopy() *VerrazzanoInformation {
	if in == nil {
		return nil
	}
	out := new(VerrazzanoInformation)
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
	in.ArgoCDRegistration.DeepCopyInto(&out.ArgoCDRegistration)
	out.Kubernetes = in.Kubernetes
	out.Verrazzano = in.Verrazzano
	out.ClusterRef = in.ClusterRef
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

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *VirtualNodePool) DeepCopyInto(out *VirtualNodePool) {
	*out = *in
	if in.Replicas != nil {
		in, out := &in.Replicas, &out.Replicas
		*out = new(int)
		**out = **in
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new VirtualNodePool.
func (in *VirtualNodePool) DeepCopy() *VirtualNodePool {
	if in == nil {
		return nil
	}
	out := new(VirtualNodePool)
	in.DeepCopyInto(out)
	return out
}
