// Copyright (c) 2021, 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
package oam

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	k8scheme "k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

var crEnabled = vzapi.Verrazzano{
	Spec: vzapi.VerrazzanoSpec{
		Components: vzapi.ComponentSpec{
			OAM: &vzapi.OAMComponent{
				Enabled: getBoolPtr(true),
			},
		},
	},
}

// TestIsOAMOperatorReady tests the isOAMReady function
// GIVEN a call to isOAMReady
//
//	WHEN the deployment object has enough replicas available
//	THEN true is returned
func TestIsOAMOperatorReady(t *testing.T) {

	fakeClient := fake.NewClientBuilder().WithScheme(k8scheme.Scheme).WithObjects(
		&appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: ComponentNamespace,
				Name:      ComponentName,
				Labels:    map[string]string{"app.kubernetes.io/name": "oam-kubernetes-runtime"},
			},
			Spec: appsv1.DeploymentSpec{
				Selector: &metav1.LabelSelector{
					MatchLabels: map[string]string{"app.kubernetes.io/name": "oam-kubernetes-runtime"},
				},
			},
			Status: appsv1.DeploymentStatus{
				AvailableReplicas: 1,
				Replicas:          1,
				UpdatedReplicas:   1,
			},
		},
		&corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: ComponentNamespace,
				Name:      ComponentName + "-95d8c5d96-m6mbr",
				Labels: map[string]string{
					"pod-template-hash":      "95d8c5d96",
					"app.kubernetes.io/name": "oam-kubernetes-runtime",
				},
			},
		},
		&appsv1.ReplicaSet{
			ObjectMeta: metav1.ObjectMeta{
				Namespace:   ComponentNamespace,
				Name:        ComponentName + "-95d8c5d96",
				Annotations: map[string]string{"deployment.kubernetes.io/revision": "1"},
			},
		},
	).Build()
	oam := NewComponent().(oamComponent)
	assert.True(t, oam.isOAMReady(spi.NewFakeContext(fakeClient, &vzapi.Verrazzano{}, nil, false)))
}

// TestIsOAMOperatorNotReady tests the isOAMReady function
// GIVEN a call to isOAMReady
//
//	WHEN the deployment object does NOT have enough replicas available
//	THEN false is returned
func TestIsOAMOperatorNotReady(t *testing.T) {

	fakeClient := fake.NewClientBuilder().WithScheme(k8scheme.Scheme).WithObjects(&appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: ComponentNamespace,
			Name:      ComponentName,
		},
		Status: appsv1.DeploymentStatus{
			AvailableReplicas: 1,
			Replicas:          1,
			UpdatedReplicas:   0,
		},
	}).Build()
	oam := NewComponent().(oamComponent)
	assert.False(t, oam.isOAMReady(spi.NewFakeContext(fakeClient, &vzapi.Verrazzano{}, nil, false)))
}

// TestIsEnabledNilOAM tests the IsEnabled function
// GIVEN a call to IsEnabled
//
//	WHEN The OAM component is nil
//	THEN true is returned
func TestIsEnabledNilOAM(t *testing.T) {
	cr := crEnabled
	cr.Spec.Components.OAM = nil
	assert.True(t, NewComponent().IsEnabled(&cr))
}

// TestIsEnabledNilComponent tests the IsEnabled function
// GIVEN a call to IsEnabled
//
//	WHEN The OAM component is nil
//	THEN false is returned
func TestIsEnabledNilComponent(t *testing.T) {
	assert.True(t, NewComponent().IsEnabled(&vzapi.Verrazzano{}))
}

// TestIsEnabledNilEnabled tests the IsEnabled function
// GIVEN a call to IsEnabled
//
//	WHEN The OAM component enabled is nil
//	THEN true is returned
func TestIsEnabledNilEnabled(t *testing.T) {
	cr := crEnabled
	cr.Spec.Components.OAM.Enabled = nil
	assert.True(t, NewComponent().IsEnabled(&cr))
}

// TestIsEnabledExplicit tests the IsEnabled function
// GIVEN a call to IsEnabled
//
//	WHEN The OAM component is explicitly enabled
//	THEN true is returned
func TestIsEnabledExplicit(t *testing.T) {
	cr := crEnabled
	cr.Spec.Components.OAM.Enabled = getBoolPtr(true)
	assert.True(t, NewComponent().IsEnabled(&cr))
}

// TestIsDisableExplicit tests the IsEnabled function
// GIVEN a call to IsEnabled
//
//	WHEN The OAM component is explicitly disabled
//	THEN false is returned
func TestIsDisableExplicit(t *testing.T) {
	cr := crEnabled
	cr.Spec.Components.OAM.Enabled = getBoolPtr(false)
	assert.False(t, NewComponent().IsEnabled(&cr))
}

func getBoolPtr(b bool) *bool {
	return &b
}

// TestEnsureClusterRoles tests the ensureClusterRoles function
func TestEnsureClusterRoles(t *testing.T) {
	// GIVEN a call to ensureClusterRoles
	// WHEN the cluster roles do not exist
	// THEN the cluster roles are created
	client := fake.NewClientBuilder().WithScheme(testScheme).Build()
	ctx := spi.NewFakeContext(client, nil, nil, false)
	ensureClusterRoles(ctx)

	var clusterRole rbacv1.ClusterRole
	err := client.Get(context.TODO(), types.NamespacedName{Name: pvcClusterRoleName}, &clusterRole)
	assert.NoError(t, err)
	assert.Equal(t, "true", clusterRole.Labels[aggregateToControllerLabel])

	// GIVEN a call to ensureClusterRoles
	// WHEN the cluster roles already exist
	// THEN the cluster roles are updated
	client = fake.NewClientBuilder().WithScheme(testScheme).WithObjects(
		&rbacv1.ClusterRole{
			ObjectMeta: metav1.ObjectMeta{
				Name: pvcClusterRoleName,
			},
			Rules: []rbacv1.PolicyRule{
				{
					Resources: []string{"deployments"},
				},
			},
		},
	).Build()
	ctx = spi.NewFakeContext(client, nil, nil, false)
	ensureClusterRoles(ctx)

	err = client.Get(context.TODO(), types.NamespacedName{Name: pvcClusterRoleName}, &clusterRole)
	assert.NoError(t, err)
	assert.Equal(t, "true", clusterRole.Labels[aggregateToControllerLabel])
	assert.Equal(t, 1, len(clusterRole.Rules))
	assert.Equal(t, 1, len(clusterRole.Rules[0].Resources))
	assert.Equal(t, "persistentvolumeclaims", clusterRole.Rules[0].Resources[0])
}
