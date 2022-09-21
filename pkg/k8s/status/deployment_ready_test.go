// Copyright (c) 2021, 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
package status

import (
	"sigs.k8s.io/controller-runtime/pkg/client"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/verrazzano/verrazzano/pkg/log/vzlog"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	k8scheme "k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

var testReadyDeployment = &appsv1.Deployment{
	ObjectMeta: metav1.ObjectMeta{
		Namespace: "bar",
		Name:      "foo",
		Labels:    map[string]string{"app": "foo"},
	},
	Spec: appsv1.DeploymentSpec{
		Selector: &metav1.LabelSelector{
			MatchLabels: map[string]string{"app": "foo"},
		},
	},
	Status: appsv1.DeploymentStatus{
		AvailableReplicas: 1,
		Replicas:          1,
		UpdatedReplicas:   1,
	},
}
var testReadyPod = &corev1.Pod{
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
}
var testReadyReplicaSet = &appsv1.ReplicaSet{
	ObjectMeta: metav1.ObjectMeta{
		Namespace:   "bar",
		Name:        "foo-95d8c5d96",
		Annotations: map[string]string{deploymentRevisionAnnotation: "1"},
	},
}

func TestDeploymentsReadyBySelectors(t *testing.T) {
	opts := []client.ListOption{
		client.MatchingLabels{
			"app": "foo",
		},
	}
	log := vzlog.DefaultLogger()
	var tests = []struct {
		name  string
		c     client.Client
		opts  []client.ListOption
		ready bool
	}{
		{
			"not ready when no matching deployments",
			fake.NewClientBuilder().WithScheme(k8scheme.Scheme).Build(),
			[]client.ListOption{},
			false,
		},
		{
			"not ready when matched deployment is not ready",
			fake.NewClientBuilder().WithScheme(k8scheme.Scheme).
				WithObjects(testReadyDeployment).
				Build(),
			opts,
			false,
		},
		{
			"ready when matched deployment is ready",
			fake.NewClientBuilder().WithScheme(k8scheme.Scheme).
				WithObjects(testReadyDeployment, testReadyPod, testReadyReplicaSet).
				Build(),
			opts,
			true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ready := DeploymentsReadyBySelectors(log, tt.c, 1, "foo", tt.opts...)
			assert.Equal(t, tt.ready, ready)
		})
	}
}

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
	fakeClient := fake.NewClientBuilder().WithScheme(k8scheme.Scheme).WithObjects(
		testReadyDeployment,
		testReadyPod,
		testReadyReplicaSet,
	).Build()
	assert.True(t, DeploymentsAreReady(vzlog.DefaultLogger(), fakeClient, namespacedName, 1, ""))
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
	fakeClient := fake.NewClientBuilder().WithScheme(k8scheme.Scheme).WithObjects(
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
	).Build()
	assert.False(t, DeploymentsAreReady(vzlog.DefaultLogger(), fakeClient, namespacedName, 1, ""))
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
	fakeClient := fake.NewClientBuilder().WithScheme(k8scheme.Scheme).WithObjects(
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
	).Build()
	assert.False(t, DeploymentsAreReady(vzlog.DefaultLogger(), fakeClient, namespacedName, 1, ""))
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
	fakeClient := fake.NewClientBuilder().WithScheme(k8scheme.Scheme).WithObjects(
		&appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: "bar",
				Name:      "foo",
				Labels:    map[string]string{"app": "foo"},
			},
			Spec: appsv1.DeploymentSpec{
				Selector: &metav1.LabelSelector{
					MatchLabels: map[string]string{"app": "foo"},
				},
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
	).Build()
	assert.True(t, DeploymentsAreReady(vzlog.DefaultLogger(), fakeClient, namespacedName, 2, ""))
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
	fakeClient := fake.NewClientBuilder().WithScheme(k8scheme.Scheme).WithObjects(
		&appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: "bar",
				Name:      "foo",
				Labels:    map[string]string{"app": "foo"},
			},
			Spec: appsv1.DeploymentSpec{
				Selector: &metav1.LabelSelector{
					MatchLabels: map[string]string{"app": "foo"},
				},
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
	).Build()
	assert.True(t, DeploymentsAreReady(vzlog.DefaultLogger(), fakeClient, namespacedName, 1, ""))
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
	fakeClient := fake.NewClientBuilder().WithScheme(k8scheme.Scheme).WithObjects(
		&appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: "bar",
				Name:      "foo",
			},
			Status: appsv1.DeploymentStatus{
				AvailableReplicas: 0,
				Replicas:          1,
				UpdatedReplicas:   1,
			},
		},
	).Build()
	assert.False(t, DeploymentsAreReady(vzlog.DefaultLogger(), fakeClient, namespacedName, 1, ""))
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
	fakeClient := fake.NewClientBuilder().WithScheme(k8scheme.Scheme).WithObjects(
		&appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: "bar",
				Name:      "foo",
			},
			Status: appsv1.DeploymentStatus{
				AvailableReplicas: 0,
				Replicas:          1,
				UpdatedReplicas:   0,
			},
		},
	).Build()
	assert.False(t, DeploymentsAreReady(vzlog.DefaultLogger(), fakeClient, namespacedName, 1, ""))
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
	fakeClient := fake.NewClientBuilder().WithScheme(k8scheme.Scheme).WithObjects(
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
	).Build()
	assert.False(t, DeploymentsAreReady(vzlog.DefaultLogger(), fakeClient, namespacedName, 3, ""))
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
	fakeClient := fake.NewClientBuilder().WithScheme(k8scheme.Scheme).Build()
	assert.False(t, DeploymentsAreReady(vzlog.DefaultLogger(), fakeClient, namespacedName, 1, ""))
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
	fakeClient := fake.NewClientBuilder().WithScheme(k8scheme.Scheme).WithObjects(
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
	).Build()
	assert.False(t, DeploymentsAreReady(vzlog.DefaultLogger(), fakeClient, namespacedName, 1, ""))
}

// TestDeploymentsReadyPodNotFound tests a deployment ready status check
// GIVEN a call validate DeploymentsReady
// WHEN the target Pod object is not found
// THEN false is returned
func TestDeploymentsReadyPodNotFound(t *testing.T) {
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
	fakeClient := fake.NewClientBuilder().WithScheme(k8scheme.Scheme).WithObjects(
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
	).Build()

	assert.False(t, DeploymentsAreReady(vzlog.DefaultLogger(), fakeClient, namespacedName, 1, ""))
}
