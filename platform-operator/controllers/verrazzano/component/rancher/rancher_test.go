// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package rancher

import (
	"fmt"
	"github.com/stretchr/testify/assert"
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"go.uber.org/zap"
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
					External: &vzapi.External{Suffix: ComponentName},
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
					External: &vzapi.External{Suffix: ComponentName},
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

func getTestLogger(t *testing.T) *zap.SugaredLogger {
	return zaptest.NewLogger(t).Sugar()
}

func createCASecret() v1.Secret {
	return v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: defaultSecretNamespace,
			Name:      defaultVerrazzanoName,
		},
		Data: map[string][]byte{
			"ca.cert": []byte("blahblah"),
		},
	}
}

func createRancherIngress() networking.Ingress {
	ingresses := []v1.LoadBalancerIngress{
		{"ip", "hostname"},
	}
	objectMeta := metav1.ObjectMeta{
		Name:      ComponentName,
		Namespace: ComponentNamespace,
	}
	return networking.Ingress{
		ObjectMeta: objectMeta,
		Status: networking.IngressStatus{
			LoadBalancer: v1.LoadBalancerStatus{
				Ingress: ingresses,
			},
		},
	}
}

func createRancherPodList() v1.PodList {
	return v1.PodList{
		Items: []v1.Pod{
			{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "rancherpod",
					Namespace: ComponentNamespace,
					Labels: map[string]string{
						"app": ComponentName,
					},
				},
			},
		},
	}
}

func createAdminSecret() v1.Secret {
	return v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: ComponentNamespace,
			Name:      adminSecretName,
		},
		Data: map[string][]byte{
			"password": []byte("foobar"),
		},
	}
}

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

func TestGetRancherHostname(t *testing.T) {
	expected := fmt.Sprintf("%s.%s.rancher", ComponentName, vzAcmeDev.Spec.EnvironmentName)
	actual, _ := getRancherHostname(fake.NewFakeClientWithScheme(getScheme()), &vzAcmeDev)
	assert.Equal(t, expected, actual)
}

func TestGetRancherHostnameNotFound(t *testing.T) {
	_, err := getRancherHostname(fake.NewFakeClientWithScheme(getScheme()), &vzapi.Verrazzano{})
	assert.NotNil(t, err)
}
