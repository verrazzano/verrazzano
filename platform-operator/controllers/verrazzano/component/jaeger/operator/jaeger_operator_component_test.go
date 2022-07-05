// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package operator

import (
	"github.com/verrazzano/verrazzano/platform-operator/constants"
	"testing"

	"github.com/stretchr/testify/assert"
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	"github.com/verrazzano/verrazzano/platform-operator/internal/config"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

const profilesRelativePath = "../../../../../manifests/profiles"

var enabled = true
var jaegerEnabledCR = &vzapi.Verrazzano{
	Spec: vzapi.VerrazzanoSpec{
		Components: vzapi.ComponentSpec{
			JaegerOperator: &vzapi.JaegerOperatorComponent{
				Enabled: &enabled,
			},
		},
	},
}

// TestIsEnabled tests the IsEnabled function for the Jaeger Operator component
func TestIsEnabled(t *testing.T) {
	falseValue := false
	tests := []struct {
		name       string
		actualCR   vzapi.Verrazzano
		expectTrue bool
	}{
		{
			// GIVEN a default Verrazzano custom resource
			// WHEN we call IsReady on the Jaeger Operator component
			// THEN the call returns false
			name:       "Test IsEnabled when using default Verrazzano CR",
			actualCR:   vzapi.Verrazzano{},
			expectTrue: false,
		},
		{
			// GIVEN a Verrazzano custom resource with the Jaeger Operator enabled
			// WHEN we call IsReady on the Jaeger Operator component
			// THEN the call returns true
			name:       "Test IsEnabled when Jaeger Operator component set to enabled",
			actualCR:   *jaegerEnabledCR,
			expectTrue: true,
		},
		{
			// GIVEN a Verrazzano custom resource with the Jaeger Operator disabled
			// WHEN we call IsReady on the Jaeger Operator component
			// THEN the call returns false
			name: "Test IsEnabled when Jaeger Operator component set to disabled",
			actualCR: vzapi.Verrazzano{
				Spec: vzapi.VerrazzanoSpec{
					Components: vzapi.ComponentSpec{
						JaegerOperator: &vzapi.JaegerOperatorComponent{
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

func TestGetMinVerrazzanoVersion(t *testing.T) {
	assert.Equal(t, constants.VerrazzanoVersion1_3_0, NewComponent().GetMinVerrazzanoVersion())
}

func TestGetDependencies(t *testing.T) {
	assert.Equal(t, []string{"cert-manager"}, NewComponent().GetDependencies())
}

// TestValidateUpdate tests the Jaeger Operator ValidateUpdate function
func TestValidateUpdate(t *testing.T) {
	oldVZ := vzapi.Verrazzano{
		Spec: vzapi.VerrazzanoSpec{
			Components: vzapi.ComponentSpec{
				JaegerOperator: &vzapi.JaegerOperatorComponent{
					Enabled: &trueValue,
				},
			},
		},
	}
	newVZ := vzapi.Verrazzano{
		Spec: vzapi.VerrazzanoSpec{
			Components: vzapi.ComponentSpec{
				JaegerOperator: &vzapi.JaegerOperatorComponent{
					Enabled: &falseValue,
				},
			},
		},
	}
	assert.Error(t, NewComponent().ValidateUpdate(&oldVZ, &newVZ))
}

// TestPostInstall tests the component PostInstall function
func TestPostInstall(t *testing.T) {
	// GIVEN the Jaeger Operator is being installed
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
	// GIVEN the Jaeger Operator is being upgraded
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
