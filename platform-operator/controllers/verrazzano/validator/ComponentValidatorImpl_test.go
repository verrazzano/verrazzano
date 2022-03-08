// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package validator

import (
	"reflect"
	"testing"

	"github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/internal/config"
)

// TestComponentValidatorImpl_ValidateInstall tests the ValidateInstall function
// GIVEN a valid CR
// WHEN ValidateInstall is called
// THEN ensure that no error is raised
func TestComponentValidatorImpl_ValidateInstall(t *testing.T) {
	tests := []struct {
		name string
		vz   *v1alpha1.Verrazzano
		want []error
	}{
		{
			name: "valid CR",
			vz:   &v1alpha1.Verrazzano{},
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
	tests := []struct {
		name string
		vz   *v1alpha1.Verrazzano
		want []error
	}{
		{
			name: "valid CR",
			vz:   &v1alpha1.Verrazzano{},
			want: nil,
		},
	}
	config.TestProfilesDir = "../../../manifests/profiles"
	defer func() { config.TestProfilesDir = "" }()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := ComponentValidatorImpl{}
			if got := c.ValidateUpdate(tt.vz, tt.vz); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ValidateInstall() = %v, want %v", got, tt.want)
			}
		})
	}
}
