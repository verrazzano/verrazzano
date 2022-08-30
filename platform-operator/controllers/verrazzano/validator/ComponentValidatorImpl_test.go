// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package validator

import (
	"testing"

	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/internal/config"
)

var disabled = false

// TestComponentValidatorImpl_ValidateInstall tests the ValidateInstall function
// GIVEN a valid CR
// WHEN ValidateInstall is called
// THEN ensure that no error is raised
func TestComponentValidatorImpl_ValidateInstall(t *testing.T) {
	tests := []struct {
		name           string
		vz             *vzapi.Verrazzano
		numberOfErrors int
	}{
		{
			name:           "default CR",
			vz:             &vzapi.Verrazzano{},
			numberOfErrors: 0,
		},
		{
			name: "disabled cert and ingress",
			vz: &vzapi.Verrazzano{
				Spec: vzapi.VerrazzanoSpec{
					Components: vzapi.ComponentSpec{
						CertManager: &vzapi.CertManagerComponent{
							Enabled: &disabled,
						},
						Ingress: &vzapi.IngressNginxComponent{
							Enabled: &disabled,
						},
					},
				},
			},
			numberOfErrors: 0,
		},
	}
	config.TestProfilesDir = "../../../manifests/profiles/v1alpha1"
	defer func() { config.TestProfilesDir = "" }()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := ComponentValidatorImpl{}
			got := c.ValidateInstall(tt.vz)
			if len(got) != tt.numberOfErrors {
				t.Errorf("ValidateInstall() = %v, numberOfErrors %v", len(got), tt.numberOfErrors)
			}
		})
	}
}

// TestComponentValidatorImpl_ValidateUpdate tests the ValidateUpdate function
// GIVEN a valid CR
// WHEN ValidateUpdate is called
// THEN ensure that no error is raised
func TestComponentValidatorImpl_ValidateUpdate(t *testing.T) {
	tests := []struct {
		name           string
		old            *vzapi.Verrazzano
		new            *vzapi.Verrazzano
		numberOfErrors int
	}{
		{
			name:           "no change",
			old:            &vzapi.Verrazzano{},
			new:            &vzapi.Verrazzano{},
			numberOfErrors: 0,
		},
		{
			name: "disable rancher",
			old:  &vzapi.Verrazzano{},
			new: &vzapi.Verrazzano{
				Spec: vzapi.VerrazzanoSpec{
					Components: vzapi.ComponentSpec{
						Rancher: &vzapi.RancherComponent{
							Enabled: &disabled,
						},
					},
				},
			},
			numberOfErrors: 1,
		},
		{
			name: "disable cert",
			old:  &vzapi.Verrazzano{},
			new: &vzapi.Verrazzano{
				Spec: vzapi.VerrazzanoSpec{
					Components: vzapi.ComponentSpec{
						CertManager: &vzapi.CertManagerComponent{
							Enabled: &disabled,
						},
					},
				},
			},
			numberOfErrors: 1,
		},
		{
			name: "disabled cert and ingress",
			old:  &vzapi.Verrazzano{},
			new: &vzapi.Verrazzano{
				Spec: vzapi.VerrazzanoSpec{
					Components: vzapi.ComponentSpec{
						CertManager: &vzapi.CertManagerComponent{
							Enabled: &disabled,
						},
						Ingress: &vzapi.IngressNginxComponent{
							Enabled: &disabled,
						},
					},
				},
			},
			numberOfErrors: 2,
		},
	}
	config.TestProfilesDir = "../../../manifests/profiles/v1alpha1"
	defer func() { config.TestProfilesDir = "" }()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := ComponentValidatorImpl{}
			got := c.ValidateUpdate(tt.old, tt.new)
			if len(got) != tt.numberOfErrors {
				t.Errorf("ValidateUpdate() = %v, numberOfErrors %v", len(got), tt.numberOfErrors)
			}
		})
	}
}
