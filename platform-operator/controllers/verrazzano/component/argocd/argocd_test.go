// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package argocd

import (
	"fmt"
	certv1 "github.com/jetstack/cert-manager/pkg/apis/certmanager/v1"
	"github.com/stretchr/testify/assert"
	"github.com/verrazzano/verrazzano/pkg/log/vzlog"
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/constants"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/common"
	admv1 "k8s.io/api/admissionregistration/v1"
	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	networking "k8s.io/api/networking/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	v12 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"testing"
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
					External: &vzapi.External{Suffix: common.ArgoCDName},
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
					External: &vzapi.External{Suffix: common.ArgoCDName},
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
			Namespace: constants.ArgoCDNamespace,
			Name:      common.ArgoCDIngressCAName,
		},
		Data: map[string][]byte{
			common.ArgoCDCACert: []byte("blahblah"),
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

// TestUseAdditionalCAs verifies that additional CAs should be used when specified in the Verrazzano CR
// GIVEN a Verrazzano CR
//
//	WHEN useAdditionalCAs is called
//	THEN useAdditionalCAs return true or false if additional CAs are required
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

// TestGetArgoCDHostname verifies the ArgocD hostname can be generated
// GIVEN a Verrazzano CR
//
//	WHEN getArgoCDHostname is called
//	THEN getArgoCDHostname should return the ArgoCD hostname
func TestGetArgoCDHostname(t *testing.T) {
	expected := fmt.Sprintf("%s.%s.argocd", common.ArgoCDName, vzAcmeDev.Spec.EnvironmentName)
	actual, _ := getArgoCDHostname(fake.NewClientBuilder().WithScheme(getScheme()).Build(), &vzAcmeDev)
	assert.Equal(t, expected, actual)
}

// TestGetArgoCDHostnameNotFound verifies the ArgoCD hostname can not be generated in the CR is invalid
// GIVEN an invalid Verrazzano CR
//
//	WHEN getArgoCDHostname is called
//	THEN getArgoCDHostname should return an error
func TestGetArgoCDHostnameNotFound(t *testing.T) {
	_, err := getArgoCDHostname(fake.NewClientBuilder().WithScheme(getScheme()).Build(), &vzapi.Verrazzano{})
	assert.NotNil(t, err)
}
