// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package keycloak

import (
	certmanager "github.com/jetstack/cert-manager/pkg/apis/certmanager/v1"
	"github.com/stretchr/testify/assert"
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8scheme "k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"testing"
)

var kcComponent = NewComponent()

func TestIsEnabled(t *testing.T) {
	disabled := false
	var tests = []struct {
		name      string
		vz        *vzapi.Verrazzano
		isEnabled bool
	}{
		{
			"disabled",
			&vzapi.Verrazzano{
				Spec: vzapi.VerrazzanoSpec{
					Components: vzapi.ComponentSpec{
						Keycloak: &vzapi.KeycloakComponent{
							Enabled: &disabled,
						},
					},
				},
			},
			false,
		},
		{
			"enabled/nil",
			&vzapi.Verrazzano{
				Spec: vzapi.VerrazzanoSpec{
					Components: vzapi.ComponentSpec{
						Keycloak: &vzapi.KeycloakComponent{
							Enabled: nil,
						},
					},
				},
			},
			true,
		},
		{
			"enabled",
			testVZ,
			true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := spi.NewFakeContext(fake.NewFakeClientWithScheme(k8scheme.Scheme), tt.vz, false)
			assert.Equal(t, tt.isEnabled, isKeycloakEnabled(ctx))
		})
	}
}

func TestIsReady(t *testing.T) {
	readyCert := &certmanager.Certificate{
		ObjectMeta: metav1.ObjectMeta{
			Name:      getCertName(testVZ),
			Namespace: ComponentName,
		},
		Status: certmanager.CertificateStatus{
			Conditions: []certmanager.CertificateCondition{
				{
					Type: "Ready",
				},
			},
		},
	}
	scheme := k8scheme.Scheme
	_ = certmanager.AddToScheme(scheme)
	var tests = []struct {
		name    string
		c       client.Client
		isReady bool
	}{
		{
			"should not be ready when certificate not found",
			fake.NewFakeClientWithScheme(scheme),
			false,
		},
		{
			"should not be ready when certificate has no status",
			fake.NewFakeClientWithScheme(scheme, &certmanager.Certificate{
				ObjectMeta: metav1.ObjectMeta{
					Name:      getCertName(testVZ),
					Namespace: ComponentName,
				},
			}),
			false,
		},
		{
			"should not be ready when certificate status is not ready",
			fake.NewFakeClientWithScheme(scheme, &certmanager.Certificate{
				ObjectMeta: metav1.ObjectMeta{
					Name:      getCertName(testVZ),
					Namespace: ComponentName,
				},
				Status: certmanager.CertificateStatus{
					Conditions: []certmanager.CertificateCondition{
						{
							Type: "NotReady",
						},
					},
				},
			}),
			false,
		},
		{
			"should not be ready when certificate status is ready but statefulset is not ready",
			fake.NewFakeClientWithScheme(scheme, readyCert),
			false,
		},
		{
			"should be ready when certificate status is ready and statefulset is ready",
			fake.NewFakeClientWithScheme(scheme, readyCert, &appsv1.StatefulSet{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: ComponentName,
					Name:      ComponentName,
				},
				Status: appsv1.StatefulSetStatus{
					ReadyReplicas: 1,
				},
			}),
			true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := spi.NewFakeContext(tt.c, testVZ, false)
			assert.Equal(t, tt.isReady, kcComponent.IsReady(ctx))
		})
	}
}
