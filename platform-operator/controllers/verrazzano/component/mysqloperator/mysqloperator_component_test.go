// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package mysqloperator

import (
	"testing"

	"github.com/stretchr/testify/assert"
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
)

// TestGetInstallOverrides tests the GetInstallOverrides function
// GIVEN a call to GetInstallOverrides
//  WHEN there is a valid MySQL Operator configuration
//  THEN the correct Helm overrides are returned
func TestGetInstallOverrides(t *testing.T) {
	vz := &vzapi.Verrazzano{
		Spec: vzapi.VerrazzanoSpec{
			Components: vzapi.ComponentSpec{
				MySQLOperator: &vzapi.MySQLOperatorComponent{
					InstallOverrides: vzapi.InstallOverrides{
						ValueOverrides: []vzapi.Overrides{
							{Values: &apiextensionsv1.JSON{
								Raw: []byte("{\"key1\": \"value1\"}")},
							},
						},
					},
				},
			},
		},
	}

	comp := NewComponent()
	overrides := comp.GetOverrides(vz)
	assert.Equal(t, []byte("{\"key1\": \"value1\"}"), overrides[0].Values.Raw)
}

// TestIsEnabled tests the IsEnabled function
// GIVEN a call to IsEnabled
//  WHEN varying the enabled states of keycloak and MySQL Operator
//  THEN check for the expected response
func TestIsEnabled(t *testing.T) {
	trueValue := true
	falseValue := false

	tests := []struct {
		name string
		vz   *vzapi.Verrazzano
		want bool
	}{
		{
			name: "MySQL Operator explicitly enabled, Keycloak disabled",
			vz: &vzapi.Verrazzano{
				Spec: vzapi.VerrazzanoSpec{
					Components: vzapi.ComponentSpec{
						MySQLOperator: &vzapi.MySQLOperatorComponent{
							Enabled: &trueValue},
						Keycloak: &vzapi.KeycloakComponent{
							Enabled: &falseValue}}}},
			want: true,
		},
		{
			name: "MySQL Operator and Keycloak explicitly disabled",
			vz: &vzapi.Verrazzano{
				Spec: vzapi.VerrazzanoSpec{
					Components: vzapi.ComponentSpec{
						MySQLOperator: &vzapi.MySQLOperatorComponent{
							Enabled: &falseValue},
						Keycloak: &vzapi.KeycloakComponent{
							Enabled: &falseValue}}}},
			want: false,
		},
		{
			name: "Keycloak enabled, MySQL Operator component nil",
			vz: &vzapi.Verrazzano{
				Spec: vzapi.VerrazzanoSpec{
					Components: vzapi.ComponentSpec{
						MySQLOperator: &vzapi.MySQLOperatorComponent{},
						Keycloak: &vzapi.KeycloakComponent{
							Enabled: &falseValue}}}},
			want: true,
		},
		{
			name: "Keycloak and MySQL Operator component nil",
			vz: &vzapi.Verrazzano{
				Spec: vzapi.VerrazzanoSpec{
					Components: vzapi.ComponentSpec{}}},
			want: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := NewComponent().IsEnabled(tt.vz)
			assert.Equal(t, tt.want, c)
		})
	}
}
