// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package argocd

import (
	"github.com/stretchr/testify/assert"
	"github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1beta1"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/common"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"testing"
)

var (
	falseValue = false
	trueValue  = true
)

// TestIsReady verifies ArgoCD is enabled or disabled as expected
// GIVEN a Verrzzano CR
//
//	WHEN IsEnabled is called
//	THEN IsEnabled should return true/false depending on the enabled state of the CR
func TestIsEnabled(t *testing.T) {
	enabled := true
	disabled := false
	c := fake.NewClientBuilder().WithScheme(getScheme()).Build()
	vzWithArgoCD := vzapi.Verrazzano{
		Spec: vzapi.VerrazzanoSpec{
			Components: vzapi.ComponentSpec{
				ArgoCD: &vzapi.ArgoCDComponent{
					Enabled: &enabled,
				},
			},
		},
	}
	vzNoArgoCD := vzapi.Verrazzano{
		Spec: vzapi.VerrazzanoSpec{
			Components: vzapi.ComponentSpec{
				ArgoCD: &vzapi.ArgoCDComponent{
					Enabled: &disabled,
				},
			},
		},
	}
	var tests = []struct {
		testName string
		ctx      spi.ComponentContext
		enabled  bool
	}{
		{
			"should be enabled",
			spi.NewFakeContext(c, &vzWithArgoCD, nil, false),
			true,
		},
		{
			"should not be enabled",
			spi.NewFakeContext(c, &vzNoArgoCD, nil, false),
			false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.testName, func(t *testing.T) {
			r := NewComponent()
			assert.Equal(t, tt.enabled, r.IsEnabled(tt.ctx.EffectiveCR()))
		})
	}
}

// TestIsReady verifies that a ready-state ArgoCD shows as ready
// GIVEN a ready ArgoCD install
//
//	WHEN IsReady is called
//	THEN IsReady should return true
func TestIsReady(t *testing.T) {
	readyClient := fake.NewClientBuilder().WithScheme(getScheme()).WithObjects(
		newReadyDeployment(ComponentNamespace, ComponentName),
		newPod(ComponentNamespace, ComponentName),
		newReplicaSet(ComponentNamespace, ComponentName),
		newReadyDeployment(ComponentNamespace, common.ArgoCDApplicationSetController),
		newPod(ComponentNamespace, common.ArgoCDApplicationSetController),
		newReplicaSet(ComponentNamespace, common.ArgoCDApplicationSetController),
		newReadyDeployment(ComponentNamespace, common.ArgoCDDexServer),
		newPod(ComponentNamespace, common.ArgoCDDexServer),
		newReplicaSet(ComponentNamespace, common.ArgoCDDexServer),
		newReadyDeployment(ComponentNamespace, common.ArgoCDNotificationController),
		newPod(ComponentNamespace, common.ArgoCDNotificationController),
		newReplicaSet(ComponentNamespace, common.ArgoCDNotificationController),
		newReadyDeployment(ComponentNamespace, common.ArgoCDRedis),
		newPod(ComponentNamespace, common.ArgoCDRedis),
		newReplicaSet(ComponentNamespace, common.ArgoCDRedis),
		newReadyDeployment(ComponentNamespace, common.ArgoCDRepoServer),
		newPod(ComponentNamespace, common.ArgoCDRepoServer),
		newReplicaSet(ComponentNamespace, common.ArgoCDRepoServer),
		newReadyDeployment(ComponentNamespace, common.ArgoCDServer),
		newPod(ComponentNamespace, common.ArgoCDServer),
		newReplicaSet(ComponentNamespace, common.ArgoCDServer),
	).Build()
	unreadyDeployClient := fake.NewClientBuilder().WithScheme(getScheme()).WithObjects(
		&appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: ComponentNamespace,
				Name:      ComponentName,
			},
			Status: appsv1.DeploymentStatus{
				AvailableReplicas: 0,
				Replicas:          1,
			},
		},
		&appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: ComponentNamespace,
				Name:      common.ArgoCDApplicationSetController,
			},
			Status: appsv1.DeploymentStatus{
				AvailableReplicas: 0,
				Replicas:          1,
			},
		},
		&appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: ComponentNamespace,
				Name:      common.ArgoCDDexServer,
			},
			Status: appsv1.DeploymentStatus{
				AvailableReplicas: 0,
				Replicas:          1,
			},
		},
		&appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: ComponentNamespace,
				Name:      common.ArgoCDNotificationController,
			},
			Status: appsv1.DeploymentStatus{
				AvailableReplicas: 0,
				Replicas:          1,
			},
		},
		&appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: ComponentNamespace,
				Name:      common.ArgoCDRedis,
			},
			Status: appsv1.DeploymentStatus{
				AvailableReplicas: 0,
				Replicas:          1,
			},
		},
		&appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: ComponentNamespace,
				Name:      common.ArgoCDRepoServer,
			},
			Status: appsv1.DeploymentStatus{
				AvailableReplicas: 0,
				Replicas:          1,
			},
		},
		&appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: ComponentNamespace,
				Name:      common.ArgoCDServer,
			},
			Status: appsv1.DeploymentStatus{
				AvailableReplicas: 0,
				Replicas:          1,
			},
		},
	).Build()

	var tests = []struct {
		testName string
		ctx      spi.ComponentContext
		isReady  bool
	}{
		{
			"should be ready",
			spi.NewFakeContext(readyClient, &vzDefaultCA, nil, true),
			true,
		},
		{
			"should not be ready due to deployment",
			spi.NewFakeContext(unreadyDeployClient, &vzDefaultCA, nil, true),
			false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.testName, func(t *testing.T) {
			assert.Equal(t, tt.isReady, NewComponent().IsReady(tt.ctx))
		})
	}
}

func newReadyDeployment(namespace string, name string) *appsv1.Deployment {
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
			AvailableReplicas: 1,
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

// TestValidateUpdate tests the ArgoCD component ValidateUpdate function
func TestValidateUpdate(t *testing.T) {
	// GIVEN an old VZ with ArgoCD enabled and a new VZ with ArgoCD disabled
	// WHEN we call the ValidateUpdate function
	// THEN the function returns an error
	oldVz := &vzapi.Verrazzano{
		Spec: vzapi.VerrazzanoSpec{
			Components: vzapi.ComponentSpec{
				ArgoCD: &vzapi.ArgoCDComponent{
					Enabled: &trueValue,
				},
			},
		},
	}

	newVz := &vzapi.Verrazzano{
		Spec: vzapi.VerrazzanoSpec{
			Components: vzapi.ComponentSpec{
				ArgoCD: &vzapi.ArgoCDComponent{
					Enabled: &falseValue,
				},
			},
		},
	}

	assert.Error(t, NewComponent().ValidateUpdate(oldVz, newVz))

	// GIVEN an old VZ with ArgoCD enabled and a new VZ with ArgoCD enabled
	// WHEN we call the ValidateUpdate function
	// THEN the function does not return an error
	newVz.Spec.Components.ArgoCD.Enabled = &trueValue
	assert.NoError(t, NewComponent().ValidateUpdate(oldVz, newVz))
}

// TestValidateUpdateV1beta1 tests the ArgoCD component ValidateUpdate function
func TestValidateUpdateV1beta1(t *testing.T) {
	// GIVEN an old VZ with ArgoCD enabled and a new VZ with ArgoCD disabled
	// WHEN we call the ValidateUpdate function
	// THEN the function returns an error
	oldVz := &v1beta1.Verrazzano{
		Spec: v1beta1.VerrazzanoSpec{
			Components: v1beta1.ComponentSpec{
				ArgoCD: &v1beta1.ArgoCDComponent{
					Enabled: &trueValue,
				},
			},
		},
	}

	newVz := &v1beta1.Verrazzano{
		Spec: v1beta1.VerrazzanoSpec{
			Components: v1beta1.ComponentSpec{
				ArgoCD: &v1beta1.ArgoCDComponent{
					Enabled: &falseValue,
				},
			},
		},
	}

	assert.Error(t, NewComponent().ValidateUpdateV1Beta1(oldVz, newVz))

	// GIVEN an old VZ with ArgoCD enabled and a new VZ with ArgoCD enabled
	// WHEN we call the ValidateUpdate function
	// THEN the function does not return an error
	newVz.Spec.Components.ArgoCD.Enabled = &trueValue
	assert.NoError(t, NewComponent().ValidateUpdateV1Beta1(oldVz, newVz))
}

func TestValidateInstall(t *testing.T) {
	tests := []struct {
		name    string
		vz      *vzapi.Verrazzano
		wantErr bool
	}{
		{
			name: "ArgoCDComponent empty",
			vz: &v1alpha1.Verrazzano{
				Spec: v1alpha1.VerrazzanoSpec{
					Components: v1alpha1.ComponentSpec{
						ArgoCD: &v1alpha1.ArgoCDComponent{},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "ArgoCDComponent enabled",
			vz: &v1alpha1.Verrazzano{
				Spec: v1alpha1.VerrazzanoSpec{
					Components: v1alpha1.ComponentSpec{
						ArgoCD: &v1alpha1.ArgoCDComponent{
							Enabled: &trueValue,
						},
					},
				},
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := NewComponent()
			if err := c.ValidateInstall(tt.vz); (err != nil) != tt.wantErr {
				t.Errorf("ValidateInstall() error = %v, wantErr %v", err, tt.wantErr)
			}
			v1beta1Vz := &v1beta1.Verrazzano{}
			err := tt.vz.ConvertTo(v1beta1Vz)
			assert.NoError(t, err)
			if err := c.ValidateInstallV1Beta1(v1beta1Vz); (err != nil) != tt.wantErr {
				t.Errorf("ValidateInstallV1Beta1() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
