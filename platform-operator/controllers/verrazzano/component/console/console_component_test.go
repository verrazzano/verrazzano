// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package console

import (
	"github.com/stretchr/testify/assert"
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8scheme "k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"testing"
)

var (
	disabled              = false
	testVZConsoleEnabled  = vzapi.Verrazzano{}
	testVZConsoleDisabled = vzapi.Verrazzano{
		Spec: vzapi.VerrazzanoSpec{
			Components: vzapi.ComponentSpec{
				Console: &vzapi.ConsoleComponent{
					Enabled: &disabled,
				},
			},
		},
	}
)

func createTestDeploy(name string, replicas int32) *appsv1.Deployment {
	return &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: ComponentNamespace,
			Name:      name,
			Labels:    map[string]string{"app": "coherence"},
		},
		Spec: appsv1.DeploymentSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{"app": "coherence"},
			},
		},
		Status: appsv1.DeploymentStatus{
			ReadyReplicas:     replicas,
			UpdatedReplicas:   replicas,
			AvailableReplicas: replicas,
		},
	}
}

func createTestPod(name string) *corev1.Pod {
	return &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: ComponentNamespace,
			Name:      name + "-95d8c5d96-m6mbr",
			Labels: map[string]string{
				"pod-template-hash": "95d8c5d96",
				"app":               "coherence",
			},
		},
	}
}

func createTestReplicaSet(name string) *appsv1.ReplicaSet {
	return &appsv1.ReplicaSet{
		ObjectMeta: metav1.ObjectMeta{
			Namespace:   ComponentNamespace,
			Name:        name + "-95d8c5d96",
			Annotations: map[string]string{"deployment.kubernetes.io/revision": "1"},
		},
	}
}

func TestValidateUpdate(t *testing.T) {
	c := NewComponent()
	var tests = []struct {
		name     string
		old      *vzapi.Verrazzano
		new      *vzapi.Verrazzano
		hasError bool
	}{
		{
			"allow update when going from enabled -> enabled",
			&testVZConsoleEnabled,
			&testVZConsoleEnabled,
			false,
		},
		{
			"allow update when going from disabled -> enabled",
			&testVZConsoleDisabled,
			&testVZConsoleEnabled,
			false,
		},
		{
			"allow update when going from disabled -> disabled",
			&testVZConsoleDisabled,
			&testVZConsoleDisabled,
			false,
		},
		{
			"allow update when going from enabled -> disabled",
			&testVZConsoleEnabled,
			&testVZConsoleDisabled,
			true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := c.ValidateUpdate(tt.old, tt.new); (err != nil) != tt.hasError {
				t.Errorf("c.ValidateUpdate() error: %v", err)
			}
		})
	}
}

func TestIsReady(t *testing.T) {
	comp := NewComponent()
	var tests = []struct {
		name       string
		deployment client.Object
		pod        client.Object
		replicaSet client.Object
		ready      bool
	}{
		{
			"ready when console deploy is ready",
			createTestDeploy(ComponentName, 1),
			createTestPod(ComponentName),
			createTestReplicaSet(ComponentName),
			true,
		},
		{
			"not ready when console deploy is not ready",
			createTestDeploy(ComponentName, 0),
			nil,
			nil,
			false,
		},
		{
			"not ready when console deploy isn't present",
			createTestDeploy("blah", 1),
			nil,
			nil,
			false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var c client.WithWatch
			if tt.ready {
				c = fake.NewClientBuilder().
					WithScheme(k8scheme.Scheme).
					WithObjects(tt.deployment, tt.pod, tt.replicaSet).
					Build()
			} else {
				c = fake.NewClientBuilder().WithScheme(k8scheme.Scheme).
					WithObjects(tt.deployment).
					Build()
			}
			ctx := spi.NewFakeContext(c, &vzapi.Verrazzano{}, nil, true)
			assert.Equal(t, tt.ready, comp.IsReady(ctx))
		})
	}
}
