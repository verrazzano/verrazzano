// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package operator

import (
	"testing"

	"github.com/stretchr/testify/assert"
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	"github.com/verrazzano/verrazzano/platform-operator/internal/config"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

const profilesRelativePath = "../../../../../manifests/profiles"

// TestIsEnabled tests the IsEnabled function for the Prometheus Operator component
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
			// WHEN we call IsReady on the Prometheus Operator component
			// THEN the call returns true
			name:       "Test IsEnabled when using default Verrazzano CR",
			actualCR:   vzapi.Verrazzano{},
			expectTrue: true,
		},
		{
			// GIVEN a Verrazzano custom resource with the Prometheus Operator enabled
			// WHEN we call IsReady on the Prometheus Operator component
			// THEN the call returns true
			name: "Test IsEnabled when Prometheus Operator component set to enabled",
			actualCR: vzapi.Verrazzano{
				Spec: vzapi.VerrazzanoSpec{
					Components: vzapi.ComponentSpec{
						PrometheusOperator: &vzapi.PrometheusOperatorComponent{
							Enabled: &trueValue,
						},
					},
				},
			},
			expectTrue: true,
		},
		{
			// GIVEN a Verrazzano custom resource with the Prometheus Operator disabled
			// WHEN we call IsReady on the Prometheus Operator component
			// THEN the call returns false
			name: "Test IsEnabled when Prometheus Operator component set to disabled",
			actualCR: vzapi.Verrazzano{
				Spec: vzapi.VerrazzanoSpec{
					Components: vzapi.ComponentSpec{
						PrometheusOperator: &vzapi.PrometheusOperatorComponent{
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
			ctx := spi.NewFakeContext(nil, &tests[i].actualCR, false, profilesRelativePath)
			assert.Equal(t, tt.expectTrue, NewComponent().IsEnabled(ctx.EffectiveCR()))
		})
	}
}

// TestValidateUpdate tests the Prometheus Operator ValidateUpdate function
func TestValidateUpdate(t *testing.T) {
	oldVZ := vzapi.Verrazzano{
		Spec: vzapi.VerrazzanoSpec{
			Components: vzapi.ComponentSpec{
				PrometheusOperator: &vzapi.PrometheusOperatorComponent{
					Enabled: &trueValue,
				},
			},
		},
	}
	newVZ := vzapi.Verrazzano{
		Spec: vzapi.VerrazzanoSpec{
			Components: vzapi.ComponentSpec{
				PrometheusOperator: &vzapi.PrometheusOperatorComponent{
					Enabled: &falseValue,
				},
			},
		},
	}
	assert.Error(t, NewComponent().ValidateUpdate(&oldVZ, &newVZ))
}

// TestPostInstall tests the component PostInstall function
func TestPostInstall(t *testing.T) {
	// GIVEN the Prometheus Operator is being installed
	// WHEN we call the PostInstall function
	// THEN no error is returned
	oldConfig := config.Get()
	defer config.Set(oldConfig)
	config.Set(config.OperatorConfig{
		VerrazzanoRootDir: "../../../../../..",
	})

	client := fake.NewClientBuilder().WithScheme(testScheme).Build()
	ctx := spi.NewFakeContext(client, &vzapi.Verrazzano{}, false, profilesRelativePath)
	err := NewComponent().PostInstall(ctx)
	assert.NoError(t, err)
}

// TestPostUpgrade tests the component PostUpgrade function
func TestPostUpgrade(t *testing.T) {
	// GIVEN the Prometheus Operator is being upgraded
	// WHEN we call the PostUpgrade function
	// THEN no error is returned
	oldConfig := config.Get()
	defer config.Set(oldConfig)
	config.Set(config.OperatorConfig{
		VerrazzanoRootDir: "../../../../../..",
	})

	client := fake.NewClientBuilder().WithScheme(testScheme).Build()
	ctx := spi.NewFakeContext(client, &vzapi.Verrazzano{}, false, profilesRelativePath)
	err := NewComponent().PostUpgrade(ctx)
	assert.NoError(t, err)
}
