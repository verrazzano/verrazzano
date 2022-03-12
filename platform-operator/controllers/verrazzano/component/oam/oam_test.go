// Copyright (c) 2021, 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
package oam

import (
	"testing"

	"github.com/stretchr/testify/assert"
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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
//  WHEN the deployment object has enough replicas available
//  THEN true is returned
func TestIsOAMOperatorReady(t *testing.T) {

	fakeClient := fake.NewFakeClientWithScheme(k8scheme.Scheme, &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: ComponentNamespace,
			Name:      ComponentName,
			Labels:    map[string]string{"app.kubernetes.io/name": "oam-kubernetes-runtime"},
		},
		Status: appsv1.DeploymentStatus{
			AvailableReplicas: 1,
			Replicas:          1,
			UpdatedReplicas:   1,
		},
	})
	assert.True(t, isOAMReady(spi.NewFakeContext(fakeClient, &vzapi.Verrazzano{}, false)))
}

// TestIsOAMOperatorNotReady tests the isOAMReady function
// GIVEN a call to isOAMReady
//  WHEN the deployment object does NOT have enough replicas available
//  THEN false is returned
func TestIsOAMOperatorNotReady(t *testing.T) {

	fakeClient := fake.NewFakeClientWithScheme(k8scheme.Scheme, &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: ComponentNamespace,
			Name:      ComponentName,
		},
		Status: appsv1.DeploymentStatus{
			AvailableReplicas: 1,
			Replicas:          1,
			UpdatedReplicas:   0,
		},
	})
	assert.False(t, isOAMReady(spi.NewFakeContext(fakeClient, &vzapi.Verrazzano{}, false)))
}

// TestIsEnabledNilOAM tests the IsEnabled function
// GIVEN a call to IsEnabled
//  WHEN The OAM component is nil
//  THEN true is returned
func TestIsEnabledNilOAM(t *testing.T) {
	cr := crEnabled
	cr.Spec.Components.OAM = nil
	assert.True(t, NewComponent().IsEnabled(&cr))
}

// TestIsEnabledNilComponent tests the IsEnabled function
// GIVEN a call to IsEnabled
//  WHEN The OAM component is nil
//  THEN false is returned
func TestIsEnabledNilComponent(t *testing.T) {
	assert.True(t, NewComponent().IsEnabled(&vzapi.Verrazzano{}))
}

// TestIsEnabledNilEnabled tests the IsEnabled function
// GIVEN a call to IsEnabled
//  WHEN The OAM component enabled is nil
//  THEN true is returned
func TestIsEnabledNilEnabled(t *testing.T) {
	cr := crEnabled
	cr.Spec.Components.OAM.Enabled = nil
	assert.True(t, NewComponent().IsEnabled(&cr))
}

// TestIsEnabledExplicit tests the IsEnabled function
// GIVEN a call to IsEnabled
//  WHEN The OAM component is explicitly enabled
//  THEN true is returned
func TestIsEnabledExplicit(t *testing.T) {
	cr := crEnabled
	cr.Spec.Components.OAM.Enabled = getBoolPtr(true)
	assert.True(t, NewComponent().IsEnabled(&cr))
}

// TestIsDisableExplicit tests the IsEnabled function
// GIVEN a call to IsEnabled
//  WHEN The OAM component is explicitly disabled
//  THEN false is returned
func TestIsDisableExplicit(t *testing.T) {
	cr := crEnabled
	cr.Spec.Components.OAM.Enabled = getBoolPtr(false)
	assert.False(t, NewComponent().IsEnabled(&cr))
}

func getBoolPtr(b bool) *bool {
	return &b
}
