// Copyright (c) 2022, 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package pushgateway

import (
	"context"
	"testing"

	"github.com/verrazzano/verrazzano/pkg/bom"
	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	"github.com/stretchr/testify/assert"
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
)

const profilesRelativePath = "../../../../../manifests/profiles"

// TestIsEnabled tests the IsEnabled function for the Prometheus Pushgateway component
func TestIsEnabled(t *testing.T) {
	falseValue := false
	trueValue := true
	tests := []struct {
		name       string
		actualCR   vzapi.Verrazzano
		expectTrue bool
	}{
		{
			// GIVEN a default Verrazzano custom resource
			// WHEN we call IsReady on the Prometheus Pushgateway component
			// THEN the call returns true
			name:       "Test IsEnabled when using default Verrazzano CR",
			actualCR:   vzapi.Verrazzano{},
			expectTrue: false,
		},
		{
			// GIVEN a Verrazzano custom resource with the Prometheus Pushgateway enabled
			// WHEN we call IsReady on the Prometheus Pushgateway component
			// THEN the call returns true
			name: "Test IsEnabled when Pushgateway component set to enabled",
			actualCR: vzapi.Verrazzano{
				Spec: vzapi.VerrazzanoSpec{
					Components: vzapi.ComponentSpec{
						PrometheusPushgateway: &vzapi.PrometheusPushgatewayComponent{
							Enabled: &trueValue,
						},
					},
				},
			},
			expectTrue: true,
		},
		{
			// GIVEN a Verrazzano custom resource with the Prometheus Pushgateway having empty section
			// WHEN we call IsReady on the Prometheus Pushgateway component
			// THEN the call returns true
			name: "Test IsEnabled when Pushgateway components Monitoring component is left empty",
			actualCR: vzapi.Verrazzano{
				Spec: vzapi.VerrazzanoSpec{
					Components: vzapi.ComponentSpec{
						PrometheusPushgateway: &vzapi.PrometheusPushgatewayComponent{},
					},
				},
			},
			expectTrue: false,
		},
		{
			// GIVEN a Verrazzano custom resource with the Prometheus Pushgateway having empty section
			// WHEN we call IsReady on the Prometheus Pushgateway component
			// THEN the call returns true
			name: "Test IsEnabled when Pushgateway component is left empty",
			actualCR: vzapi.Verrazzano{
				Spec: vzapi.VerrazzanoSpec{
					Components: vzapi.ComponentSpec{
						PrometheusPushgateway: &vzapi.PrometheusPushgatewayComponent{},
					},
				},
			},
			expectTrue: false,
		},
		{
			// GIVEN a Verrazzano custom resource with the Prometheus Pushgateway disabled
			// WHEN we call IsReady on the Prometheus Pushgateway component
			// THEN the call returns false
			name: "Test IsEnabled when Prometheus Pushgateway component set to disabled",
			actualCR: vzapi.Verrazzano{
				Spec: vzapi.VerrazzanoSpec{
					Components: vzapi.ComponentSpec{
						PrometheusPushgateway: &vzapi.PrometheusPushgatewayComponent{
							Enabled: &falseValue,
						},
					},
				},
			},
			expectTrue: false,
		},
	}

	for i, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := spi.NewFakeContext(nil, &tests[i].actualCR, nil, false, profilesRelativePath)
			assert.Equal(t, tt.expectTrue, NewComponent().IsEnabled(ctx.EffectiveCR()))
		})
	}
}

// Test isReady when it's called with component context
func TestIsReadyCompenent(t *testing.T) {
	c := fake.NewClientBuilder().Build()
	ctx := spi.NewFakeContext(c, &vzapi.Verrazzano{}, nil, false, profilesRelativePath)
	assert.False(t, NewComponent().IsReady(ctx))
}

// test Monitoroverrides method
func TestMonitorOverride(t *testing.T) {
	falseValue := false
	trueValue := true
	tests := []struct {
		name       string
		actualCR   vzapi.Verrazzano
		expectTrue bool
	}{
		{
			// GIVEN a default Verrazzano custom resource
			// WHEN we call MonitorOverride on the PrometheusPushgatewayComponent
			// THEN the call returns false
			name:       "Test MonitorOverride when using default Verrazzano CR",
			actualCR:   vzapi.Verrazzano{},
			expectTrue: true,
		},
		{
			// GIVEN a Verrazzano custom resource with the PrometheusPushgatewayComponent enabled
			// WHEN we call MonitorOverride on the PrometheusPushgatewayComponent
			// THEN the call returns true
			name: "Test MonitorOverride when PrometheusPushgatewayComponent set to enabled",
			actualCR: vzapi.Verrazzano{
				Spec: vzapi.VerrazzanoSpec{
					Components: vzapi.ComponentSpec{
						PrometheusPushgateway: &vzapi.PrometheusPushgatewayComponent{
							Enabled:          &trueValue,
							InstallOverrides: vzapi.InstallOverrides{MonitorChanges: &trueValue},
						},
					},
				},
			},
			expectTrue: true,
		},
		{
			// GIVEN a Verrazzano custom resource with the PrometheusPushgatewayComponent disabled
			// WHEN we call MonitorOverride on the PrometheusPushgatewayComponent
			// THEN the call returns true
			name: "Test MonitorOverride when PrometheusPushgatewayComponent set to disabled",
			actualCR: vzapi.Verrazzano{
				Spec: vzapi.VerrazzanoSpec{
					Components: vzapi.ComponentSpec{
						PrometheusPushgateway: &vzapi.PrometheusPushgatewayComponent{
							Enabled: &falseValue,
						},
					},
				},
			},
			expectTrue: true,
		},
	}

	for i, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := spi.NewFakeContext(nil, &tests[i].actualCR, nil, false, profilesRelativePath)
			assert.Equal(t, tt.expectTrue, NewComponent().MonitorOverrides(ctx))
		})
	}
}

// test preinstall for component class
func TestPreInstallcomponent(t *testing.T) {
	c := fake.NewClientBuilder().Build()
	ctx := spi.NewFakeContext(c, &vzapi.Verrazzano{}, nil, true)
	assert.Nil(t, NewComponent().PreInstall(ctx))
}

// TestAppendNGINXOverrides tests the AppendOverrides fn
// GIVEN a call to AppendOverrides
//
//	WHEN we pass a VZ spec with defaults
//	THEN the values created properly
func TestAppendNGINXOverrides(t *testing.T) {
	vz := &vzapi.Verrazzano{}
	kvs, err := AppendOverrides(spi.NewFakeContext(nil, vz, nil, false), ComponentName, ComponentNamespace, "", []bom.KeyValue{})
	assert.NoError(t, err)
	assert.Len(t, kvs, 1)
}

// TestPreUpgrade tests the component PreUpgrade function
func TestPreUpgrade(t *testing.T) {
	// GIVEN a previous installation of pushgateway and the pushgateway deployment exists
	// WHEN the component PreUpgrade function is called
	// THEN the deployment is deleted and PreUpgrade does not return an error
	deployment := &appsv1.Deployment{ObjectMeta: metav1.ObjectMeta{Namespace: ComponentNamespace, Name: deploymentName}}

	c := fake.NewClientBuilder().WithObjects(deployment).Build()
	ctx := spi.NewFakeContext(c, &vzapi.Verrazzano{}, nil, true)
	assert.Nil(t, NewComponent().PreUpgrade(ctx))

	err := c.Get(context.TODO(), types.NamespacedName{Namespace: ComponentNamespace, Name: deploymentName}, deployment)
	assert.Error(t, err)
	assert.True(t, errors.IsNotFound(err))

	// GIVEN a previous installation of pushgateway and the pushgateway deployment does not exist
	// WHEN the component PreUpgrade function is called
	// THEN PreUpgrade does not return an error
	assert.Nil(t, NewComponent().PreUpgrade(ctx))
}
