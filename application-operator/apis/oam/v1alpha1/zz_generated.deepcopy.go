//go:build !ignore_autogenerated
// +build !ignore_autogenerated

// Copyright (c) 2021, 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

// Code generated by controller-gen. DO NOT EDIT.

package v1alpha1

import (
	corev1alpha1 "github.com/crossplane/crossplane-runtime/apis/core/v1alpha1"
	"k8s.io/apimachinery/pkg/runtime"
)

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *AuthorizationPolicy) DeepCopyInto(out *AuthorizationPolicy) {
	*out = *in
	if in.Rules != nil {
		in, out := &in.Rules, &out.Rules
		*out = make([]*AuthorizationRule, len(*in))
		for i := range *in {
			if (*in)[i] != nil {
				in, out := &(*in)[i], &(*out)[i]
				*out = new(AuthorizationRule)
				(*in).DeepCopyInto(*out)
			}
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new AuthorizationPolicy.
func (in *AuthorizationPolicy) DeepCopy() *AuthorizationPolicy {
	if in == nil {
		return nil
	}
	out := new(AuthorizationPolicy)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *AuthorizationRule) DeepCopyInto(out *AuthorizationRule) {
	*out = *in
	if in.From != nil {
		in, out := &in.From, &out.From
		*out = new(AuthorizationRuleFrom)
		(*in).DeepCopyInto(*out)
	}
	if in.When != nil {
		in, out := &in.When, &out.When
		*out = make([]*AuthorizationRuleCondition, len(*in))
		for i := range *in {
			if (*in)[i] != nil {
				in, out := &(*in)[i], &(*out)[i]
				*out = new(AuthorizationRuleCondition)
				(*in).DeepCopyInto(*out)
			}
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new AuthorizationRule.
func (in *AuthorizationRule) DeepCopy() *AuthorizationRule {
	if in == nil {
		return nil
	}
	out := new(AuthorizationRule)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *AuthorizationRuleCondition) DeepCopyInto(out *AuthorizationRuleCondition) {
	*out = *in
	if in.Values != nil {
		in, out := &in.Values, &out.Values
		*out = make([]string, len(*in))
		copy(*out, *in)
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new AuthorizationRuleCondition.
func (in *AuthorizationRuleCondition) DeepCopy() *AuthorizationRuleCondition {
	if in == nil {
		return nil
	}
	out := new(AuthorizationRuleCondition)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *AuthorizationRuleFrom) DeepCopyInto(out *AuthorizationRuleFrom) {
	*out = *in
	if in.RequestPrincipals != nil {
		in, out := &in.RequestPrincipals, &out.RequestPrincipals
		*out = make([]string, len(*in))
		copy(*out, *in)
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new AuthorizationRuleFrom.
func (in *AuthorizationRuleFrom) DeepCopy() *AuthorizationRuleFrom {
	if in == nil {
		return nil
	}
	out := new(AuthorizationRuleFrom)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *DeploymentTemplate) DeepCopyInto(out *DeploymentTemplate) {
	*out = *in
	in.Metadata.DeepCopyInto(&out.Metadata)
	in.PodSpec.DeepCopyInto(&out.PodSpec)
	in.Selector.DeepCopyInto(&out.Selector)
	in.Strategy.DeepCopyInto(&out.Strategy)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new DeploymentTemplate.
func (in *DeploymentTemplate) DeepCopy() *DeploymentTemplate {
	if in == nil {
		return nil
	}
	out := new(DeploymentTemplate)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *IngressDestination) DeepCopyInto(out *IngressDestination) {
	*out = *in
	if in.HTTPCookie != nil {
		in, out := &in.HTTPCookie, &out.HTTPCookie
		*out = new(IngressDestinationHTTPCookie)
		**out = **in
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new IngressDestination.
func (in *IngressDestination) DeepCopy() *IngressDestination {
	if in == nil {
		return nil
	}
	out := new(IngressDestination)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *IngressDestinationHTTPCookie) DeepCopyInto(out *IngressDestinationHTTPCookie) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new IngressDestinationHTTPCookie.
func (in *IngressDestinationHTTPCookie) DeepCopy() *IngressDestinationHTTPCookie {
	if in == nil {
		return nil
	}
	out := new(IngressDestinationHTTPCookie)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *IngressPath) DeepCopyInto(out *IngressPath) {
	*out = *in
	if in.Policy != nil {
		in, out := &in.Policy, &out.Policy
		*out = new(AuthorizationPolicy)
		(*in).DeepCopyInto(*out)
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new IngressPath.
func (in *IngressPath) DeepCopy() *IngressPath {
	if in == nil {
		return nil
	}
	out := new(IngressPath)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *IngressRule) DeepCopyInto(out *IngressRule) {
	*out = *in
	in.Destination.DeepCopyInto(&out.Destination)
	if in.Hosts != nil {
		in, out := &in.Hosts, &out.Hosts
		*out = make([]string, len(*in))
		copy(*out, *in)
	}
	if in.Paths != nil {
		in, out := &in.Paths, &out.Paths
		*out = make([]IngressPath, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new IngressRule.
func (in *IngressRule) DeepCopy() *IngressRule {
	if in == nil {
		return nil
	}
	out := new(IngressRule)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *IngressSecurity) DeepCopyInto(out *IngressSecurity) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new IngressSecurity.
func (in *IngressSecurity) DeepCopy() *IngressSecurity {
	if in == nil {
		return nil
	}
	out := new(IngressSecurity)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *IngressTrait) DeepCopyInto(out *IngressTrait) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	in.Spec.DeepCopyInto(&out.Spec)
	in.Status.DeepCopyInto(&out.Status)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new IngressTrait.
func (in *IngressTrait) DeepCopy() *IngressTrait {
	if in == nil {
		return nil
	}
	out := new(IngressTrait)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *IngressTrait) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *IngressTraitList) DeepCopyInto(out *IngressTraitList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ListMeta.DeepCopyInto(&out.ListMeta)
	if in.Items != nil {
		in, out := &in.Items, &out.Items
		*out = make([]IngressTrait, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new IngressTraitList.
func (in *IngressTraitList) DeepCopy() *IngressTraitList {
	if in == nil {
		return nil
	}
	out := new(IngressTraitList)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *IngressTraitList) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *IngressTraitSpec) DeepCopyInto(out *IngressTraitSpec) {
	*out = *in
	if in.Rules != nil {
		in, out := &in.Rules, &out.Rules
		*out = make([]IngressRule, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
	out.TLS = in.TLS
	out.WorkloadReference = in.WorkloadReference
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new IngressTraitSpec.
func (in *IngressTraitSpec) DeepCopy() *IngressTraitSpec {
	if in == nil {
		return nil
	}
	out := new(IngressTraitSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *IngressTraitStatus) DeepCopyInto(out *IngressTraitStatus) {
	*out = *in
	in.ConditionedStatus.DeepCopyInto(&out.ConditionedStatus)
	if in.Resources != nil {
		in, out := &in.Resources, &out.Resources
		*out = make([]corev1alpha1.TypedReference, len(*in))
		copy(*out, *in)
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new IngressTraitStatus.
func (in *IngressTraitStatus) DeepCopy() *IngressTraitStatus {
	if in == nil {
		return nil
	}
	out := new(IngressTraitStatus)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *LoggingTrait) DeepCopyInto(out *LoggingTrait) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	out.Spec = in.Spec
	in.Status.DeepCopyInto(&out.Status)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new LoggingTrait.
func (in *LoggingTrait) DeepCopy() *LoggingTrait {
	if in == nil {
		return nil
	}
	out := new(LoggingTrait)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *LoggingTrait) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *LoggingTraitList) DeepCopyInto(out *LoggingTraitList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ListMeta.DeepCopyInto(&out.ListMeta)
	if in.Items != nil {
		in, out := &in.Items, &out.Items
		*out = make([]LoggingTrait, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new LoggingTraitList.
func (in *LoggingTraitList) DeepCopy() *LoggingTraitList {
	if in == nil {
		return nil
	}
	out := new(LoggingTraitList)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *LoggingTraitList) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *LoggingTraitSpec) DeepCopyInto(out *LoggingTraitSpec) {
	*out = *in
	out.WorkloadReference = in.WorkloadReference
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new LoggingTraitSpec.
func (in *LoggingTraitSpec) DeepCopy() *LoggingTraitSpec {
	if in == nil {
		return nil
	}
	out := new(LoggingTraitSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *LoggingTraitStatus) DeepCopyInto(out *LoggingTraitStatus) {
	*out = *in
	in.ConditionedStatus.DeepCopyInto(&out.ConditionedStatus)
	if in.Resources != nil {
		in, out := &in.Resources, &out.Resources
		*out = make([]corev1alpha1.TypedReference, len(*in))
		copy(*out, *in)
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new LoggingTraitStatus.
func (in *LoggingTraitStatus) DeepCopy() *LoggingTraitStatus {
	if in == nil {
		return nil
	}
	out := new(LoggingTraitStatus)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *MetricsTrait) DeepCopyInto(out *MetricsTrait) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	in.Spec.DeepCopyInto(&out.Spec)
	in.Status.DeepCopyInto(&out.Status)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new MetricsTrait.
func (in *MetricsTrait) DeepCopy() *MetricsTrait {
	if in == nil {
		return nil
	}
	out := new(MetricsTrait)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *MetricsTrait) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *MetricsTraitList) DeepCopyInto(out *MetricsTraitList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ListMeta.DeepCopyInto(&out.ListMeta)
	if in.Items != nil {
		in, out := &in.Items, &out.Items
		*out = make([]MetricsTrait, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new MetricsTraitList.
func (in *MetricsTraitList) DeepCopy() *MetricsTraitList {
	if in == nil {
		return nil
	}
	out := new(MetricsTraitList)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *MetricsTraitList) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *MetricsTraitSpec) DeepCopyInto(out *MetricsTraitSpec) {
	*out = *in
	if in.Enabled != nil {
		in, out := &in.Enabled, &out.Enabled
		*out = new(bool)
		**out = **in
	}
	if in.Path != nil {
		in, out := &in.Path, &out.Path
		*out = new(string)
		**out = **in
	}
	if in.Port != nil {
		in, out := &in.Port, &out.Port
		*out = new(int)
		**out = **in
	}
	if in.Ports != nil {
		in, out := &in.Ports, &out.Ports
		*out = make([]PortSpec, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
	if in.Scraper != nil {
		in, out := &in.Scraper, &out.Scraper
		*out = new(string)
		**out = **in
	}
	if in.Secret != nil {
		in, out := &in.Secret, &out.Secret
		*out = new(string)
		**out = **in
	}
	out.WorkloadReference = in.WorkloadReference
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new MetricsTraitSpec.
func (in *MetricsTraitSpec) DeepCopy() *MetricsTraitSpec {
	if in == nil {
		return nil
	}
	out := new(MetricsTraitSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *MetricsTraitStatus) DeepCopyInto(out *MetricsTraitStatus) {
	*out = *in
	in.ConditionedStatus.DeepCopyInto(&out.ConditionedStatus)
	if in.Resources != nil {
		in, out := &in.Resources, &out.Resources
		*out = make([]QualifiedResourceRelation, len(*in))
		copy(*out, *in)
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new MetricsTraitStatus.
func (in *MetricsTraitStatus) DeepCopy() *MetricsTraitStatus {
	if in == nil {
		return nil
	}
	out := new(MetricsTraitStatus)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *PortSpec) DeepCopyInto(out *PortSpec) {
	*out = *in
	if in.Path != nil {
		in, out := &in.Path, &out.Path
		*out = new(string)
		**out = **in
	}
	if in.Port != nil {
		in, out := &in.Port, &out.Port
		*out = new(int)
		**out = **in
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new PortSpec.
func (in *PortSpec) DeepCopy() *PortSpec {
	if in == nil {
		return nil
	}
	out := new(PortSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *QualifiedResourceRelation) DeepCopyInto(out *QualifiedResourceRelation) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new QualifiedResourceRelation.
func (in *QualifiedResourceRelation) DeepCopy() *QualifiedResourceRelation {
	if in == nil {
		return nil
	}
	out := new(QualifiedResourceRelation)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *VerrazzanoCoherenceWorkload) DeepCopyInto(out *VerrazzanoCoherenceWorkload) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	in.Spec.DeepCopyInto(&out.Spec)
	out.Status = in.Status
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new VerrazzanoCoherenceWorkload.
func (in *VerrazzanoCoherenceWorkload) DeepCopy() *VerrazzanoCoherenceWorkload {
	if in == nil {
		return nil
	}
	out := new(VerrazzanoCoherenceWorkload)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *VerrazzanoCoherenceWorkload) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *VerrazzanoCoherenceWorkloadList) DeepCopyInto(out *VerrazzanoCoherenceWorkloadList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ListMeta.DeepCopyInto(&out.ListMeta)
	if in.Items != nil {
		in, out := &in.Items, &out.Items
		*out = make([]VerrazzanoCoherenceWorkload, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new VerrazzanoCoherenceWorkloadList.
func (in *VerrazzanoCoherenceWorkloadList) DeepCopy() *VerrazzanoCoherenceWorkloadList {
	if in == nil {
		return nil
	}
	out := new(VerrazzanoCoherenceWorkloadList)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *VerrazzanoCoherenceWorkloadList) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *VerrazzanoCoherenceWorkloadSpec) DeepCopyInto(out *VerrazzanoCoherenceWorkloadSpec) {
	*out = *in
	in.Template.DeepCopyInto(&out.Template)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new VerrazzanoCoherenceWorkloadSpec.
func (in *VerrazzanoCoherenceWorkloadSpec) DeepCopy() *VerrazzanoCoherenceWorkloadSpec {
	if in == nil {
		return nil
	}
	out := new(VerrazzanoCoherenceWorkloadSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *VerrazzanoCoherenceWorkloadStatus) DeepCopyInto(out *VerrazzanoCoherenceWorkloadStatus) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new VerrazzanoCoherenceWorkloadStatus.
func (in *VerrazzanoCoherenceWorkloadStatus) DeepCopy() *VerrazzanoCoherenceWorkloadStatus {
	if in == nil {
		return nil
	}
	out := new(VerrazzanoCoherenceWorkloadStatus)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *VerrazzanoHelidonWorkload) DeepCopyInto(out *VerrazzanoHelidonWorkload) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	in.Spec.DeepCopyInto(&out.Spec)
	in.Status.DeepCopyInto(&out.Status)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new VerrazzanoHelidonWorkload.
func (in *VerrazzanoHelidonWorkload) DeepCopy() *VerrazzanoHelidonWorkload {
	if in == nil {
		return nil
	}
	out := new(VerrazzanoHelidonWorkload)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *VerrazzanoHelidonWorkload) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *VerrazzanoHelidonWorkloadList) DeepCopyInto(out *VerrazzanoHelidonWorkloadList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ListMeta.DeepCopyInto(&out.ListMeta)
	if in.Items != nil {
		in, out := &in.Items, &out.Items
		*out = make([]VerrazzanoHelidonWorkload, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new VerrazzanoHelidonWorkloadList.
func (in *VerrazzanoHelidonWorkloadList) DeepCopy() *VerrazzanoHelidonWorkloadList {
	if in == nil {
		return nil
	}
	out := new(VerrazzanoHelidonWorkloadList)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *VerrazzanoHelidonWorkloadList) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *VerrazzanoHelidonWorkloadSpec) DeepCopyInto(out *VerrazzanoHelidonWorkloadSpec) {
	*out = *in
	in.DeploymentTemplate.DeepCopyInto(&out.DeploymentTemplate)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new VerrazzanoHelidonWorkloadSpec.
func (in *VerrazzanoHelidonWorkloadSpec) DeepCopy() *VerrazzanoHelidonWorkloadSpec {
	if in == nil {
		return nil
	}
	out := new(VerrazzanoHelidonWorkloadSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *VerrazzanoHelidonWorkloadStatus) DeepCopyInto(out *VerrazzanoHelidonWorkloadStatus) {
	*out = *in
	in.ConditionedStatus.DeepCopyInto(&out.ConditionedStatus)
	if in.Resources != nil {
		in, out := &in.Resources, &out.Resources
		*out = make([]QualifiedResourceRelation, len(*in))
		copy(*out, *in)
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new VerrazzanoHelidonWorkloadStatus.
func (in *VerrazzanoHelidonWorkloadStatus) DeepCopy() *VerrazzanoHelidonWorkloadStatus {
	if in == nil {
		return nil
	}
	out := new(VerrazzanoHelidonWorkloadStatus)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *VerrazzanoWebLogicWorkload) DeepCopyInto(out *VerrazzanoWebLogicWorkload) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	in.Spec.DeepCopyInto(&out.Spec)
	out.Status = in.Status
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new VerrazzanoWebLogicWorkload.
func (in *VerrazzanoWebLogicWorkload) DeepCopy() *VerrazzanoWebLogicWorkload {
	if in == nil {
		return nil
	}
	out := new(VerrazzanoWebLogicWorkload)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *VerrazzanoWebLogicWorkload) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *VerrazzanoWebLogicWorkloadList) DeepCopyInto(out *VerrazzanoWebLogicWorkloadList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ListMeta.DeepCopyInto(&out.ListMeta)
	if in.Items != nil {
		in, out := &in.Items, &out.Items
		*out = make([]VerrazzanoWebLogicWorkload, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new VerrazzanoWebLogicWorkloadList.
func (in *VerrazzanoWebLogicWorkloadList) DeepCopy() *VerrazzanoWebLogicWorkloadList {
	if in == nil {
		return nil
	}
	out := new(VerrazzanoWebLogicWorkloadList)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *VerrazzanoWebLogicWorkloadList) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *VerrazzanoWebLogicWorkloadSpec) DeepCopyInto(out *VerrazzanoWebLogicWorkloadSpec) {
	*out = *in
	in.Template.DeepCopyInto(&out.Template)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new VerrazzanoWebLogicWorkloadSpec.
func (in *VerrazzanoWebLogicWorkloadSpec) DeepCopy() *VerrazzanoWebLogicWorkloadSpec {
	if in == nil {
		return nil
	}
	out := new(VerrazzanoWebLogicWorkloadSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *VerrazzanoWebLogicWorkloadStatus) DeepCopyInto(out *VerrazzanoWebLogicWorkloadStatus) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new VerrazzanoWebLogicWorkloadStatus.
func (in *VerrazzanoWebLogicWorkloadStatus) DeepCopy() *VerrazzanoWebLogicWorkloadStatus {
	if in == nil {
		return nil
	}
	out := new(VerrazzanoWebLogicWorkloadStatus)
	in.DeepCopyInto(out)
	return out
}
