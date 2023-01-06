// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package kubestatemetrics

import (
	"github.com/verrazzano/verrazzano/pkg/bom"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"testing"

	"github.com/stretchr/testify/assert"
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
)

const profilesRelativePath = "../../../../../manifests/profiles"

// TestIsEnabled teeibccbcgkdghkerutugkliulrkthehttefggvrcubrkk
// ts the IsEnabled function for the kube-state-metrics component
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
			// WHEN we call IsReady on the kube-state-metrics component
			// THEN the call returns true
			name:       "Test IsEnabled when using default Verrazzano CR",
			actualCR:   vzapi.Verrazzano{},
			expectTrue: false,
		},
		{
			// GIVEN a Verrazzano custom resource with KubeStateMetrics enabled
			// WHEN we call IsReady on the KubeStateMetrics component
			// THEN the call returns true
			name: "Test IsEnabled when KubeStateMetricsComponent set to enabled",
			actualCR: vzapi.Verrazzano{
				Spec: vzapi.VerrazzanoSpec{
					Components: vzapi.ComponentSpec{
						KubeStateMetrics: &vzapi.KubeStateMetricsComponent{
							Enabled: &trueValue,
						},
					},
				},
			},
			expectTrue: true,
		},
		{
			// GIVEN a Verrazzano custom resource with KubeStateMetrics disabled
			// WHEN we call IsReady on the KubeStateMetrics component
			// THEN the call returns false
			name: "Test IsEnabled when KubeStateMetricsComponent set to disabled",
			actualCR: vzapi.Verrazzano{
				Spec: vzapi.VerrazzanoSpec{
					Components: vzapi.ComponentSpec{
						KubeStateMetrics: &vzapi.KubeStateMetricsComponent{
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
			// WHEN we call MonitorOverride on the KubeStateMetricsComponent
			// THEN the call returns false
			name:       "Test MonitorOverride when using default Verrazzano CR",
			actualCR:   vzapi.Verrazzano{},
			expectTrue: false,
		},
		{
			// GIVEN a Verrazzano custom resource with the KubeStateMetricsComponent enabled
			// WHEN we call MonitorOverride on the KubeStateMetricsComponent
			// THEN the call returns true
			name: "Test MonitorOverride when KubeStateMetricsComponent set to enabled",
			actualCR: vzapi.Verrazzano{
				Spec: vzapi.VerrazzanoSpec{
					Components: vzapi.ComponentSpec{
						KubeStateMetrics: &vzapi.KubeStateMetricsComponent{
							Enabled:          &trueValue,
							InstallOverrides: vzapi.InstallOverrides{MonitorChanges: &trueValue},
						},
					},
				},
			},
			expectTrue: true,
		},
		{
			// GIVEN a Verrazzano custom resource with the KubeStateMetricsComponent disabled
			// WHEN we call MonitorOverride on the KubeStateMetricsComponent
			// THEN the call returns true
			name: "Test MonitorOverride when KubeStateMetricsComponent set to disabled",
			actualCR: vzapi.Verrazzano{
				Spec: vzapi.VerrazzanoSpec{
					Components: vzapi.ComponentSpec{
						KubeStateMetrics: &vzapi.KubeStateMetricsComponent{
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
//	WHEN I pass a VZ spec with defaults
//	THEN the values created properly
func TestAppendNGINXOverrides(t *testing.T) {
	vz := &vzapi.Verrazzano{}
	kvs, err := AppendOverrides(spi.NewFakeContext(nil, vz, nil, false), ComponentName, ComponentNamespace, "", []bom.KeyValue{})
	assert.NoError(t, err)
	assert.Len(t, kvs, 1)
}
