// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package coherence

import (
	"github.com/stretchr/testify/assert"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"testing"

	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1beta1"
)

// TestValidateUpdate tests the Coherence ValidateUpdate call for v1alpha1.Verrazzano
func TestValidateUpdate(t *testing.T) {
	disabled := false
	tests := []struct {
		name    string
		old     *vzapi.Verrazzano
		new     *vzapi.Verrazzano
		wantErr bool
	}{
		// GIVEN a VZ CR with coherence component disabled,
		// WHEN I call update the VZ CR to enable coherence component
		// THEN the update succeeds with no error.
		{
			name: "enable",
			old: &vzapi.Verrazzano{
				Spec: vzapi.VerrazzanoSpec{
					Components: vzapi.ComponentSpec{
						CoherenceOperator: &vzapi.CoherenceOperatorComponent{
							Enabled: &disabled,
						},
					},
				},
			},
			new:     &vzapi.Verrazzano{},
			wantErr: false,
		},
		// GIVEN a VZ CR with coherence component enabled,
		// WHEN I call update the VZ CR to disable coherence component
		// THEN the update fails with an error.
		{
			name: "disable",
			old:  &vzapi.Verrazzano{},
			new: &vzapi.Verrazzano{
				Spec: vzapi.VerrazzanoSpec{
					Components: vzapi.ComponentSpec{
						CoherenceOperator: &vzapi.CoherenceOperatorComponent{
							Enabled: &disabled,
						},
					},
				},
			},
			wantErr: true,
		},
		// GIVEN a default VZ CR with coherence component,
		// WHEN I call update with no change to the coherence component
		// THEN the update succeeds and no error is returned.
		{
			name:    "no change",
			old:     &vzapi.Verrazzano{},
			new:     &vzapi.Verrazzano{},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := NewComponent()
			if err := c.ValidateUpdate(tt.old, tt.new); (err != nil) != tt.wantErr {
				t.Errorf("ValidateUpdate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// TestValidateUpdateV1beta1 tests the Coherence ValidateUpdate call for v1beta1.Verrazzano
func TestValidateUpdateV1beta1(t *testing.T) {
	disabled := false
	tests := []struct {
		name    string
		old     *v1beta1.Verrazzano
		new     *v1beta1.Verrazzano
		wantErr bool
	}{
		// GIVEN a VZ CR with coherence component disabled,
		// WHEN I call update the VZ CR to enable coherence component
		// THEN the update succeeds with no error.
		{
			name: "enable",
			old: &v1beta1.Verrazzano{
				Spec: v1beta1.VerrazzanoSpec{
					Components: v1beta1.ComponentSpec{
						CoherenceOperator: &v1beta1.CoherenceOperatorComponent{
							Enabled: &disabled,
						},
					},
				},
			},
			new:     &v1beta1.Verrazzano{},
			wantErr: false,
		},
		// GIVEN a VZ CR with coherence component enabled,
		// WHEN I call update the VZ CR to disable coherence component
		// THEN the update fails with an error.
		{
			name: "disable",
			old:  &v1beta1.Verrazzano{},
			new: &v1beta1.Verrazzano{
				Spec: v1beta1.VerrazzanoSpec{
					Components: v1beta1.ComponentSpec{
						CoherenceOperator: &v1beta1.CoherenceOperatorComponent{
							Enabled: &disabled,
						},
					},
				},
			},
			wantErr: true,
		},
		// GIVEN a default VZ CR with coherence component,
		// WHEN I call update with no change to the coherence component
		// THEN the update succeeds and no error is returned.
		{
			name:    "no change",
			old:     &v1beta1.Verrazzano{},
			new:     &v1beta1.Verrazzano{},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := NewComponent()
			if err := c.ValidateUpdateV1Beta1(tt.old, tt.new); (err != nil) != tt.wantErr {
				t.Errorf("ValidateUpdateV1Beta1() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// Test isReady when it's called with component context
func TestIsReady(t *testing.T) {
	c := fake.NewClientBuilder().Build()
	ctx := spi.NewFakeContext(c, &vzapi.Verrazzano{}, nil, false)
	assert.False(t, NewComponent().IsReady(ctx))
}

// test Monitoroverrides method
func TestMonitorOverride(t *testing.T) {
	falseValue := false
	trueValue := true
	tests := []struct {
		name       string
		actualCR   *vzapi.Verrazzano
		expectTrue bool
	}{
		{
			// GIVEN a default Verrazzano custom resource
			// WHEN we call MonitorOverride on the CoherenceOperatorComponent
			// THEN the call returns false
			name:       "Test MonitorOverride when using default Verrazzano CR",
			actualCR:   &vzapi.Verrazzano{},
			expectTrue: false,
		},
		{
			// GIVEN a Verrazzano custom resource with the CoherenceOperatorComponent enabled
			// WHEN we call MonitorOverride on the CoherenceOperatorComponent
			// THEN the call returns true
			name: "Test MonitorOverride when CoherenceOperatorComponent set to enabled",
			actualCR: &vzapi.Verrazzano{
				Spec: vzapi.VerrazzanoSpec{
					Components: vzapi.ComponentSpec{
						CoherenceOperator: &vzapi.CoherenceOperatorComponent{
							Enabled:          &trueValue,
							InstallOverrides: vzapi.InstallOverrides{MonitorChanges: &trueValue},
						},
					},
				},
			},
			expectTrue: true,
		},
		{
			// GIVEN a Verrazzano custom resource with the CoherenceOperatorComponent disabled
			// WHEN we call MonitorOverride on the CoherenceOperatorComponent
			// THEN the call returns true
			name: "Test MonitorOverride when CoherenceOperatorComponent set to disabled",
			actualCR: &vzapi.Verrazzano{
				Spec: vzapi.VerrazzanoSpec{
					Components: vzapi.ComponentSpec{
						CoherenceOperator: &vzapi.CoherenceOperatorComponent{
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
			ctx := spi.NewFakeContext(nil, tests[i].actualCR, nil, false)
			assert.Equal(t, tt.expectTrue, NewComponent().MonitorOverrides(ctx))
		})
	}
}

// test PostUninstall for component class
func TestPostUninstallcomponent(t *testing.T) {
	c := fake.NewClientBuilder().Build()
	ctx := spi.NewFakeContext(c, &vzapi.Verrazzano{}, nil, true)
	assert.Nil(t, NewComponent().PostUninstall(ctx))
}
