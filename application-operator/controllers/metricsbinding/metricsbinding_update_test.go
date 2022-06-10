package metricsbinding

import (
	"context"
	"testing"

	promoperapi "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	asserts "github.com/stretchr/testify/assert"
	vzapi "github.com/verrazzano/verrazzano/application-operator/apis/app/v1alpha1"
	"github.com/verrazzano/verrazzano/application-operator/constants"
	vzconst "github.com/verrazzano/verrazzano/pkg/constants"
	"github.com/verrazzano/verrazzano/pkg/log/vzlog"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

var metricsTemplate = vzapi.MetricsTemplate{
	TypeMeta: metav1.TypeMeta{
		Kind:       vzconst.MetricsTemplateKind,
		APIVersion: vzconst.MetricsTemplateAPIVersion,
	},
	ObjectMeta: metav1.ObjectMeta{
		Namespace: testMetricsTemplateNamespace,
		Name:      testMetricsTemplateName,
	},
}

var metricsBinding = vzapi.MetricsBinding{
	ObjectMeta: metav1.ObjectMeta{
		Namespace: testMetricsBindingNamespace,
		Name:      testMetricsBindingName,
	},
	Spec: vzapi.MetricsBindingSpec{
		MetricsTemplate: vzapi.NamespaceName{
			Namespace: testMetricsTemplateNamespace,
			Name:      testMetricsTemplateName,
		},
		PrometheusConfigMap: vzapi.NamespaceName{
			Namespace: constants.VerrazzanoSystemNamespace,
			Name:      testConfigMapName,
		},
		Workload: vzapi.Workload{
			Name: testDeploymentName,
			TypeMeta: metav1.TypeMeta{
				Kind:       vzconst.DeploymentWorkloadKind,
				APIVersion: deploymentGroup + "/" + deploymentVersion,
			},
		},
	},
}

// TestGetMetricsTemplate tests the retrieval process of the metrics template
// GIVEN a metrics binding
// WHEN the function receives the binding
// THEN return the metrics template without error
func TestGetMetricsTemplate(t *testing.T) {
	assert := asserts.New(t)

	scheme := runtime.NewScheme()
	vzapi.AddToScheme(scheme)
	c := fake.NewClientBuilder().WithScheme(scheme).WithRuntimeObjects(&metricsTemplate).Build()

	localMetricsBinding := metricsBinding.DeepCopy()

	log := vzlog.DefaultLogger()
	r := newReconciler(c)
	template, err := r.getMetricsTemplate(context.Background(), localMetricsBinding, log)
	assert.NoError(err, "Expected no error getting the MetricsTemplate from the MetricsBinding")
	assert.NotNil(template)
}

// TestGetMetricsTemplate tests the retrieval process of the metrics template
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

	plainNs := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: testMetricsBindingNamespace,
		},
	}
	labeledNs := plainNs.DeepCopy()
	labeledNs.Labels = map[string]string{constants.LabelIstioInjection: "enabled"}

	plainWorkload := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: testMetricsBindingNamespace,
			Name:      testDeploymentName,
		},
	}
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
				&metricsTemplate,
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
