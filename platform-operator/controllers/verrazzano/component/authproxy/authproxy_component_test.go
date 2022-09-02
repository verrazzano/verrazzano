// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package authproxy

import (
	"fmt"
	"testing"

	helmcli "github.com/verrazzano/verrazzano/pkg/helm"
	"github.com/verrazzano/verrazzano/pkg/os"
	"github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1beta1"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	"github.com/stretchr/testify/assert"
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/constants"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
)

const profilesRelativePath = "../../../../manifests/profiles/v1alpha1"

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
			ctx := spi.NewFakeContext(nil, &tests[i].actualCR, nil, false, profilesRelativePath)
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

// Test_authProxyComponent_ValidateUpdate tests the AuthProxy ValidateUpdate call for v1alpha1.Verrazzano
func Test_authProxyComponent_ValidateUpdate(t *testing.T) {
	disabled := false
	tests := []struct {
		name    string
		old     *vzapi.Verrazzano
		new     *vzapi.Verrazzano
		wantErr bool
	}{
		// GIVEN a VZ CR with auth proxy component disabled,
		// WHEN I call update the VZ CR to enable auth proxy component
		// THEN the update succeeds with no errors.
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
		// GIVEN a VZ CR with auth proxy component enabled,
		// WHEN I call update the VZ CR to disable auth proxy component
		// THEN the update fails with an error.
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
		// GIVEN a default VZ CR with auth proxy component,
		// WHEN I call update with no change to the auth proxy component
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

// Test_authProxyComponent_ValidateUpdateV1beta1 tests the AuthProxy ValidateUpdate call for v1beta1.Verrazzano
func Test_authProxyComponent_ValidateUpdateV1beta1(t *testing.T) {
	disabled := false
	tests := []struct {
		name    string
		old     *v1beta1.Verrazzano
		new     *v1beta1.Verrazzano
		wantErr bool
	}{
		// GIVEN a VZ CR with auth proxy component disabled,
		// WHEN I call update the VZ CR to enable auth proxy component
		// THEN the update succeeds with no error.
		{
			name: "enable",
			old: &v1beta1.Verrazzano{
				Spec: v1beta1.VerrazzanoSpec{
					Components: v1beta1.ComponentSpec{
						AuthProxy: &v1beta1.AuthProxyComponent{
							Enabled: &disabled,
						},
					},
				},
			},
			new:     &v1beta1.Verrazzano{},
			wantErr: false,
		},
		// GIVEN a VZ CR with auth proxy component enabled,
		// WHEN I call update the VZ CR to disable auth proxy component
		// THEN the update fails with an error.
		{
			name: "disable",
			old:  &v1beta1.Verrazzano{},
			new: &v1beta1.Verrazzano{
				Spec: v1beta1.VerrazzanoSpec{
					Components: v1beta1.ComponentSpec{
						AuthProxy: &v1beta1.AuthProxyComponent{
							Enabled: &disabled,
						},
					},
				},
			},
			wantErr: true,
		},
		// GIVEN a default VZ CR with auth proxy component,
		// WHEN I call update with no change to the auth proxy component
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

// TestUninstallHelmChartInstalled tests the Fluentd Uninstall call
// GIVEN a Fluentd component
//  WHEN I call Uninstall with the Fluentd helm chart installed
//  THEN no error is returned
func TestUninstallHelmChartInstalled(t *testing.T) {
	helmcli.SetCmdRunner(os.GenericTestRunner{
		StdOut: []byte(""),
		StdErr: []byte{},
		Err:    nil,
	})
	defer helmcli.SetDefaultRunner()

	err := NewComponent().Uninstall(spi.NewFakeContext(fake.NewClientBuilder().Build(), &vzapi.Verrazzano{}, nil, false))
	assert.NoError(t, err)
}

// TestUninstallHelmChartNotInstalled tests the Fluentd Uninstall call
// GIVEN a Fluentd component
//  WHEN I call Uninstall with the Fluentd helm chart not installed
//  THEN no error is returned
func TestUninstallHelmChartNotInstalled(t *testing.T) {
	helmcli.SetCmdRunner(os.GenericTestRunner{
		StdOut: []byte(""),
		StdErr: []byte{},
		Err:    fmt.Errorf("Not installed"),
	})
	defer helmcli.SetDefaultRunner()

	err := NewComponent().Uninstall(spi.NewFakeContext(fake.NewClientBuilder().Build(), &vzapi.Verrazzano{}, nil, false))
	assert.NoError(t, err)
}
