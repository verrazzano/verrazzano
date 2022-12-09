// Copyright (c) 2021, 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package v1alpha1

import (
	oamrt "github.com/crossplane/crossplane-runtime/apis/core/v1alpha1"
	"github.com/crossplane/oam-kubernetes-runtime/pkg/oam"
)

// Enforce that LoggingTrait adheres to Trait interface.
var _ oam.Trait = &LoggingTrait{}

// GetCondition gets the status condition of this trait.
func (in *LoggingTrait) GetCondition(ct oamrt.ConditionType) oamrt.Condition {
	return in.Status.GetCondition(ct)
}

// SetConditions sets the status condition of this trait.
func (in *LoggingTrait) SetConditions(c ...oamrt.Condition) {
	in.Status.SetConditions(c...)
}

// GetWorkloadReference gets the workload reference of this trait.
func (in *LoggingTrait) GetWorkloadReference() oamrt.TypedReference {
	return in.Spec.WorkloadReference
}

// SetWorkloadReference sets the workload reference of this trait.
func (in *LoggingTrait) SetWorkloadReference(r oamrt.TypedReference) {
	in.Spec.WorkloadReference = r
}
