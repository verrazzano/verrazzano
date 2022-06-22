// Copyright (c) 2021, 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package metricsbinding

import (
	"fmt"
	"strings"

	"github.com/Jeffail/gabs/v2"
	vzapi "github.com/verrazzano/verrazzano/application-operator/apis/app/v1alpha1"
	"github.com/verrazzano/verrazzano/application-operator/constants"
	vzconst "github.com/verrazzano/verrazzano/pkg/constants"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	k8smetav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/yaml"
)

const (
	prometheusConfigKey          = "prometheus.yml"
	prometheusScrapeConfigsLabel = "scrape_configs"

	configMapKind   = "ConfigMap"
	k8sV1APIVersion = "v1"

	finalizerName = "metricsbinding.finalizers.verrazzano.io/finalizer"

	workloadSourceLabel = "__meta_kubernetes_pod_label_app_verrazzano_io_workload"
)

// Creates a job name in the format <namespace>_<name>_<kind>
func createJobName(metricsbinding *vzapi.MetricsBinding) string {
	reference := metricsbinding.Spec.Workload
	return fmt.Sprintf("%s_%s_%s_%s", metricsbinding.Namespace, reference.Name, strings.Replace(reference.TypeMeta.APIVersion, "/", "_", 1), reference.TypeMeta.Kind)
}

// returns a container of the Prometheus config data from the configmap
func getConfigData(configMap *v1.ConfigMap) (*gabs.Container, error) {
	if configMap.Data == nil {
		return nil, errors.NewNotFound(schema.GroupResource{
			Group:    "default",
			Resource: configMap.Kind,
		}, configMap.Name)
	}
	oldPromConfigData := configMap.Data[prometheusConfigKey]
	promConfigJSON, err := yaml.YAMLToJSON([]byte(oldPromConfigData))
	if err != nil {
		return nil, err
	}
	promConfig, err := gabs.ParseJSON(promConfigJSON)
	if err != nil {
		return nil, err
	}
	return promConfig, nil
}

// returns a container of the Prometheus config data from the given secret
func getConfigDataFromSecret(secret *v1.Secret, key string) (*gabs.Container, error) {
	if secret.Data == nil {
		return nil, errors.NewNotFound(schema.GroupResource{
			Group:    "default",
			Resource: secret.Kind,
		}, secret.Name)
	}
	oldPromConfigData := secret.Data[key]
	promConfigJSON, err := yaml.YAMLToJSON([]byte(oldPromConfigData))
	if err != nil {
		return nil, err
	}
	promConfig, err := gabs.ParseJSON(promConfigJSON)
	if err != nil {
		return nil, err
	}
	return promConfig, nil
}

// Formats job name as specified by the Prometheus config
func formatJobName(jobName string) string {
	return fmt.Sprintf("%s: %s\n", vzconst.PrometheusJobNameKey, jobName)
}

// getPromConfigMap returns the Prometheus ConfigMap given in the MetricsBinding
func getPromConfigMap(metricsBinding *vzapi.MetricsBinding) *v1.ConfigMap {
	targetConfigMap := metricsBinding.Spec.PrometheusConfigMap
	if targetConfigMap.Name == "" {
		return nil
	}
	return &v1.ConfigMap{
		TypeMeta: k8smetav1.TypeMeta{
			Kind:       configMapKind,
			APIVersion: k8sV1APIVersion,
		},
		ObjectMeta: k8smetav1.ObjectMeta{
			Name:      targetConfigMap.Name,
			Namespace: targetConfigMap.Namespace,
		},
	}
}

// getPromConfigSecret returns the Prometheus Config Secret given in the MetricsBinding, along with the key
func getPromConfigSecret(metricsBinding *vzapi.MetricsBinding) (*v1.Secret, string) {
	targetSecret := metricsBinding.Spec.PrometheusConfigSecret
	if targetSecret.Name == "" {
		return nil, ""
	}
	return &v1.Secret{
		TypeMeta: k8smetav1.TypeMeta{
			Kind:       vzconst.SecretKind,
			APIVersion: k8sV1APIVersion,
		},
		ObjectMeta: k8smetav1.ObjectMeta{
			Name:      targetSecret.Name,
			Namespace: targetSecret.Namespace,
		},
	}, targetSecret.Key
}

// isLegacyDefaultMetricsBinding determines whether the given binding uses the
// "default" metrics template used pre-Verrazzano 1.4 AND the legacy VMI system prometheus config map
func isLegacyDefaultMetricsBinding(metricsBinding *vzapi.MetricsBinding) bool {
	templateName := metricsBinding.Spec.MetricsTemplate
	configMapName := metricsBinding.Spec.PrometheusConfigMap
	return templateName.Namespace == constants.LegacyDefaultMetricsTemplateNamespace &&
		templateName.Name == constants.LegacyDefaultMetricsTemplateName &&
		isLegacyVmiPrometheusConfigMapName(configMapName)
}

// isLegacyVmiPrometheusConfigMapName returns true if the given NamespaceName is that of the legacy
// vmi system prometheus config map
func isLegacyVmiPrometheusConfigMapName(configMapName vzapi.NamespaceName) bool {
	return configMapName.Namespace == constants.VerrazzanoSystemNamespace &&
		configMapName.Name == vzconst.VmiPromConfigName
}
