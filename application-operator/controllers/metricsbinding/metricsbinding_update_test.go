package metricsbinding

import (
	"context"
	"strings"
	"testing"

	promoperapi "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	asserts "github.com/stretchr/testify/assert"
	vzapi "github.com/verrazzano/verrazzano/application-operator/apis/app/v1alpha1"
	"github.com/verrazzano/verrazzano/application-operator/constants"
	vzconst "github.com/verrazzano/verrazzano/pkg/constants"
	"github.com/verrazzano/verrazzano/pkg/log/vzlog"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

// TestGetMetricsTemplate tests the retrieval process of the metrics template
// GIVEN a metrics binding
// WHEN the function receives the binding
// THEN return the metrics template without error
func TestGetMetricsTemplate(t *testing.T) {
	assert := asserts.New(t)

	scheme := runtime.NewScheme()
	_ = vzapi.AddToScheme(scheme)
	c := fake.NewClientBuilder().WithScheme(scheme).WithRuntimeObjects(metricsTemplate).Build()

	localMetricsBinding := metricsBinding.DeepCopy()

	log := vzlog.DefaultLogger()
	r := newReconciler(c)
	template, err := r.getMetricsTemplate(context.Background(), localMetricsBinding, log)
	assert.NoError(err, "Expected no error getting the MetricsTemplate from the MetricsBinding")
	assert.NotNil(template)
}

// TestHandleDefaultMetricsTemplate tests the retrieval process of the metrics template
// GIVEN a metrics binding
// WHEN the function receives the binding
// THEN a scrape config gets generated for the target workload
func TestHandleDefaultMetricsTemplate(t *testing.T) {
	assert := asserts.New(t)

	scheme := runtime.NewScheme()
	_ = vzapi.AddToScheme(scheme)
	_ = corev1.AddToScheme(scheme)
	_ = appsv1.AddToScheme(scheme)
	_ = promoperapi.AddToScheme(scheme)

	labeledNs := plainNs.DeepCopy()
	labeledNs.Labels = map[string]string{constants.LabelIstioInjection: "enabled"}

	labeledWorkload := plainWorkload.DeepCopy()
	labeledWorkload.Labels = map[string]string{constants.MetricsWorkloadLabel: testDeploymentName}

	serviceMonitorNSN := types.NamespacedName{Namespace: testMetricsBindingNamespace, Name: testMetricsBindingName}

	tests := []struct {
		name        string
		workload    *appsv1.Deployment
		namespace   *corev1.Namespace
		expectError bool
	}{
		{
			name:        "test no workload",
			workload:    &appsv1.Deployment{},
			namespace:   labeledNs,
			expectError: true,
		},
		{
			name:        "test no namespace",
			workload:    labeledWorkload,
			namespace:   &corev1.Namespace{},
			expectError: true,
		},
		{
			name:        "test workload no label",
			workload:    plainWorkload,
			namespace:   labeledNs,
			expectError: true,
		},
		{
			name:        "test workload and namespace label",
			workload:    labeledWorkload,
			namespace:   labeledNs,
			expectError: false,
		},
		{
			name:        "test workload label only",
			workload:    labeledWorkload,
			namespace:   plainNs,
			expectError: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := fake.NewClientBuilder().WithScheme(scheme).WithRuntimeObjects([]runtime.Object{
				metricsTemplate,
				tt.workload,
				tt.namespace,
			}...).Build()

			localMetricsBinding := metricsBinding.DeepCopy()

			log := vzlog.DefaultLogger()
			r := newReconciler(c)
			err := r.handleDefaultMetricsTemplate(context.Background(), localMetricsBinding, log)
			if tt.expectError {
				assert.Error(err, "Expected error handling the default MetricsTemplate")
				return
			}
			assert.NoError(err, "Expected no error handling the default MetricsTemplate")

			// Get the service monitor for analysis
			serviceMonitor := promoperapi.ServiceMonitor{}
			err = c.Get(context.TODO(), serviceMonitorNSN, &serviceMonitor)
			assert.NoError(err, "Expected no error getting the Service Monitor")

			assert.Equal(1, len(serviceMonitor.Spec.Endpoints))
			assert.Contains(serviceMonitor.Spec.Endpoints[0].RelabelConfigs[1].SourceLabels, promoperapi.LabelName(workloadSourceLabel))
			if _, ok := tt.namespace.Labels[constants.LabelIstioInjection]; ok {
				assert.Equal("https", serviceMonitor.Spec.Endpoints[0].Scheme)
			} else {
				assert.Equal("http", serviceMonitor.Spec.Endpoints[0].Scheme)
			}
		})
	}
}

// TestHandleCustomMetricsTemplate tests the retrieval process of the metrics template
// GIVEN a metrics binding
// WHEN the function receives the binding
// THEN a scrape config gets generated for the target workload
func TestHandleCustomMetricsTemplate(t *testing.T) {
	assert := asserts.New(t)

	scheme := runtime.NewScheme()
	_ = vzapi.AddToScheme(scheme)
	_ = corev1.AddToScheme(scheme)
	_ = appsv1.AddToScheme(scheme)
	_ = promoperapi.AddToScheme(scheme)

	labeledNs := plainNs.DeepCopy()
	labeledNs.Labels = map[string]string{constants.LabelIstioInjection: "enabled"}

	labeledWorkload := plainWorkload.DeepCopy()
	labeledWorkload.Labels = map[string]string{constants.MetricsWorkloadLabel: testDeploymentName}

	populatedTemplate, err := getTemplateTestFile()
	assert.NoError(err)
	testFileCM, err := getConfigMapFromTestFile(true)
	assert.NoError(err)
	testFileSec, err := getSecretFromTestFile(true)
	assert.NoError(err)

	tests := []struct {
		name        string
		workload    *appsv1.Deployment
		namespace   *corev1.Namespace
		configMap   *corev1.ConfigMap
		secret      *corev1.Secret
		expectError bool
	}{
		{
			name:        "test configmap",
			workload:    labeledWorkload,
			namespace:   labeledNs,
			configMap:   testFileCM,
			expectError: false,
		},
		{
			name:        "test secret",
			workload:    labeledWorkload,
			namespace:   labeledNs,
			secret:      testFileSec,
			expectError: false,
		},
		{
			name:        "test configmap no Istio",
			workload:    labeledWorkload,
			namespace:   plainNs,
			configMap:   testFileCM,
			expectError: false,
		},
		{
			name:        "test secret no Istio",
			workload:    labeledWorkload,
			namespace:   plainNs,
			secret:      testFileSec,
			expectError: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := fake.NewClientBuilder().WithScheme(scheme).WithRuntimeObjects([]runtime.Object{
				populatedTemplate,
				tt.workload,
				tt.namespace,
			}...)

			localMetricsBinding := metricsBinding.DeepCopy()

			if tt.configMap != nil {
				c = c.WithRuntimeObjects(tt.configMap)
			}
			if tt.secret != nil {
				c = c.WithRuntimeObjects(tt.secret)
				localMetricsBinding.Spec.PrometheusConfigMap = vzapi.NamespaceName{}
				localMetricsBinding.Spec.PrometheusConfigSecret = vzapi.SecretKey{
					Namespace: vzconst.PrometheusOperatorNamespace,
					Name:      vzconst.PromAdditionalScrapeConfigsSecretName,
					Key:       prometheusConfigKey,
				}
			}

			client := c.Build()

			log := vzlog.DefaultLogger()
			r := newReconciler(client)
			err = r.handleCustomMetricsTemplate(context.Background(), localMetricsBinding, log)
			if tt.expectError {
				assert.Error(err, "Expected error handling the default MetricsTemplate")
				return
			}
			assert.NoError(err, "Expected no error handling the default MetricsTemplate")

			if tt.configMap != nil {
				var newCM corev1.ConfigMap
				err := client.Get(context.TODO(), types.NamespacedName{Namespace: vzconst.VerrazzanoSystemNamespace, Name: testConfigMapName}, &newCM)
				assert.NoError(err)
				assert.True(strings.Contains(newCM.Data[prometheusConfigKey], createJobName(localMetricsBinding)))
			}
			if tt.secret != nil {
				var newSecret corev1.Secret
				err := client.Get(context.TODO(), types.NamespacedName{Namespace: vzconst.PrometheusOperatorNamespace, Name: vzconst.PromAdditionalScrapeConfigsSecretName}, &newSecret)
				assert.NoError(err)
				assert.True(strings.Contains(string(newSecret.Data[prometheusConfigKey]), createJobName(localMetricsBinding)))
			}
		})
	}
}
