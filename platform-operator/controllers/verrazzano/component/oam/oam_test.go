// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
package oam

import (
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/verrazzano/verrazzano/platform-operator/constants"
	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8scheme "k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

const profilesRelativePath = "../../../../manifests/profiles"

var crEnabled = vzapi.Verrazzano{
	Spec: vzapi.VerrazzanoSpec{
		Components: vzapi.ComponentSpec{
			OAM: &vzapi.OAMComponent{
				Enabled: getBoolPtr(true),
			},
		},
	},
}

// TestIsOAMOperatorReady tests the IsOAMReady function
// GIVEN a call to IsOAMReady
//  WHEN the deployment object has enough replicas available
//  THEN true is returned
func TestIsOAMOperatorReady(t *testing.T) {

	fakeClient := fake.NewFakeClientWithScheme(k8scheme.Scheme, &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: constants.VerrazzanoSystemNamespace,
			Name:      oamOperatorDeploymentName,
		},
		Status: appsv1.DeploymentStatus{
			Replicas:            1,
			ReadyReplicas:       1,
			AvailableReplicas:   1,
			UnavailableReplicas: 0,
		},
	})
	assert.True(t, IsOAMReady(spi.NewFakeContext(fakeClient, &vzapi.Verrazzano{}, false), "", constants.VerrazzanoSystemNamespace))
}

// TestIsOAMOperatorNotReady tests the IsOAMReady function
// GIVEN a call to IsOAMReady
//  WHEN the deployment object does NOT have enough replicas available
//  THEN false is returned
func TestIsOAMOperatorNotReady(t *testing.T) {

	fakeClient := fake.NewFakeClientWithScheme(k8scheme.Scheme, &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: constants.VerrazzanoSystemNamespace,
			Name:      oamOperatorDeploymentName,
		},
		Status: appsv1.DeploymentStatus{
			Replicas:            1,
			ReadyReplicas:       0,
			AvailableReplicas:   0,
			UnavailableReplicas: 1,
		},
	})
	assert.False(t, IsOAMReady(spi.NewFakeContext(fakeClient, &vzapi.Verrazzano{}, false), "", constants.VerrazzanoSystemNamespace))
}

// TestIsEnabledNilOAM tests the IsEnabled function
// GIVEN a call to IsEnabled
//  WHEN The OAM component is nil
//  THEN true is returned
func TestIsEnabledNilOAM(t *testing.T) {
	cr := crEnabled
	cr.Spec.Components.OAM = nil
	assert.True(t, NewComponent().IsEnabled(spi.NewFakeContext(nil, &cr, false, profilesRelativePath)))
}

// TestIsEnabledNilComponent tests the IsEnabled function
// GIVEN a call to IsEnabled
//  WHEN The OAM component is nil
//  THEN false is returned
func TestIsEnabledNilComponent(t *testing.T) {
	assert.True(t, NewComponent().IsEnabled(spi.NewFakeContext(nil, &vzapi.Verrazzano{}, false, profilesRelativePath)))
}

// TestIsEnabledNilEnabled tests the IsEnabled function
// GIVEN a call to IsEnabled
//  WHEN The OAM component enabled is nil
//  THEN true is returned
func TestIsEnabledNilEnabled(t *testing.T) {
	cr := crEnabled
	cr.Spec.Components.OAM.Enabled = nil
	assert.True(t, NewComponent().IsEnabled(spi.NewFakeContext(nil, &cr, false, profilesRelativePath)))
}

// TestIsEnabledExplicit tests the IsEnabled function
// GIVEN a call to IsEnabled
//  WHEN The OAM component is explicitly enabled
//  THEN true is returned
func TestIsEnabledExplicit(t *testing.T) {
	cr := crEnabled
	cr.Spec.Components.OAM.Enabled = getBoolPtr(true)
	assert.True(t, NewComponent().IsEnabled(spi.NewFakeContext(nil, &cr, false, profilesRelativePath)))
}

// TestIsDisableExplicit tests the IsEnabled function
// GIVEN a call to IsEnabled
//  WHEN The OAM component is explicitly disabled
//  THEN false is returned
func TestIsDisableExplicit(t *testing.T) {
	cr := crEnabled
	cr.Spec.Components.OAM.Enabled = getBoolPtr(false)
	assert.False(t, NewComponent().IsEnabled(spi.NewFakeContext(nil, &cr, false, profilesRelativePath)))
}

func getBoolPtr(b bool) *bool {
	return &b
}
