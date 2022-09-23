// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package validator

import (
	"testing"

	"github.com/verrazzano/verrazzano/pkg/k8sutil"
	"github.com/verrazzano/verrazzano/pkg/log/vzlog"
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/internal/config"
	k8sfake "k8s.io/client-go/kubernetes/fake"
	appsv1Cli "k8s.io/client-go/kubernetes/typed/apps/v1"
	corev1Cli "k8s.io/client-go/kubernetes/typed/core/v1"
)

var disabled = false

// TestComponentValidatorImpl_ValidateInstall tests the ValidateInstall function
// GIVEN a valid CR
// WHEN ValidateInstall is called
// THEN ensure that no error is raised
func TestComponentValidatorImpl_ValidateInstall(t *testing.T) {
	k8sutil.GetCoreV1Func = func(_ ...vzlog.VerrazzanoLogger) (corev1Cli.CoreV1Interface, error) {
		return k8sfake.NewSimpleClientset().CoreV1(), nil
	}
	k8sutil.GetAppsV1Func = func(_ ...vzlog.VerrazzanoLogger) (appsv1Cli.AppsV1Interface, error) {
		return k8sfake.NewSimpleClientset().AppsV1(), nil
	}
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
	config.TestProfilesDir = "../../../manifests/profiles"
	defer func() {
		config.TestProfilesDir = ""
		k8sutil.GetCoreV1Func = k8sutil.GetCoreV1Client
		k8sutil.GetAppsV1Func = k8sutil.GetAppsV1Client
	}()
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
	config.TestProfilesDir = "../../../manifests/profiles"
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
