// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package operator

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	asserts "github.com/stretchr/testify/assert"
	vmoconst "github.com/verrazzano/verrazzano-monitoring-operator/pkg/constants"
	"github.com/verrazzano/verrazzano/pkg/bom"
	vzconst "github.com/verrazzano/verrazzano/pkg/constants"
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/constants"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	"github.com/verrazzano/verrazzano/platform-operator/internal/config"
	istioclisecv1beta1 "istio.io/api/security/v1beta1"
	istioclisec "istio.io/client-go/pkg/apis/security/v1beta1"
	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	k8scheme "k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

const (
	testBomFilePath = "../../../testdata/test_bom.json"
)

var (
	testScheme = runtime.NewScheme()

	falseValue = false
	trueValue  = true
)

func init() {
	_ = clientgoscheme.AddToScheme(testScheme)
	_ = vzapi.AddToScheme(testScheme)
}

// TestIsPrometheusOperatorReady tests the isPrometheusOperatorReady function for the Prometheus Operator
func TestIsPrometheusOperatorReady(t *testing.T) {
	tests := []struct {
		name       string
		client     client.Client
		expectTrue bool
	}{
		{
			// GIVEN the Prometheus Operator deployment exists and there are available replicas
			// WHEN we call isPrometheusOperatorReady
			// THEN the call returns true
			name: "Test IsReady when Prometheus Operator is successfully deployed",
			client: fake.NewClientBuilder().WithScheme(testScheme).WithObjects(
				&appsv1.Deployment{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: ComponentNamespace,
						Name:      deploymentName,
					},
					Status: appsv1.DeploymentStatus{
						AvailableReplicas: 1,
						Replicas:          1,
						UpdatedReplicas:   1,
					},
				}).Build(),
			expectTrue: true,
		},
		{
			// GIVEN the Prometheus Operator deployment exists and there are no available replicas
			// WHEN we call isPrometheusOperatorReady
			// THEN the call returns false
			name: "Test IsReady when Prometheus Operator deployment is not ready",
			client: fake.NewClientBuilder().WithScheme(testScheme).WithObjects(
				&appsv1.Deployment{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: ComponentNamespace,
						Name:      deploymentName,
					},
					Status: appsv1.DeploymentStatus{
						AvailableReplicas: 0,
						Replicas:          1,
						UpdatedReplicas:   0,
					},
				}).Build(),
			expectTrue: false,
		},
		{
			// GIVEN the Prometheus Operator deployment does not exist
			// WHEN we call isPrometheusOperatorReady
			// THEN the call returns false
			name:       "Test IsReady when Prometheus Operator deployment does not exist",
			client:     fake.NewClientBuilder().WithScheme(testScheme).Build(),
			expectTrue: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := spi.NewFakeContext(tt.client, &vzapi.Verrazzano{}, false)
			assert.Equal(t, tt.expectTrue, isPrometheusOperatorReady(ctx))
		})
	}
}

// TestAppendOverrides tests that helm overrides are set properly
func TestAppendOverrides(t *testing.T) {
	oldBomPath := config.GetDefaultBOMFilePath()
	config.SetDefaultBomFilePath(testBomFilePath)
	defer config.SetDefaultBomFilePath(oldBomPath)

	client := fake.NewClientBuilder().WithScheme(testScheme).Build()
	kvs := make([]bom.KeyValue, 0)

	// GIVEN a Verrazzano CR with the CertManager component enabled
	// WHEN the AppendOverrides function is called
	// THEN the key/value slice contains the expected helm override keys and values
	// AND the admission webhook cert manager helm override is set to true
	vz := &vzapi.Verrazzano{
		Spec: vzapi.VerrazzanoSpec{
			Components: vzapi.ComponentSpec{
				CertManager: &vzapi.CertManagerComponent{
					Enabled: &trueValue,
				},
				Keycloak: &vzapi.KeycloakComponent{
					Enabled: &falseValue,
				},
			},
		},
	}

	ctx := spi.NewFakeContext(client, vz, false)

	var err error
	kvs, err = AppendOverrides(ctx, "", "", "", kvs)
	assert.NoError(t, err)
	assert.Len(t, kvs, 27)

	assert.Equal(t, "ghcr.io/verrazzano/prometheus-config-reloader", bom.FindKV(kvs, "prometheusOperator.prometheusConfigReloader.image.repository"))
	assert.NotEmpty(t, bom.FindKV(kvs, "prometheusOperator.prometheusConfigReloader.image.tag"))

	assert.Equal(t, "ghcr.io/verrazzano/alertmanager", bom.FindKV(kvs, "alertmanager.alertmanagerSpec.image.repository"))
	assert.NotEmpty(t, bom.FindKV(kvs, "alertmanager.alertmanagerSpec.image.tag"))

	assert.True(t, strings.HasPrefix(bom.FindKV(kvs, "prometheusOperator.alertmanagerDefaultBaseImage"), "ghcr.io/verrazzano/alertmanager:"))
	assert.True(t, strings.HasPrefix(bom.FindKV(kvs, "prometheusOperator.prometheusDefaultBaseImage"), "ghcr.io/verrazzano/prometheus:"))

	assert.Equal(t, "true", bom.FindKV(kvs, "prometheusOperator.admissionWebhooks.certManager.enabled"))

	// GIVEN a Verrazzano CR with the CertManager component disabled
	// WHEN the AppendOverrides function is called
	// THEN the key/value slice contains the expected helm override keys and values
	// AND the admission webhook cert manager helm override is set to false
	vz = &vzapi.Verrazzano{
		Spec: vzapi.VerrazzanoSpec{
			Components: vzapi.ComponentSpec{
				CertManager: &vzapi.CertManagerComponent{
					Enabled: &falseValue,
				},
				Keycloak: &vzapi.KeycloakComponent{
					Enabled: &falseValue,
				},
			},
		},
	}

	ctx = spi.NewFakeContext(client, vz, false)
	kvs = make([]bom.KeyValue, 0)

	kvs, err = AppendOverrides(ctx, "", "", "", kvs)
	assert.NoError(t, err)
	assert.Len(t, kvs, 27)

	assert.Equal(t, "false", bom.FindKV(kvs, "prometheusOperator.admissionWebhooks.certManager.enabled"))
}

// TestPreInstall tests the preInstall function.
func TestPreInstall(t *testing.T) {
	// GIVEN the Prometheus Operator is being installed
	// WHEN the preInstall function is called
	// THEN the component namespace is created in the cluster
	client := fake.NewClientBuilder().WithScheme(testScheme).Build()
	ctx := spi.NewFakeContext(client, &vzapi.Verrazzano{}, false)

	err := preInstall(ctx)
	assert.NoError(t, err)

	ns := v1.Namespace{}
	err = client.Get(context.TODO(), types.NamespacedName{Name: ComponentNamespace}, &ns)
	assert.NoError(t, err)
}

// TestAppendIstioOverrides tests that the Istio overrides get applied
func TestAppendIstioOverrides(t *testing.T) {
	annotationKey := "annKey"
	volumeMountKey := "vmKey"
	volumeKey := "volKey"
	tests := []struct {
		name            string
		expectOverrides []bom.KeyValue
	}{
		{
			name: "test expect overrides",
			expectOverrides: []bom.KeyValue{
				{
					Key:   fmt.Sprintf(`%s.traffic\.sidecar\.istio\.io/excludeOutboundIPRanges`, annotationKey),
					Value: "0.0.0.0/0",
				},
				{
					Key:   fmt.Sprintf(`%s.proxy\.istio\.io/config`, annotationKey),
					Value: `{"proxyMetadata":{ "OUTPUT_CERTS": "/etc/istio-output-certs"}}`,
				},
				{
					Key:   fmt.Sprintf(`%s.sidecar\.istio\.io/userVolumeMount`, annotationKey),
					Value: `[{"name": "istio-certs-dir", "mountPath": "/etc/istio-output-certs"}]`,
				},
				{
					Key:   fmt.Sprintf("%s[0].name", volumeMountKey),
					Value: istioVolumeName,
				},
				{
					Key:   fmt.Sprintf("%s[0].mountPath", volumeMountKey),
					Value: vmoconst.IstioCertsMountPath,
				},
				{
					Key:   fmt.Sprintf("%s[0].name", volumeKey),
					Value: istioVolumeName,
				},
				{
					Key:   fmt.Sprintf("%s[0].emptyDir.medium", volumeKey),
					Value: string(v1.StorageMediumMemory),
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			kvs, err := appendIstioOverrides(annotationKey, volumeMountKey, volumeKey, []bom.KeyValue{})

			assert.Equal(t, len(tt.expectOverrides), len(kvs))

			for _, kvsVal := range kvs {
				found := false
				for _, expVal := range tt.expectOverrides {
					if expVal == kvsVal {
						found = true
						break
					}
				}
				assert.True(t, found, fmt.Sprintf("Could not find key %s, value %s in expected key value pairs", kvsVal.Key, kvsVal.Value))
			}
			assert.NoError(t, err)
		})
	}
}

// TestValidatePrometheusOperator tests the validation of the Prometheus Operator installation and the Verrazzano CR
func TestValidatePrometheusOperator(t *testing.T) {
	tests := []struct {
		name        string
		vz          vzapi.Verrazzano
		expectError bool
	}{
		{
			name:        "test nothing enabled",
			vz:          vzapi.Verrazzano{},
			expectError: false,
		},
		{
			name: "test only Prometheus enabled",
			vz: vzapi.Verrazzano{
				Spec: vzapi.VerrazzanoSpec{
					Components: vzapi.ComponentSpec{
						Prometheus:         &vzapi.PrometheusComponent{Enabled: &trueValue},
						PrometheusOperator: &vzapi.PrometheusOperatorComponent{Enabled: &falseValue},
					},
				},
			},
			expectError: true,
		},
		{
			name: "test only Prometheus Operator enabled",
			vz: vzapi.Verrazzano{
				Spec: vzapi.VerrazzanoSpec{
					Components: vzapi.ComponentSpec{
						Prometheus:         &vzapi.PrometheusComponent{Enabled: &falseValue},
						PrometheusOperator: &vzapi.PrometheusOperatorComponent{Enabled: &trueValue},
					},
				},
			},
			expectError: false,
		},
		{
			name: "test all enabled",
			vz: vzapi.Verrazzano{
				Spec: vzapi.VerrazzanoSpec{
					Components: vzapi.ComponentSpec{
						Prometheus:         &vzapi.PrometheusComponent{Enabled: &trueValue},
						PrometheusOperator: &vzapi.PrometheusOperatorComponent{Enabled: &trueValue},
					},
				},
			},
			expectError: false,
		},
		{
			name: "test all disabled",
			vz: vzapi.Verrazzano{
				Spec: vzapi.VerrazzanoSpec{
					Components: vzapi.ComponentSpec{
						Prometheus:         &vzapi.PrometheusComponent{Enabled: &falseValue},
						PrometheusOperator: &vzapi.PrometheusOperatorComponent{Enabled: &falseValue},
					},
				},
			},
			expectError: false,
		},
	}
	c := prometheusComponent{}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := c.validatePrometheusOperator(&tt.vz)
			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// TestApplySystemMonitors tests the applySystemMonitors function
func TestApplySystemMonitors(t *testing.T) {
	// GIVEN the Prometheus Operator is being installed or upgraded
	// WHEN we call the applySystemMonitors function
	// THEN ServiceMonitor and PodMonitor resources are applied so that
	// Verrazzano system components will have their metrics collected
	oldConfig := config.Get()
	defer config.Set(oldConfig)
	config.Set(config.OperatorConfig{
		VerrazzanoRootDir: "../../../../../..",
	})

	client := fake.NewClientBuilder().WithScheme(testScheme).Build()
	ctx := spi.NewFakeContext(client, &vzapi.Verrazzano{}, false)

	err := applySystemMonitors(ctx)
	assert.NoError(t, err)

	// expect that 3 PodMonitors are created
	monitors := &unstructured.UnstructuredList{}
	monitors.SetGroupVersionKind(schema.GroupVersionKind{Group: "monitoring.coreos.com", Version: "v1", Kind: "PodMonitor"})
	err = client.List(context.TODO(), monitors)
	assert.NoError(t, err)
	assert.Len(t, monitors.Items, 3)

	// expect that 1 ServiceMonitor is created
	monitors = &unstructured.UnstructuredList{}
	monitors.SetGroupVersionKind(schema.GroupVersionKind{Group: "monitoring.coreos.com", Version: "v1", Kind: "ServiceMonitor"})
	err = client.List(context.TODO(), monitors)
	assert.NoError(t, err)
	assert.Len(t, monitors.Items, 1)
}

// TestValidatePrometheusOperator tests the validation of the Prometheus Operator installation and the Verrazzano CR
func TestUpdateApplicationAuthorizationPolicies(t *testing.T) {
	assert := asserts.New(t)
	scheme := k8scheme.Scheme
	_ = vzapi.AddToScheme(scheme)
	_ = istioclisec.AddToScheme(scheme)

	testNsName := "test-ns"
	testAuthPolicyName := "test-authpolicy"
	principal := "cluster.local/ns/verrazzano-monitoring/sa/prometheus-operator-kube-p-prometheus"
	namespace := v1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name:   testNsName,
			Labels: map[string]string{vzconst.VerrazzanoManagedLabelKey: "true"},
		},
	}

	tests := []struct {
		name               string
		objects            []client.Object
		expectedPrincipals *[]string
	}{
		{
			name:               "test no namespaces",
			objects:            []client.Object{},
			expectedPrincipals: nil,
		},
		{
			name: "test no authpolicy",
			objects: []client.Object{
				&namespace,
			},
			expectedPrincipals: nil,
		},
		{
			name: "test no rules",
			objects: []client.Object{
				&namespace,
				&istioclisec.AuthorizationPolicy{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: testNsName,
						Name:      testAuthPolicyName,
						Labels:    map[string]string{constants.IstioAppLabel: testAuthPolicyName},
					},
				},
			},
			expectedPrincipals: nil,
		},
		{
			name: "test nil rule",
			objects: []client.Object{
				&namespace,
				&istioclisec.AuthorizationPolicy{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: testNsName,
						Name:      testAuthPolicyName,
						Labels:    map[string]string{constants.IstioAppLabel: testAuthPolicyName},
					},
					Spec: istioclisecv1beta1.AuthorizationPolicy{
						Rules: []*istioclisecv1beta1.Rule{
							nil,
						},
					},
				},
			},
			expectedPrincipals: nil,
		},
		{
			name: "test nil from",
			objects: []client.Object{
				&namespace,
				&istioclisec.AuthorizationPolicy{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: testNsName,
						Name:      testAuthPolicyName,
						Labels:    map[string]string{constants.IstioAppLabel: testAuthPolicyName},
					},
					Spec: istioclisecv1beta1.AuthorizationPolicy{
						Rules: []*istioclisecv1beta1.Rule{
							{
								From: []*istioclisecv1beta1.Rule_From{
									nil,
								},
							},
						},
					},
				},
			},
			expectedPrincipals: nil,
		},
		{
			name: "test nil source",
			objects: []client.Object{
				&namespace,
				&istioclisec.AuthorizationPolicy{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: testNsName,
						Name:      testAuthPolicyName,
						Labels:    map[string]string{constants.IstioAppLabel: testAuthPolicyName},
					},
					Spec: istioclisecv1beta1.AuthorizationPolicy{
						Rules: []*istioclisecv1beta1.Rule{
							{
								From: []*istioclisecv1beta1.Rule_From{
									{
										Source: nil,
									},
								},
							},
						},
					},
				},
			},
			expectedPrincipals: nil,
		},
		{
			name: "test empty principals",
			objects: []client.Object{
				&namespace,
				&istioclisec.AuthorizationPolicy{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: testNsName,
						Name:      testAuthPolicyName,
						Labels:    map[string]string{constants.IstioAppLabel: testAuthPolicyName},
					},
					Spec: istioclisecv1beta1.AuthorizationPolicy{
						Rules: []*istioclisecv1beta1.Rule{
							{
								From: []*istioclisecv1beta1.Rule_From{
									{
										Source: &istioclisecv1beta1.Source{Principals: []string{}},
									},
								},
							},
						},
					},
				},
			},
			expectedPrincipals: &[]string{principal},
		},
		{
			name: "test non-empty principals",
			objects: []client.Object{
				&namespace,
				&istioclisec.AuthorizationPolicy{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: testNsName,
						Name:      testAuthPolicyName,
						Labels:    map[string]string{constants.IstioAppLabel: testAuthPolicyName},
					},
					Spec: istioclisecv1beta1.AuthorizationPolicy{
						Rules: []*istioclisecv1beta1.Rule{
							{
								From: []*istioclisecv1beta1.Rule_From{
									{
										Source: &istioclisecv1beta1.Source{Principals: []string{
											"p1", "p2", "p3",
										}},
									},
								},
							},
						},
					},
				},
			},
			expectedPrincipals: &[]string{"p1", "p2", "p3", principal},
		},
		{
			name: "test multiple authpolicies",
			objects: []client.Object{
				&namespace,
				&istioclisec.AuthorizationPolicy{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: testNsName,
						Name:      testAuthPolicyName,
						Labels:    map[string]string{constants.IstioAppLabel: testAuthPolicyName},
					},
					Spec: istioclisecv1beta1.AuthorizationPolicy{
						Rules: []*istioclisecv1beta1.Rule{
							{
								From: []*istioclisecv1beta1.Rule_From{
									{
										Source: &istioclisecv1beta1.Source{Principals: []string{
											"p1", "p2", "p3",
										}},
									},
								},
							},
						},
					},
				},
				&istioclisec.AuthorizationPolicy{},
			},
			expectedPrincipals: &[]string{"p1", "p2", "p3", principal},
		},
		{
			name: "test no authpolicy label",
			objects: []client.Object{
				&namespace,
				&istioclisec.AuthorizationPolicy{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: testNsName,
						Name:      testAuthPolicyName,
					},
					Spec: istioclisecv1beta1.AuthorizationPolicy{
						Rules: []*istioclisecv1beta1.Rule{
							{
								From: []*istioclisecv1beta1.Rule_From{
									{
										Source: &istioclisecv1beta1.Source{Principals: []string{
											"p1", "p2", "p3",
										}},
									},
								},
							},
						},
					},
				},
			},
			expectedPrincipals: nil,
		},
		{
			name: "test existing principal",
			objects: []client.Object{
				&namespace,
				&istioclisec.AuthorizationPolicy{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: testNsName,
						Name:      testAuthPolicyName,
					},
					Spec: istioclisecv1beta1.AuthorizationPolicy{
						Rules: []*istioclisecv1beta1.Rule{
							{
								From: []*istioclisecv1beta1.Rule_From{
									{
										Source: &istioclisecv1beta1.Source{Principals: []string{
											"p1", "p2", "p3", principal,
										}},
									},
								},
							},
						},
					},
				},
			},
			expectedPrincipals: &[]string{"p1", "p2", "p3", principal},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fakes := fake.NewClientBuilder().WithScheme(scheme).WithObjects(tt.objects...).Build()
			ctx := spi.NewFakeContext(fakes, &vzapi.Verrazzano{}, false)
			err := updateApplicationAuthorizationPolicies(ctx)
			assert.NoError(err)
			if tt.expectedPrincipals != nil {
				nsList := v1.NamespaceList{}
				err = fakes.List(context.TODO(), &nsList)
				assert.NoError(err)
				for _, ns := range nsList.Items {
					authPolicyList := istioclisec.AuthorizationPolicyList{}
					err = fakes.List(context.TODO(), &authPolicyList, &client.ListOptions{Namespace: ns.Name})
					assert.NoError(err)
					for _, authPolicy := range authPolicyList.Items {
						foundPrincipals := []string{}
						for _, principal := range authPolicy.Spec.Rules[0].From[0].Source.Principals {
							assert.NotContains(foundPrincipals, principal)
							assert.Contains(*tt.expectedPrincipals, principal)
							foundPrincipals = append(foundPrincipals, principal)
						}
					}
				}
			}
		})
	}
}
