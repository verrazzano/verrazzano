// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package thanos

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1beta1"
	"github.com/verrazzano/verrazzano/platform-operator/constants"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/authproxy"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	"k8s.io/apimachinery/pkg/types"
	k8scheme "k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

const profilesRelativePath = "../../../../manifests/profiles"

// TestThanosEnabled tests if Thanos is enabled
// GIVEN a call to IsEnabled
// WHEN the VZ CR is populated
// THEN a boolean is returned
func TestThanosEnabled(t *testing.T) {
	trueVal := true
	falseVal := false
	crA1 := &v1alpha1.Verrazzano{}
	crB1 := &v1beta1.Verrazzano{}

	crA1NilComp := crA1.DeepCopy()
	crA1NilComp.Spec.Components.Thanos = nil
	crA1NilEnabled := crA1.DeepCopy()
	crA1NilEnabled.Spec.Components.Thanos = &v1alpha1.ThanosComponent{Enabled: nil}
	crA1Enabled := crA1.DeepCopy()
	crA1Enabled.Spec.Components.Thanos = &v1alpha1.ThanosComponent{Enabled: &trueVal}
	crA1Disabled := crA1.DeepCopy()
	crA1Disabled.Spec.Components.Thanos = &v1alpha1.ThanosComponent{Enabled: &falseVal}

	crB1NilComp := crB1.DeepCopy()
	crB1NilComp.Spec.Components.Thanos = nil
	crB1NilEnabled := crB1.DeepCopy()
	crB1NilEnabled.Spec.Components.Thanos = &v1beta1.ThanosComponent{Enabled: nil}
	crB1Enabled := crB1.DeepCopy()
	crB1Enabled.Spec.Components.Thanos = &v1beta1.ThanosComponent{Enabled: &trueVal}
	crB1Disabled := crB1.DeepCopy()
	crB1Disabled.Spec.Components.Thanos = &v1beta1.ThanosComponent{Enabled: &falseVal}

	tests := []struct {
		name         string
		verrazzanoA1 *v1alpha1.Verrazzano
		verrazzanoB1 *v1beta1.Verrazzano
		assertion    func(t assert.TestingT, value bool, msgAndArgs ...interface{}) bool
	}{
		{
			name:         "test v1alpha1 component nil",
			verrazzanoA1: crA1NilComp,
			assertion:    assert.False,
		},
		{
			name:         "test v1alpha1 enabled nil",
			verrazzanoA1: crA1NilEnabled,
			assertion:    assert.False,
		},
		{
			name:         "test v1alpha1 enabled",
			verrazzanoA1: crA1Enabled,
			assertion:    assert.True,
		},
		{
			name:         "test v1alpha1 disabled",
			verrazzanoA1: crA1Disabled,
			assertion:    assert.False,
		},
		{
			name:         "test v1beta1 component nil",
			verrazzanoB1: crB1NilComp,
			assertion:    assert.False,
		},
		{
			name:         "test v1beta1 enabled nil",
			verrazzanoB1: crB1NilEnabled,
			assertion:    assert.False,
		},
		{
			name:         "test v1beta1 enabled",
			verrazzanoB1: crB1Enabled,
			assertion:    assert.True,
		},
		{
			name:         "test v1beta1 disabled",
			verrazzanoB1: crB1Disabled,
			assertion:    assert.False,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.verrazzanoA1 != nil {
				tt.assertion(t, NewComponent().IsEnabled(spi.NewFakeContext(nil, tt.verrazzanoA1, tt.verrazzanoB1, false, profilesRelativePath).EffectiveCR()))
			}
			if tt.verrazzanoB1 != nil {
				tt.assertion(t, NewComponent().IsEnabled(spi.NewFakeContext(nil, tt.verrazzanoA1, tt.verrazzanoB1, false, profilesRelativePath).EffectiveCRV1Beta1()))
			}
		})
	}
}

// TestGetIngressNames tests the GetIngressNames for the Thanos component
func TestGetIngressNames(t *testing.T) {
	enabled := true
	disabled := false

	scheme := k8scheme.Scheme

	var tests = []struct {
		name     string
		vz       v1alpha1.Verrazzano
		ingNames []types.NamespacedName
	}{
		// GIVEN a call to GetIngressNames
		// WHEN all components are disabled
		// THEN no ingresses are returned
		{
			name: "TestGetIngress when all disabled",
			vz: v1alpha1.Verrazzano{
				Spec: v1alpha1.VerrazzanoSpec{
					Components: v1alpha1.ComponentSpec{
						AuthProxy: &v1alpha1.AuthProxyComponent{Enabled: &disabled},
						Ingress:   &v1alpha1.IngressNginxComponent{Enabled: &disabled},
						Thanos:    &v1alpha1.ThanosComponent{Enabled: &disabled},
						DNS: &v1alpha1.DNSComponent{
							OCI: &v1alpha1.OCI{
								DNSZoneName: "mydomain.com",
							},
						},
					},
				},
			},
		},
		// GIVEN a call to GetIngressNames
		// WHEN all Thanos is disabled
		// THEN no ingresses are returned
		{
			name: "TestGetIngress when Thanos disabled",
			vz: v1alpha1.Verrazzano{
				Spec: v1alpha1.VerrazzanoSpec{
					Components: v1alpha1.ComponentSpec{
						AuthProxy: &v1alpha1.AuthProxyComponent{Enabled: &enabled},
						Ingress:   &v1alpha1.IngressNginxComponent{Enabled: &enabled},
						Thanos:    &v1alpha1.ThanosComponent{Enabled: &disabled},
						DNS: &v1alpha1.DNSComponent{
							OCI: &v1alpha1.OCI{
								DNSZoneName: "mydomain.com",
							},
						},
					},
				},
			},
		},
		// GIVEN a call to GetIngressNames
		// WHEN all NGINX is disabled
		// THEN no ingresses are returned
		{
			name: "TestGetIngress when NGINX disabled",
			vz: v1alpha1.Verrazzano{
				Spec: v1alpha1.VerrazzanoSpec{
					Components: v1alpha1.ComponentSpec{
						AuthProxy: &v1alpha1.AuthProxyComponent{Enabled: &enabled},
						Ingress:   &v1alpha1.IngressNginxComponent{Enabled: &disabled},
						Thanos:    &v1alpha1.ThanosComponent{Enabled: &enabled},
						DNS: &v1alpha1.DNSComponent{
							OCI: &v1alpha1.OCI{
								DNSZoneName: "mydomain.com",
							},
						},
					},
				},
			},
		},
		// GIVEN a call to GetIngressNames
		// WHEN the authproxy is disabled
		// THEN and ingress in the verrazzano-system namespace is returned
		{
			name: "TestGetIngress when Authproxy disabled",
			vz: v1alpha1.Verrazzano{
				Spec: v1alpha1.VerrazzanoSpec{
					Components: v1alpha1.ComponentSpec{
						AuthProxy: &v1alpha1.AuthProxyComponent{Enabled: &disabled},
						Ingress:   &v1alpha1.IngressNginxComponent{Enabled: &enabled},
						Thanos:    &v1alpha1.ThanosComponent{Enabled: &enabled},
						DNS: &v1alpha1.DNSComponent{
							OCI: &v1alpha1.OCI{
								DNSZoneName: "mydomain.com",
							},
						},
					},
				},
			},
			ingNames: []types.NamespacedName{
				{Namespace: constants.VerrazzanoSystemNamespace, Name: constants.ThanosQueryIngress},
			},
		},
		// GIVEN a call to GetIngressNames
		// WHEN all components are enabled
		// THEN an ingress in the authproxy namespace is returned
		{
			name: "TestGetIngress when all enabled",
			vz: v1alpha1.Verrazzano{
				Spec: v1alpha1.VerrazzanoSpec{
					Components: v1alpha1.ComponentSpec{
						AuthProxy: &v1alpha1.AuthProxyComponent{Enabled: &enabled},
						Ingress:   &v1alpha1.IngressNginxComponent{Enabled: &enabled},
						Thanos:    &v1alpha1.ThanosComponent{Enabled: &enabled},
						DNS: &v1alpha1.DNSComponent{
							OCI: &v1alpha1.OCI{
								DNSZoneName: "mydomain.com",
							},
						},
					},
				},
			},
			ingNames: []types.NamespacedName{
				{Namespace: authproxy.ComponentNamespace, Name: constants.ThanosQueryIngress},
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			client := fake.NewClientBuilder().WithScheme(scheme).Build()
			ctx := spi.NewFakeContext(client, &test.vz, nil, false)
			nsn := NewComponent().GetIngressNames(ctx)
			assert.Equal(t, nsn, test.ingNames)
		})
	}
}

// TestGetCertificateNames tests the GetCertificateNames for the Thanos component
func TestGetCertificateNames(t *testing.T) {
	enabled := true
	disabled := false

	scheme := k8scheme.Scheme

	var tests = []struct {
		name     string
		vz       v1alpha1.Verrazzano
		ingNames []types.NamespacedName
	}{
		// GIVEN a call to GetCertificateNames
		// WHEN all components are disabled
		// THEN no certificates are returned
		{
			name: "TestGetIngress when all disabled",
			vz: v1alpha1.Verrazzano{
				Spec: v1alpha1.VerrazzanoSpec{
					Components: v1alpha1.ComponentSpec{
						AuthProxy: &v1alpha1.AuthProxyComponent{Enabled: &disabled},
						Ingress:   &v1alpha1.IngressNginxComponent{Enabled: &disabled},
						Thanos:    &v1alpha1.ThanosComponent{Enabled: &disabled},
						DNS: &v1alpha1.DNSComponent{
							OCI: &v1alpha1.OCI{
								DNSZoneName: "mydomain.com",
							},
						},
					},
				},
			},
		},
		// GIVEN a call to GetCertificateNames
		// WHEN all Thanos is disabled
		// THEN no certificates are returned
		{
			name: "TestGetIngress when Thanos disabled",
			vz: v1alpha1.Verrazzano{
				Spec: v1alpha1.VerrazzanoSpec{
					Components: v1alpha1.ComponentSpec{
						AuthProxy: &v1alpha1.AuthProxyComponent{Enabled: &enabled},
						Ingress:   &v1alpha1.IngressNginxComponent{Enabled: &enabled},
						Thanos:    &v1alpha1.ThanosComponent{Enabled: &disabled},
						DNS: &v1alpha1.DNSComponent{
							OCI: &v1alpha1.OCI{
								DNSZoneName: "mydomain.com",
							},
						},
					},
				},
			},
		},
		// GIVEN a call to GetCertificateNames
		// WHEN all NGINX is disabled
		// THEN no certificates are returned
		{
			name: "TestGetIngress when NGINX disabled",
			vz: v1alpha1.Verrazzano{
				Spec: v1alpha1.VerrazzanoSpec{
					Components: v1alpha1.ComponentSpec{
						AuthProxy: &v1alpha1.AuthProxyComponent{Enabled: &enabled},
						Ingress:   &v1alpha1.IngressNginxComponent{Enabled: &disabled},
						Thanos:    &v1alpha1.ThanosComponent{Enabled: &enabled},
						DNS: &v1alpha1.DNSComponent{
							OCI: &v1alpha1.OCI{
								DNSZoneName: "mydomain.com",
							},
						},
					},
				},
			},
		},
		// GIVEN a call to GetCertificateNames
		// WHEN the authproxy is disabled
		// THEN and certificate in the verrazzano-system namespace is returned
		{
			name: "TestGetIngress when Authproxy disabled",
			vz: v1alpha1.Verrazzano{
				Spec: v1alpha1.VerrazzanoSpec{
					Components: v1alpha1.ComponentSpec{
						AuthProxy: &v1alpha1.AuthProxyComponent{Enabled: &disabled},
						Ingress:   &v1alpha1.IngressNginxComponent{Enabled: &enabled},
						Thanos:    &v1alpha1.ThanosComponent{Enabled: &enabled},
						DNS: &v1alpha1.DNSComponent{
							OCI: &v1alpha1.OCI{
								DNSZoneName: "mydomain.com",
							},
						},
					},
				},
			},
			ingNames: []types.NamespacedName{
				{Namespace: constants.VerrazzanoSystemNamespace, Name: queryCertificateName},
			},
		},
		// GIVEN a call to GetCertificateNames
		// WHEN all components are enabled
		// THEN an certificate in the authproxy namespace is returned
		{
			name: "TestGetIngress when all enabled",
			vz: v1alpha1.Verrazzano{
				Spec: v1alpha1.VerrazzanoSpec{
					Components: v1alpha1.ComponentSpec{
						AuthProxy: &v1alpha1.AuthProxyComponent{Enabled: &enabled},
						Ingress:   &v1alpha1.IngressNginxComponent{Enabled: &enabled},
						Thanos:    &v1alpha1.ThanosComponent{Enabled: &enabled},
						DNS: &v1alpha1.DNSComponent{
							OCI: &v1alpha1.OCI{
								DNSZoneName: "mydomain.com",
							},
						},
					},
				},
			},
			ingNames: []types.NamespacedName{
				{Namespace: authproxy.ComponentNamespace, Name: queryCertificateName},
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			client := fake.NewClientBuilder().WithScheme(scheme).Build()
			ctx := spi.NewFakeContext(client, &test.vz, nil, false)
			nsn := NewComponent().GetCertificateNames(ctx)
			assert.Equal(t, nsn, test.ingNames)
		})
	}
}
