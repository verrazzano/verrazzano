// Copyright (c) 2022, 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package validator

import (
	"github.com/verrazzano/verrazzano/pkg/k8sutil"
	"github.com/verrazzano/verrazzano/pkg/log/vzlog"
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	vzapibeta "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1beta1"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/common"
	"github.com/verrazzano/verrazzano/platform-operator/internal/config"
	k8sfake "k8s.io/client-go/kubernetes/fake"
	appsv1Cli "k8s.io/client-go/kubernetes/typed/apps/v1"
	corev1Cli "k8s.io/client-go/kubernetes/typed/core/v1"
	"testing"
)

var disabled = false

const disabledCertAndIngress string = "disabled cert and ingress"
const testProfilesDirectory string = "../../../manifests/profiles"

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
	k8sutil.GetDynamicClientFunc = common.MockDynamicClient()

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
			name: disabledCertAndIngress,
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
	config.TestProfilesDir = testProfilesDirectory
	defer func() {
		config.TestProfilesDir = ""
		k8sutil.GetCoreV1Func = k8sutil.GetCoreV1Client
		k8sutil.GetAppsV1Func = k8sutil.GetAppsV1Client
		k8sutil.GetDynamicClientFunc = k8sutil.GetDynamicClient
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

// TestComponentValidatorImpl_ValidateInstallV1Beta1 tests the ValidateInstallV1Beta1 function
// GIVEN a valid CR
// WHEN ValidateInstallV1Beta1 is called
// THEN ensure that no error is raised
func TestComponentValidatorImpl_ValidateInstallV1Beta1(t *testing.T) {
	k8sutil.GetCoreV1Func = func(_ ...vzlog.VerrazzanoLogger) (corev1Cli.CoreV1Interface, error) {
		return k8sfake.NewSimpleClientset().CoreV1(), nil
	}
	k8sutil.GetAppsV1Func = func(_ ...vzlog.VerrazzanoLogger) (appsv1Cli.AppsV1Interface, error) {
		return k8sfake.NewSimpleClientset().AppsV1(), nil
	}
	k8sutil.GetDynamicClientFunc = common.MockDynamicClient()
	tests := []struct {
		name           string
		vz             *vzapibeta.Verrazzano
		numberOfErrors int
	}{
		{
			name:           "default CR",
			vz:             &vzapibeta.Verrazzano{},
			numberOfErrors: 0,
		},
		{
			name: disabledCertAndIngress,
			vz: &vzapibeta.Verrazzano{
				Spec: vzapibeta.VerrazzanoSpec{
					Components: vzapibeta.ComponentSpec{
						CertManager: &vzapibeta.CertManagerComponent{
							Enabled: &disabled,
						},
						IngressNGINX: &vzapibeta.IngressNginxComponent{
							Enabled: &disabled,
						},
					},
				},
			},
			numberOfErrors: 0,
		},
	}

	config.TestProfilesDir = testProfilesDirectory
	defer func() {
		config.TestProfilesDir = ""
		k8sutil.GetCoreV1Func = k8sutil.GetCoreV1Client
		k8sutil.GetAppsV1Func = k8sutil.GetAppsV1Client
		k8sutil.GetDynamicClientFunc = k8sutil.GetDynamicClient
	}()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := ComponentValidatorImpl{}
			got := c.ValidateInstallV1Beta1(tt.vz)
			if len(got) != tt.numberOfErrors {
				t.Errorf("ValidateInstallV1Beta1() = %v, numberOfErrors %v", len(got), tt.numberOfErrors)
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
			name: disabledCertAndIngress,
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
	config.TestProfilesDir = testProfilesDirectory
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

// TestComponentValidatorImpl_ValidateUpdateV1Beta1 tests the ValidateUpdate function
// GIVEN a valid CR
// WHEN ValidateUpdateV1Beta1 is called
// THEN ensure that no error is raised
func TestComponentValidatorImpl_ValidateUpdateV1Beta1(t *testing.T) {
	tests := []struct {
		name           string
		old            *vzapibeta.Verrazzano
		new            *vzapibeta.Verrazzano
		numberOfErrors int
	}{
		{
			name:           "no change",
			old:            &vzapibeta.Verrazzano{},
			new:            &vzapibeta.Verrazzano{},
			numberOfErrors: 0,
		},
		{
			name: "disable rancher",
			old:  &vzapibeta.Verrazzano{},
			new: &vzapibeta.Verrazzano{
				Spec: vzapibeta.VerrazzanoSpec{
					Components: vzapibeta.ComponentSpec{
						Rancher: &vzapibeta.RancherComponent{
							Enabled: &disabled,
						},
					},
				},
			},
			numberOfErrors: 1,
		},
		{
			name: "disable cert",
			old:  &vzapibeta.Verrazzano{},
			new: &vzapibeta.Verrazzano{
				Spec: vzapibeta.VerrazzanoSpec{
					Components: vzapibeta.ComponentSpec{
						CertManager: &vzapibeta.CertManagerComponent{
							Enabled: &disabled,
						},
					},
				},
			},
			numberOfErrors: 1,
		},
		{

			name: disabledCertAndIngress,
			old:  &vzapibeta.Verrazzano{},
			new: &vzapibeta.Verrazzano{
				Spec: vzapibeta.VerrazzanoSpec{
					Components: vzapibeta.ComponentSpec{
						CertManager: &vzapibeta.CertManagerComponent{
							Enabled: &disabled,
						},
						IngressNGINX: &vzapibeta.IngressNginxComponent{
							Enabled: &disabled,
						},
					},
				},
			},
			numberOfErrors: 2,
		},
	}

	config.TestProfilesDir = testProfilesDirectory
	defer func() { config.TestProfilesDir = "" }()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := ComponentValidatorImpl{}
			got := c.ValidateUpdateV1Beta1(tt.old, tt.new)
			if len(got) != tt.numberOfErrors {
				t.Errorf("ValidateUpdate() = %v, numberOfErrors %v", len(got), tt.numberOfErrors)
			}
		})
	}
}
