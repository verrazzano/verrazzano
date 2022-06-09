// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package authproxy

import (
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"testing"

	"github.com/stretchr/testify/assert"
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/constants"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
)

const profilesRelativePath = "../../../../manifests/profiles"

// TestIsEnabled tests the AuthProxy IsEnabled call
// GIVEN a AuthProxy component
//  WHEN I call IsEnabled when all requirements are met
//  THEN true or false is returned
func TestIsEnabled(t *testing.T) {
	falseValue := false
	tests := []struct {
		name       string
		actualCR   vzapi.Verrazzano
		expectTrue bool
	}{
		{
			name:       "Test IsEnabled when using default Verrazzano CR",
			actualCR:   vzapi.Verrazzano{},
			expectTrue: true,
		},
		{
			name: "Test IsEnabled when using AuthProxy component set to disabled",
			actualCR: vzapi.Verrazzano{
				Spec: vzapi.VerrazzanoSpec{
					Components: vzapi.ComponentSpec{
						AuthProxy: &vzapi.AuthProxyComponent{
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
			if tt.expectTrue {
				assert.True(t, NewComponent().IsEnabled(ctx.EffectiveCR()))
			} else {
				assert.False(t, NewComponent().IsEnabled(ctx.EffectiveCR()))
			}
		})
	}
}

// TestGetIngressNames tests the AuthProxy GetIngressNames call
// GIVEN a AuthProxy component
//  WHEN I call GetIngressNames
//  THEN the correct list of names is returned
func TestGetIngressNames(t *testing.T) {
	ingressNames := NewComponent().GetIngressNames(nil)
	assert.True(t, len(ingressNames) == 1)
	assert.Equal(t, constants.VzConsoleIngress, ingressNames[0].Name)
	assert.Equal(t, ComponentNamespace, ingressNames[0].Namespace)
}

// TestAuthProxyComponentValidateUpdate tests the AuthProxy GetIngressNames call
// GIVEN a AuthProxy component
//  WHEN I call ValidateUpdate,
//  THEN it should return an error only when the update disables a previously enabled auth proxy component.
func TestAuthProxyComponentValidateUpdate(t *testing.T) {
	disabled := false
	tests := []struct {
		name    string
		old     *vzapi.Verrazzano
		new     *vzapi.Verrazzano
		wantErr bool
	}{
		{
			name: "enable",
			old: &vzapi.Verrazzano{
				Spec: vzapi.VerrazzanoSpec{
					Components: vzapi.ComponentSpec{
						AuthProxy: &vzapi.AuthProxyComponent{
							Enabled: &disabled,
						},
					},
				},
			},
			new:     &vzapi.Verrazzano{},
			wantErr: false,
		},
		{
			name: "disable",
			old:  &vzapi.Verrazzano{},
			new: &vzapi.Verrazzano{
				Spec: vzapi.VerrazzanoSpec{
					Components: vzapi.ComponentSpec{
						AuthProxy: &vzapi.AuthProxyComponent{
							Enabled: &disabled,
						},
					},
				},
			},
			wantErr: true,
		},
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

// TestAuthProxyComponentIsReady tests the AuthProxy IsReady call
// GIVEN a AuthProxy component
//  WHEN I call IsReady,
//  THEN it should return false since none of the replicas of the authproxy deployment is ready by default.
func TestAuthProxyComponentIsReady(t *testing.T) {
	client := fake.NewClientBuilder().WithScheme(testScheme).Build()
	trueValue := true
	vz := &vzapi.Verrazzano{
		Spec: vzapi.VerrazzanoSpec{
			Components: vzapi.ComponentSpec{
				AuthProxy: &vzapi.AuthProxyComponent{
					Enabled: &trueValue,
				},
			},
		},
	}
	authProxy := NewComponent()
	isReady := authProxy.IsReady(spi.NewFakeContext(client, vz, true))
	assert.False(t, isReady, "When dry run flag is enabled, IsReady should return false")
	isReady = authProxy.IsReady(spi.NewFakeContext(client, vz, false))
	assert.False(t, isReady, "When dry run flag is disabled, IsReady should return false")
}

// TestAuthProxyComponentPreInstall tests the AuthProxy PreInstall call
// GIVEN a AuthProxy component
// WHEN I call PreUpgrade,
// THEN it should not return any error.
func TestAuthProxyComponentPreInstall(t *testing.T) {
	client := fake.NewClientBuilder().WithScheme(testScheme).Build()
	trueValue := true
	vz := &vzapi.Verrazzano{
		Spec: vzapi.VerrazzanoSpec{
			Components: vzapi.ComponentSpec{
				AuthProxy: &vzapi.AuthProxyComponent{
					Enabled: &trueValue,
				},
			},
		},
	}
	ctx := spi.NewFakeContext(client, vz, false)
	authProxy := NewComponent()
	err := authProxy.PreInstall(ctx)
	assert.Nilf(t, err, "There should not be any error when pre install is invoked")
}

// TestAuthProxyComponentPreUpgrade tests the AuthProxy PreUpgrade call
// GIVEN a AuthProxy component
// WHEN I call PreUpgrade,
// THEN it should not return any error.
func TestAuthProxyComponentPreUpgrade(t *testing.T) {
	client := fake.NewClientBuilder().WithScheme(testScheme).Build()
	trueValue := true
	vz := &vzapi.Verrazzano{
		Spec: vzapi.VerrazzanoSpec{
			Components: vzapi.ComponentSpec{
				AuthProxy: &vzapi.AuthProxyComponent{
					Enabled: &trueValue,
				},
			},
		},
	}
	ctx := spi.NewFakeContext(client, vz, false)
	authProxy := NewComponent()
	err := authProxy.PreUpgrade(ctx)
	assert.Nilf(t, err, "There should not be any error when pre upgrade is invoked")
}

// TestAuthProxyComponentMonitorOverrides tests the AuthProxy MonitorOverrides call
// GIVEN a AuthProxy component
//  WHEN I call IsReady,
//  THEN it should return false if no authproxy component is defined in VZ CR or if explicitly disabled
//       and return true otherwise.
func TestAuthProxyComponentMonitorOverrides(t *testing.T) {
	trueValue := true
	falseValue := false
	tests := []struct {
		name              string
		actualCR          vzapi.Verrazzano
		expectMonitorFlag bool
	}{
		{
			name:              "Test MonitorOverrides return false when no custom authproxy spec is set",
			actualCR:          vzapi.Verrazzano{},
			expectMonitorFlag: false,
		},
		{
			name: "Test MonitorOverrides return true when authproxy spec does not define MonitorOverrides flag",
			actualCR: vzapi.Verrazzano{
				Spec: vzapi.VerrazzanoSpec{
					Components: vzapi.ComponentSpec{
						AuthProxy: &vzapi.AuthProxyComponent{},
					},
				},
			},
			expectMonitorFlag: true,
		},
		{
			name: "Test MonitorOverrides return true when authproxy spec does not define MonitorOverrides flag",
			actualCR: vzapi.Verrazzano{
				Spec: vzapi.VerrazzanoSpec{
					Components: vzapi.ComponentSpec{
						AuthProxy: &vzapi.AuthProxyComponent{
							InstallOverrides: vzapi.InstallOverrides{
								MonitorChanges: &trueValue,
							},
						},
					},
				},
			},
			expectMonitorFlag: true,
		},
		{
			name: "Test MonitorOverrides returns false when authproxy spec explicitly defines MonitorOverrides flag",
			actualCR: vzapi.Verrazzano{
				Spec: vzapi.VerrazzanoSpec{
					Components: vzapi.ComponentSpec{
						AuthProxy: &vzapi.AuthProxyComponent{
							InstallOverrides: vzapi.InstallOverrides{
								MonitorChanges: &falseValue,
							},
						},
					},
				},
			},
			expectMonitorFlag: false,
		},
	}
	client := fake.NewClientBuilder().WithScheme(testScheme).Build()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := spi.NewFakeContext(client, &tt.actualCR, false)
			c := NewComponent()
			monitorFlag := c.MonitorOverrides(ctx)
			assert.Equal(t, tt.expectMonitorFlag, monitorFlag)
		})
	}
}
