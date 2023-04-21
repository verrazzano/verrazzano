// Copyright (c) 2022, 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package adapter

import (
	"testing"

	"github.com/stretchr/testify/assert"
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1beta1"
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

// TestValidateUpdate tests the validate update functions
func TestValidateUpdate(t *testing.T) {
	falseValue := false
	trueValue := true
	tests := []struct {
		name    string
		old     *vzapi.Verrazzano
		new     *vzapi.Verrazzano
		wantErr bool
	}{
		{
			// GIVEN the component is disabled
			// WHEN the component is enabled and we call the validate update function
			// THEN no error is returned
			name: "enable",
			old: &vzapi.Verrazzano{
				Spec: vzapi.VerrazzanoSpec{
					Components: vzapi.ComponentSpec{
						PrometheusAdapter: &vzapi.PrometheusAdapterComponent{
							Enabled: &falseValue,
						},
					},
				},
			},
			new: &vzapi.Verrazzano{
				Spec: vzapi.VerrazzanoSpec{
					Components: vzapi.ComponentSpec{
						PrometheusAdapter: &vzapi.PrometheusAdapterComponent{
							Enabled: &trueValue,
						},
					},
				},
			},
			wantErr: false,
		},
		{
			// GIVEN the component is enabled
			// WHEN the component is disabled and we call the validate update function
			// THEN an error is returned
			name: "disable",
			old: &vzapi.Verrazzano{
				Spec: vzapi.VerrazzanoSpec{
					Components: vzapi.ComponentSpec{
						PrometheusAdapter: &vzapi.PrometheusAdapterComponent{
							Enabled: &trueValue,
						},
					},
				},
			},
			new: &vzapi.Verrazzano{
				Spec: vzapi.VerrazzanoSpec{
					Components: vzapi.ComponentSpec{
						PrometheusAdapter: &vzapi.PrometheusAdapterComponent{
							Enabled: &falseValue,
						},
					},
				},
			},
			wantErr: true,
		},
		{
			// GIVEN the component is enabled
			// WHEN the component is not changed and we call the validate update function
			// THEN no error is returned
			name: "no change",
			old: &vzapi.Verrazzano{
				Spec: vzapi.VerrazzanoSpec{
					Components: vzapi.ComponentSpec{
						PrometheusAdapter: &vzapi.PrometheusAdapterComponent{
							Enabled: &trueValue,
						},
					},
				},
			},
			new: &vzapi.Verrazzano{
				Spec: vzapi.VerrazzanoSpec{
					Components: vzapi.ComponentSpec{
						PrometheusAdapter: &vzapi.PrometheusAdapterComponent{
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
			if err := c.ValidateUpdate(tt.old, tt.new); (err != nil) != tt.wantErr {
				t.Errorf("ValidateUpdate() error = %v, wantErr %v", err, tt.wantErr)
			}

			v1beta1New := &v1beta1.Verrazzano{}
			v1beta1Old := &v1beta1.Verrazzano{}
			err := tt.new.ConvertTo(v1beta1New)
			assert.NoError(t, err)
			err = tt.old.ConvertTo(v1beta1Old)
			assert.NoError(t, err)

			if err := c.ValidateUpdateV1Beta1(v1beta1Old, v1beta1New); (err != nil) != tt.wantErr {
				t.Errorf("ValidateUpdateV1Beta1() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
