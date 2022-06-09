package metrics

import (
	"context"
	"testing"

	asserts "github.com/stretchr/testify/assert"
	vzapi "github.com/verrazzano/verrazzano/application-operator/apis/app/v1alpha1"
	"github.com/verrazzano/verrazzano/pkg/constants"
	"github.com/verrazzano/verrazzano/pkg/log/vzlog"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

const (
	testConfigMapName            = "test-cm-name"
	testMetricsTemplateNamespace = "test-namespace"
	testMetricsTemplateName      = "test-template-name"
	testMetricsBindingNamespace  = "test-namespace"
	testMetricsBindingName       = "test-binding-name"
	testDeploymentName           = "test-deployment"
	deploymentGroup              = "apps"
	deploymentVersion            = "v1"
)

var metricsTemplate = vzapi.MetricsTemplate{
	TypeMeta: metav1.TypeMeta{
		Kind:       constants.MetricsTemplateKind,
		APIVersion: constants.MetricsTemplateAPIVersion,
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
				Kind:       constants.DeploymentWorkloadKind,
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
	template, err := GetMetricsTemplate(context.Background(), c, localMetricsBinding, log.GetZapLogger())
	assert.NoError(err, "Expected no error getting the MetricsTemplate from the MetricsBinding")
	assert.NotNil(template)
}
