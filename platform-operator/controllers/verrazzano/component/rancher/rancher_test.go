// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package rancher

import (
	"fmt"
	"github.com/stretchr/testify/assert"
	vzlog "github.com/verrazzano/verrazzano/pkg/log/progress"
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/common"
	"go.uber.org/zap/zaptest"
	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/api/core/v1"
	networking "k8s.io/api/networking/v1"
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
	return scheme
}

func getTestLogger(t *testing.T) vzlog.VerrazzanoLogger {
	l := zaptest.NewLogger(t).Sugar()
	return vzlog.EnsureLogContext("test").EnsureLogger("test", l,l)
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

func createRancherPodList() v1.PodList {
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
