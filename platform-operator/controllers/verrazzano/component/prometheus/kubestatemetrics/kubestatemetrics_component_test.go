// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package kubestatemetrics

import (
	"testing"

	"github.com/stretchr/testify/assert"
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
)

const profilesRelativePath = "../../../../../manifests/profiles/v1alpha1"

// TestIsEnabled tests the IsEnabled function for the kube-state-metrics component
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
