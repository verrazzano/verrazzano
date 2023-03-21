// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package thanos

import (
	"context"
	"fmt"
	"testing"

	certapiv1 "github.com/cert-manager/cert-manager/pkg/apis/certmanager/v1"
	cmmeta "github.com/cert-manager/cert-manager/pkg/apis/meta/v1"
	asserts "github.com/stretchr/testify/assert"
	"github.com/verrazzano/verrazzano/pkg/bom"
	"github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1beta1"
	"github.com/verrazzano/verrazzano/platform-operator/constants"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	"github.com/verrazzano/verrazzano/platform-operator/internal/config"
	v1 "k8s.io/api/core/v1"
	netv1 "k8s.io/api/networking/v1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	k8scheme "k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

const bomFilePathOverride = "../../../../verrazzano-bom.json"

// TestGetOverrides tests if Thanos overrides are properly collected
// GIVEN a call to GetOverrides
// WHEN the VZ CR Thanos component has overrides
// THEN the overrides are returned from this function
func TestGetOverrides(t *testing.T) {
	testKey := "test-key"
	testVal := "test-val"
	jsonVal := []byte(fmt.Sprintf("{\"%s\":\"%s\"}", testKey, testVal))

	vzA1CR := &v1alpha1.Verrazzano{}
	vzA1CROverrides := vzA1CR.DeepCopy()
	vzA1CROverrides.Spec.Components.Thanos = &v1alpha1.ThanosComponent{
		InstallOverrides: v1alpha1.InstallOverrides{
			ValueOverrides: []v1alpha1.Overrides{
				{
					Values: &apiextensionsv1.JSON{
						Raw: jsonVal,
					},
				},
			},
		},
	}

	vzB1CR := &v1beta1.Verrazzano{}
	vzB1CROverrides := vzB1CR.DeepCopy()
	vzB1CROverrides.Spec.Components.Thanos = &v1beta1.ThanosComponent{
		InstallOverrides: v1beta1.InstallOverrides{
			ValueOverrides: []v1beta1.Overrides{
				{
					Values: &apiextensionsv1.JSON{
						Raw: jsonVal,
					},
				},
			},
		},
	}

	tests := []struct {
		name           string
		verrazzanoA1   *v1alpha1.Verrazzano
		verrazzanoB1   *v1beta1.Verrazzano
		expA1Overrides interface{}
		expB1Overrides interface{}
	}{
		{
			name:           "test no overrides",
			verrazzanoA1:   vzA1CR,
			verrazzanoB1:   vzB1CR,
			expA1Overrides: []v1alpha1.Overrides{},
			expB1Overrides: []v1beta1.Overrides{},
		},
		{
			name:           "test v1alpha1 enabled nil",
			verrazzanoA1:   vzA1CROverrides,
			verrazzanoB1:   vzB1CROverrides,
			expA1Overrides: vzA1CROverrides.Spec.Components.Thanos.ValueOverrides,
			expB1Overrides: vzB1CROverrides.Spec.Components.Thanos.ValueOverrides,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			asserts.Equal(t, tt.expA1Overrides, NewComponent().GetOverrides(tt.verrazzanoA1))
			asserts.Equal(t, tt.expB1Overrides, NewComponent().GetOverrides(tt.verrazzanoB1))
		})
	}
}

// TestAppendOverrides tests if Thanos overrides are appendded
// GIVEN a call to AppendOverrides
// WHEN the bom is populated with the Thanos image
// THEN the overrides are returned from this function
func TestAppendOverrides(t *testing.T) {
	config.SetDefaultBomFilePath(bomFilePathOverride)
	kvs, err := AppendOverrides(nil, "", "", "", []bom.KeyValue{})
	asserts.NoError(t, err)

	expectedKVS := map[string]string{
		"image.registry":   "ghcr.io",
		"image.repository": "verrazzano/thanos",
	}
	for _, kv := range kvs {
		if kv.Key == "image.tag" {
			asserts.NotEmpty(t, kv.Value)
			continue
		}
		val, ok := expectedKVS[kv.Key]
		asserts.True(t, ok)
		asserts.Equal(t, val, kv.Value)
	}
}

// TestPreInstall tests the pre-install for the Thanos component
// GIVEN a call to PreInstall
// WHEN Thanos is enabled
// THEN the Thanos pre-install components get installed
func TestPreInstall(t *testing.T) {
	scheme := k8scheme.Scheme
	client := fake.NewClientBuilder().WithScheme(scheme).Build()
	ctx := spi.NewFakeContext(client, &v1alpha1.Verrazzano{}, nil, false, profilesRelativePath)
	asserts.NoError(t, NewComponent().PreInstall(ctx))

	ns := v1.Namespace{}
	asserts.NoError(t, client.Get(context.TODO(), types.NamespacedName{Name: constants.VerrazzanoMonitoringNamespace}, &ns))
}

// TestPreInstallUpgrade tests the preInstallUpgrade for the Thanos component
// GIVEN a call to preInstallUpgrade
// WHEN Thanos is enabled
// THEN the Thanos preInstallUpgrade components get installed
func TestPreInstallUpgrade(t *testing.T) {
	scheme := k8scheme.Scheme
	client := fake.NewClientBuilder().WithScheme(scheme).Build()
	ctx := spi.NewFakeContext(client, &v1alpha1.Verrazzano{}, nil, false, profilesRelativePath)
	asserts.NoError(t, preInstallUpgrade(ctx))

	ns := v1.Namespace{}
	asserts.NoError(t, client.Get(context.TODO(), types.NamespacedName{Name: constants.VerrazzanoMonitoringNamespace}, &ns))
}

// TestPostInstallUpgrade tests the postInstallUpgrade for the Thanos component
func TestPostInstallUpgrade(t *testing.T) {
	enabled := true
	disabled := false
	time := metav1.Now()
	scheme := k8scheme.Scheme
	asserts.NoError(t, certapiv1.AddToScheme(scheme))

	var tests = []struct {
		name    string
		vz      v1alpha1.Verrazzano
		ingress netv1.Ingress
		cert    certapiv1.Certificate
	}{
		// GIVEN a call to postInstallUpgrade
		// WHEN everything is disabled
		// THEN the Thanos postInstallUpgrade returns no error
		{
			name: "TestPostInstallUpgrade When everything is disabled",
			vz: v1alpha1.Verrazzano{
				Spec: v1alpha1.VerrazzanoSpec{
					Components: v1alpha1.ComponentSpec{
						AuthProxy: &v1alpha1.AuthProxyComponent{Enabled: &disabled},
						Ingress:   &v1alpha1.IngressNginxComponent{Enabled: &disabled},
						DNS: &v1alpha1.DNSComponent{
							OCI: &v1alpha1.OCI{
								DNSZoneName: "mydomain.com",
							},
						},
					},
				},
			},
			ingress: netv1.Ingress{
				ObjectMeta: metav1.ObjectMeta{Name: constants.ThanosQueryIngress, Namespace: constants.VerrazzanoSystemNamespace},
			},
			cert: certapiv1.Certificate{
				ObjectMeta: metav1.ObjectMeta{Name: queryCertificateName, Namespace: constants.VerrazzanoSystemNamespace},
				Status: certapiv1.CertificateStatus{
					Conditions: []certapiv1.CertificateCondition{
						{Type: certapiv1.CertificateConditionReady, Status: cmmeta.ConditionTrue, LastTransitionTime: &time},
					},
				},
			},
		},
		// GIVEN a call to postInstallUpgrade
		// WHEN the authproxy is disabled
		// THEN the Thanos postInstallUpgrade returns no error
		{
			name: "TestPostInstallUpgrade When authproxy is disabled",
			vz: v1alpha1.Verrazzano{
				Spec: v1alpha1.VerrazzanoSpec{
					Components: v1alpha1.ComponentSpec{
						AuthProxy: &v1alpha1.AuthProxyComponent{Enabled: &disabled},
						Ingress:   &v1alpha1.IngressNginxComponent{Enabled: &enabled},
						DNS: &v1alpha1.DNSComponent{
							OCI: &v1alpha1.OCI{
								DNSZoneName: "mydomain.com",
							},
						},
					},
				},
			},
			ingress: netv1.Ingress{
				ObjectMeta: metav1.ObjectMeta{Name: constants.ThanosQueryIngress, Namespace: constants.VerrazzanoSystemNamespace},
			},
			cert: certapiv1.Certificate{
				ObjectMeta: metav1.ObjectMeta{Name: queryCertificateName, Namespace: constants.VerrazzanoSystemNamespace},
				Status: certapiv1.CertificateStatus{
					Conditions: []certapiv1.CertificateCondition{
						{Type: certapiv1.CertificateConditionReady, Status: cmmeta.ConditionTrue, LastTransitionTime: &time},
					},
				},
			},
		},
		// GIVEN a call to postInstallUpgrade
		// WHEN NGINX is disabled
		// THEN the Thanos postInstallUpgrade returns no error
		{
			name: "TestPostInstallUpgrade When NGINX is disabled",
			vz: v1alpha1.Verrazzano{
				Spec: v1alpha1.VerrazzanoSpec{
					Components: v1alpha1.ComponentSpec{
						AuthProxy: &v1alpha1.AuthProxyComponent{Enabled: &enabled},
						Ingress:   &v1alpha1.IngressNginxComponent{Enabled: &disabled},
						DNS: &v1alpha1.DNSComponent{
							OCI: &v1alpha1.OCI{
								DNSZoneName: "mydomain.com",
							},
						},
					},
				},
			},
			ingress: netv1.Ingress{
				ObjectMeta: metav1.ObjectMeta{Name: constants.ThanosQueryIngress, Namespace: constants.VerrazzanoSystemNamespace},
			},
			cert: certapiv1.Certificate{
				ObjectMeta: metav1.ObjectMeta{Name: queryCertificateName, Namespace: constants.VerrazzanoSystemNamespace},
				Status: certapiv1.CertificateStatus{
					Conditions: []certapiv1.CertificateCondition{
						{Type: certapiv1.CertificateConditionReady, Status: cmmeta.ConditionTrue, LastTransitionTime: &time},
					},
				},
			},
		},
		// GIVEN a call to postInstallUpgrade
		// WHEN all enabled
		// THEN the Thanos postInstallUpgrade returns no error
		{
			name: "TestPostInstallUpgrade When all enabled",
			vz: v1alpha1.Verrazzano{
				Spec: v1alpha1.VerrazzanoSpec{
					Components: v1alpha1.ComponentSpec{
						AuthProxy: &v1alpha1.AuthProxyComponent{Enabled: &enabled},
						Ingress:   &v1alpha1.IngressNginxComponent{Enabled: &enabled},
						DNS: &v1alpha1.DNSComponent{
							OCI: &v1alpha1.OCI{
								DNSZoneName: "mydomain.com",
							},
						},
					},
				},
			},
			ingress: netv1.Ingress{
				ObjectMeta: metav1.ObjectMeta{Name: constants.ThanosQueryIngress, Namespace: constants.VerrazzanoSystemNamespace},
			},
			cert: certapiv1.Certificate{
				ObjectMeta: metav1.ObjectMeta{Name: queryCertificateName, Namespace: constants.VerrazzanoSystemNamespace},
				Status: certapiv1.CertificateStatus{
					Conditions: []certapiv1.CertificateCondition{
						{Type: certapiv1.CertificateConditionReady, Status: cmmeta.ConditionTrue, LastTransitionTime: &time},
					},
				},
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			client := fake.NewClientBuilder().WithScheme(scheme).WithObjects(&test.ingress, &test.cert).Build()
			ctx := spi.NewFakeContext(client, &test.vz, nil, false)
			err := postInstallUpgrade(ctx)
			asserts.NoError(t, err)
		})
	}
}
