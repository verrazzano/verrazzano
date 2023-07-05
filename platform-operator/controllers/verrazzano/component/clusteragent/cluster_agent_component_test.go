// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package clusteragent

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

// TestIsEnabled tests the IsEnabled function for the Cluster Agent
// GIVEN a Verrazzano CR
//
//	WHEN I call IsEnabled
//	THEN the function returns true if the cluster agent is enabled in the Verrazzano CR
func TestIsEnabled(t *testing.T) {
	a := assert.New(t)
	trueVal := true
	falseVal := false
	vzEmpty := &v1alpha1.Verrazzano{}
	vzEnabled := vzEmpty.DeepCopy()
	vzEnabled.Spec.Components.ClusterAgent = &v1alpha1.ClusterAgentComponent{
		Enabled: &trueVal,
	}
	vzDisabled := vzEmpty.DeepCopy()
	vzDisabled.Spec.Components.ClusterAgent = &v1alpha1.ClusterAgentComponent{
		Enabled: &falseVal,
	}

	component := NewComponent()
	a.True(component.IsEnabled(vzEmpty), "Expected empty cluster agent to return true")
	a.True(component.IsEnabled(vzEnabled), "Expected enabled cluster agent to return true")
	a.False(component.IsEnabled(vzDisabled), "Expected disabled cluster agent to return false")
}

// Test isReady when it's called with component context
func TestIsReady(t *testing.T) {
	scheme := runtime.NewScheme()
	_ = corev1.AddToScheme(scheme)
	_ = appsv1.AddToScheme(scheme)
	tests := []struct {
		name        string
		objects     []client.Object
		expectReady bool
	}{
		{"ready", []client.Object{
			newDeployment(ComponentNamespace, ComponentName, 1),
			newPod(ComponentNamespace, ComponentName),
			newReplicaSet(ComponentNamespace, ComponentName)}, true},

		{"not enough replicas", []client.Object{
			newDeployment(ComponentNamespace, ComponentName, 0),
			newPod(ComponentNamespace, ComponentName),
			newReplicaSet(ComponentNamespace, ComponentName)}, false},

		{"no pods", []client.Object{
			newDeployment(ComponentNamespace, ComponentName, 1),
			newReplicaSet(ComponentNamespace, ComponentName)}, false},

		{"no replicaset", []client.Object{
			newDeployment(ComponentNamespace, ComponentName, 1),
			newPod(ComponentNamespace, ComponentName)}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := fake.NewClientBuilder().WithScheme(scheme).WithObjects(tt.objects...).Build()
			ctx := spi.NewFakeContext(c, &v1alpha1.Verrazzano{}, nil, true)
			fmt.Printf("Expecting ready=%v\n", tt.expectReady)
			assert.Equal(t, tt.expectReady, NewComponent().IsReady(ctx))
		})
	}
}

func newDeployment(namespace string, name string, availableReplicas int32) *appsv1.Deployment {
	return &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: namespace,
			Name:      name,
			Labels:    map[string]string{"app": name},
		},
		Spec: appsv1.DeploymentSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{"app": name},
			},
		},
		Status: appsv1.DeploymentStatus{
			AvailableReplicas: availableReplicas,
			Replicas:          1,
			UpdatedReplicas:   1,
		},
	}
}

func newPod(namespace string, name string) *corev1.Pod {
	return &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: namespace,
			Name:      name + "-95d8c5d96-m6mbr",
			Labels: map[string]string{
				"pod-template-hash": "95d8c5d96",
				"app":               name,
			},
		},
	}
}

func newReplicaSet(namespace string, name string) *appsv1.ReplicaSet {
	return &appsv1.ReplicaSet{
		ObjectMeta: metav1.ObjectMeta{
			Namespace:   namespace,
			Name:        name + "-95d8c5d96",
			Annotations: map[string]string{"deployment.kubernetes.io/revision": "1"},
		},
	}
}
