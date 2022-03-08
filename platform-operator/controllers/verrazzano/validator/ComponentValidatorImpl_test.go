// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package validator

import (
	"reflect"
	"testing"

	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/internal/config"
)

// TestComponentValidatorImpl_ValidateInstall tests the ValidateInstall function
// GIVEN a valid CR
// WHEN ValidateInstall is called
// THEN ensure that no error is raised
func TestComponentValidatorImpl_ValidateInstall(t *testing.T) {
	tests := []struct {
		name string
		vz   *vzapi.Verrazzano
		want []error
	}{
		{
			name: "valid CR",
			vz:   &vzapi.Verrazzano{},
			want: nil,
		},
	}
	config.TestProfilesDir = "../../../manifests/profiles"
	defer func() { config.TestProfilesDir = "" }()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := ComponentValidatorImpl{}
			if got := c.ValidateInstall(tt.vz); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ValidateInstall() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestComponentValidatorImpl_ValidateInstall tests the ValidateUpdate function
// GIVEN a valid CR
// WHEN ValidateUpdate is called
// THEN ensure that no error is raised
func TestComponentValidatorImpl_ValidateUpdate(t *testing.T) {
	disabled := false
	tests := []struct {
		name    string
		old     *vzapi.Verrazzano
		new     *vzapi.Verrazzano
		wantErr bool
	}{
		{
			name:    "no change",
			old:     &vzapi.Verrazzano{},
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
	}
	config.TestProfilesDir = "../../../manifests/profiles"
	defer func() { config.TestProfilesDir = "" }()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := ComponentValidatorImpl{}
			if err := c.ValidateUpdate(tt.old, tt.new); (err != nil) != tt.wantErr {
				t.Errorf("ValidateUpdate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
