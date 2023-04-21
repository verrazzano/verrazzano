// Copyright (c) 2022, 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package kubestatemetrics

import (
	"testing"

	"github.com/verrazzano/verrazzano/pkg/bom"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	"github.com/stretchr/testify/assert"
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1beta1"
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
			// WHEN we call IsEnabled on the kube-state-metrics component
			// THEN the call returns true
			name:       "Test IsEnabled when using default Verrazzano CR",
			actualCR:   vzapi.Verrazzano{},
			expectTrue: true,
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
			// THEN the call returns true (since kube-state-metrics is enabled by default and
			// monitorOverrides defaults to true)
			name:       "Test MonitorOverride when using default Verrazzano CR",
			actualCR:   vzapi.Verrazzano{},
			expectTrue: true,
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
						KubeStateMetrics: &vzapi.KubeStateMetricsComponent{
							Enabled: &falseValue,
						},
					},
				},
			},
			new: &vzapi.Verrazzano{
				Spec: vzapi.VerrazzanoSpec{
					Components: vzapi.ComponentSpec{
						KubeStateMetrics: &vzapi.KubeStateMetricsComponent{
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
						KubeStateMetrics: &vzapi.KubeStateMetricsComponent{
							Enabled: &trueValue,
						},
					},
				},
			},
			new: &vzapi.Verrazzano{
				Spec: vzapi.VerrazzanoSpec{
					Components: vzapi.ComponentSpec{
						KubeStateMetrics: &vzapi.KubeStateMetricsComponent{
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
						KubeStateMetrics: &vzapi.KubeStateMetricsComponent{
							Enabled: &trueValue,
						},
					},
				},
			},
			new: &vzapi.Verrazzano{
				Spec: vzapi.VerrazzanoSpec{
					Components: vzapi.ComponentSpec{
						KubeStateMetrics: &vzapi.KubeStateMetricsComponent{
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
