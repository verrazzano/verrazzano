// +build !ignore_autogenerated

// Copyright (c) 2020, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

// Code generated by controller-gen. DO NOT EDIT.

package v1alpha1

import (
	"k8s.io/api/core/v1"
	runtime "k8s.io/apimachinery/pkg/runtime"
)

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *Acme) DeepCopyInto(out *Acme) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new Acme.
func (in *Acme) DeepCopy() *Acme {
	if in == nil {
		return nil
	}
	out := new(Acme)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *CA) DeepCopyInto(out *CA) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new CA.
func (in *CA) DeepCopy() *CA {
	if in == nil {
		return nil
	}
	out := new(CA)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *CertManagerComponent) DeepCopyInto(out *CertManagerComponent) {
	*out = *in
	out.Certificate = in.Certificate
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new CertManagerComponent.
func (in *CertManagerComponent) DeepCopy() *CertManagerComponent {
	if in == nil {
		return nil
	}
	out := new(CertManagerComponent)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *Certificate) DeepCopyInto(out *Certificate) {
	*out = *in
	out.Acme = in.Acme
	out.CA = in.CA
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new Certificate.
func (in *Certificate) DeepCopy() *Certificate {
	if in == nil {
		return nil
	}
	out := new(Certificate)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ComponentSpec) DeepCopyInto(out *ComponentSpec) {
	*out = *in
	out.CertManager = in.CertManager
	out.DNS = in.DNS
	in.Ingress.DeepCopyInto(&out.Ingress)
	in.Istio.DeepCopyInto(&out.Istio)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ComponentSpec.
func (in *ComponentSpec) DeepCopy() *ComponentSpec {
	if in == nil {
		return nil
	}
	out := new(ComponentSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *Condition) DeepCopyInto(out *Condition) {
	*out = *in
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
func (in *DNSComponent) DeepCopyInto(out *DNSComponent) {
	*out = *in
	out.XIPIO = in.XIPIO
	out.OCI = in.OCI
	out.External = in.External
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new DNSComponent.
func (in *DNSComponent) DeepCopy() *DNSComponent {
	if in == nil {
		return nil
	}
	out := new(DNSComponent)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *External) DeepCopyInto(out *External) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new External.
func (in *External) DeepCopy() *External {
	if in == nil {
		return nil
	}
	out := new(External)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *IngressNginxComponent) DeepCopyInto(out *IngressNginxComponent) {
	*out = *in
	if in.NGINXInstallArgs != nil {
		in, out := &in.NGINXInstallArgs, &out.NGINXInstallArgs
		*out = make([]InstallArgs, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
	if in.Ports != nil {
		in, out := &in.Ports, &out.Ports
		*out = make([]v1.ServicePort, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new IngressNginxComponent.
func (in *IngressNginxComponent) DeepCopy() *IngressNginxComponent {
	if in == nil {
		return nil
	}
	out := new(IngressNginxComponent)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *InstallArgs) DeepCopyInto(out *InstallArgs) {
	*out = *in
	if in.ValueList != nil {
		in, out := &in.ValueList, &out.ValueList
		*out = make([]string, len(*in))
		copy(*out, *in)
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new InstallArgs.
func (in *InstallArgs) DeepCopy() *InstallArgs {
	if in == nil {
		return nil
	}
	out := new(InstallArgs)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *IstioComponent) DeepCopyInto(out *IstioComponent) {
	*out = *in
	if in.IstioInstallArgs != nil {
		in, out := &in.IstioInstallArgs, &out.IstioInstallArgs
		*out = make([]InstallArgs, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new IstioComponent.
func (in *IstioComponent) DeepCopy() *IstioComponent {
	if in == nil {
		return nil
	}
	out := new(IstioComponent)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *OCI) DeepCopyInto(out *OCI) {
	*out = *in
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
func (in *PrivateKeyPassphraseSecretRef) DeepCopyInto(out *PrivateKeyPassphraseSecretRef) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new PrivateKeyPassphraseSecretRef.
func (in *PrivateKeyPassphraseSecretRef) DeepCopy() *PrivateKeyPassphraseSecretRef {
	if in == nil {
		return nil
	}
	out := new(PrivateKeyPassphraseSecretRef)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *Verrazzano) DeepCopyInto(out *Verrazzano) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	in.Spec.DeepCopyInto(&out.Spec)
	in.Status.DeepCopyInto(&out.Status)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new Verrazzano.
func (in *Verrazzano) DeepCopy() *Verrazzano {
	if in == nil {
		return nil
	}
	out := new(Verrazzano)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *Verrazzano) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *VerrazzanoList) DeepCopyInto(out *VerrazzanoList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ListMeta.DeepCopyInto(&out.ListMeta)
	if in.Items != nil {
		in, out := &in.Items, &out.Items
		*out = make([]Verrazzano, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new VerrazzanoList.
func (in *VerrazzanoList) DeepCopy() *VerrazzanoList {
	if in == nil {
		return nil
	}
	out := new(VerrazzanoList)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *VerrazzanoList) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *VerrazzanoSpec) DeepCopyInto(out *VerrazzanoSpec) {
	*out = *in
	in.Components.DeepCopyInto(&out.Components)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new VerrazzanoSpec.
func (in *VerrazzanoSpec) DeepCopy() *VerrazzanoSpec {
	if in == nil {
		return nil
	}
	out := new(VerrazzanoSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *VerrazzanoStatus) DeepCopyInto(out *VerrazzanoStatus) {
	*out = *in
	if in.Conditions != nil {
		in, out := &in.Conditions, &out.Conditions
		*out = make([]Condition, len(*in))
		copy(*out, *in)
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new VerrazzanoStatus.
func (in *VerrazzanoStatus) DeepCopy() *VerrazzanoStatus {
	if in == nil {
		return nil
	}
	out := new(VerrazzanoStatus)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *XIPIO) DeepCopyInto(out *XIPIO) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new XIPIO.
func (in *XIPIO) DeepCopy() *XIPIO {
	if in == nil {
		return nil
	}
	out := new(XIPIO)
	in.DeepCopyInto(out)
	return out
}
