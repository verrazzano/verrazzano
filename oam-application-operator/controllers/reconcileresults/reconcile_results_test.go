// Copyright (c) 2020, 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package reconcileresults

import (
	"fmt"
	oamrt "github.com/crossplane/crossplane-runtime/apis/core/v1alpha1"
	asserts "github.com/stretchr/testify/assert"
	"github.com/verrazzano/verrazzano/oam-application-operator/apis/oam/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"testing"
)

// TestContainsErrors tests positive and negative use cases using ContainsErrors
func TestContainsErrors(t *testing.T) {
	assert := asserts.New(t)
	var status ReconcileResults

	// GIVEN status with all default fields
	// WHEN the status is checked for errors
	// THEN verify that errors are not detected
	status = ReconcileResults{}
	assert.False(status.ContainsErrors())

	// GIVEN status that contains an error
	// WHEN the status is checked for errors
	// THEN verify that errors are detected
	status = ReconcileResults{Errors: []error{fmt.Errorf("test-error")}}
	assert.True(status.ContainsErrors())

	// GIVEN status that contains no errors
	// WHEN the status is checked for errors
	// THEN verify that errors are not detected
	status = ReconcileResults{Errors: []error{nil}}
	assert.False(status.ContainsErrors())

	// GIVEN status that contains both errors and nils
	// WHEN the status is checked for errors
	// THEN verify that errors are detected
	status = ReconcileResults{Errors: []error{nil, fmt.Errorf("test-error"), nil}}
	assert.True(status.ContainsErrors())
}

// TestContainsUpdates tests the ContainsUpdates method
func TestContainsUpdates(t *testing.T) {
	assert := asserts.New(t)
	var status ReconcileResults

	// GIVEN an empty reconcile result
	// WHEN a check is made for updates
	// THEN verify false is returned
	assert.False(status.ContainsUpdates())

	// GIVEN an empty reconcile result with an update
	// WHEN a check is made for updates
	// THEN verify true is returned
	status = ReconcileResults{}
	status.RecordOutcome(v1alpha1.QualifiedResourceRelation{}, controllerutil.OperationResultUpdated, nil)
	assert.True(status.ContainsUpdates())

	// GIVEN an empty reconcile result with no updates
	// WHEN a check is made for updates
	// THEN verify false is returned
	status = ReconcileResults{}
	status.RecordOutcome(v1alpha1.QualifiedResourceRelation{}, controllerutil.OperationResultNone, nil)
	assert.False(status.ContainsUpdates())
}

// func (s *ReconcileResults) ContainsRelation(relation v1alpha1.QualifiedResourceRelation) bool {
// TestContainsRelation tests the ContainsRelation method.
func TestContainsRelation(t *testing.T) {
	assert := asserts.New(t)
	var results ReconcileResults
	var rel v1alpha1.QualifiedResourceRelation

	rel = v1alpha1.QualifiedResourceRelation{
		APIVersion: "test-apiver",
		Kind:       "test-kind",
		Name:       "test-name",
		Role:       "test-role"}

	// GIVEN a reconcile result with no relations
	// WHEN a search is made for a relation
	// THEN verify no relation is found
	results = ReconcileResults{}
	assert.False(results.ContainsRelation(rel))

	// GIVEN a reconcile result with at least one relation
	// WHEN a search is made for a relation that was recorded
	// THEN verify the relation is found
	results = ReconcileResults{}
	results.RecordOutcome(rel, controllerutil.OperationResultNone, nil)
	assert.True(results.ContainsRelation(rel))
}

// TestCreateConditionedStatus tests the CreateConditionedStatus method.
func TestCreateConditionedStatus(t *testing.T) {
	assert := asserts.New(t)
	var recStatus ReconcileResults
	var condStatus oamrt.ConditionedStatus

	// GIVEN a default reconcile status
	// WHEN the conditioned status is created from the reconcile status
	// THEN verify that the conditioned status indicates success
	recStatus = ReconcileResults{}
	condStatus = recStatus.CreateConditionedStatus()
	assert.Len(condStatus.Conditions, 1)
	assert.Equal(oamrt.ReasonReconcileSuccess, condStatus.Conditions[0].Reason)

	// GIVEN a reconcile status with one error
	// WHEN the conditioned status is created from the reconcile status
	// THEN verify that the error is included in the conditioned status
	recStatus = ReconcileResults{Errors: []error{fmt.Errorf("test-error")}}
	condStatus = recStatus.CreateConditionedStatus()
	assert.Len(condStatus.Conditions, 1)
	assert.Equal(oamrt.ReasonReconcileError, condStatus.Conditions[0].Reason)
	assert.Equal("test-error", condStatus.Conditions[0].Message)

	// GIVEN a reconcile status with one success
	// WHEN the conditioned status is created from the reconcile status
	// THEN verify that the conditioned status indicates success
	recStatus = ReconcileResults{Errors: []error{nil}}
	condStatus = recStatus.CreateConditionedStatus()
	assert.Len(condStatus.Conditions, 1)
	assert.Equal(oamrt.ReasonReconcileSuccess, condStatus.Conditions[0].Reason)

	// GIVEN a reconcile status with both success and error
	// WHEN the conditioned status is created from the reconcile status
	// THEN verify that the conditioned status indicates failure
	recStatus = ReconcileResults{Errors: []error{fmt.Errorf("test-error"), nil}}
	condStatus = recStatus.CreateConditionedStatus()
	assert.Len(condStatus.Conditions, 1)
	assert.Equal(oamrt.ReasonReconcileError, condStatus.Conditions[0].Reason)
	assert.Equal("test-error", condStatus.Conditions[0].Message)
}

// TestCreateResources tests the CreateResources method
func TestCreateResources(t *testing.T) {
	assert := asserts.New(t)
	var status ReconcileResults
	var resources []oamrt.TypedReference

	// GIVEN a default reconcile results
	// WHEN a resources slice is retrieved
	// THEN verify the slice is empty.
	status = ReconcileResults{}
	resources = status.CreateResources()
	assert.Len(resources, 0)

	// GIVEN a reconcile status with a related resource
	// WHEN related resources are retrieved
	// THEN verify the retrieved slice matches the recorded resources.
	relation := v1alpha1.QualifiedResourceRelation{
		APIVersion: "test-apiver",
		Kind:       "test-kind",
		Name:       "test-name",
		Role:       "test-role"}
	resource := oamrt.TypedReference{
		APIVersion: "test-apiver",
		Kind:       "test-kind",
		Name:       "test-name"}
	status.RecordOutcome(relation, controllerutil.OperationResultUpdated, nil)
	resources = status.CreateResources()
	assert.Len(resources, 1)
	assert.Equal(resource, resources[0])
}

// TestCreateStatusRelations tests the CreateStatusRelations method.
func TestCreateRelations(t *testing.T) {
	assert := asserts.New(t)
	var status ReconcileResults
	var rels []v1alpha1.QualifiedResourceRelation

	// GIVEN a default reconcile results
	// WHEN a relations slice is retrieved
	// THEN verify the slice is empty.
	status = ReconcileResults{}
	rels = status.CreateRelations()
	assert.Len(rels, 0)

	// GIVEN a reconcile status with a related resource
	// WHEN related resources are retrieved
	// THEN verify the retrieved slice matches the recorded relations.
	rel := v1alpha1.QualifiedResourceRelation{
		APIVersion: "test-apiver",
		Kind:       "test-kind",
		Name:       "test-name",
		Role:       "test-role"}
	status.RecordOutcome(rel, controllerutil.OperationResultUpdated, nil)
	rels = status.CreateRelations()
	assert.Len(rels, 1)
	assert.Equal(rel, rels[0])
}

// TestRecordOutcomeIfError tests the RecordOutcomeIfError method.
func TestRecordOutcomeIfError(t *testing.T) {
	assert := asserts.New(t)
	var status ReconcileResults
	var rels []v1alpha1.QualifiedResourceRelation
	var rel v1alpha1.QualifiedResourceRelation
	var res controllerutil.OperationResult

	// GIVEN an empty reconcile results
	// WHEN no error outcomes have been recorded
	// THEN verify that no relations have been recorded
	rel = v1alpha1.QualifiedResourceRelation{
		APIVersion: "test-api-ver",
		Kind:       "test-kind",
		Namespace:  "test-space",
		Name:       "test-name",
		Role:       "test-role"}
	res = controllerutil.OperationResultNone
	status = ReconcileResults{}
	status.RecordOutcomeIfError(rel, res, nil)
	rels = status.CreateRelations()
	assert.Len(rels, 0)

	// GIVEN reconcile results that has recorded some errors and some successes.
	// WHEN the recorded relations are returned
	// THEN verify that only error relations have been recorded
	err := fmt.Errorf("test-error")
	relError := v1alpha1.QualifiedResourceRelation{
		APIVersion: "test-api-ver",
		Kind:       "test-kind",
		Namespace:  "test-space",
		Name:       "test-error-name",
		Role:       "test-role"}
	relSuccess := v1alpha1.QualifiedResourceRelation{
		APIVersion: "test-api-ver",
		Kind:       "test-kind",
		Namespace:  "test-space",
		Name:       "test-error-name",
		Role:       "test-role"}
	res = controllerutil.OperationResultNone
	status = ReconcileResults{}
	status.RecordOutcomeIfError(relSuccess, res, nil)
	status.RecordOutcomeIfError(relError, res, err)
	status.RecordOutcomeIfError(relSuccess, res, nil)
	rels = status.CreateRelations()
	assert.Len(rels, 1)
	assert.Equal(relError, rels[0])
}

// TestConditionsEquivalent tests the ConditionsEquivalent method.
func TestConditionsEquivalent(t *testing.T) {
	assert := asserts.New(t)
	var one oamrt.Condition
	var two oamrt.Condition

	one = oamrt.Condition{
		Type:               "test-type",
		Status:             "test-status-1",
		LastTransitionTime: metav1.Unix(0, 0),
		Reason:             "test-reason",
		Message:            "test-message"}
	two = oamrt.Condition{
		Type:               "test-type",
		Status:             "test-status-2",
		LastTransitionTime: metav1.Unix(1, 1),
		Reason:             "test-reason",
		Message:            "test-message"}

	// GIVEN two nil conditions
	// WHEN the conditions are compared for equivalence
	// THEN verify they are considered equivalent
	assert.True(ConditionsEquivalent(nil, nil))

	// GIVEN one nil and one non-nil condition
	// WHEN the conditions are compared for equivalence
	// THEN verify they are not considered equivalent
	assert.False(ConditionsEquivalent(&one, nil))

	// GIVEN one nil and one non-nil condition
	// WHEN the conditions are compared for equivalence
	// THEN verify they are not considered equivalent
	assert.False(ConditionsEquivalent(nil, &two))

	// GIVEN two identical conditions
	// WHEN the conditions are compared for equivalence
	// THEN verify they are considered equivalent
	assert.True(ConditionsEquivalent(&one, &one))

	// GIVEN two non-equivalent conditions
	// WHEN the conditions are compared for equivalence
	// THEN verify they are not considered equivalent
	assert.False(ConditionsEquivalent(&one, &two))
}

// TestConditionedStatusEquivalent tests the ConditionedStatusEquivalent method.
func TestConditionedStatusEquivalent(t *testing.T) {
	assert := asserts.New(t)

	one := oamrt.ConditionedStatus{Conditions: []oamrt.Condition{{
		Type:               "test-type",
		Status:             "test-status-1",
		LastTransitionTime: metav1.Unix(1, 1),
		Reason:             "test-reason",
		Message:            "test-message"}}}
	two := oamrt.ConditionedStatus{Conditions: []oamrt.Condition{{
		Type:               "test-type",
		Status:             "test-status-2",
		LastTransitionTime: metav1.Unix(2, 2),
		Reason:             "test-reason",
		Message:            "test-message"}}}
	three := oamrt.ConditionedStatus{Conditions: []oamrt.Condition{one.Conditions[0], one.Conditions[0]}}

	// GIVEN two nil conditioned statuses
	// WHEN they are compared for equivalence
	// THEN verify they are considered equivalent
	assert.True(ConditionedStatusEquivalent(nil, nil))

	// GIVEN a nil and non-nil conditioned statuses
	// WHEN they are compared for equivalence
	// THEN verify they are considered not equivalent
	assert.False(ConditionedStatusEquivalent(&one, nil))

	// GIVEN a nil and non-nil conditioned statuses
	// WHEN they are compared for equivalence
	// THEN verify they are considered not equivalent
	assert.False(ConditionedStatusEquivalent(nil, &two))

	// GIVEN two identical conditioned statuses
	// WHEN they are compared for equivalence
	// THEN verify they are considered equivalent
	assert.True(ConditionedStatusEquivalent(&one, &one))

	// GIVEN two non-identical conditioned statuses
	// WHEN they are compared for equivalence
	// THEN verify they are considered not equivalent
	assert.False(ConditionedStatusEquivalent(&one, &two))

	// GIVEN two conditioned statuses of different lengths
	// WHEN they are compared for equivalence
	// THEN verify they are considered not equivalent
	assert.False(ConditionedStatusEquivalent(&one, &three))
}
