// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package thanos

import (
	"context"
	"fmt"
	constants2 "github.com/verrazzano/verrazzano/pkg/constants"
	"github.com/verrazzano/verrazzano/pkg/nginxutil"
	"strconv"
	"testing"

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

// TestAppendOverrides tests if Thanos overrides are appended
func TestAppendOverrides(t *testing.T) {
	config.SetDefaultBomFilePath(bomFilePathOverride)
	scheme := k8scheme.Scheme
	falseVal := false
	trueVal := true

	service := &v1.Service{
		ObjectMeta: metav1.ObjectMeta{Name: constants.NGINXControllerServiceName, Namespace: nginxutil.IngressNGINXNamespace()},
		Spec: v1.ServiceSpec{
			Type: v1.ServiceTypeLoadBalancer,
		},
		Status: v1.ServiceStatus{
			LoadBalancer: v1.LoadBalancerStatus{
				Ingress: []v1.LoadBalancerIngress{
					{IP: "11.22.33.44"},
				},
			},
		},
	}

	imageKVS := map[string]string{
		"image.registry":   "ghcr.io",
		"image.repository": "verrazzano/thanos",
		"image.tag":        `v\d+\.\d+\.\d+-.+-.+`,
	}

	ingressKVs := map[string]string{
		"queryFrontend.ingress.namespace":                                                      constants.VerrazzanoSystemNamespace,
		"queryFrontend.ingress.ingressClassName":                                               "verrazzano-nginx",
		"queryFrontend.ingress.extraRules[0].host":                                             "thanos-query.default.11.22.33.44.nip.io",
		"queryFrontend.ingress.extraRules[0].http.paths[0].backend.service.name":               constants.VerrazzanoAuthProxyServiceName,
		"queryFrontend.ingress.extraRules[0].http.paths[0].backend.service.port.number":        strconv.Itoa(constants.VerrazzanoAuthProxyServicePort),
		"queryFrontend.ingress.extraRules[0].http.paths[0].path":                               "/(.*)()",
		"queryFrontend.ingress.extraRules[0].http.paths[0].pathType":                           string(netv1.PathTypeImplementationSpecific),
		"queryFrontend.ingress.extraTls[0].hosts[0]":                                           "thanos-query.default.11.22.33.44.nip.io",
		"queryFrontend.ingress.extraTls[0].secretName":                                         queryCertificateName,
		`queryFrontend.ingress.annotations.nginx\.ingress\.kubernetes\.io/session-cookie-name`: queryHostName,
		`queryFrontend.ingress.annotations.cert-manager\.io/cluster-issuer`:                    constants2.VerrazzanoClusterIssuerName,
		`queryFrontend.ingress.annotations.cert-manager\.io/common-name`:                       "thanos-query.default.11.22.33.44.nip.io",
		"query.ingress.grpc.namespace":                                                         constants.VerrazzanoSystemNamespace,
		"query.ingress.grpc.ingressClassName":                                                  "verrazzano-nginx",
		"query.ingress.grpc.extraRules[0].host":                                                "thanos-query-store.default.11.22.33.44.nip.io",
		"query.ingress.grpc.extraRules[0].http.paths[0].backend.service.name":                  constants.VerrazzanoAuthProxyServiceName,
		"query.ingress.grpc.extraRules[0].http.paths[0].backend.service.port.number":           strconv.Itoa(constants.VerrazzanoAuthProxyGRPCServicePort),
		"query.ingress.grpc.extraRules[0].http.paths[0].path":                                  "/",
		"query.ingress.grpc.extraRules[0].http.paths[0].pathType":                              string(netv1.PathTypeImplementationSpecific),
		"query.ingress.grpc.extraTls[0].hosts[0]":                                              "thanos-query-store.default.11.22.33.44.nip.io",
		"query.ingress.grpc.extraTls[0].secretName":                                            queryStoreCertificateName,
		`query.ingress.grpc.annotations.cert-manager\.io/cluster-issuer`:                       constants2.VerrazzanoClusterIssuerName,
		`query.ingress.grpc.annotations.cert-manager\.io/common-name`:                          "thanos-query-store.default.11.22.33.44.nip.io",
	}
	sslipioKVs := map[string]string{
		"queryFrontend.ingress.extraRules[0].host":                       "thanos-query.default.11.22.33.44.sslip.io",
		"queryFrontend.ingress.extraTls[0].hosts[0]":                     "thanos-query.default.11.22.33.44.sslip.io",
		`queryFrontend.ingress.annotations.cert-manager\.io/common-name`: "thanos-query.default.11.22.33.44.sslip.io",
		"query.ingress.grpc.extraRules[0].host":                          "thanos-query-store.default.11.22.33.44.sslip.io",
		"query.ingress.grpc.extraTls[0].hosts[0]":                        "thanos-query-store.default.11.22.33.44.sslip.io",
		`query.ingress.grpc.annotations.cert-manager\.io/common-name`:    "thanos-query-store.default.11.22.33.44.sslip.io",
	}
	istioEnabledKV := map[string]string{
		"verrazzano.isIstioEnabled": "true",
	}

	istioDisabledKV := map[string]string{
		"verrazzano.isIstioEnabled": "false",
	}

	externalDNSKVs := map[string]string{
		"queryFrontend.ingress.extraRules[0].host":                                     "thanos-query.default.mydomain.com",
		"queryFrontend.ingress.extraTls[0].hosts[0]":                                   "thanos-query.default.mydomain.com",
		`queryFrontend.ingress.annotations.external-dns\.alpha\.kubernetes\.io/target`: "verrazzano-ingress.default.mydomain.com",
		`queryFrontend.ingress.annotations.external-dns\.alpha\.kubernetes\.io/ttl`:    "60",
		`queryFrontend.ingress.annotations.cert-manager\.io/common-name`:               "thanos-query.default.mydomain.com",
		"query.ingress.grpc.extraRules[0].host":                                        "thanos-query-store.default.mydomain.com",
		"query.ingress.grpc.extraTls[0].hosts[0]":                                      "thanos-query-store.default.mydomain.com",
		`query.ingress.grpc.annotations.external-dns\.alpha\.kubernetes\.io/target`:    "verrazzano-ingress.default.mydomain.com",
		`query.ingress.grpc.annotations.external-dns\.alpha\.kubernetes\.io/ttl`:       "60",
		`query.ingress.grpc.annotations.cert-manager\.io/common-name`:                  "thanos-query-store.default.mydomain.com",
	}
	externalDNSZone := "mydomain.com"

	tests := []struct {
		name     string
		vz       *v1alpha1.Verrazzano
		extraKVS map[string]string
	}{
		// GIVEN a call to AppendOverrides
		// WHEN the NGINX is disabled
		// THEN query ingresses are disabled
		{
			name: "test NGINX Disabled",
			vz: &v1alpha1.Verrazzano{
				Spec: v1alpha1.VerrazzanoSpec{
					Components: v1alpha1.ComponentSpec{
						Ingress: &v1alpha1.IngressNginxComponent{
							Enabled: &falseVal,
						},
					},
				},
			},
			extraKVS: mergeMaps(map[string]string{
				"query.ingress.grpc.enabled":    "false",
				"queryFrontend.ingress.enabled": "false",
			}, istioEnabledKV),
		},
		// GIVEN a call to AppendOverrides
		// WHEN wildcard is enabled
		// THEN the proper overrides are populated
		{
			name: "test ExternalDNS Disabled",
			vz: &v1alpha1.Verrazzano{
				Spec: v1alpha1.VerrazzanoSpec{
					Components: v1alpha1.ComponentSpec{
						DNS: &v1alpha1.DNSComponent{
							Wildcard: &v1alpha1.Wildcard{
								Domain: "sslip.io",
							},
						},
					},
				},
			},
			extraKVS: mergeMaps(ingressKVs, istioEnabledKV, sslipioKVs),
		},
		// GIVEN a call to AppendOverrides
		// WHEN wildcard is enabled
		// THEN the extra external DNS overrides are added
		{
			name: "test ExternalDNS Enabled",
			vz: &v1alpha1.Verrazzano{
				Spec: v1alpha1.VerrazzanoSpec{
					Components: v1alpha1.ComponentSpec{
						DNS: &v1alpha1.DNSComponent{
							OCI: &v1alpha1.OCI{
								DNSZoneName: externalDNSZone,
							},
						},
					},
				},
			},
			extraKVS: mergeMaps(ingressKVs, istioEnabledKV, externalDNSKVs),
		},
		// GIVEN a call to AppendOverrides
		// WHEN Istio is disabled
		// THEN isIstioEnabled is set to false
		{
			name: "test Istio Disabled",
			vz: &v1alpha1.Verrazzano{
				Spec: v1alpha1.VerrazzanoSpec{
					Components: v1alpha1.ComponentSpec{
						Istio: &v1alpha1.IstioComponent{
							Enabled: &falseVal,
						},
					},
				},
			},
			extraKVS: mergeMaps(ingressKVs, istioDisabledKV),
		},
		// GIVEN a call to AppendOverrides
		// WHEN Istio is enabled but Istio injection is disabled
		// THEN isIstioEnabled is set to false
		{
			name: "test Istio Injection Disabled",
			vz: &v1alpha1.Verrazzano{
				Spec: v1alpha1.VerrazzanoSpec{
					Components: v1alpha1.ComponentSpec{
						Istio: &v1alpha1.IstioComponent{
							Enabled:          &trueVal,
							InjectionEnabled: &falseVal,
						},
					},
				},
			},
			extraKVS: mergeMaps(ingressKVs, istioDisabledKV),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := fake.NewClientBuilder().WithScheme(scheme).WithObjects(service).Build()
			ctx := spi.NewFakeContext(client, tt.vz, nil, false)
			kvs, err := AppendOverrides(ctx, "", "", "", []bom.KeyValue{})
			asserts.NoError(t, err)

			expectedKVS := map[string]string{}
			for k, v := range imageKVS {
				expectedKVS[k] = v
			}
			for k, v := range tt.extraKVS {
				expectedKVS[k] = v
			}

			for _, kv := range kvs {
				val, ok := expectedKVS[kv.Key]
				asserts.Truef(t, ok, "Key %s not located in the Key Value pairs", kv.Key)
				asserts.Regexp(t, val, kv.Value)
			}
		})
	}
}

// mergeMaps merges the contents of the given maps, starting from the first, and treating the
// rest as overlays on top of the first.
func mergeMaps(mapsToMerge ...map[string]string) map[string]string {
	mergedMap := map[string]string{}
	for _, eachMap := range mapsToMerge {
		for k, v := range eachMap {
			mergedMap[k] = v
		}
	}
	return mergedMap
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
