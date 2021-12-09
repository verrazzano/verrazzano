// +build !ignore_autogenerated

// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

// Code generated by controller-gen. DO NOT EDIT.

package v1alpha1

import (
	runtime "k8s.io/apimachinery/pkg/runtime"
)

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *MetricsTemplate) DeepCopyInto(out *MetricsTemplate) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	in.Spec.DeepCopyInto(&out.Spec)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new MetricsTemplate.
func (in *MetricsTemplate) DeepCopy() *MetricsTemplate {
	if in == nil {
		return nil
	}
	out := new(MetricsTemplate)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *MetricsTemplate) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *MetricsTemplateList) DeepCopyInto(out *MetricsTemplateList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ListMeta.DeepCopyInto(&out.ListMeta)
	if in.Items != nil {
		in, out := &in.Items, &out.Items
		*out = make([]MetricsTemplate, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new MetricsTemplateList.
func (in *MetricsTemplateList) DeepCopy() *MetricsTemplateList {
	if in == nil {
		return nil
	}
	out := new(MetricsTemplateList)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *MetricsTemplateList) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *MetricsTemplateSpec) DeepCopyInto(out *MetricsTemplateSpec) {
	*out = *in
	in.WorkloadSelector.DeepCopyInto(&out.WorkloadSelector)
	out.PrometheusConfig = in.PrometheusConfig
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new MetricsTemplateSpec.
func (in *MetricsTemplateSpec) DeepCopy() *MetricsTemplateSpec {
	if in == nil {
		return nil
	}
	out := new(MetricsTemplateSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *MetricsTemplateStatus) DeepCopyInto(out *MetricsTemplateStatus) {
	*out = *in
	in.ConditionedStatus.DeepCopyInto(&out.ConditionedStatus)
	if in.Resources != nil {
		in, out := &in.Resources, &out.Resources
		*out = make([]QualifiedResourceRelation, len(*in))
		copy(*out, *in)
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new MetricsTemplateStatus.
func (in *MetricsTemplateStatus) DeepCopy() *MetricsTemplateStatus {
	if in == nil {
		return nil
	}
	out := new(MetricsTemplateStatus)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *PrometheusConfig) DeepCopyInto(out *PrometheusConfig) {
	*out = *in
	out.TargetConfigMap = in.TargetConfigMap
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new PrometheusConfig.
func (in *PrometheusConfig) DeepCopy() *PrometheusConfig {
	if in == nil {
		return nil
	}
	out := new(PrometheusConfig)
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
func (in *TargetConfigMap) DeepCopyInto(out *TargetConfigMap) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new TargetConfigMap.
func (in *TargetConfigMap) DeepCopy() *TargetConfigMap {
	if in == nil {
		return nil
	}
	out := new(TargetConfigMap)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *TargetWorkload) DeepCopyInto(out *TargetWorkload) {
	*out = *in
	in.Selector.DeepCopyInto(&out.Selector)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new TargetWorkload.
func (in *TargetWorkload) DeepCopy() *TargetWorkload {
	if in == nil {
		return nil
	}
	out := new(TargetWorkload)
	in.DeepCopyInto(out)
	return out
}
