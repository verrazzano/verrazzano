// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package k8sutil

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	k8scheme "k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

// TestDeploymentsReady tests a deployment ready check
// GIVEN a call validate DeploymentsReady
// WHEN the target Deployment object has an available condition of true
// THEN true is returned
func TestDeploymentsReady(t *testing.T) {
	lastTransitionTime := metav1.Now()
	namespacedName := []types.NamespacedName{
		{
			Name:      "foo",
			Namespace: "bar",
		},
	}
	client := fake.NewFakeClientWithScheme(k8scheme.Scheme,
		&appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: "bar",
				Name:      "foo",
			},
			Status: appsv1.DeploymentStatus{
				Conditions: []appsv1.DeploymentCondition{
					{
						Type:               appsv1.DeploymentAvailable,
						Status:             corev1.ConditionTrue,
						LastTransitionTime: metav1.NewTime(lastTransitionTime.Add(time.Second)),
					},
				},
			},
		},
	)
	ready, err := DeploymentsAreReady(client, namespacedName, 1, lastTransitionTime)
	assert.NoError(t, err)
	assert.True(t, ready)
}

// TestDeploymentsNotReady tests a deployment ready check
// GIVEN a call validate DeploymentsReady
// WHEN the target Deployment object does not have an available condition of true
// THEN false is returned
func TestDeploymentsNotReady(t *testing.T) {
	lastTransitionTime := metav1.Now()
	namespacedName := []types.NamespacedName{
		{
			Name:      "foo",
			Namespace: "bar",
		},
	}
	client := fake.NewFakeClientWithScheme(k8scheme.Scheme,
		&appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: "bar",
				Name:      "foo",
			},
			Status: appsv1.DeploymentStatus{
				Conditions: []appsv1.DeploymentCondition{
					{
						Type:               appsv1.DeploymentAvailable,
						Status:             corev1.ConditionFalse,
						LastTransitionTime: metav1.NewTime(lastTransitionTime.Add(time.Second)),
					},
				},
			},
		},
	)
	ready, err := DeploymentsAreReady(client, namespacedName, 1, lastTransitionTime)
	assert.Error(t, err)
	assert.EqualError(t, err, "waiting for deployment bar/foo condition to be Available")
	assert.False(t, ready)
}

// TestDeploymentsNotReadyOldAvailable tests a deployment ready check
// GIVEN a call validate DeploymentsReady
// WHEN the target Deployment object has an older available condition of true
// THEN false is returned
func TestDeploymentsNotReadyOldAvailable(t *testing.T) {
	oldTransitionTime := metav1.Now()
	lastTransitionTime := metav1.NewTime(oldTransitionTime.Add(time.Second))
	namespacedName := []types.NamespacedName{
		{
			Name:      "foo",
			Namespace: "bar",
		},
	}
	client := fake.NewFakeClientWithScheme(k8scheme.Scheme,
		&appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: "bar",
				Name:      "foo",
			},
			Status: appsv1.DeploymentStatus{
				Conditions: []appsv1.DeploymentCondition{
					{
						Type:               appsv1.DeploymentAvailable,
						Status:             corev1.ConditionFalse,
						LastTransitionTime: oldTransitionTime,
					},
				},
			},
		},
	)
	ready, err := DeploymentsAreReady(client, namespacedName, 1, lastTransitionTime)
	assert.Error(t, err)
	assert.EqualError(t, err, "waiting for deployment bar/foo condition to be Available")
	assert.False(t, ready)
}
