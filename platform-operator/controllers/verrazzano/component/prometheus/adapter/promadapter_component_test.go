// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package adapter

import (
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"testing"

	"github.com/stretchr/testify/assert"
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
)

const profilesRelativePath = "../../../../../manifests/profiles"

// TestIsEnabled tests the IsEnabled function for the Prometheus Adapter component
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
			// WHEN we call IsReady on the Prometheus Adapter component
			// THEN the call returns true
			name:       "Test IsEnabled when using default Verrazzano CR",
			actualCR:   vzapi.Verrazzano{},
			expectTrue: false,
		},
		{
			// GIVEN a Verrazzano custom resource with the Prometheus Adapter enabled
			// WHEN we call IsReady on the Prometheus Adapter component
			// THEN the call returns true
			name: "Test IsEnabled when Prometheus Adapter component set to enabled",
			actualCR: vzapi.Verrazzano{
				Spec: vzapi.VerrazzanoSpec{
					Components: vzapi.ComponentSpec{
						PrometheusAdapter: &vzapi.PrometheusAdapterComponent{
							Enabled: &trueValue,
						},
					},
				},
			},
			expectTrue: true,
		},
		{
			// GIVEN a Verrazzano custom resource with the Prometheus Adapter disabled
			// WHEN we call IsReady on the Prometheus Adapter component
			// THEN the call returns false
			name: "Test IsEnabled when Prometheus Adapter component set to disabled",
			actualCR: vzapi.Verrazzano{
				Spec: vzapi.VerrazzanoSpec{
					Components: vzapi.ComponentSpec{
						PrometheusAdapter: &vzapi.PrometheusAdapterComponent{
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
func TestIsReady(t *testing.T) {
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
			// WHEN we call MonitorOverride on the Prometheus Adapter component
			// THEN the call returns false
			name:       "Test MonitorOverride when using default Verrazzano CR",
			actualCR:   vzapi.Verrazzano{},
			expectTrue: false,
		},
		{
			// GIVEN a Verrazzano custom resource with the Prometheus Adapter enabled
			// WHEN we call MonitorOverride on the Prometheus Adapter component
			// THEN the call returns true
			name: "Test MonitorOverride when Prometheus Adapter component set to enabled",
			actualCR: vzapi.Verrazzano{
				Spec: vzapi.VerrazzanoSpec{
					Components: vzapi.ComponentSpec{
						PrometheusAdapter: &vzapi.PrometheusAdapterComponent{
							Enabled:          &trueValue,
							InstallOverrides: vzapi.InstallOverrides{MonitorChanges: &trueValue},
						},
					},
				},
			},
			expectTrue: true,
		},
		{
			// GIVEN a Verrazzano custom resource with the Prometheus Adapter disabled
			// WHEN we call MonitorOverride on the Prometheus Adapter component
			// THEN the call returns true
			name: "Test MonitorOverride when Prometheus Adapter component set to disabled",
			actualCR: vzapi.Verrazzano{
				Spec: vzapi.VerrazzanoSpec{
					Components: vzapi.ComponentSpec{
						PrometheusAdapter: &vzapi.PrometheusAdapterComponent{
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
