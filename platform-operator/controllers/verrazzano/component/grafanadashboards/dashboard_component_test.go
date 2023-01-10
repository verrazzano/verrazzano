// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package grafanadashboards

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1beta1"
	"github.com/verrazzano/verrazzano/platform-operator/constants"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

var enabledFalse = false
var grafanaDisabledVZBeta = v1beta1.Verrazzano{
	Spec: v1beta1.VerrazzanoSpec{
		Components: v1beta1.ComponentSpec{
			Grafana: &v1beta1.GrafanaComponent{Enabled: &enabledFalse},
		},
	},
}

var grafanaDisabledVZAlpha = vzapi.Verrazzano{
	Spec: vzapi.VerrazzanoSpec{
		Components: vzapi.ComponentSpec{
			Grafana: &vzapi.GrafanaComponent{Enabled: &enabledFalse},
		},
	},
}

// GIVEN a Grafana dashboards helm component
//
//	WHEN the IsEnabled function is called
//	THEN the call returns true if Grafana is implicitly or explicitly enabled, false otherwise
func TestIsEnabled(t *testing.T) {
	comp := NewComponent()
	assert.True(t, comp.IsEnabled(nil))
	assert.False(t, comp.IsEnabled(&grafanaDisabledVZBeta))
	assert.False(t, comp.IsEnabled(&grafanaDisabledVZAlpha))
	grafanaEnabledBeta := grafanaDisabledVZBeta.DeepCopy()
	trueValue := true
	grafanaEnabledBeta.Spec.Components.Grafana.Enabled = &trueValue
	grafanaEnabledAlpha := grafanaDisabledVZAlpha.DeepCopy()
	grafanaEnabledAlpha.Spec.Components.Grafana.Enabled = &trueValue
	assert.True(t, comp.IsEnabled(grafanaEnabledBeta))
	assert.True(t, comp.IsEnabled(grafanaEnabledAlpha))
}

// TestPostInstall tests the component PostInstall function.
func TestPostInstall(t *testing.T) {
	// GIVEN a Grafana dashboards helm component and the legacy dashboards configmap exists
	//
	//	WHEN the PostInstall function is called
	//	THEN the call returns no error and the legacy configmap is deleted
	cm := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      legacyDashboardConfigMapName,
			Namespace: constants.VerrazzanoSystemNamespace,
		},
	}
	client := fake.NewClientBuilder().WithObjects(cm).Build()
	ctx := spi.NewFakeContext(client, nil, nil, false)
	err := NewComponent().PostInstall(ctx)
	assert.NoError(t, err)

	// confirm that the configmap was deleted
	err = client.Get(context.TODO(), types.NamespacedName{Namespace: constants.VerrazzanoSystemNamespace, Name: legacyDashboardConfigMapName}, &corev1.ConfigMap{})
	assert.True(t, errors.IsNotFound(err), "Expected to get a NotFound error")

	// GIVEN a Grafana dashboards helm component and the legacy dashboards configmap does NOT exist
	//
	//	WHEN the PostInstall function is called
	//	THEN the call returns no error
	client = fake.NewClientBuilder().Build()
	ctx = spi.NewFakeContext(client, nil, nil, false)
	err = NewComponent().PostInstall(ctx)
	assert.NoError(t, err)
}

// TestPostUpgrade tests the component PostUpgrade function.
func TestPostUpgrade(t *testing.T) {
	// GIVEN a Grafana dashboards helm component and the legacy dashboards configmap exists
	//
	//	WHEN the PostUpgrade function is called
	//	THEN the call returns no error and the legacy configmap is deleted
	cm := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      legacyDashboardConfigMapName,
			Namespace: constants.VerrazzanoSystemNamespace,
		},
	}
	client := fake.NewClientBuilder().WithObjects(cm).Build()
	ctx := spi.NewFakeContext(client, nil, nil, false)
	err := NewComponent().PostUpgrade(ctx)
	assert.NoError(t, err)

	// confirm that the configmap was deleted
	err = client.Get(context.TODO(), types.NamespacedName{Namespace: constants.VerrazzanoSystemNamespace, Name: legacyDashboardConfigMapName}, &corev1.ConfigMap{})
	assert.True(t, errors.IsNotFound(err), "Expected to get a NotFound error")

	// GIVEN a Grafana dashboards helm component and the legacy dashboards configmap does NOT exist
	//
	//	WHEN the PostUpgrade function is called
	//	THEN the call returns no error
	client = fake.NewClientBuilder().Build()
	ctx = spi.NewFakeContext(client, nil, nil, false)
	err = NewComponent().PostUpgrade(ctx)
	assert.NoError(t, err)
}
