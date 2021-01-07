// Copyright (c) 2020, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package v1alpha1

import (
	oamrt "github.com/crossplane/crossplane-runtime/apis/core/v1alpha1"
	asserts "github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"testing"
)

// TestGetCondition tests the GetCondition method
func TestGetCondition(t *testing.T) {
	assert := asserts.New(t)
	var trait MetricsTrait
	var cond oamrt.Condition

	trait = MetricsTrait{
		Status: MetricsTraitStatus{
			ConditionedStatus: oamrt.ConditionedStatus{Conditions: []oamrt.Condition{{
				Type:               oamrt.TypeSynced,
				Status:             corev1.ConditionTrue,
				LastTransitionTime: metav1.Now(),
				Reason:             "test-reason",
				Message:            "test-message"}}}}}

	// GIVEN a trait with a synced condition
	// WHEN the syned condition is retrieved
	// THEN verify that the correct condition was retrieved
	cond = trait.GetCondition(oamrt.TypeSynced)
	assert.Equal(oamrt.TypeSynced, cond.Type)
	assert.Equal(corev1.ConditionTrue, cond.Status)
	assert.Equal(oamrt.ConditionReason("test-reason"), cond.Reason)
	assert.Equal("test-message", cond.Message)

	// GIVEN a trait with a synced condition
	// WHEN the ready condition is retrieved
	// THEN verify that an unknown status condition is returned
	cond = trait.GetCondition(oamrt.TypeReady)
	assert.Equal(oamrt.TypeReady, cond.Type)
	assert.Equal(corev1.ConditionUnknown, cond.Status)
}

// TestSetCondition tests the SetConditions method.
func TestSetCondition(t *testing.T) {
	assert := asserts.New(t)
	var trait MetricsTrait
	var cond oamrt.Condition

	cond = oamrt.Condition{
		Type:               oamrt.TypeSynced,
		Status:             corev1.ConditionTrue,
		LastTransitionTime: metav1.Now(),
		Reason:             "test-reason",
		Message:            "test-message"}

	// GIVEN an trait with no conditions
	// WHEN a condition is set
	// THEN verify that the fields are correctly populated
	trait = MetricsTrait{}
	trait.SetConditions(cond)
	assert.Len(trait.Status.Conditions, 1)
	assert.Equal(cond, trait.Status.Conditions[0])
}

// TestGetWorkloadReference tests the GetWorkloadReference method.
func TestGetWorkloadReference(t *testing.T) {
	assert := asserts.New(t)
	var trait MetricsTrait
	var expected oamrt.TypedReference
	var actual oamrt.TypedReference

	// GIVEN a trait with a workload reference
	// WHEN the workload reference is retrieved
	// THEN verify the correct workload reference information is returned
	expected = oamrt.TypedReference{
		APIVersion: "test-api/ver",
		Kind:       "test-kind",
		Name:       "test-name",
		UID:        "test-uid"}
	trait = MetricsTrait{}
	trait.Spec.WorkloadReference = expected
	actual = trait.GetWorkloadReference()
	assert.Equal(expected, actual)
}

// TestSetWorkloadReference test the SetWorkloadReference method.
func TestSetWorkloadReference(t *testing.T) {
	assert := asserts.New(t)
	var trait MetricsTrait
	var expected oamrt.TypedReference
	var actual oamrt.TypedReference

	// GIVEN a trait with a workload reference
	// WHEN the workload reference is retrieved
	// THEN verify the correct workload reference information is returned
	expected = oamrt.TypedReference{
		APIVersion: "test-api/ver",
		Kind:       "test-kind",
		Name:       "test-name",
		UID:        "test-uid"}
	trait = MetricsTrait{}
	trait.SetWorkloadReference(expected)
	actual = trait.Spec.WorkloadReference
	assert.Equal(expected, actual)
}
