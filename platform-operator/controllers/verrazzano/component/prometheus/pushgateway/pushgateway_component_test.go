// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package pushgateway

import (
	"testing"

	"github.com/stretchr/testify/assert"
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
)

const profilesRelativePath = "../../../../../manifests/profiles/v1alpha1"

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
