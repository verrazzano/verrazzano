// Copyright (c) 2021, 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
package status

import (
	"testing"

	"github.com/verrazzano/verrazzano/pkg/log/vzlog"

	"github.com/stretchr/testify/assert"
	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	k8scheme "k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

// TestDeploymentsReady tests a deployment ready status check
// GIVEN a call validate DeploymentsReady
// WHEN the target Deployment object has a minimum of desired available replicas
// THEN true is returned
func TestDeploymentsReady(t *testing.T) {
	name := types.NamespacedName{Name: "foo", Namespace: "bar"}
	client := fake.NewFakeClientWithScheme(k8scheme.Scheme, &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: name.Namespace,
			Name:      name.Name,
		},
		Status: appsv1.DeploymentStatus{
			AvailableReplicas: 1,
			UpdatedReplicas:   1,
		},
	})
	assert.True(t, DeploymentsReady(vzlog.DefaultLogger(), client, []types.NamespacedName{name}, 1, ""))
}

// TestMultipleReplicasReady tests a deployment ready status check
// GIVEN a call validate DeploymentsReady for more than one replica
// WHEN the target Deployment object has met the minimum of desired available replicas > 1
// THEN true is returned
func TestMultipleReplicasReady(t *testing.T) {
	name := types.NamespacedName{Name: "foo", Namespace: "bar"}
	client := fake.NewFakeClientWithScheme(k8scheme.Scheme, &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: name.Namespace,
			Name:      name.Name,
		},
		Status: appsv1.DeploymentStatus{
			AvailableReplicas: 3,
			UpdatedReplicas:   3,
		},
	})
	assert.True(t, DeploymentsReady(vzlog.DefaultLogger(), client, []types.NamespacedName{name}, 3, ""))
}

// TestMultipleReplicasReadyAboveThreshold tests a deployment ready status check
// GIVEN a call validate DeploymentsReady for more than one replica
// WHEN the target Deployment object has more than the minimum desired replicas available
// THEN true is returned
func TestMultipleReplicasReadyAboveThreshold(t *testing.T) {
	name := types.NamespacedName{Name: "foo", Namespace: "bar"}
	client := fake.NewFakeClientWithScheme(k8scheme.Scheme, &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: name.Namespace,
			Name:      name.Name,
		},
		Status: appsv1.DeploymentStatus{
			AvailableReplicas: 3,
			UpdatedReplicas:   3,
		},
	})
	assert.True(t, DeploymentsReady(vzlog.DefaultLogger(), client, []types.NamespacedName{name}, 2, ""))
}

// TestDeploymentsNotAvailableOrReady tests a deployment ready status check
// GIVEN a call validate DeploymentsReady
// WHEN the target Deployment object does not have a minimium number of desired available replicas
// THEN false is returned
func TestDeploymentsNotAvailableOrReady(t *testing.T) {
	name := types.NamespacedName{Name: "foo", Namespace: "bar"}
	client := fake.NewFakeClientWithScheme(k8scheme.Scheme, &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: name.Namespace,
			Name:      name.Name,
		},
		Status: appsv1.DeploymentStatus{
			AvailableReplicas: 1,
			UpdatedReplicas:   0,
		},
	})
	assert.False(t, DeploymentsReady(vzlog.DefaultLogger(), client, []types.NamespacedName{name}, 1, ""))
}

// TestMultipleReplicasReadyBelowThreshold tests a deployment ready status check
// GIVEN a call validate DeploymentsReady for more than one replica
// WHEN the target Deployment object has less than the minimum desired replicas available
// THEN false is returned
func TestMultipleReplicasReadyBelowThreshold(t *testing.T) {
	name := types.NamespacedName{Name: "foo", Namespace: "bar"}
	client := fake.NewFakeClientWithScheme(k8scheme.Scheme, &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: name.Namespace,
			Name:      name.Name,
		},
		Status: appsv1.DeploymentStatus{
			AvailableReplicas: 3,
			UpdatedReplicas:   1,
		},
	})
	assert.False(t, DeploymentsReady(vzlog.DefaultLogger(), client, []types.NamespacedName{name}, 2, ""))
}

// TestDeploymentsReadyDeploymentNotFound tests a deployment ready status check
// GIVEN a call validate DeploymentsReady
// WHEN the target Deployment object is not found
// THEN false is returned
func TestDeploymentsReadyDeploymentNotFound(t *testing.T) {
	name := types.NamespacedName{Name: "foo", Namespace: "bar"}
	client := fake.NewFakeClientWithScheme(k8scheme.Scheme)
	assert.False(t, DeploymentsReady(vzlog.DefaultLogger(), client, []types.NamespacedName{name}, 1, ""))
}

// TestMultipleDeploymentsReplicasReadyBelowThreshold tests a deployment ready status check
// GIVEN a call validate DeploymentsReady for more than one replica for multiple deployments
// WHEN the one of the target Deployment objects has less than the minimum desired replicas available
// THEN false is returned
func TestMultipleDeploymentsReplicasReadyBelowThreshold(t *testing.T) {
	name1 := types.NamespacedName{Name: "foo", Namespace: "bar"}
	name2 := types.NamespacedName{Name: "thud", Namespace: "bar"}
	name3 := types.NamespacedName{Name: "thud", Namespace: "thwack"}
	client := fake.NewFakeClientWithScheme(k8scheme.Scheme,
		&appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: name1.Namespace,
				Name:      name1.Name,
			},
			Status: appsv1.DeploymentStatus{
				AvailableReplicas: 3,
				UpdatedReplicas:   3,
			},
		},
		&appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: name2.Namespace,
				Name:      name2.Name,
			},
			Status: appsv1.DeploymentStatus{
				AvailableReplicas: 3,
				UpdatedReplicas:   1,
			},
		},
		&appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: name3.Namespace,
				Name:      name3.Name,
			},
			Status: appsv1.DeploymentStatus{
				AvailableReplicas: 3,
				UpdatedReplicas:   3,
			},
		},
	)
	assert.False(t, DeploymentsReady(vzlog.DefaultLogger(), client, []types.NamespacedName{name1, name2, name3}, 2, ""))
}

// TestMultipleDeploymentsReplicasReady tests a deployment ready status check
// GIVEN a call validate DeploymentsReady for more than one replica for multiple deployments
// WHEN the all of the target Deployment objects meet the minimum desired replicas available threshold
// THEN true is returned
func TestMultipleDeploymentsReplicasReady(t *testing.T) {
	name1 := types.NamespacedName{Name: "foo", Namespace: "bar"}
	name2 := types.NamespacedName{Name: "thud", Namespace: "bar"}
	name3 := types.NamespacedName{Name: "thud", Namespace: "thwack"}
	client := fake.NewFakeClientWithScheme(k8scheme.Scheme,
		&appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: name1.Namespace,
				Name:      name1.Name,
			},
			Status: appsv1.DeploymentStatus{
				AvailableReplicas: 3,
				UpdatedReplicas:   3,
			},
		},
		&appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: name2.Namespace,
				Name:      name2.Name,
			},
			Status: appsv1.DeploymentStatus{
				AvailableReplicas: 3,
				UpdatedReplicas:   3,
			},
		},
		&appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: name3.Namespace,
				Name:      name3.Name,
			},
			Status: appsv1.DeploymentStatus{
				AvailableReplicas: 3,
				UpdatedReplicas:   3,
			},
		},
	)
	assert.True(t, DeploymentsReady(vzlog.DefaultLogger(), client, []types.NamespacedName{name1, name2, name3}, 2, ""))
}
