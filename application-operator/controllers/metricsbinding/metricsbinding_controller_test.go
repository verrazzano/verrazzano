// Copyright (c) 2021, 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package metricsbinding

import (
	"context"
	"fmt"
	"testing"
	"time"

	promoperapi "github.com/prometheus-wls/prometheus-wls/pkg/apis/monitoring/v1"
	asserts "github.com/stretchr/testify/assert"
	vzapi "github.com/verrazzano/verrazzano/application-operator/apis/app/v1alpha1"
	"github.com/verrazzano/verrazzano/application-operator/constants"
	vzconst "github.com/verrazzano/verrazzano/pkg/constants"
	k8sapps "k8s.io/api/apps/v1"
	k8score "k8s.io/api/core/v1"
	k8smeta "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

var metricsTemplate = &vzapi.MetricsTemplate{
	TypeMeta: k8smeta.TypeMeta{
		Kind:       vzconst.MetricsTemplateKind,
		APIVersion: vzconst.MetricsTemplateAPIVersion,
	},
	ObjectMeta: k8smeta.ObjectMeta{
		Namespace: testMetricsTemplateNamespace,
		Name:      testMetricsTemplateName,
	},
}

var trueValue = true
var metricsBinding = &vzapi.MetricsBinding{
	ObjectMeta: k8smeta.ObjectMeta{
		Namespace: testMetricsBindingNamespace,
		Name:      testMetricsBindingName,
		OwnerReferences: []k8smeta.OwnerReference{
			{
				APIVersion:         fmt.Sprintf("%s/%s", deploymentGroup, deploymentVersion),
				BlockOwnerDeletion: &trueValue,
				Controller:         &trueValue,
				Kind:               deploymentKind,
				Name:               testDeploymentName,
			},
		},
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
			TypeMeta: k8smeta.TypeMeta{
				Kind:       vzconst.DeploymentWorkloadKind,
				APIVersion: deploymentGroup + "/" + deploymentVersion,
			},
		},
	},
}

var plainWorkload = &k8sapps.Deployment{
	ObjectMeta: k8smeta.ObjectMeta{
		Namespace: testMetricsBindingNamespace,
		Name:      testDeploymentName,
		UID:       types.UID(testDeploymentUID),
	},
}

// The namespace has to contain labels for the template
var plainNs = &k8score.Namespace{
	ObjectMeta: k8smeta.ObjectMeta{
		Name: testMetricsBindingNamespace,
		Labels: map[string]string{
			"test": "test",
		},
	},
}

// TestReconcile tests the reconcile process of the Metrics Binding
// GIVEN a metrics binding
// WHEN the function receives the binding
// THEN the reconcile process occurs for updating or deleting the Metrics Binding
func TestReconcile(t *testing.T) {
	assert := asserts.New(t)

	scheme := runtime.NewScheme()
	_ = vzapi.AddToScheme(scheme)
	_ = k8score.AddToScheme(scheme)
	_ = k8sapps.AddToScheme(scheme)
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

	CMMetricsBinding := metricsBinding.DeepCopy()
	secMetricsBinding := metricsBinding.DeepCopy()
	secMetricsBinding.Spec.PrometheusConfigMap = vzapi.NamespaceName{}
	secMetricsBinding.Spec.PrometheusConfigSecret = vzapi.SecretKey{
		Namespace: vzconst.PrometheusOperatorNamespace,
		Name:      vzconst.PromAdditionalScrapeConfigsSecretName,
		Key:       prometheusConfigKey,
	}
	legacyBinding := metricsBinding.DeepCopy()
	legacyBinding.Spec.MetricsTemplate.Namespace = constants.LegacyDefaultMetricsTemplateNamespace
	legacyBinding.Spec.MetricsTemplate.Name = constants.LegacyDefaultMetricsTemplateName
	legacyBinding.Spec.PrometheusConfigMap.Namespace = vzconst.VerrazzanoSystemNamespace
	legacyBinding.Spec.PrometheusConfigMap.Name = vzconst.VmiPromConfigName
	deleteBinding := metricsBinding.DeepCopy()
	deleteBinding.DeletionTimestamp = &k8smeta.Time{Time: time.Now()}

	tests := []struct {
		name           string
		metricsBinding *vzapi.MetricsBinding
		workload       *k8sapps.Deployment
		namespace      *k8score.Namespace
		configMap      *k8score.ConfigMap
		secret         *k8score.Secret
		request        *reconcile.Request
		requeue        bool
		expectError    bool
	}{
		{
			name:           "test kube-system request",
			metricsBinding: CMMetricsBinding,
			workload:       labeledWorkload,
			namespace:      labeledNs,
			request:        &reconcile.Request{NamespacedName: types.NamespacedName{Namespace: vzconst.KubeSystem}},
			requeue:        false,
		},
		{
			name:           "test configmap",
			metricsBinding: CMMetricsBinding,
			workload:       labeledWorkload,
			namespace:      labeledNs,
			configMap:      testFileCM,
			requeue:        true,
			expectError:    false,
		},
		{
			name:           "test secret",
			metricsBinding: secMetricsBinding,
			workload:       labeledWorkload,
			namespace:      labeledNs,
			secret:         testFileSec,
			requeue:        true,
			expectError:    false,
		},
		{
			name:           "test delete",
			metricsBinding: deleteBinding,
			workload:       labeledWorkload,
			namespace:      labeledNs,
			configMap:      testFileCM,
			requeue:        false,
			expectError:    false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := fake.NewClientBuilder().WithScheme(scheme).WithRuntimeObjects([]runtime.Object{
				populatedTemplate,
				tt.workload,
				tt.namespace,
				tt.metricsBinding,
			}...)

			if tt.configMap != nil {
				c = c.WithRuntimeObjects(tt.configMap)
			}
			if tt.secret != nil {
				c = c.WithRuntimeObjects(tt.secret)
			}

			client := c.Build()
			r := newReconciler(client)
			if tt.request == nil {
				tt.request = &reconcile.Request{NamespacedName: types.NamespacedName{Namespace: tt.metricsBinding.Namespace, Name: tt.metricsBinding.Name}}
			}
			result, err := r.Reconcile(context.TODO(), *tt.request)
			if tt.expectError {
				assert.Error(err, "Expected error Reconciling the Metrics Binding")
				return
			}
			assert.NoError(err, "Expected no error Reconciling the Metrics Binding")
			assert.Equal(tt.requeue, result.Requeue)
		})
	}
}
