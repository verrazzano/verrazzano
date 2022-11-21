// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package metricsbinding

import (
	"context"
	"strings"
	"testing"

	promoperapi "github.com/prometheus-wls/prometheus-wls/pkg/apis/monitoring/v1"
	asserts "github.com/stretchr/testify/assert"
	vzapi "github.com/verrazzano/verrazzano/application-operator/apis/app/v1alpha1"
	"github.com/verrazzano/verrazzano/application-operator/constants"
	vzconst "github.com/verrazzano/verrazzano/pkg/constants"
	"github.com/verrazzano/verrazzano/pkg/log/vzlog"
	"github.com/verrazzano/verrazzano/pkg/metricsutils"
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
			assert.Equal("local", serviceMonitor.Spec.Endpoints[0].RelabelConfigs[0].Replacement)
			if _, ok := tt.namespace.Labels[constants.LabelIstioInjection]; ok {
				assert.Equal("https", serviceMonitor.Spec.Endpoints[0].Scheme)
			} else {
				assert.Equal("http", serviceMonitor.Spec.Endpoints[0].Scheme)
			}
		})
	}
}

// TestHandleCustomMetricsTemplate tests the custom metrics path implementation
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
	testFileCMEmpty, err := getConfigMapFromTestFile(true)
	assert.NoError(err)
	testFileCMFilled, err := getConfigMapFromTestFile(false)
	assert.NoError(err)
	testFileCMOtherScrapeConfigs, err := readConfigMapData("testdata/cmDataHasOtherScrapeConfigs.yaml")
	assert.NoError(err)
	testFileSec, err := getSecretFromTestFile(true)
	assert.NoError(err)

	tests := []struct {
		name               string
		workload           *appsv1.Deployment
		namespace          *corev1.Namespace
		configMap          *corev1.ConfigMap
		expectConfigMapAdd bool
		secret             *corev1.Secret
		expectError        bool
	}{
		{
			name:               "test configmap empty",
			workload:           labeledWorkload,
			namespace:          labeledNs,
			configMap:          testFileCMEmpty,
			expectError:        false,
			expectConfigMapAdd: true,
		},
		{
			name:               "test configmap with other scrape configs",
			workload:           labeledWorkload,
			namespace:          labeledNs,
			configMap:          testFileCMOtherScrapeConfigs,
			expectError:        false,
			expectConfigMapAdd: true,
		},
		{
			name:               "test configmap filled",
			workload:           labeledWorkload,
			namespace:          labeledNs,
			configMap:          testFileCMFilled,
			expectError:        false,
			expectConfigMapAdd: false,
		},
		{
			name:        "test secret",
			workload:    labeledWorkload,
			namespace:   labeledNs,
			secret:      testFileSec,
			expectError: false,
		},
		{
			name:               "test configmap no Istio",
			workload:           labeledWorkload,
			namespace:          plainNs,
			configMap:          testFileCMEmpty,
			expectError:        false,
			expectConfigMapAdd: true,
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
			configMapNumScrapeConfigs := 0
			if tt.configMap != nil {
				c = c.WithRuntimeObjects(tt.configMap)
				parsedConfigMap, err := getConfigData(tt.configMap)
				assert.NoError(err, "Could not parse test config map")
				configMapNumScrapeConfigs = len(parsedConfigMap.Search(prometheusScrapeConfigsLabel).Children())
			}
			if tt.secret != nil {
				c = c.WithRuntimeObjects(tt.secret)
				localMetricsBinding.Spec.PrometheusConfigMap = vzapi.NamespaceName{}
				localMetricsBinding.Spec.PrometheusConfigSecret = vzapi.SecretKey{
					Namespace: vzconst.PrometheusOperatorNamespace,
					Name:      vzconst.PromAdditionalScrapeConfigsSecretName,
					Key:       vzconst.PromAdditionalScrapeConfigsSecretKey,
				}
			}

			client := c.Build()

			log := vzlog.DefaultLogger()
			r := newReconciler(client)
			err = r.handleCustomMetricsTemplate(context.TODO(), localMetricsBinding, log)
			if tt.expectError {
				assert.Error(err, "Expected error handling the custom MetricsTemplate")
				return
			}
			assert.NoError(err, "Expected no error handling the custom MetricsTemplate")

			if tt.configMap != nil {
				var newCM corev1.ConfigMap
				err := client.Get(context.TODO(), types.NamespacedName{Namespace: vzconst.VerrazzanoSystemNamespace, Name: testConfigMapName}, &newCM)
				assert.NoError(err)
				assert.True(strings.Contains(newCM.Data[prometheusConfigKey], createJobName(localMetricsBinding)))
				parsedPrometheusConfig, err := getConfigData(&newCM)
				assert.NoError(err)
				newScrapeConfigs := parsedPrometheusConfig.Search(prometheusScrapeConfigsLabel)
				assert.NotNil(newScrapeConfigs)
				if tt.expectConfigMapAdd {
					assert.Equal(configMapNumScrapeConfigs+1, len(newScrapeConfigs.Children()))
				} else {
					assert.Equal(configMapNumScrapeConfigs, len(newScrapeConfigs.Children()))
				}
				foundJob := metricsutils.FindScrapeJob(newScrapeConfigs, createJobName(localMetricsBinding))
				assert.NotNil(foundJob)
			}
			if tt.secret != nil {
				var newSecret corev1.Secret
				err := client.Get(context.TODO(), types.NamespacedName{Namespace: vzconst.PrometheusOperatorNamespace, Name: vzconst.PromAdditionalScrapeConfigsSecretName}, &newSecret)
				assert.NoError(err)
				assert.True(strings.Contains(string(newSecret.Data[vzconst.PromAdditionalScrapeConfigsSecretKey]), createJobName(localMetricsBinding)))
			}
		})
	}
}

// TestReconcileCreateOrUpdate tests the create or update process of the reconiler
// GIVEN a metrics binding
// WHEN the function receives the binding
// THEN the binding gets updated accordingly
func TestReconcileCreateOrUpdate(t *testing.T) {
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

	CMMetricsBinding := metricsBinding.DeepCopy()
	secMetricsBinding := metricsBinding.DeepCopy()
	secMetricsBinding.Spec.PrometheusConfigMap = vzapi.NamespaceName{}
	secMetricsBinding.Spec.PrometheusConfigSecret = vzapi.SecretKey{
		Namespace: vzconst.PrometheusOperatorNamespace,
		Name:      vzconst.PromAdditionalScrapeConfigsSecretName,
		Key:       vzconst.PromAdditionalScrapeConfigsSecretKey,
	}
	legacyBinding := metricsBinding.DeepCopy()
	legacyBinding.Spec.MetricsTemplate.Namespace = constants.LegacyDefaultMetricsTemplateNamespace
	legacyBinding.Spec.MetricsTemplate.Name = constants.LegacyDefaultMetricsTemplateName
	legacyBinding.Spec.PrometheusConfigMap.Namespace = vzconst.VerrazzanoSystemNamespace
	legacyBinding.Spec.PrometheusConfigMap.Name = vzconst.VmiPromConfigName

	legacyBindingCustomTemplateDefaultCM := metricsBinding.DeepCopy()
	legacyBindingCustomTemplateDefaultCM.Spec.PrometheusConfigMap.Namespace = vzconst.VerrazzanoSystemNamespace
	legacyBindingCustomTemplateDefaultCM.Spec.PrometheusConfigMap.Name = vzconst.VmiPromConfigName

	tests := []struct {
		name           string
		metricsBinding *vzapi.MetricsBinding
		workload       *appsv1.Deployment
		namespace      *corev1.Namespace
		configMap      *corev1.ConfigMap
		secret         *corev1.Secret
		requeue        bool
		expectError    bool
	}{
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
			name:           "test legacy",
			metricsBinding: legacyBinding,
			workload:       labeledWorkload,
			namespace:      labeledNs,
			secret:         testFileSec,
			requeue:        true,
			expectError:    false,
		},
		{
			name:           "test legacy with custom template default CM",
			metricsBinding: legacyBindingCustomTemplateDefaultCM,
			workload:       labeledWorkload,
			namespace:      labeledNs,
			secret:         testFileSec,
			requeue:        true,
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
			log := vzlog.DefaultLogger()
			r := newReconciler(client)
			reconcileMetricsBinding := tt.metricsBinding.DeepCopy()
			result, err := r.reconcileBindingCreateOrUpdate(context.TODO(), reconcileMetricsBinding, log)
			if tt.expectError {
				assert.Error(err, "Expected error reconciling the Metrics Binding")
				return
			}
			assert.NoError(err, "Expected no error reconciling the Metrics Binding")
			assert.Equal(tt.requeue, result.Requeue)

			if isLegacyDefaultMetricsBinding(tt.metricsBinding) {
				newMB := vzapi.MetricsBinding{}
				err := client.Get(context.TODO(), types.NamespacedName{Namespace: tt.metricsBinding.Namespace, Name: tt.metricsBinding.Name}, &newMB)
				assert.Error(err, "Expected not to find the Metrics Binding in the cluster")
				return
			}

			if isLegacyVmiPrometheusConfigMapName(tt.metricsBinding.Spec.PrometheusConfigMap) {
				newMB := vzapi.MetricsBinding{}
				err := client.Get(context.TODO(), types.NamespacedName{Namespace: tt.metricsBinding.Namespace, Name: tt.metricsBinding.Name}, &newMB)
				assert.NoError(err)

				// for legacy VMI config map in metrics binding, it should be updated to have
				// the additional scrape configs secret, and the config map should be removed from
				// the metrics binding
				assert.Empty(newMB.Spec.PrometheusConfigMap.Namespace)
				assert.Empty(newMB.Spec.PrometheusConfigMap.Name)
				assert.Equal(vzconst.PrometheusOperatorNamespace, newMB.Spec.PrometheusConfigSecret.Namespace)
				assert.Equal(vzconst.PromAdditionalScrapeConfigsSecretName, newMB.Spec.PrometheusConfigSecret.Name)
				assert.Equal(vzconst.PromAdditionalScrapeConfigsSecretKey, newMB.Spec.PrometheusConfigSecret.Key)

				updatedSecret := corev1.Secret{}
				err = client.Get(context.TODO(), types.NamespacedName{Namespace: vzconst.PrometheusOperatorNamespace, Name: vzconst.PromAdditionalScrapeConfigsSecretName}, &updatedSecret)
				assert.NoError(err)
				scrapeConfigs := updatedSecret.Data[vzconst.PromAdditionalScrapeConfigsSecretKey]
				assert.NotNil(scrapeConfigs, "Expected additional scrape config secret to contain the scrape config")
				assert.Contains(string(scrapeConfigs), tt.workload.Name)
			}
			newMB := vzapi.MetricsBinding{}
			err = client.Get(context.TODO(), types.NamespacedName{Namespace: tt.metricsBinding.Namespace, Name: tt.metricsBinding.Name}, &newMB)
			assert.NoError(err)
			assert.Equal(1, len(newMB.Finalizers))
			assert.Equal(finalizerName, newMB.Finalizers[0])
			assert.Equal(1, len(newMB.OwnerReferences))
			assert.Equal(tt.workload.Name, newMB.OwnerReferences[0].Name)
		})
	}
}
