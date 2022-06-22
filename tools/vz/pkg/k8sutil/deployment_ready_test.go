// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package k8sutil

import (
	"testing"

	"github.com/stretchr/testify/assert"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
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
				AvailableReplicas: 1,
				Replicas:          1,
				UpdatedReplicas:   1,
			},
		},
		&corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: "bar",
				Name:      "foo-95d8c5d96-m6mbr",
				Labels: map[string]string{
					podTemplateHashLabel: "95d8c5d96",
					"app":                "foo",
				},
			},
			Status: corev1.PodStatus{
				InitContainerStatuses: []corev1.ContainerStatus{
					{
						Ready: true,
					},
				},
				ContainerStatuses: []corev1.ContainerStatus{
					{
						Ready: true,
					},
				},
			},
		},
		&appsv1.ReplicaSet{
			ObjectMeta: metav1.ObjectMeta{
				Namespace:   "bar",
				Name:        "foo-95d8c5d96",
				Annotations: map[string]string{deploymentRevisionAnnotation: "1"},
			},
		},
	)
	ready, err := DeploymentsAreReady(client, namespacedName, 1)
	assert.NoError(t, err)
	assert.True(t, ready)
}

// TestDeploymentsContainerNotReady tests a deployment ready status check
// GIVEN a call validate DeploymentsReady
// WHEN the target Deployment object has a minimum of number of containers ready
// THEN false is returned
func TestDeploymentsContainerNotReady(t *testing.T) {
	selector := &metav1.LabelSelector{
		MatchLabels: map[string]string{
			"app": "foo",
		},
	}
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
			Spec: appsv1.DeploymentSpec{
				Selector: selector,
			},
			Status: appsv1.DeploymentStatus{
				AvailableReplicas: 1,
				Replicas:          1,
				UpdatedReplicas:   1,
			},
		},
		&corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: "bar",
				Name:      "foo-95d8c5d96-m6mbr",
				Labels: map[string]string{
					podTemplateHashLabel: "95d8c5d96",
					"app":                "foo",
				},
			},
			Status: corev1.PodStatus{
				ContainerStatuses: []corev1.ContainerStatus{
					{
						Ready: false,
					},
				},
			},
		},
		&appsv1.ReplicaSet{
			ObjectMeta: metav1.ObjectMeta{
				Namespace:   "bar",
				Name:        "foo-95d8c5d96",
				Annotations: map[string]string{deploymentRevisionAnnotation: "1"},
			},
		},
	)
	ready, err := DeploymentsAreReady(client, namespacedName, 1)
	assert.Error(t, err)
	assert.EqualError(t, err, "waiting for container of pod foo-95d8c5d96-m6mbr to be ready")
	assert.False(t, ready)
}

// TestDeploymentsInitContainerNotReady tests a deployment ready status check
// GIVEN a call validate DeploymentsReady
// WHEN the target Deployment object has a minimum of number of init containers ready
// THEN false is returned
func TestDeploymentsInitContainerNotReady(t *testing.T) {
	selector := &metav1.LabelSelector{
		MatchLabels: map[string]string{
			"app": "foo",
		},
	}
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
			Spec: appsv1.DeploymentSpec{
				Selector: selector,
			},
			Status: appsv1.DeploymentStatus{
				AvailableReplicas: 1,
				Replicas:          1,
				UpdatedReplicas:   1,
			},
		},
		&corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: "bar",
				Name:      "foo-95d8c5d96-m6mbr",
				Labels: map[string]string{
					podTemplateHashLabel: "95d8c5d96",
					"app":                "foo",
				},
			},
			Status: corev1.PodStatus{
				InitContainerStatuses: []corev1.ContainerStatus{
					{
						Ready: false,
					},
				},
			},
		},
		&appsv1.ReplicaSet{
			ObjectMeta: metav1.ObjectMeta{
				Namespace:   "bar",
				Name:        "foo-95d8c5d96",
				Annotations: map[string]string{deploymentRevisionAnnotation: "1"},
			},
		},
	)
	ready, err := DeploymentsAreReady(client, namespacedName, 1)
	assert.Error(t, err)
	assert.EqualError(t, err, "waiting for init container of pod foo-95d8c5d96-m6mbr to be ready")
	assert.False(t, ready)
}

// TestMultipleReplicasReady tests a deployment ready status check
// GIVEN a call validate DeploymentsReady for more than one replica
// WHEN the target Deployment object has met the minimum of desired available replicas > 1
// THEN true is returned
func TestMultipleReplicasReady(t *testing.T) {
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
				AvailableReplicas: 2,
				Replicas:          2,
				UpdatedReplicas:   2,
			},
		},
		&corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: "bar",
				Name:      "foo-95d8c5d96-m6mbr",
				Labels: map[string]string{
					podTemplateHashLabel: "95d8c5d96",
					"app":                "foo",
				},
			},
			Status: corev1.PodStatus{
				ContainerStatuses: []corev1.ContainerStatus{
					{
						Ready: true,
					},
				},
			},
		},
		&corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: "bar",
				Name:      "foo-95d8c5d96-l6r96",
				Labels: map[string]string{
					podTemplateHashLabel: "95d8c5d96",
					"app":                "foo",
				},
			},
			Status: corev1.PodStatus{
				ContainerStatuses: []corev1.ContainerStatus{
					{
						Ready: true,
					},
				},
			},
		},
		&appsv1.ReplicaSet{
			ObjectMeta: metav1.ObjectMeta{
				Namespace:   "bar",
				Name:        "foo-95d8c5d96",
				Annotations: map[string]string{deploymentRevisionAnnotation: "1"},
			},
		},
	)
	ready, err := DeploymentsAreReady(client, namespacedName, 2)
	assert.NoError(t, err)
	assert.True(t, ready)
}

// TestMultipleReplicasReadyAboveThreshold tests a deployment ready status check
// GIVEN a call validate DeploymentsReady for more than one replica
// WHEN the target Deployment object has more than the minimum desired replicas available
// THEN true is returned
func TestMultipleReplicasReadyAboveThreshold(t *testing.T) {
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
				AvailableReplicas: 2,
				Replicas:          2,
				UpdatedReplicas:   2,
			},
		},
		&corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: "bar",
				Name:      "foo-95d8c5d96-m6mbr",
				Labels: map[string]string{
					podTemplateHashLabel: "95d8c5d96",
					"app":                "foo",
				},
			},
			Status: corev1.PodStatus{
				ContainerStatuses: []corev1.ContainerStatus{
					{
						Ready: true,
					},
				},
			},
		},
		&corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: "bar",
				Name:      "foo-95d8c5d96-l6r96",
				Labels: map[string]string{
					podTemplateHashLabel: "95d8c5d96",
					"app":                "foo",
				},
			},
			Status: corev1.PodStatus{
				ContainerStatuses: []corev1.ContainerStatus{
					{
						Ready: true,
					},
				},
			},
		},
		&appsv1.ReplicaSet{
			ObjectMeta: metav1.ObjectMeta{
				Namespace:   "bar",
				Name:        "foo-95d8c5d96",
				Annotations: map[string]string{deploymentRevisionAnnotation: "1"},
			},
		},
	)
	ready, err := DeploymentsAreReady(client, namespacedName, 1)
	assert.NoError(t, err)
	assert.True(t, ready)
}

// TestDeploymentsNoneAvailable tests a deployment ready status check
// GIVEN a call validate DeploymentsReady
// WHEN the target Deployment object does not have a minimum number of desired available replicas
// THEN false is returned
func TestDeploymentsNoneAvailable(t *testing.T) {
	namespacedName := []types.NamespacedName{
		{
			Name:      "foo",
			Namespace: "bar",
		},
	}
	client := fake.NewFakeClientWithScheme(k8scheme.Scheme, &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "bar",
			Name:      "foo",
		},
		Status: appsv1.DeploymentStatus{
			AvailableReplicas: 0,
			Replicas:          1,
			UpdatedReplicas:   1,
		},
	})
	ready, err := DeploymentsAreReady(client, namespacedName, 1)
	assert.Error(t, err)
	assert.EqualError(t, err, "waiting for deployment bar/foo replicas to be 1, current available replicas is 0")
	assert.False(t, ready)
}

// TestDeploymentsNoneUpdated tests a deployment ready status check
// GIVEN a call validate DeploymentsReady
// WHEN the target Deployment object does not have a minimum number of desired updated replicas
// THEN false is returned
func TestDeploymentsNoneUpdated(t *testing.T) {
	namespacedName := []types.NamespacedName{
		{
			Name:      "foo",
			Namespace: "bar",
		},
	}
	client := fake.NewFakeClientWithScheme(k8scheme.Scheme, &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "bar",
			Name:      "foo",
		},
		Status: appsv1.DeploymentStatus{
			AvailableReplicas: 0,
			Replicas:          1,
			UpdatedReplicas:   0,
		},
	})
	ready, err := DeploymentsAreReady(client, namespacedName, 1)
	assert.Error(t, err)
	assert.EqualError(t, err, "waiting for deployment bar/foo replicas to be 1, current updated replicas is 0")
	assert.False(t, ready)
}

// TestMultipleReplicasReadyBelowThreshold tests a deployment ready status check
// GIVEN a call validate DeploymentsReady for more than one replica
// WHEN the target Deployment object has less than the minimum desired replicas available
// THEN false is returned
func TestMultipleReplicasReadyBelowThreshold(t *testing.T) {
	selector := &metav1.LabelSelector{
		MatchLabels: map[string]string{
			"app": "foo",
		},
	}
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
			Spec: appsv1.DeploymentSpec{
				Selector: selector,
			},
			Status: appsv1.DeploymentStatus{
				AvailableReplicas: 3,
				Replicas:          3,
				UpdatedReplicas:   3,
			},
		},
		&corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: "bar",
				Name:      "foo-95d8c5d96-m6mbr",
				Labels: map[string]string{
					podTemplateHashLabel: "95d8c5d96",
					"app":                "foo",
				},
			},
			Status: corev1.PodStatus{
				ContainerStatuses: []corev1.ContainerStatus{
					{
						Ready: true,
					},
				},
			},
		},
		&appsv1.ReplicaSet{
			ObjectMeta: metav1.ObjectMeta{
				Namespace:   "bar",
				Name:        "foo-95d8c5d96",
				Annotations: map[string]string{deploymentRevisionAnnotation: "1"},
			},
		},
	)
	ready, err := DeploymentsAreReady(client, namespacedName, 3)
	assert.Error(t, err)
	assert.EqualError(t, err, "waiting for deployment bar/foo pods to be 3, current available pods are 1")
	assert.False(t, ready)
}

// TestDeploymentsReadyDeploymentNotFound tests a deployment ready status check
// GIVEN a call validate DeploymentsReady
// WHEN the target Deployment object is not found
// THEN false is returned
func TestDeploymentsReadyDeploymentNotFound(t *testing.T) {
	namespacedName := []types.NamespacedName{
		{
			Name:      "foo",
			Namespace: "bar",
		},
	}
	client := fake.NewFakeClientWithScheme(k8scheme.Scheme)
	ready, err := DeploymentsAreReady(client, namespacedName, 1)
	assert.Error(t, err)
	assert.EqualError(t, err, "waiting for deployment bar/foo to exist")
	assert.False(t, ready)
}

// TestDeploymentsReadyReplicaSetNotFound tests a deployment ready status check
// GIVEN a call validate DeploymentsReady
// WHEN the target ReplicaSet object is not found
// THEN false is returned
func TestDeploymentsReadyReplicaSetNotFound(t *testing.T) {
	selector := &metav1.LabelSelector{
		MatchLabels: map[string]string{
			"app": "foo",
		},
	}
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
			Spec: appsv1.DeploymentSpec{
				Selector: selector,
			},
			Status: appsv1.DeploymentStatus{
				AvailableReplicas: 1,
				Replicas:          1,
				UpdatedReplicas:   1,
			},
		},
		&corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: "bar",
				Name:      "foo-95d8c5d96-m6mbr",
				Labels: map[string]string{
					podTemplateHashLabel: "95d8c5d96",
					"app":                "foo",
				},
			},
			Status: corev1.PodStatus{
				ContainerStatuses: []corev1.ContainerStatus{
					{
						Ready: true,
					},
				},
			},
		},
	)
	ready, err := DeploymentsAreReady(client, namespacedName, 1)
	assert.Error(t, err)
	assert.EqualError(t, err, "failed to get replicaset bar/foo")
	assert.False(t, ready)
}
