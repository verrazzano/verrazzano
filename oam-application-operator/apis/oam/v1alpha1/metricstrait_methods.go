// Copyright (c) 2020, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package v1alpha1

import (
	oamrt "github.com/crossplane/crossplane-runtime/apis/core/v1alpha1"
	"github.com/crossplane/oam-kubernetes-runtime/pkg/oam"
)

// Ensure that MetricsTrait adheres to Scope interface.
var _ oam.Trait = &MetricsTrait{}

// GetCondition gets the condition of this trait.
func (t *MetricsTrait) GetCondition(ct oamrt.ConditionType) oamrt.Condition {
	return t.Status.GetCondition(ct)
}

// SetConditions sets the condition of this trait.
func (t *MetricsTrait) SetConditions(c ...oamrt.Condition) {
	t.Status.SetConditions(c...)
}

// GetWorkloadReference gets the workload reference of this trait.
func (t *MetricsTrait) GetWorkloadReference() oamrt.TypedReference {
	return t.Spec.WorkloadReference
}

// SetWorkloadReference sets the workload reference of this trait.
func (t *MetricsTrait) SetWorkloadReference(r oamrt.TypedReference) {
	t.Spec.WorkloadReference = r
}
