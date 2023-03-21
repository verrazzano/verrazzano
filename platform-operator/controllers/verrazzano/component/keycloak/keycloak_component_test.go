// Copyright (c) 2021, 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package keycloak

import (
	"github.com/verrazzano/verrazzano/pkg/helm"
	"github.com/verrazzano/verrazzano/pkg/log/vzlog"
	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/cli"
	"helm.sh/helm/v3/pkg/release"
	"testing"

	"github.com/stretchr/testify/assert"
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1beta1"
	"github.com/verrazzano/verrazzano/platform-operator/constants"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	k8scheme "k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

var kcComponent = NewComponent()

// TestIsEnabled tests the Keycloak IsEnabled call
// GIVEN a Keycloak component
//
//	WHEN I call IsEnabled
//	THEN true is returned unless Keycloak is explicitly disabled
func TestIsEnabled(t *testing.T) {
	disabled := false
	var tests = []struct {
		name      string
		vz        *vzapi.Verrazzano
		isEnabled bool
	}{
		{
			"disabled",
			&vzapi.Verrazzano{
				Spec: vzapi.VerrazzanoSpec{
					Components: vzapi.ComponentSpec{
						Keycloak: &vzapi.KeycloakComponent{
							Enabled: &disabled,
						},
					},
				},
			},
			false,
		},
		{
			"enabled/nil",
			&vzapi.Verrazzano{
				Spec: vzapi.VerrazzanoSpec{
					Components: vzapi.ComponentSpec{
						Keycloak: &vzapi.KeycloakComponent{
							Enabled: nil,
						},
					},
				},
			},
			true,
		},
		{
			"enabled",
			testVZ,
			true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := spi.NewFakeContext(fake.NewClientBuilder().WithScheme(k8scheme.Scheme).Build(), tt.vz, nil, false)
			assert.Equal(t, tt.isEnabled, kcComponent.IsEnabled(ctx.EffectiveCR()))
		})
	}
}

func testActionConfigWithoutInstallation(log vzlog.VerrazzanoLogger, settings *cli.EnvSettings, namespace string) (*action.Configuration, error) {
	return helm.CreateActionConfig(false, ComponentName, release.StatusDeployed, vzlog.DefaultLogger(), nil)
}

// TestReconcileBeforeInstall tests the Keycloak Reconcile call
// GIVEN a Keycloak component
//
// WHEN I call Reconcile with defaults, before Keycloak is actually installed
// THEN a nil error is returned
func TestReconcileBeforeInstall(t *testing.T) {
	// simulate the case that Keycloak is not installed
	defer helm.SetDefaultActionConfigFunction()
	helm.SetActionConfigFunction(testActionConfigWithoutInstallation)

	c := fake.NewClientBuilder().WithScheme(k8scheme.Scheme).Build()
	ctx := spi.NewFakeContext(c, &crEnabled, nil, false)
	err := NewComponent().Reconcile(ctx)
	assert.NoError(t, err)
}

// TestPreinstall tests the Keycloak PreInstall call
// GIVEN a Keycloak component
//
//	WHEN I call PreInstall
//	THEN an error is returned unless the post-install validation criteria are met
func TestPreinstall(t *testing.T) {
	vzSecret := &v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "verrazzano",
			Namespace: constants.VerrazzanoSystemNamespace,
		},
		Data: map[string][]byte{
			"password": []byte("password"),
		},
	}

	var tests = []struct {
		name   string
		client client.Client
		isErr  bool
	}{
		{
			"should pass when vz secret secret is present",
			fake.NewClientBuilder().WithScheme(k8scheme.Scheme).WithObjects(vzSecret).Build(),
			false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := spi.NewFakeContext(tt.client, testVZ, nil, false)
			err := NewComponent().PreInstall(ctx)
			if tt.isErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// TestKeycloakComponentValidateUpdate tests the Keycloak ValidateUpdate call
// GIVEN a Keycloak component
//
//	WHEN I call ValidateUpdate
//	THEN an error is returned if the validation is expected to fail
func TestKeycloakComponentValidateUpdate(t *testing.T) {
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
						Keycloak: &vzapi.KeycloakComponent{
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
						Keycloak: &vzapi.KeycloakComponent{
							Enabled: &disabled,
						},
					},
				},
			},
			wantErr: true,
		},
		{
			name: "disable",
			old:  &vzapi.Verrazzano{},
			new: &vzapi.Verrazzano{
				Spec: vzapi.VerrazzanoSpec{
					Components: vzapi.ComponentSpec{
						Keycloak: &vzapi.KeycloakComponent{
							KeycloakInstallArgs: []vzapi.InstallArgs{{Name: "foo", Value: "bar"}},
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

// TestKeycloakComponentValidateUpdateV1Beta1 tests the Keycloak ValidateUpdateV1beta1 call
// GIVEN a Keycloak component
//
//	WHEN I call ValidateUpdate
//	THEN an error is returned if the validation is expected to fail
func TestKeycloakComponentValidateUpdateV1Beta1(t *testing.T) {
	disabled := false
	tests := []struct {
		name    string
		old     *v1beta1.Verrazzano
		new     *v1beta1.Verrazzano
		wantErr bool
	}{
		{
			name: "enable",
			old: &v1beta1.Verrazzano{
				Spec: v1beta1.VerrazzanoSpec{
					Components: v1beta1.ComponentSpec{
						Keycloak: &v1beta1.KeycloakComponent{
							Enabled: &disabled,
						},
					},
				},
			},
			new:     &v1beta1.Verrazzano{},
			wantErr: false,
		},
		{
			name: "disable",
			old:  &v1beta1.Verrazzano{},
			new: &v1beta1.Verrazzano{
				Spec: v1beta1.VerrazzanoSpec{
					Components: v1beta1.ComponentSpec{
						Keycloak: &v1beta1.KeycloakComponent{
							Enabled: &disabled,
						},
					},
				},
			},
			wantErr: true,
		},
		{
			name: "Dummy install overrides",
			old:  &v1beta1.Verrazzano{},
			new: &v1beta1.Verrazzano{
				Spec: v1beta1.VerrazzanoSpec{
					Components: v1beta1.ComponentSpec{
						Keycloak: &v1beta1.KeycloakComponent{
							InstallOverrides: v1beta1.InstallOverrides{
								ValueOverrides: []v1beta1.Overrides{
									{},
								},
							},
						},
					},
				},
			},
			wantErr: true,
		},
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

// TestKeycloakComponent_GetCertificateNames tests the Keycloak GetCertificateNames call
// GIVEN a Keycloak component
//
//	WHEN I call GetCertificateNames
//	THEN the correct number of certificate names are returned based on what is enabled
func TestKeycloakComponent_GetCertificateNames(t *testing.T) {
	enabled := true
	vz := &vzapi.Verrazzano{
		Spec: vzapi.VerrazzanoSpec{
			EnvironmentName: "myenv",
			Components: vzapi.ComponentSpec{
				Keycloak: &vzapi.KeycloakComponent{
					Enabled: &enabled,
				},
			},
		},
	}

	c := fake.NewClientBuilder().WithScheme(k8scheme.Scheme).Build()
	ctx := spi.NewFakeContext(c, vz, nil, false)
	names := NewComponent().GetCertificateNames(ctx)
	assert.Len(t, names, 1)
	assert.Equal(t, types.NamespacedName{Name: keycloakCertificateName, Namespace: ComponentNamespace}, names[0])
}
