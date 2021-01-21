// Copyright (c) 2020, 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package reconcileresults

import (
	oamrt "github.com/crossplane/crossplane-runtime/apis/core/v1alpha1"
	"github.com/verrazzano/verrazzano/application-operator/apis/oam/v1alpha1"
	"reflect"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

// ReconcileResults is used to collect the results of creating or updating child resources during reconciliation.
// The contained arrays are parallel arrays
type ReconcileResults struct {
	Relations []v1alpha1.QualifiedResourceRelation
	Results   []controllerutil.OperationResult
	Errors    []error
}

// ContainsErrors scans the errors to determine if any errors were recorded.
func (s *ReconcileResults) ContainsErrors() bool {
	for _, err := range s.Errors {
		if err != nil {
			return true
		}
	}
	return false
}

// ContainsUpdates scans the updates to determine if any updates were recorded.
func (s *ReconcileResults) ContainsUpdates() bool {
	for _, result := range s.Results {
		if result != controllerutil.OperationResultNone {
			return true
		}
	}
	return false
}

// ContainsRelation determines if the reconcile result contains the provided relation.
func (s *ReconcileResults) ContainsRelation(relation v1alpha1.QualifiedResourceRelation) bool {
	for _, existing := range s.Relations {
		if reflect.DeepEqual(relation, existing) {
			return true
		}
	}
	return false
}

// CreateConditionedStatus creates conditioned status for use in object status.
// If no errors are found in the reconcile status a success condition is returned.
// Otherwise reconcile errors statuses are returned for the first error.
func (s *ReconcileResults) CreateConditionedStatus() oamrt.ConditionedStatus {
	// Return the first error if there are any.
	for _, err := range s.Errors {
		if err != nil {
			return oamrt.ConditionedStatus{Conditions: []oamrt.Condition{oamrt.ReconcileError(err)}}
		}
	}
	// If no errors are found then return success.
	return oamrt.ConditionedStatus{Conditions: []oamrt.Condition{oamrt.ReconcileSuccess()}}
}

// CreateResources creates a typed reference slice for use in an object status.
func (s *ReconcileResults) CreateResources() []oamrt.TypedReference {
	resources := []oamrt.TypedReference{}
	for _, relation := range s.Relations {
		resources = append(resources, oamrt.TypedReference{
			APIVersion: relation.APIVersion,
			Kind:       relation.Kind,
			Name:       relation.Name,
		})
	}
	return resources
}

// CreateRelations creates a qualified resource relation slice for use in an object status.
func (s *ReconcileResults) CreateRelations() []v1alpha1.QualifiedResourceRelation {
	// Copies the slice.
	return append([]v1alpha1.QualifiedResourceRelation{}, s.Relations...)
}

// RecordOutcome records the outcome of an operation during a reconcile.
func (s *ReconcileResults) RecordOutcome(rel v1alpha1.QualifiedResourceRelation, res controllerutil.OperationResult, err error) {
	s.Relations = append(s.Relations, rel)
	s.Results = append(s.Results, res)
	s.Errors = append(s.Errors, err)
}

// RecordOutcomeIfError records the outcome of an operation during a reconcile only the err is non-nil.
func (s *ReconcileResults) RecordOutcomeIfError(rel v1alpha1.QualifiedResourceRelation, res controllerutil.OperationResult, err error) {
	if err != nil {
		s.Relations = append(s.Relations, rel)
		s.Results = append(s.Results, res)
		s.Errors = append(s.Errors, err)
	}
}

// ConditionsEquivalent determines if two conditions are equivalent.
// The type, status, reason and message are compared to determine equivalence.
func ConditionsEquivalent(left *oamrt.Condition, right *oamrt.Condition) bool {
	if left == nil && right == nil {
		return true
	}
	if left == nil || right == nil {
		return false
	}
	if left.Type != right.Type || left.Status != right.Status || left.Reason != right.Reason || left.Message != right.Message {
		return false
	}
	return true
}

// ConditionedStatusEquivalent determines if two conditioned status are equivalent.
// This is done by searching for all of the conditions from the left in the right.
// Then the conditions in the right are searched for in the left.
// False is returned at any point when a condition cannot be found.
// True is returned if all conditions in the left can be found in the right and vice versa.
func ConditionedStatusEquivalent(left *oamrt.ConditionedStatus, right *oamrt.ConditionedStatus) bool {
	if left == nil && right == nil {
		return true
	}
	if left == nil || right == nil {
		return false
	}
	if len(left.Conditions) != len(right.Conditions) {
		return false
	}
	for i := range left.Conditions {
		if !ConditionsEquivalent(&left.Conditions[i], &right.Conditions[i]) {
			return false
		}
	}
	return true
}
