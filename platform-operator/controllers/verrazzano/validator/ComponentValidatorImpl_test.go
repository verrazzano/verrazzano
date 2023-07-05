// Copyright (c) 2022, 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package validator

import (
	"github.com/verrazzano/verrazzano/pkg/constants"
	"github.com/verrazzano/verrazzano/pkg/k8sutil"
	"github.com/verrazzano/verrazzano/pkg/log/vzlog"
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	vzapibeta "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1beta1"
	"github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/validators"
	cmcommon "github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/certmanager/common"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/common"
	"github.com/verrazzano/verrazzano/platform-operator/internal/config"
	"github.com/verrazzano/verrazzano/platform-operator/internal/vzconfig"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	apiextv1fake "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset/fake"
	apiextv1 "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset/typed/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	k8sfake "k8s.io/client-go/kubernetes/fake"
	k8scheme "k8s.io/client-go/kubernetes/scheme"
	appsv1Cli "k8s.io/client-go/kubernetes/typed/apps/v1"
	corev1Cli "k8s.io/client-go/kubernetes/typed/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
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
	corev1Client := func(_ ...vzlog.VerrazzanoLogger) (corev1Cli.CoreV1Interface, error) {
		return k8sfake.NewSimpleClientset(
			&corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name: constants.CertManagerNamespace,
					Labels: map[string]string{
						constants.LabelVerrazzanoNamespace: constants.CertManagerNamespace,
					},
				},
			}).CoreV1(), nil
	}
	k8sutil.GetCoreV1Func = corev1Client
	cmcommon.GetClientFunc = corev1Client
	k8sutil.GetAppsV1Func = func(_ ...vzlog.VerrazzanoLogger) (appsv1Cli.AppsV1Interface, error) {
		return k8sfake.NewSimpleClientset().AppsV1(), nil
	}
	k8sutil.GetDynamicClientFunc = common.MockDynamicClient()

	k8sutil.GetAPIExtV1ClientFunc = func() (apiextv1.ApiextensionsV1Interface, error) {
		return apiextv1fake.NewSimpleClientset(createCertManagerCRDs()...).ApiextensionsV1(), nil
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
		{
			name: "CertManager Certificates and ClusterIssuer configured",
			vz: &vzapi.Verrazzano{
				Spec: vzapi.VerrazzanoSpec{
					Components: vzapi.ComponentSpec{
						CertManager: &vzapi.CertManagerComponent{
							Certificate: vzapi.Certificate{
								Acme: vzapi.Acme{
									EmailAddress: "joe@blow.com",
									Environment:  "production",
									Provider:     vzapi.LetsEncrypt,
								},
							},
						},
						ClusterIssuer: &vzapi.ClusterIssuerComponent{
							IssuerConfig: vzapi.IssuerConfig{
								LetsEncrypt: &vzapi.LetsEncryptACMEIssuer{
									EmailAddress: "foo@bar.com",
									Environment:  "staging",
								},
							},
						},
					},
				},
			},
			numberOfErrors: 1,
		},
	}
	config.TestProfilesDir = testProfilesDirectory
	defer func() {
		config.TestProfilesDir = ""
		k8sutil.GetCoreV1Func = k8sutil.GetCoreV1Client
		k8sutil.GetAppsV1Func = k8sutil.GetAppsV1Client
		k8sutil.GetDynamicClientFunc = k8sutil.GetDynamicClient
		cmcommon.ResetCoreV1ClientFunc()
		k8sutil.ResetGetAPIExtV1ClientFunc()
	}()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := ComponentValidatorImpl{}
			got := c.ValidateInstall(tt.vz)
			if len(got) != tt.numberOfErrors {
				for _, err := range got {
					t.Logf("Unexpected error: %s", err.Error())
				}
				t.Errorf("ValidateInstall() = %v, numberOfErrors %v", tt.numberOfErrors, len(got))
			}
		})
	}
}

// TestComponentValidatorImpl_ValidateInstallV1Beta1 tests the ValidateInstallV1Beta1 function
// GIVEN a valid CR
// WHEN ValidateInstallV1Beta1 is called
// THEN ensure that no error is raised
func TestComponentValidatorImpl_ValidateInstallV1Beta1(t *testing.T) {
	vzconfig.GetControllerRuntimeClient = func(scheme *runtime.Scheme) (client.Client, error) {
		return fake.NewClientBuilder().WithScheme(newScheme()).WithObjects().Build(), nil
	}
	defer func() { vzconfig.GetControllerRuntimeClient = validators.GetClient }()
	corev1Client := func(_ ...vzlog.VerrazzanoLogger) (corev1Cli.CoreV1Interface, error) {
		return k8sfake.NewSimpleClientset(
			&corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name: constants.CertManagerNamespace,
					Labels: map[string]string{
						constants.LabelVerrazzanoNamespace: constants.CertManagerNamespace,
					},
				},
			}).CoreV1(), nil
	}
	k8sutil.GetCoreV1Func = corev1Client
	cmcommon.GetClientFunc = corev1Client
	k8sutil.GetAPIExtV1ClientFunc = func() (apiextv1.ApiextensionsV1Interface, error) {
		return apiextv1fake.NewSimpleClientset(createCertManagerCRDs()...).ApiextensionsV1(), nil
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
		{
			name: "CertManager Certificates and ClusterIssuer configured",
			vz: &vzapibeta.Verrazzano{
				Spec: vzapibeta.VerrazzanoSpec{
					Components: vzapibeta.ComponentSpec{
						CertManager: &vzapibeta.CertManagerComponent{
							Certificate: vzapibeta.Certificate{
								Acme: vzapibeta.Acme{
									EmailAddress: "joe@blow.com",
									Environment:  "production",
									Provider:     vzapibeta.LetsEncrypt,
								},
							},
						},
						ClusterIssuer: &vzapibeta.ClusterIssuerComponent{
							IssuerConfig: vzapibeta.IssuerConfig{
								LetsEncrypt: &vzapibeta.LetsEncryptACMEIssuer{
									EmailAddress: "foo@bar.com",
									Environment:  "staging",
								},
							},
						},
					},
				},
			},
			numberOfErrors: 1,
		},
	}

	config.TestProfilesDir = testProfilesDirectory
	defer func() {
		config.TestProfilesDir = ""
		k8sutil.GetCoreV1Func = k8sutil.GetCoreV1Client
		k8sutil.GetAppsV1Func = k8sutil.GetAppsV1Client
		k8sutil.GetDynamicClientFunc = k8sutil.GetDynamicClient
		cmcommon.ResetCoreV1ClientFunc()
		k8sutil.ResetGetAPIExtV1ClientFunc()
	}()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := ComponentValidatorImpl{}
			got := c.ValidateInstallV1Beta1(tt.vz)
			if len(got) != tt.numberOfErrors {
				for _, err := range got {
					t.Logf("Unexpected error: %s", err.Error())
				}
				t.Errorf("ValidateInstallV1Beta1() = %v, numberOfErrors %v", tt.numberOfErrors, len(got))
			}
		})
	}
}

// TestComponentValidatorImpl_ValidateUpdate tests the ValidateUpdate function
// GIVEN a valid CR
// WHEN ValidateUpdate is called
// THEN ensure that no error is raised
func TestComponentValidatorImpl_ValidateUpdate(t *testing.T) {
	corev1Client := func(_ ...vzlog.VerrazzanoLogger) (corev1Cli.CoreV1Interface, error) {
		return k8sfake.NewSimpleClientset(
			&corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name: constants.CertManagerNamespace,
					Labels: map[string]string{
						constants.LabelVerrazzanoNamespace: constants.CertManagerNamespace,
					},
				},
			}).CoreV1(), nil
	}
	k8sutil.GetCoreV1Func = corev1Client
	cmcommon.GetClientFunc = corev1Client
	k8sutil.GetAPIExtV1ClientFunc = func() (apiextv1.ApiextensionsV1Interface, error) {
		return apiextv1fake.NewSimpleClientset(createCertManagerCRDs()...).ApiextensionsV1(), nil
	}
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
		{
			name: "CertManager Certificates and ClusterIssuer configured",
			old:  &vzapi.Verrazzano{},
			new: &vzapi.Verrazzano{
				Spec: vzapi.VerrazzanoSpec{
					Components: vzapi.ComponentSpec{
						CertManager: &vzapi.CertManagerComponent{
							Certificate: vzapi.Certificate{
								Acme: vzapi.Acme{
									EmailAddress: "joe@blow.com",
									Environment:  "production",
									Provider:     vzapi.LetsEncrypt,
								},
							},
						},
						ClusterIssuer: &vzapi.ClusterIssuerComponent{
							IssuerConfig: vzapi.IssuerConfig{
								LetsEncrypt: &vzapi.LetsEncryptACMEIssuer{
									EmailAddress: "foo@bar.com",
									Environment:  "staging",
								},
							},
						},
					},
				},
			},
			numberOfErrors: 1,
		},
	}
	config.TestProfilesDir = testProfilesDirectory
	defer func() {
		config.TestProfilesDir = ""
		cmcommon.ResetCoreV1ClientFunc()
		k8sutil.ResetGetAPIExtV1ClientFunc()
	}()
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
	corev1Client := func(_ ...vzlog.VerrazzanoLogger) (corev1Cli.CoreV1Interface, error) {
		return k8sfake.NewSimpleClientset(
			&corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name: constants.CertManagerNamespace,
					Labels: map[string]string{
						constants.LabelVerrazzanoNamespace: constants.CertManagerNamespace,
					},
				},
			}).CoreV1(), nil
	}
	k8sutil.GetCoreV1Func = corev1Client
	cmcommon.GetClientFunc = corev1Client
	k8sutil.GetAPIExtV1ClientFunc = func() (apiextv1.ApiextensionsV1Interface, error) {
		return apiextv1fake.NewSimpleClientset(createCertManagerCRDs()...).ApiextensionsV1(), nil
	}
	vzconfig.GetControllerRuntimeClient = func(scheme *runtime.Scheme) (client.Client, error) {
		return fake.NewClientBuilder().WithScheme(newScheme()).WithObjects().Build(), nil
	}
	defer func() {
		vzconfig.GetControllerRuntimeClient = validators.GetClient
		cmcommon.ResetCoreV1ClientFunc()
		k8sutil.ResetGetAPIExtV1ClientFunc()
	}()
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
			name: "CertManager Certificates and ClusterIssuer configured",
			old:  &vzapibeta.Verrazzano{},
			new: &vzapibeta.Verrazzano{
				Spec: vzapibeta.VerrazzanoSpec{
					Components: vzapibeta.ComponentSpec{
						CertManager: &vzapibeta.CertManagerComponent{
							Certificate: vzapibeta.Certificate{
								Acme: vzapibeta.Acme{
									EmailAddress: "joe@blow.com",
									Environment:  "production",
									Provider:     vzapibeta.LetsEncrypt,
								},
							},
						},
						ClusterIssuer: &vzapibeta.ClusterIssuerComponent{
							IssuerConfig: vzapibeta.IssuerConfig{
								LetsEncrypt: &vzapibeta.LetsEncryptACMEIssuer{
									EmailAddress: "foo@bar.com",
									Environment:  "staging",
								},
							},
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
				t.Logf("Errors: %v", got)
				t.Errorf("ValidateUpdate() = %v, numberOfErrors %v", len(got), tt.numberOfErrors)
			}
		})
	}
}

func newScheme() *runtime.Scheme {
	scheme := runtime.NewScheme()
	k8scheme.AddToScheme(scheme)
	return scheme
}

func createCertManagerCRDs() []runtime.Object {
	var runtimeObjs []runtime.Object
	for _, crd := range cmcommon.GetRequiredCertManagerCRDNames() {
		runtimeObjs = append(runtimeObjs, newCRD(crd))
	}
	return runtimeObjs
}

func newCRD(name string) client.Object {
	crd := &v1.CustomResourceDefinition{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
	}
	return crd
}
