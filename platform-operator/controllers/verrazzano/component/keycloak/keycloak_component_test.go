// Copyright (c) 2021, 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package keycloak

import (
	"testing"

	"github.com/stretchr/testify/assert"
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/constants"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/mysql"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8scheme "k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

var kcComponent = NewComponent()

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
			ctx := spi.NewFakeContext(fake.NewFakeClientWithScheme(k8scheme.Scheme), tt.vz, false)
			assert.Equal(t, tt.isEnabled, kcComponent.IsEnabled(ctx))
		})
	}
}

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

	mysqlSecret := &v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      mysql.ComponentName,
			Namespace: ComponentNamespace,
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
			"should fail when verrazzano secret is not present",
			fake.NewFakeClientWithScheme(k8scheme.Scheme, mysqlSecret),
			true,
		},
		{
			"should fail when mysql secret is not present",
			fake.NewFakeClientWithScheme(k8scheme.Scheme, vzSecret),
			true,
		},
		{
			"should pass when both secrets are present",
			fake.NewFakeClientWithScheme(k8scheme.Scheme, vzSecret, mysqlSecret),
			false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := spi.NewFakeContext(tt.client, testVZ, false)
			err := NewComponent().PreInstall(ctx)
			if tt.isErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
