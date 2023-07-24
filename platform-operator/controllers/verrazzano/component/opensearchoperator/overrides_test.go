// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package opensearchoperator

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	vmov1 "github.com/verrazzano/verrazzano-monitoring-operator/pkg/apis/vmcontroller/v1"
	"github.com/verrazzano/verrazzano/pkg/bom"
	"github.com/verrazzano/verrazzano/pkg/nginxutil"
	"github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1beta1"
	"github.com/verrazzano/verrazzano/platform-operator/constants"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"

	"github.com/stretchr/testify/assert"
	v1 "k8s.io/api/core/v1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/yaml"
)

// TestGetOverrides tests if OpenSearchOperator overrides are properly collected
// GIVEN a call to GetOverrides
// WHEN the VZ CR OpenSearchOperator component has overrides
// THEN the overrides are returned from this function
func TestGetOverrides(t *testing.T) {
	testKey := "test-key"
	testVal := "test-val"
	jsonVal := []byte(fmt.Sprintf("{\"%s\":\"%s\"}", testKey, testVal))

	vzA1CR := &v1alpha1.Verrazzano{}
	vzA1CROverrides := vzA1CR.DeepCopy()
	vzA1CROverrides.Spec.Components.OpenSearchOperator = &v1alpha1.OpenSearchOperatorComponent{
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
	vzB1CROverrides.Spec.Components.OpenSearchOperator = &v1beta1.OpenSearchOperatorComponent{
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
			expA1Overrides: vzA1CROverrides.Spec.Components.OpenSearchOperator.ValueOverrides,
			expB1Overrides: vzB1CROverrides.Spec.Components.OpenSearchOperator.ValueOverrides,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expA1Overrides, NewComponent().GetOverrides(tt.verrazzanoA1))
			assert.Equal(t, tt.expB1Overrides, NewComponent().GetOverrides(tt.verrazzanoB1))
		})
	}
}

// TestMergeNodePoolOverrides tests the MergeNodePoolOverrides function
// GIVEN a CR with user overrides
// WHEN MergeNodePoolOverrides is called
// THEN a yaml with correct merged node pool overrides is returned
func TestMergeNodePoolOverrides(t *testing.T) {
	defer func() {
		GetControllerRuntimeClient = GetClient
	}()

	var tests = []struct {
		name         string
		expectedYAML string
		actualCR     string
	}{
		{
			name:         "TestBasicUserOverrides-1",
			expectedYAML: "testdata/expectedMergedOverrides-1.yaml",
			actualCR:     "testdata/userOverrides-1.yaml",
		},
		{
			name:         "TestBasicUserOverrides-2",
			expectedYAML: "testdata/expectedMergedOverrides-2.yaml",
			actualCR:     "testdata/userOverrides-2.yaml",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			a := assert.New(t)
			// Load the actual CR
			actualCR, err := loadYamlAsVZObject(test.actualCR)
			a.NoError(err)
			a.NotNil(actualCR)

			fakeClient := fake.NewClientBuilder().WithScheme(testScheme).Build()
			fakeCtx := spi.NewFakeContext(fakeClient, actualCR, nil, false, profilesRelativePath)
			GetControllerRuntimeClient = func() (client.Client, error) {
				return fakeClient, nil
			}

			mergedYaml, err := MergeNodePoolOverrides(fakeCtx.EffectiveCR(), fakeClient, []NodePool{})
			a.NoError(err)
			bYaml, err := os.ReadFile(filepath.Join(test.expectedYAML))
			a.NoError(err)
			a.YAMLEq(string(bYaml), mergedYaml)
		})
	}
}

func TestBuildNodePoolOverrides(t *testing.T) {
	defer func() {
		GetControllerRuntimeClient = GetClient
	}()

	var tests = []struct {
		name         string
		expectedYAML string
		actualCR     string
	}{
		{
			name:         "TestBasicUserOverrides-1",
			expectedYAML: "testdata/expectedMergedValuesOverrides-1.yaml",
			actualCR:     "testdata/userOverrides-1.yaml",
		},
		{
			name:         "TestBasicUserOverrides-2",
			expectedYAML: "testdata/expectedMergedValuesOverrides-2.yaml",
			actualCR:     "testdata/userOverrides-2.yaml",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			a := assert.New(t)
			// Load the actual CR
			actualCR, err := loadYamlAsVZObject(test.actualCR)
			a.NoError(err)
			a.NotNil(actualCR)

			fakeClient := fake.NewClientBuilder().WithScheme(testScheme).Build()
			fakeCtx := spi.NewFakeContext(fakeClient, actualCR, nil, false, profilesRelativePath)
			GetControllerRuntimeClient = func() (client.Client, error) {
				return fakeClient, nil
			}

			mergedOverrides := BuildNodePoolOverrides(fakeCtx.EffectiveCR())
			actualJSON, err := json.Marshal(mergedOverrides)
			a.NoError(err)

			bYaml, err := os.ReadFile(filepath.Join(test.expectedYAML))
			a.NoError(err)
			expectedJSON, err := yaml.YAMLToJSON(bYaml)
			a.NoError(err)

			a.JSONEq(string(expectedJSON), string(actualJSON))

		})
	}
}

func TestBuildv1beta1NodePoolOverrides(t *testing.T) {
	defer func() {
		GetControllerRuntimeClient = GetClient
	}()

	var tests = []struct {
		name         string
		expectedYAML string
		actualCR     string
	}{
		{
			name:         "TestBasicUserOverrides-1",
			expectedYAML: "testdata/expectedMergedValuesOverrides-1.yaml",
			actualCR:     "testdata/userOverrides-1.yaml",
		},
		{
			name:         "TestBasicUserOverrides-2",
			expectedYAML: "testdata/expectedMergedValuesOverrides-2.yaml",
			actualCR:     "testdata/userOverrides-2.yaml",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			a := assert.New(t)
			// Load the actual CR
			actualCR, err := loadYamlAsVZObject(test.actualCR)
			a.NoError(err)
			a.NotNil(actualCR)

			fakeClient := fake.NewClientBuilder().WithScheme(testScheme).Build()
			fakeCtx := spi.NewFakeContext(fakeClient, actualCR, nil, false, profilesRelativePath)
			GetControllerRuntimeClient = func() (client.Client, error) {
				return fakeClient, nil
			}

			mergedOverrides := Buildv1beta1NodePoolOverrides(fakeCtx.EffectiveCRV1Beta1())
			actualJSON, err := json.Marshal(mergedOverrides)
			a.NoError(err)

			bYaml, err := os.ReadFile(filepath.Join(test.expectedYAML))
			a.NoError(err)
			expectedJSON, err := yaml.YAMLToJSON(bYaml)
			a.NoError(err)

			a.JSONEq(string(expectedJSON), string(actualJSON))

		})
	}
}

func TestAppendOverrides(t *testing.T) {
	defer func() {
		GetControllerRuntimeClient = GetClient
	}()
	falseVal := false

	nodeOverride := "{\"openSearchCluster\":{\"nodePools\":[{\"component\": \"es-master\",\"replicas\": %d,\"roles\":[\"master\"]}]}}"

	singleNodeOS := v1alpha1.OpenSearchOperatorComponent{
		InstallOverrides: v1alpha1.InstallOverrides{
			ValueOverrides: []v1alpha1.Overrides{{
				Values: &apiextensionsv1.JSON{
					Raw: []byte(fmt.Sprintf(nodeOverride, 1)),
				}},
			},
		},
	}

	multiNodeOS := v1alpha1.OpenSearchOperatorComponent{
		InstallOverrides: v1alpha1.InstallOverrides{
			ValueOverrides: []v1alpha1.Overrides{{
				Values: &apiextensionsv1.JSON{
					Raw: []byte(fmt.Sprintf(nodeOverride, 3)),
				}},
			},
		},
	}

	service := &v1.Service{
		ObjectMeta: metav1.ObjectMeta{Name: constants.NGINXControllerServiceName, Namespace: nginxutil.IngressNGINXNamespace()},
		Spec: v1.ServiceSpec{
			Type: v1.ServiceTypeLoadBalancer,
		},
		Status: v1.ServiceStatus{
			LoadBalancer: v1.LoadBalancerStatus{
				Ingress: []v1.LoadBalancerIngress{
					{IP: "1.2.3.4"},
				},
			},
		},
	}
	externalDNSZone := "mydomain.com"

	ingressKVS := map[string]string{
		`ingress.openSearch.annotations.cert-manager\.io/cluster-issuer`: "verrazzano-cluster-issuer",
		`ingress.openSearch.annotations.cert-manager\.io/common-name`:    "opensearch.vmi.system.default.1.2.3.4.nip.io",
		"ingress.openSearch.host":                                        "opensearch.vmi.system.default.1.2.3.4.nip.io",
		"ingress.openSearch.ingressClassName":                            "verrazzano-nginx",
		"ingress.openSearch.tls[0].secretName":                           "system-tls-os-ingest",
		"ingress.openSearch.tls[0].hosts[0]":                             "opensearch.vmi.system.default.1.2.3.4.nip.io",

		`ingress.openSearchDashboards.annotations.cert-manager\.io/cluster-issuer`: "verrazzano-cluster-issuer",
		`ingress.openSearchDashboards.annotations.cert-manager\.io/common-name`:    "osd.vmi.system.default.1.2.3.4.nip.io",
		"ingress.openSearchDashboards.host":                                        "osd.vmi.system.default.1.2.3.4.nip.io",
		"ingress.openSearchDashboards.ingressClassName":                            "verrazzano-nginx",
		"ingress.openSearchDashboards.tls[0].secretName":                           "system-tls-osd",
		"ingress.openSearchDashboards.tls[0].hosts[0]":                             "osd.vmi.system.default.1.2.3.4.nip.io",
	}

	ingressSslipKVS := map[string]string{
		`ingress.openSearch.annotations.cert-manager\.io/common-name`:           "opensearch.vmi.system.default.1.2.3.4.sslip.io",
		"ingress.openSearch.host":                                               "opensearch.vmi.system.default.1.2.3.4.sslip.io",
		"ingress.openSearch.tls[0].hosts[0]":                                    "opensearch.vmi.system.default.1.2.3.4.sslip.io",
		`ingress.openSearchDashboards.annotations.cert-manager\.io/common-name`: "osd.vmi.system.default.1.2.3.4.sslip.io",
		"ingress.openSearchDashboards.host":                                     "osd.vmi.system.default.1.2.3.4.sslip.io",
		"ingress.openSearchDashboards.tls[0].hosts[0]":                          "osd.vmi.system.default.1.2.3.4.sslip.io",
	}

	ingressDisabledKVS := map[string]string{
		"ingress.openSearch.enable":           "false",
		"ingress.openSearchDashboards.enable": "false",
	}

	bootstrapKVS := map[string]string{
		`openSearchCluster.bootstrap.additionalConfig.cluster\.initial_master_nodes`: "opensearch-es-master-0",
	}

	pluginKVS := map[string]string{
		"openSearchCluster.general.pluginsList[0]":    "pluginA",
		"openSearchCluster.general.pluginsList[1]":    "pluginB",
		"openSearchCluster.dashboards.pluginsList[0]": "pluginA",
		"openSearchCluster.dashboards.pluginsList[1]": "pluginB",
	}

	dashboardReplicaKVS := map[string]string{
		"openSearchCluster.dashboards.replicas": "3",
	}

	externalDNSKVs := map[string]string{
		`ingress.openSearch.annotations.cert-manager\.io/common-name`:                         "opensearch.vmi.system.default.mydomain.com",
		"ingress.openSearch.host":                                                             "opensearch.vmi.system.default.mydomain.com",
		"ingress.openSearch.tls[0].hosts[0]":                                                  "opensearch.vmi.system.default.mydomain.com",
		`ingress.openSearchDashboards.annotations.cert-manager\.io/common-name`:               "osd.vmi.system.default.mydomain.com",
		"ingress.openSearchDashboards.host":                                                   "osd.vmi.system.default.mydomain.com",
		"ingress.openSearchDashboards.tls[0].hosts[0]":                                        "osd.vmi.system.default.mydomain.com",
		`ingress.openSearch.annotations.external-dns\.alpha\.kubernetes\.io/target`:           "verrazzano-ingress.default.mydomain.com",
		`ingress.openSearch.annotations.external-dns\.alpha\.kubernetes\.io/ttl`:              "60",
		`ingress.openSearchDashboards.annotations.external-dns\.alpha\.kubernetes\.io/target`: "verrazzano-ingress.default.mydomain.com",
		`ingress.openSearchDashboards.annotations.external-dns\.alpha\.kubernetes\.io/ttl`:    "60",
	}

	tests := []struct {
		name        string
		vz          *v1alpha1.Verrazzano
		expectedKVS map[string]string
	}{
		// GIVEN a call to AppendOverrides
		// WHEN the NGINX is disabled
		// THEN ingresses are disabled
		{
			name: "test NGINX Disabled",
			vz: &v1alpha1.Verrazzano{
				Spec: v1alpha1.VerrazzanoSpec{
					Components: v1alpha1.ComponentSpec{
						OpenSearchOperator: &singleNodeOS,
						Ingress: &v1alpha1.IngressNginxComponent{
							Enabled: &falseVal,
						},
					},
				},
			},
			expectedKVS: mergeMaps(bootstrapKVS, ingressDisabledKVS),
		},
		// GIVEN a call to AppendOverrides
		// WHEN wildcard is enabled
		// THEN the proper overrides are populated
		{
			name: "test ExternalDNS Disabled",
			vz: &v1alpha1.Verrazzano{
				Spec: v1alpha1.VerrazzanoSpec{
					Components: v1alpha1.ComponentSpec{
						OpenSearchOperator: &multiNodeOS,
						DNS: &v1alpha1.DNSComponent{
							Wildcard: &v1alpha1.Wildcard{
								Domain: "sslip.io",
							},
						},
					},
				},
			},
			expectedKVS: mergeMaps(ingressKVS, ingressSslipKVS),
		},
		// GIVEN a call to AppendOverrides
		// WHEN wildcard is enabled
		// THEN the extra external DNS overrides are added
		{
			name: "test ExternalDNS Enabled",
			vz: &v1alpha1.Verrazzano{
				Spec: v1alpha1.VerrazzanoSpec{
					Components: v1alpha1.ComponentSpec{
						OpenSearchOperator: &singleNodeOS,
						DNS: &v1alpha1.DNSComponent{
							OCI: &v1alpha1.OCI{
								DNSZoneName: externalDNSZone,
							},
						},
					},
				},
			},
			expectedKVS: mergeMaps(bootstrapKVS, ingressKVS, externalDNSKVs),
		},
		// GIVEN a call to AppendOverrides
		// WHEN plugin list is present in OS
		// THEN plugin overrides are added
		{
			name: "test Istio Disabled",
			vz: &v1alpha1.Verrazzano{
				Spec: v1alpha1.VerrazzanoSpec{
					Components: v1alpha1.ComponentSpec{
						OpenSearchOperator: &multiNodeOS,
						Elasticsearch: &v1alpha1.ElasticsearchComponent{
							Plugins: vmov1.OpenSearchPlugins{
								Enabled:     true,
								InstallList: []string{"pluginA", "pluginB"},
							},
						},
						Kibana: &v1alpha1.KibanaComponent{
							Plugins: vmov1.OpenSearchDashboardsPlugins{
								Enabled:     true,
								InstallList: []string{"pluginA", "pluginB"},
							},
						},
					},
				},
			},
			expectedKVS: mergeMaps(ingressKVS, pluginKVS),
		},
		// GIVEN a call to AppendOverrides
		// WHEN OpenSearchDashboard replica is configured
		// THEN replica value is added
		{
			name: "test Istio Injection Disabled",
			vz: &v1alpha1.Verrazzano{
				Spec: v1alpha1.VerrazzanoSpec{
					Components: v1alpha1.ComponentSpec{
						OpenSearchOperator: &singleNodeOS,
						Kibana: &v1alpha1.KibanaComponent{
							Replicas: v1alpha1.Int32Ptr(3),
						},
					},
				},
			},
			expectedKVS: mergeMaps(bootstrapKVS, ingressKVS, dashboardReplicaKVS),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fakeClient := fake.NewClientBuilder().WithScheme(testScheme).WithObjects(service).Build()
			GetControllerRuntimeClient = func() (client.Client, error) {
				return fakeClient, nil
			}
			ctx := spi.NewFakeContext(fakeClient, tt.vz, nil, false)
			kvs, err := AppendOverrides(ctx, "", "", "", []bom.KeyValue{})
			assert.NoError(t, err)

			assert.Equal(t, len(tt.expectedKVS), len(kvs))
			for _, kv := range kvs {
				val, ok := tt.expectedKVS[kv.Key]
				assert.Truef(t, ok, "Key %s not located in the Key Value pairs", kv.Key)
				assert.Regexp(t, val, kv.Value)
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

func loadYamlAsVZObject(expectedYamlFile string) (*v1alpha1.Verrazzano, error) {
	bYaml, err := os.ReadFile(filepath.Join(expectedYamlFile))
	if err != nil {
		return nil, err
	}
	vz := v1alpha1.Verrazzano{}
	err = yaml.Unmarshal(bYaml, &vz)
	return &vz, err
}
