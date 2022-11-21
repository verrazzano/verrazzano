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
	vzconst "github.com/verrazzano/verrazzano/pkg/constants"
	"github.com/verrazzano/verrazzano/pkg/log/vzlog"
	"github.com/verrazzano/verrazzano/pkg/metricsutils"
	"github.com/verrazzano/verrazzano/platform-operator/constants"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

// TestReconcileBindingDelete tests the reconciliation for a deletion
// GIVEN an object and a request
// WHEN the reconciler processes the request
// THEN verify the process returns no error
func TestReconcileBindingDelete(t *testing.T) {
	assert := asserts.New(t)

	scheme := runtime.NewScheme()
	_ = vzapi.AddToScheme(scheme)
	_ = corev1.AddToScheme(scheme)
	_ = appsv1.AddToScheme(scheme)
	_ = promoperapi.AddToScheme(scheme)

	testFileCMEmpty, err := getConfigMapFromTestFile(true)
	assert.NoError(err)
	testFileCMFilled, err := getConfigMapFromTestFile(false)
	assert.NoError(err)
	testFileCMOtherScrapeConfigs, err := readConfigMapData("testdata/cmDataHasOtherScrapeConfigs.yaml")
	assert.NoError(err)
	testFileSec, err := getSecretFromTestFile(false)
	assert.NoError(err)

	tests := []struct {
		name                     string
		workload                 *appsv1.Deployment
		namespace                *corev1.Namespace
		configMap                *corev1.ConfigMap
		secret                   *corev1.Secret
		expectScrapeConfigRemove bool
	}{
		{
			name:                     "test configmap empty",
			workload:                 plainWorkload,
			namespace:                plainNs,
			configMap:                testFileCMEmpty,
			expectScrapeConfigRemove: false,
		},
		{
			name:                     "test configmap with other scrape configs",
			workload:                 plainWorkload,
			namespace:                plainNs,
			configMap:                testFileCMOtherScrapeConfigs,
			expectScrapeConfigRemove: false,
		},
		{
			name:                     "test configmap filled",
			workload:                 plainWorkload,
			namespace:                plainNs,
			configMap:                testFileCMFilled,
			expectScrapeConfigRemove: true,
		},
		{
			name:                     "test secret",
			workload:                 plainWorkload,
			namespace:                plainNs,
			secret:                   testFileSec,
			expectScrapeConfigRemove: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			localMetricsBinding := metricsBinding.DeepCopy()
			localMetricsBinding.OwnerReferences = []metav1.OwnerReference{
				{
					Name: tt.workload.Name,
				},
			}

			c := fake.NewClientBuilder().WithScheme(scheme).WithRuntimeObjects([]runtime.Object{
				metricsTemplate,
				tt.workload,
				tt.namespace,
			}...)
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
					Key:       prometheusConfigKey,
				}
			}
			client := c.Build()

			log := vzlog.DefaultLogger()
			r := newReconciler(client)
			_, err = r.reconcileBindingDelete(context.TODO(), localMetricsBinding, log)
			assert.NoError(err)

			// Get the configMap for analysis
			if tt.configMap != nil {
				var newCM corev1.ConfigMap
				err = client.Get(context.TODO(), types.NamespacedName{Namespace: constants.VerrazzanoSystemNamespace, Name: testConfigMapName}, &newCM)
				assert.NoError(err)
				jobName := createJobName(localMetricsBinding)
				assert.False(strings.Contains(newCM.Data[prometheusConfigKey], jobName))
				parsedPrometheusConfig, err := getConfigData(&newCM)
				assert.NoError(err)
				newScrapeConfigs := parsedPrometheusConfig.Search(prometheusScrapeConfigsLabel)
				jobIndex := metricsutils.FindScrapeJob(newScrapeConfigs, jobName)
				assert.Equal(-1, jobIndex)
				expectedNumScrapeConfigs := configMapNumScrapeConfigs
				if tt.expectScrapeConfigRemove {
					expectedNumScrapeConfigs = configMapNumScrapeConfigs - 1
				}
				assert.Equal(expectedNumScrapeConfigs, len(newScrapeConfigs.Children()))
			}
			if tt.secret != nil {
				var newSecret corev1.Secret
				err = client.Get(context.TODO(), types.NamespacedName{Namespace: vzconst.PrometheusOperatorNamespace, Name: vzconst.PromAdditionalScrapeConfigsSecretName}, &newSecret)
				assert.NoError(err)
				assert.False(strings.Contains(string(newSecret.Data[prometheusConfigKey]), createJobName(localMetricsBinding)))
			}
		})
	}
}
