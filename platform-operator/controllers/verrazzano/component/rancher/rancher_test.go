// Copyright (c) 2021, 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package rancher

import (
	"fmt"
	"testing"

	certv1 "github.com/jetstack/cert-manager/pkg/apis/certmanager/v1"
	"github.com/stretchr/testify/assert"
	"github.com/verrazzano/verrazzano/pkg/log/vzlog"
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/common"
	admv1 "k8s.io/api/admissionregistration/v1"
	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	networking "k8s.io/api/networking/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	v12 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

var (
	vzAcmeDev = vzapi.Verrazzano{
		Spec: vzapi.VerrazzanoSpec{
			EnvironmentName: "ACME_DEV",
			Components: vzapi.ComponentSpec{
				CertManager: &vzapi.CertManagerComponent{
					Certificate: vzapi.Certificate{
						Acme: vzapi.Acme{
							Provider:     "foobar",
							EmailAddress: "foo@bar.com",
							Environment:  "dev",
						},
					},
				},
				DNS: &vzapi.DNSComponent{
					External: &vzapi.External{Suffix: common.RancherName},
				},
			},
		},
	}
	vzDefaultCA = vzapi.Verrazzano{
		Spec: vzapi.VerrazzanoSpec{
			EnvironmentName: "DefaultCA",
			Components: vzapi.ComponentSpec{
				CertManager: &vzapi.CertManagerComponent{Certificate: vzapi.Certificate{CA: vzapi.CA{
					SecretName:               defaultVerrazzanoName,
					ClusterResourceNamespace: defaultSecretNamespace,
				}}},
				DNS: &vzapi.DNSComponent{
					External: &vzapi.External{Suffix: common.RancherName},
				},
			},
		},
	}
)

func getScheme() *runtime.Scheme {
	scheme := runtime.NewScheme()
	_ = networking.AddToScheme(scheme)
	_ = appsv1.AddToScheme(scheme)
	_ = v1.AddToScheme(scheme)
	_ = certv1.AddToScheme(scheme)
	_ = admv1.AddToScheme(scheme)
	_ = rbacv1.AddToScheme(scheme)
	_ = v12.AddToScheme(scheme)
	return scheme
}

func getTestLogger(t *testing.T) vzlog.VerrazzanoLogger {
	return vzlog.DefaultLogger()
}

func createRootCASecret() v1.Secret {
	return v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: common.CattleSystem,
			Name:      common.RancherIngressCAName,
		},
		Data: map[string][]byte{
			common.RancherCACert: []byte("blahblah"),
		},
	}
}

func createCASecret() v1.Secret {
	return v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: defaultSecretNamespace,
			Name:      defaultVerrazzanoName,
		},
		Data: map[string][]byte{
			caCert: []byte("blahblah"),
		},
	}
}

func createRancherPodListWithAllRunning() v1.PodList {
	return v1.PodList{
		Items: []v1.Pod{
			{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "rancherpod",
					Namespace: common.CattleSystem,
					Labels: map[string]string{
						"app": common.RancherName,
					},
				},
				Status: v1.PodStatus{
					Conditions: []v1.PodCondition{
						{Type: "Ready", Status: "True"},
					},
				},
			},
		},
	}
}

func createClusterRoles(roleName string) rbacv1.ClusterRole {
	return rbacv1.ClusterRole{ObjectMeta: metav1.ObjectMeta{Name: roleName}}
}

func createRancherPodListWithNoneRunning() v1.PodList {
	return v1.PodList{
		Items: []v1.Pod{
			{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "rancherpod",
					Namespace: common.CattleSystem,
					Labels: map[string]string{
						"app": common.RancherName,
					},
				},
			},
		},
	}
}

func createRancherPodListWithLastRunning() v1.PodList {
	return v1.PodList{
		Items: []v1.Pod{
			{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "rancherpod1",
					Namespace: common.CattleSystem,
					Labels: map[string]string{
						"app": common.RancherName,
					},
				},
				Status: v1.PodStatus{
					Conditions: []v1.PodCondition{
						{Type: "Ready", Status: "False"},
					},
				},
			},
			{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "rancherpod2",
					Namespace: common.CattleSystem,
					Labels: map[string]string{
						"app": common.RancherName,
					},
				},
				Status: v1.PodStatus{
					Conditions: []v1.PodCondition{
						{Type: "Ready", Status: "True"},
					},
				},
			},
		},
	}
}

func createAdminSecret() v1.Secret {
	return v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: common.CattleSystem,
			Name:      common.RancherAdminSecret,
		},
		Data: map[string][]byte{
			"password": []byte("foobar"),
		},
	}
}

func createServerURLSetting() unstructured.Unstructured {
	serverURLSetting := unstructured.Unstructured{
		Object: map[string]interface{}{},
	}
	serverURLSetting.SetGroupVersionKind(GVKSetting)
	serverURLSetting.SetName(SettingServerURL)
	return serverURLSetting
}

func createFirstLoginSetting() unstructured.Unstructured {
	firstLoginSetting := unstructured.Unstructured{
		Object: map[string]interface{}{},
	}
	firstLoginSetting.SetGroupVersionKind(GVKSetting)
	firstLoginSetting.SetName(SettingFirstLogin)
	return firstLoginSetting
}

func createOciDriver() unstructured.Unstructured {
	ociDriver := unstructured.Unstructured{
		Object: map[string]interface{}{
			"spec": map[string]interface{}{
				"active": false,
			},
		},
	}
	ociDriver.SetGroupVersionKind(GVKNodeDriver)
	ociDriver.SetName(NodeDriverOCI)
	return ociDriver
}

func createOkeDriver() unstructured.Unstructured {
	okeDriver := unstructured.Unstructured{
		Object: map[string]interface{}{
			"spec": map[string]interface{}{
				"active": false,
			},
		},
	}
	okeDriver.SetGroupVersionKind(GVKKontainerDriver)
	okeDriver.SetName(KontainerDriverOKE)
	return okeDriver
}

func createKeycloakAuthConfig() unstructured.Unstructured {
	authConfig := unstructured.Unstructured{
		Object: map[string]interface{}{},
	}
	authConfig.SetGroupVersionKind(common.GVKAuthConfig)
	authConfig.SetName(common.AuthConfigKeycloak)
	return authConfig
}

func createLocalAuthConfig() unstructured.Unstructured {
	authConfig := unstructured.Unstructured{
		Object: map[string]interface{}{},
	}
	authConfig.SetGroupVersionKind(common.GVKAuthConfig)
	authConfig.SetName(AuthConfigLocal)
	return authConfig
}

func createKeycloakSecret() v1.Secret {
	return v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "keycloak",
			Name:      "keycloak-http",
		},
		Data: map[string][]byte{
			"password": []byte("blahblah"),
		},
	}
}

// TestUseAdditionalCAs verifies that additional CAs should be used when specified in the Verrazzano CR
// GIVEN a Verrazzano CR
//  WHEN useAdditionalCAs is called
//  THEN useAdditionalCAs return true or false if additional CAs are required
func TestUseAdditionalCAs(t *testing.T) {
	var tests = []struct {
		in  vzapi.Acme
		out bool
	}{
		{vzapi.Acme{Environment: "dev"}, true},
		{vzapi.Acme{Environment: "production"}, false},
	}

	for _, tt := range tests {
		t.Run(tt.in.Environment, func(t *testing.T) {
			assert.Equal(t, tt.out, useAdditionalCAs(tt.in))
		})
	}
}

// TestGetRancherHostname verifies the Rancher hostname can be generated
// GIVEN a Verrazzano CR
//  WHEN getRancherHostname is called
//  THEN getRancherHostname should return the Rancher hostname
func TestGetRancherHostname(t *testing.T) {
	expected := fmt.Sprintf("%s.%s.rancher", common.RancherName, vzAcmeDev.Spec.EnvironmentName)
	actual, _ := getRancherHostname(fake.NewFakeClientWithScheme(getScheme()), &vzAcmeDev)
	assert.Equal(t, expected, actual)
}

// TestGetRancherHostnameNotFound verifies the Rancher hostname can not be generated in the CR is invalid
// GIVEN an invalid Verrazzano CR
//  WHEN getRancherHostname is called
//  THEN getRancherHostname should return an error
func TestGetRancherHostnameNotFound(t *testing.T) {
	_, err := getRancherHostname(fake.NewFakeClientWithScheme(getScheme()), &vzapi.Verrazzano{})
	assert.NotNil(t, err)
}
