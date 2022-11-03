// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package operator

import (
	"context"
	"fmt"
	"strings"
	"testing"

	certapiv1 "github.com/jetstack/cert-manager/pkg/apis/certmanager/v1"
	cmmeta "github.com/jetstack/cert-manager/pkg/apis/meta/v1"
	promoperapi "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	"github.com/stretchr/testify/assert"
	"github.com/verrazzano/verrazzano/pkg/bom"
	vzconst "github.com/verrazzano/verrazzano/pkg/constants"
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1beta1"
	"github.com/verrazzano/verrazzano/platform-operator/constants"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/authproxy"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/common"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	"github.com/verrazzano/verrazzano/platform-operator/internal/config"
	istioclisecv1beta1 "istio.io/api/security/v1beta1"
	istioclisec "istio.io/client-go/pkg/apis/security/v1beta1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	netv1 "k8s.io/api/networking/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	k8scheme "k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

const (
	testBomFilePath = "../../../testdata/test_bom.json"

	disableMountSubPathKey = "prometheus.prometheusSpec.storageSpec.disableMountSubPath"
	requestsStorageKey     = "prometheus.prometheusSpec.storageSpec.volumeClaimTemplate.spec.resources.requests.storage"

	requestsMemoryKey = "prometheus.prometheusSpec.resources.requests.memory"
)

var (
	testScheme = runtime.NewScheme()

	falseValue = false
	trueValue  = true
)

func init() {
	_ = k8scheme.AddToScheme(testScheme)
	_ = vzapi.AddToScheme(testScheme)
	_ = certapiv1.AddToScheme(testScheme)
	_ = istioclisec.AddToScheme(testScheme)
	_ = promoperapi.AddToScheme(testScheme)
}

// TestIsPrometheusOperatorReady tests the isPrometheusOperatorReady function for the Prometheus Operator
func TestIsPrometheusOperatorReady(t *testing.T) {
	operatorObjectMeta := metav1.ObjectMeta{
		Namespace: ComponentNamespace,
		Name:      deploymentName,
		Labels: map[string]string{
			"app.kubernetes.io/instance":          ComponentName,
			constants.VerrazzanoComponentLabelKey: ComponentName,
		},
	}
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
					ObjectMeta: operatorObjectMeta,
					Spec: appsv1.DeploymentSpec{
						Selector: &metav1.LabelSelector{
							MatchLabels: map[string]string{"app.kubernetes.io/instance": ComponentName},
						},
					},
					Status: appsv1.DeploymentStatus{
						AvailableReplicas: 1,
						Replicas:          1,
						UpdatedReplicas:   1,
					},
				},
				&corev1.Pod{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: ComponentNamespace,
						Name:      deploymentName + "-95d8c5d96-m6mbr",
						Labels: map[string]string{
							"pod-template-hash":          "95d8c5d96",
							"app.kubernetes.io/instance": ComponentName,
						},
					},
				},
				&appsv1.ReplicaSet{
					ObjectMeta: metav1.ObjectMeta{
						Namespace:   ComponentNamespace,
						Name:        deploymentName + "-95d8c5d96",
						Annotations: map[string]string{"deployment.kubernetes.io/revision": "1"},
					},
				},
			).Build(),
			expectTrue: true,
		},
		{
			// GIVEN the Prometheus Operator deployment exists and there are no available replicas
			// WHEN we call isPrometheusOperatorReady
			// THEN the call returns false
			name: "Test IsReady when Prometheus Operator deployment is not ready",
			client: fake.NewClientBuilder().WithScheme(testScheme).WithObjects(
				&appsv1.Deployment{
					ObjectMeta: operatorObjectMeta,
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
			ctx := spi.NewFakeContext(tt.client, &vzapi.Verrazzano{}, nil, false)
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

	ctx := spi.NewFakeContext(client, vz, nil, false)

	var err error
	kvs, err = AppendOverrides(ctx, "", "", "", kvs)
	assert.NoError(t, err)
	assert.Len(t, kvs, 26)

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

	ctx = spi.NewFakeContext(client, vz, nil, false)
	kvs = make([]bom.KeyValue, 0)

	kvs, err = AppendOverrides(ctx, "", "", "", kvs)
	assert.NoError(t, err)
	assert.Len(t, kvs, 26)

	assert.Equal(t, "false", bom.FindKV(kvs, "prometheusOperator.admissionWebhooks.certManager.enabled"))

	// GIVEN a Verrazzano CR with Prometheus disabled
	// WHEN the AppendOverrides function is called
	// THEN the key/value slice contains the expected helm override keys and values
	vz = &vzapi.Verrazzano{
		Spec: vzapi.VerrazzanoSpec{
			Components: vzapi.ComponentSpec{
				Prometheus: &vzapi.PrometheusComponent{
					Enabled: &falseValue,
				},
			},
		},
	}

	ctx = spi.NewFakeContext(client, vz, nil, false)
	kvs = make([]bom.KeyValue, 0)

	kvs, err = AppendOverrides(ctx, "", "", "", kvs)
	assert.NoError(t, err)
	assert.Len(t, kvs, 12)

	assert.Equal(t, "false", bom.FindKV(kvs, "prometheus.enabled"))
}

// TestPreInstallUpgrade tests the preInstallUpgrade function.
func TestPreInstallUpgrade(t *testing.T) {
	// GIVEN the Prometheus Operator is being installed or upgraded
	// WHEN the preInstallUpgrade function is called
	// THEN the component namespace is created in the cluster
	client := fake.NewClientBuilder().WithScheme(testScheme).Build()
	ctx := spi.NewFakeContext(client, &vzapi.Verrazzano{}, nil, false)

	err := preInstallUpgrade(ctx)
	assert.NoError(t, err)

	ns := corev1.Namespace{}
	err = client.Get(context.TODO(), types.NamespacedName{Name: ComponentNamespace}, &ns)
	assert.NoError(t, err)
}

// TestPostInstallUpgrade tests the postInstallUpgrade function.
func TestPostInstallUpgrade(t *testing.T) {
	// GIVEN the Prometheus Operator is being installed or upgraded
	// WHEN the postInstallUpgrade function is called
	// THEN the function does not return an error
	oldConfig := config.Get()
	defer config.Set(oldConfig)
	config.Set(config.OperatorConfig{
		VerrazzanoRootDir: "../../../../../..",
	})

	vz := &vzapi.Verrazzano{
		Spec: vzapi.VerrazzanoSpec{
			Components: vzapi.ComponentSpec{
				DNS: &vzapi.DNSComponent{
					OCI: &vzapi.OCI{
						DNSZoneName: "mydomain.com",
					},
				},
			},
		},
	}

	ingress := &netv1.Ingress{
		ObjectMeta: metav1.ObjectMeta{Name: constants.PrometheusIngress, Namespace: authproxy.ComponentNamespace},
	}

	time := metav1.Now()
	cert := &certapiv1.Certificate{
		ObjectMeta: metav1.ObjectMeta{Name: prometheusCertificateName, Namespace: authproxy.ComponentNamespace},
		Status: certapiv1.CertificateStatus{
			Conditions: []certapiv1.CertificateCondition{
				{Type: certapiv1.CertificateConditionReady, Status: cmmeta.ConditionTrue, LastTransitionTime: &time},
			},
		},
	}

	client := fake.NewClientBuilder().WithScheme(testScheme).WithObjects(ingress, cert).Build()
	ctx := spi.NewFakeContext(client, vz, nil, false)

	err := postInstallUpgrade(ctx)
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
					Value: istioCertMountPath,
				},
				{
					Key:   fmt.Sprintf("%s[0].name", volumeKey),
					Value: istioVolumeName,
				},
				{
					Key:   fmt.Sprintf("%s[0].emptyDir.medium", volumeKey),
					Value: string(corev1.StorageMediumMemory),
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
		{
			name: "test Prometheus Operator disabled, Prometheus not specified (implicitly enabled)",
			vz: vzapi.Verrazzano{
				Spec: vzapi.VerrazzanoSpec{
					Components: vzapi.ComponentSpec{
						PrometheusOperator: &vzapi.PrometheusOperatorComponent{Enabled: &falseValue},
					},
				},
			},
			expectError: true,
		},
	}
	c := prometheusComponent{}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			convertedVZ := v1beta1.Verrazzano{}
			err := common.ConvertVerrazzanoCR(&tt.vz, &convertedVZ)
			assert.NoError(t, err)
			err = c.validatePrometheusOperator(&convertedVZ)
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
	ctx := spi.NewFakeContext(client, &vzapi.Verrazzano{}, nil, false)

	err := applySystemMonitors(ctx)
	assert.NoError(t, err)

	// expected number of PodMonitors is created
	monitors := &unstructured.UnstructuredList{}
	monitors.SetGroupVersionKind(schema.GroupVersionKind{Group: "monitoring.coreos.com", Version: "v1", Kind: "PodMonitor"})
	err = client.List(context.TODO(), monitors)
	assert.NoError(t, err)
	assert.Len(t, monitors.Items, 2)

	monitors = &unstructured.UnstructuredList{}
	monitors.SetGroupVersionKind(schema.GroupVersionKind{Group: "monitoring.coreos.com", Version: "v1", Kind: "ServiceMonitor"})
	err = client.List(context.TODO(), monitors)
	assert.NoError(t, err)
	// expect that 9 ServiceMonitors are created
	assert.Len(t, monitors.Items, 9)
}

// TestValidatePrometheusOperator tests the validation of the Prometheus Operator installation and the Verrazzano CR
func TestUpdateApplicationAuthorizationPolicies(t *testing.T) {
	assertions := assert.New(t)
	scheme := k8scheme.Scheme
	_ = vzapi.AddToScheme(scheme)
	_ = istioclisec.AddToScheme(scheme)

	testNsName := "test-ns"
	testAuthPolicyName := "test-authpolicy"
	principal := "cluster.local/ns/verrazzano-monitoring/sa/prometheus-operator-kube-p-prometheus"
	namespace := corev1.Namespace{
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
			ctx := spi.NewFakeContext(fakes, &vzapi.Verrazzano{}, nil, false)
			err := updateApplicationAuthorizationPolicies(ctx)
			assertions.NoError(err)
			if tt.expectedPrincipals != nil {
				nsList := corev1.NamespaceList{}
				err = fakes.List(context.TODO(), &nsList)
				assertions.NoError(err)
				for _, ns := range nsList.Items {
					authPolicyList := istioclisec.AuthorizationPolicyList{}
					err = fakes.List(context.TODO(), &authPolicyList, &client.ListOptions{Namespace: ns.Name})
					assertions.NoError(err)
					for _, authPolicy := range authPolicyList.Items {
						foundPrincipals := []string{}
						for _, principal := range authPolicy.Spec.Rules[0].From[0].Source.Principals {
							assertions.NotContains(foundPrincipals, principal)
							assertions.Contains(*tt.expectedPrincipals, principal)
							foundPrincipals = append(foundPrincipals, principal)
						}
					}
				}
			}
		})
	}
}

// TestCreateOrUpdatePrometheusAuthPolicy tests the createOrUpdatePrometheusAuthPolicy function
func TestCreateOrUpdatePrometheusAuthPolicy(t *testing.T) {
	assertions := assert.New(t)

	// GIVEN Prometheus Operator is being installed or upgraded
	// WHEN  we call the createOrUpdatePrometheusAuthPolicy function
	// THEN  the expected Istio authorization policy is created
	client := fake.NewClientBuilder().WithScheme(testScheme).Build()
	ctx := spi.NewFakeContext(client, &vzapi.Verrazzano{}, nil, false)

	err := createOrUpdatePrometheusAuthPolicy(ctx)
	assertions.NoError(err)

	authPolicy := &istioclisec.AuthorizationPolicy{}
	err = client.Get(context.TODO(), types.NamespacedName{Namespace: ComponentNamespace, Name: prometheusAuthPolicyName}, authPolicy)
	assertions.NoError(err)

	assertions.Len(authPolicy.Spec.Rules, 3)
	assertions.Contains(authPolicy.Spec.Rules[0].From[0].Source.Principals, "cluster.local/ns/verrazzano-system/sa/verrazzano-authproxy")
	assertions.Contains(authPolicy.Spec.Rules[0].From[0].Source.Principals, "cluster.local/ns/verrazzano-system/sa/verrazzano-monitoring-operator")
	assertions.Contains(authPolicy.Spec.Rules[0].From[0].Source.Principals, "cluster.local/ns/verrazzano-system/sa/vmi-system-kiali")
	assertions.Contains(authPolicy.Spec.Rules[1].From[0].Source.Principals, serviceAccount)
	assertions.Contains(authPolicy.Spec.Rules[2].From[0].Source.Principals, "cluster.local/ns/verrazzano-monitoring/sa/jaeger-operator-jaeger")

	// GIVEN Prometheus Operator is being installed or upgraded
	// AND   Istio is disabled
	// WHEN  we call the createOrUpdatePrometheusAuthPolicy function
	// THEN  no Istio authorization policy is created
	client = fake.NewClientBuilder().WithScheme(testScheme).Build()
	vz := &vzapi.Verrazzano{
		Spec: vzapi.VerrazzanoSpec{
			Components: vzapi.ComponentSpec{
				Istio: &vzapi.IstioComponent{
					Enabled: &falseValue,
				},
			},
		},
	}
	ctx = spi.NewFakeContext(client, vz, nil, false)

	err = createOrUpdatePrometheusAuthPolicy(ctx)
	assertions.NoError(err)

	authPolicy = &istioclisec.AuthorizationPolicy{}
	err = client.Get(context.TODO(), types.NamespacedName{Namespace: ComponentNamespace, Name: prometheusAuthPolicyName}, authPolicy)
	assertions.ErrorContains(err, "not found")
}

// TestCreateOrUpdateNetworkPolicies tests the createOrUpdateNetworkPolicies function
func TestCreateOrUpdateNetworkPolicies(t *testing.T) {
	// GIVEN a Prometheus Operator component
	// WHEN  the createOrUpdateNetworkPolicies function is called
	// THEN  no error is returned and the expected network policies have been created
	client := fake.NewClientBuilder().WithScheme(testScheme).Build()
	ctx := spi.NewFakeContext(client, &vzapi.Verrazzano{}, nil, false)

	err := createOrUpdateNetworkPolicies(ctx)
	assert.NoError(t, err)

	netPolicy := &netv1.NetworkPolicy{}
	err = client.Get(context.TODO(), types.NamespacedName{Name: networkPolicyName, Namespace: ComponentNamespace}, netPolicy)
	assert.NoError(t, err)
	assert.Len(t, netPolicy.Spec.Ingress, 2)
	assert.Equal(t, []netv1.PolicyType{netv1.PolicyTypeIngress}, netPolicy.Spec.PolicyTypes)
	assert.Equal(t, int32(9090), netPolicy.Spec.Ingress[0].Ports[0].Port.IntVal)
	assert.Equal(t, int32(9090), netPolicy.Spec.Ingress[1].Ports[0].Port.IntVal)
	assert.Contains(t, netPolicy.Spec.Ingress[0].From[0].PodSelector.MatchExpressions[0].Values, "verrazzano-authproxy")
	assert.Contains(t, netPolicy.Spec.Ingress[0].From[0].PodSelector.MatchExpressions[0].Values, "system-grafana")
	assert.Contains(t, netPolicy.Spec.Ingress[0].From[0].PodSelector.MatchExpressions[0].Values, "kiali")
	assert.Contains(t, netPolicy.Spec.Ingress[1].From[0].PodSelector.MatchExpressions[0].Values, "jaeger")
}

// erroringFakeClient wraps a k8s client and returns an error when Update is called
type erroringFakeClient struct {
	client.Client
}

// Update always returns an error - used to simulate an error updating a resource
func (e *erroringFakeClient) Update(_ context.Context, _ client.Object, _ ...client.UpdateOption) error {
	return errors.NewConflict(schema.GroupResource{}, "", nil)
}

// TestRemoveOldClaimFromPrometheusVolume tests the updateExistingVolumeClaims function
func TestRemoveOldClaimFromPrometheusVolume(t *testing.T) {
	const volumeName = "pvc-5ab58a05-71f9-4f09-8911-a5c029f6305f"

	// GIVEN a persistent volume that has a released status and a claim that references vmi-system-prometheus
	// WHEN the updateExistingVolumeClaims function is called
	// THEN the persistent volume is updated and the claim is removed
	client := fake.NewClientBuilder().WithScheme(testScheme).WithObjects(
		&corev1.PersistentVolume{
			ObjectMeta: metav1.ObjectMeta{
				Name: volumeName,
				Labels: map[string]string{
					constants.StorageForLabel: constants.PrometheusStorageLabelValue,
				},
			},
			Spec: corev1.PersistentVolumeSpec{
				ClaimRef: &corev1.ObjectReference{
					Name:      constants.VMISystemPrometheusVolumeClaim,
					Namespace: constants.VerrazzanoSystemNamespace,
				},
			},
			Status: corev1.PersistentVolumeStatus{
				Phase: corev1.VolumeReleased,
			},
		}).Build()
	ctx := spi.NewFakeContext(client, &vzapi.Verrazzano{}, nil, false)

	err := updateExistingVolumeClaims(ctx)
	assert.NoError(t, err)

	// validate that the ClaimRef is now nil
	pv := &corev1.PersistentVolume{}
	err = client.Get(context.TODO(), types.NamespacedName{Name: volumeName}, pv)
	assert.NoError(t, err)
	assert.Nil(t, pv.Spec.ClaimRef)

	// GIVEN no persistent volumes
	// WHEN the updateExistingVolumeClaims function is called
	// THEN no error is returned
	client = fake.NewClientBuilder().WithScheme(testScheme).WithObjects(
		&corev1.PersistentVolume{
			ObjectMeta: metav1.ObjectMeta{
				Name: volumeName,
			},
		}).Build()
	ctx = spi.NewFakeContext(client, &vzapi.Verrazzano{}, nil, false)

	err = updateExistingVolumeClaims(ctx)
	assert.NoError(t, err)

	// GIVEN a persistent volume that is bound and has a claim that references vmi-system-prometheus
	// WHEN the updateExistingVolumeClaims function is called
	// THEN the persistent volume is not updated
	client = fake.NewClientBuilder().WithScheme(testScheme).WithObjects(
		&corev1.PersistentVolume{
			ObjectMeta: metav1.ObjectMeta{
				Name: volumeName,
				Labels: map[string]string{
					constants.StorageForLabel: constants.PrometheusStorageLabelValue,
				},
			},
			Spec: corev1.PersistentVolumeSpec{
				ClaimRef: &corev1.ObjectReference{
					Name:      constants.VMISystemPrometheusVolumeClaim,
					Namespace: constants.VerrazzanoSystemNamespace,
				},
			},
			Status: corev1.PersistentVolumeStatus{
				Phase: corev1.VolumeBound,
			},
		}).Build()
	ctx = spi.NewFakeContext(client, &vzapi.Verrazzano{}, nil, false)

	err = updateExistingVolumeClaims(ctx)
	assert.NoError(t, err)

	// validate that the ClaimRef is not nil
	pv = &corev1.PersistentVolume{}
	err = client.Get(context.TODO(), types.NamespacedName{Name: volumeName}, pv)
	assert.NoError(t, err)
	assert.NotNil(t, pv.Spec.ClaimRef)

	// GIVEN a persistent volume that has a released status and a claim that references vmi-system-prometheus
	// WHEN the updateExistingVolumeClaims function is called and the call to update the volume fails
	// THEN an error is returned
	client = fake.NewClientBuilder().WithScheme(testScheme).WithObjects(
		&corev1.PersistentVolume{
			ObjectMeta: metav1.ObjectMeta{
				Name: volumeName,
				Labels: map[string]string{
					constants.StorageForLabel: constants.PrometheusStorageLabelValue,
				},
			},
			Spec: corev1.PersistentVolumeSpec{
				ClaimRef: &corev1.ObjectReference{
					Name:      constants.VMISystemPrometheusVolumeClaim,
					Namespace: constants.VerrazzanoSystemNamespace,
				},
			},
			Status: corev1.PersistentVolumeStatus{
				Phase: corev1.VolumeReleased,
			},
		}).Build()
	erroringClient := &erroringFakeClient{Client: client}
	ctx = spi.NewFakeContext(erroringClient, &vzapi.Verrazzano{}, nil, false)

	// validate that the expected error is returned
	err = updateExistingVolumeClaims(ctx)
	assert.ErrorContains(t, err, "Failed removing claim")
}

// TestResetVolumeReclaimPolicy tests the resetVolumeReclaimPolicy function
func TestResetVolumeReclaimPolicy(t *testing.T) {
	const volumeName = "pvc-5ab58a05-71f9-4f09-8911-a5c029f6305a"

	// GIVEN a persistent volume that has a bound status
	// WHEN the resetVolumeReclaimPolicy function is called
	// THEN the persistent volume reclaim policy is reset to the original value
	// AND the old-reclaim-policy label is removed
	client := fake.NewClientBuilder().WithScheme(testScheme).WithObjects(
		&corev1.PersistentVolume{
			ObjectMeta: metav1.ObjectMeta{
				Name: volumeName,
				Labels: map[string]string{
					constants.StorageForLabel:       constants.PrometheusStorageLabelValue,
					constants.OldReclaimPolicyLabel: string(corev1.PersistentVolumeReclaimDelete),
				},
			},
			Spec: corev1.PersistentVolumeSpec{
				PersistentVolumeReclaimPolicy: corev1.PersistentVolumeReclaimRetain,
			},
			Status: corev1.PersistentVolumeStatus{
				Phase: corev1.VolumeBound,
			},
		}).Build()
	ctx := spi.NewFakeContext(client, &vzapi.Verrazzano{}, nil, false)

	err := resetVolumeReclaimPolicy(ctx)
	assert.NoError(t, err)

	// validate that the reclaim policy is now "Delete" and the label has been removed
	pv := &corev1.PersistentVolume{}
	err = client.Get(context.TODO(), types.NamespacedName{Name: volumeName}, pv)
	assert.NoError(t, err)
	assert.Equal(t, corev1.PersistentVolumeReclaimDelete, pv.Spec.PersistentVolumeReclaimPolicy)
	assert.NotContains(t, pv.Labels, constants.OldReclaimPolicyLabel)

	// GIVEN a persistent volume that has an available status
	// WHEN the resetVolumeReclaimPolicy function is called
	// THEN the persistent volume is not updated
	client = fake.NewClientBuilder().WithScheme(testScheme).WithObjects(
		&corev1.PersistentVolume{
			ObjectMeta: metav1.ObjectMeta{
				Name: volumeName,
				Labels: map[string]string{
					constants.StorageForLabel:       constants.PrometheusStorageLabelValue,
					constants.OldReclaimPolicyLabel: string(corev1.PersistentVolumeReclaimDelete),
				},
			},
			Spec: corev1.PersistentVolumeSpec{
				PersistentVolumeReclaimPolicy: corev1.PersistentVolumeReclaimRetain,
			},
			Status: corev1.PersistentVolumeStatus{
				Phase: corev1.VolumeAvailable,
			},
		}).Build()
	ctx = spi.NewFakeContext(client, &vzapi.Verrazzano{}, nil, false)

	err = resetVolumeReclaimPolicy(ctx)
	assert.NoError(t, err)

	// validate that the reclaim policy has not changed and that the label still exists
	pv = &corev1.PersistentVolume{}
	err = client.Get(context.TODO(), types.NamespacedName{Name: volumeName}, pv)
	assert.NoError(t, err)
	assert.Equal(t, corev1.PersistentVolumeReclaimRetain, pv.Spec.PersistentVolumeReclaimPolicy)
	assert.Contains(t, pv.Labels, constants.OldReclaimPolicyLabel)

	// GIVEN a persistent volume that has a bound status
	// WHEN the resetVolumeReclaimPolicy function is called and the call to update the volume fails
	// THEN an error is returned
	client = fake.NewClientBuilder().WithScheme(testScheme).WithObjects(
		&corev1.PersistentVolume{
			ObjectMeta: metav1.ObjectMeta{
				Name: volumeName,
				Labels: map[string]string{
					constants.StorageForLabel:       constants.PrometheusStorageLabelValue,
					constants.OldReclaimPolicyLabel: string(corev1.PersistentVolumeReclaimDelete),
				},
			},
			Spec: corev1.PersistentVolumeSpec{
				PersistentVolumeReclaimPolicy: corev1.PersistentVolumeReclaimRetain,
			},
			Status: corev1.PersistentVolumeStatus{
				Phase: corev1.VolumeBound,
			},
		}).Build()
	erroringClient := &erroringFakeClient{Client: client}
	ctx = spi.NewFakeContext(erroringClient, &vzapi.Verrazzano{}, nil, false)

	// validate that the expected error is returned
	err = resetVolumeReclaimPolicy(ctx)
	assert.ErrorContains(t, err, "Failed resetting reclaim policy")
}

// TestCreateUpdateServiceMonitors tests the createUpdateServiceMonitors function
func TestCreateUpdateServiceMonitors(t *testing.T) {
	endPoints := make([]promoperapi.Endpoint, 0)
	endPoint := promoperapi.Endpoint{
		Port:                 "8080",
		TargetPort:           nil,
		Path:                 "test",
		Scheme:               "https",
		Params:               nil,
		Interval:             "10",
		ScrapeTimeout:        "5",
		TLSConfig:            nil,
		BearerTokenFile:      "",
		BearerTokenSecret:    corev1.SecretKeySelector{},
		Authorization:        nil,
		HonorLabels:          false,
		HonorTimestamps:      nil,
		BasicAuth:            nil,
		OAuth2:               nil,
		MetricRelabelConfigs: nil,
		RelabelConfigs:       nil,
		ProxyURL:             nil,
		FollowRedirects:      nil,
		EnableHttp2:          nil,
	}

	endPoints = append(endPoints, endPoint)

	clientWithSM := fake.NewClientBuilder().WithScheme(testScheme).WithObjects(
		&promoperapi.ServiceMonitor{
			TypeMeta:   metav1.TypeMeta{Kind: promoperapi.ServiceMonitorsKind, APIVersion: promoperapi.Version},
			ObjectMeta: metav1.ObjectMeta{Namespace: "test", Name: "testServiceMonitor"},
			Spec: promoperapi.ServiceMonitorSpec{
				JobLabel:              "test",
				TargetLabels:          nil,
				PodTargetLabels:       nil,
				Endpoints:             endPoints,
				Selector:              metav1.LabelSelector{},
				NamespaceSelector:     promoperapi.NamespaceSelector{},
				SampleLimit:           0,
				TargetLimit:           0,
				LabelLimit:            0,
				LabelNameLengthLimit:  0,
				LabelValueLengthLimit: 0,
			},
		}).Build()

	err := createOrUpdateServiceMonitors(spi.NewFakeContext(clientWithSM, &vzapi.Verrazzano{}, nil, false))
	assert.NoError(t, err)
}

// TestAppendResourceRequestOverrides tests the appendResourceRequestOverrides function
func TestAppendResourceRequestOverrides(t *testing.T) {
	const (
		storageSize = "1Gi"
		memorySize  = "128Mi"
	)
	clientNoPV := fake.NewClientBuilder().WithScheme(testScheme).Build()
	clientWithPV := fake.NewClientBuilder().WithScheme(testScheme).WithObjects(
		&corev1.PersistentVolume{
			ObjectMeta: metav1.ObjectMeta{
				Name: "pvc-5ab58a05-71f9-4f09-8911-a5c029f6305b",
				Labels: map[string]string{
					constants.StorageForLabel: constants.PrometheusStorageLabelValue,
				},
			},
		}).Build()

	tests := []struct {
		name            string
		client          client.Client
		request         common.ResourceRequestValues
		expectOverrides []bom.KeyValue
		expectError     bool
	}{
		{
			// GIVEN a resource request with both storage and memory set and there are no existing Prometheus persistent volumes
			// WHEN the appendResourceRequestOverrides function is called
			// THEN no error is returned and the expected key/value overrides are returned
			name:   "both storage and memory set, no existing Prometheus persistent volume",
			client: clientNoPV,
			request: common.ResourceRequestValues{
				Storage: storageSize,
				Memory:  memorySize,
			},
			expectOverrides: []bom.KeyValue{
				{
					Key:   disableMountSubPathKey,
					Value: "true",
				},
				{
					Key:   requestsStorageKey,
					Value: storageSize,
				},
				{
					Key:   requestsMemoryKey,
					Value: memorySize,
				},
			},
			expectError: false,
		},
		{
			// GIVEN a resource request with both storage and memory set and there is an existing Prometheus persistent volume
			// WHEN the appendResourceRequestOverrides function is called
			// THEN no error is returned and the expected key/value overrides are returned
			name:   "both storage and memory set, and existing Prometheus persistent volume",
			client: clientWithPV,
			request: common.ResourceRequestValues{
				Storage: storageSize,
				Memory:  memorySize,
			},
			expectOverrides: []bom.KeyValue{
				{
					Key:   disableMountSubPathKey,
					Value: "true",
				},
				{
					Key:   requestsStorageKey,
					Value: storageSize,
				},
				{
					Key:   requestsMemoryKey,
					Value: memorySize,
				},
			},
			expectError: false,
		},
		{
			// GIVEN a resource request with no storage or memory requests
			// WHEN the appendResourceRequestOverrides function is called
			// THEN no error is returned and no key/value overrides are returned
			name:            "neither storage nor memory set",
			client:          clientNoPV,
			request:         common.ResourceRequestValues{},
			expectOverrides: []bom.KeyValue{},
			expectError:     false,
		},
		{
			// GIVEN a resource request with only storage set and there are no existing Prometheus persistent volumes
			// WHEN the appendResourceRequestOverrides function is called
			// THEN no error is returned and the expected key/value overrides are returned
			name:   "only storage set, no persistent volumes",
			client: clientNoPV,
			request: common.ResourceRequestValues{
				Storage: storageSize,
			},
			expectOverrides: []bom.KeyValue{
				{
					Key:   disableMountSubPathKey,
					Value: "true",
				},
				{
					Key:   requestsStorageKey,
					Value: storageSize,
				},
			},
			expectError: false,
		},
		{
			// GIVEN a resource request with only storage set and there are is an existing Prometheus persistent volume
			// WHEN the appendResourceRequestOverrides function is called
			// THEN no error is returned and the expected key/value overrides are returned
			name:   "only storage set, and persistent volume exists",
			client: clientWithPV,
			request: common.ResourceRequestValues{
				Storage: storageSize,
			},
			expectOverrides: []bom.KeyValue{
				{
					Key:   disableMountSubPathKey,
					Value: "true",
				},
				{
					Key:   requestsStorageKey,
					Value: storageSize,
				},
			},
			expectError: false,
		},
		{
			// GIVEN a resource request with only memory set
			// WHEN the appendResourceRequestOverrides function is called
			// THEN no error is returned and the expected key/value overrides are returned
			name:   "only memory set",
			client: clientNoPV,
			request: common.ResourceRequestValues{
				Memory: memorySize,
			},
			expectOverrides: []bom.KeyValue{
				{
					Key:   requestsMemoryKey,
					Value: memorySize,
				},
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := spi.NewFakeContext(tt.client, &vzapi.Verrazzano{}, nil, false)

			kvs, err := appendResourceRequestOverrides(ctx, &tt.request, []bom.KeyValue{})

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
			assert.Equal(t, tt.expectOverrides, kvs)
		})
	}
}
