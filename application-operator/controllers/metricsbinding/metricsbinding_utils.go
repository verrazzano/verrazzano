// Copyright (c) 2021, 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package metricsbinding

import (
	"fmt"
	"strings"

	"github.com/Jeffail/gabs/v2"
	vzapi "github.com/verrazzano/verrazzano/application-operator/apis/app/v1alpha1"
	"github.com/verrazzano/verrazzano/pkg/constants"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/yaml"
)

const (
	prometheusConfigKey          = "prometheus.yml"
	prometheusScrapeConfigsLabel = "scrape_configs"

	configMapKind       = "ConfigMap"
	configMapAPIVersion = "v1"

	metricsTemplateKind       = "MetricsTemplate"
	metricsTemplateAPIVersion = "app.verrazzano.io/v1alpha1"

	finalizerName = "metricsbinding.finalizers.verrazzano.io/finalizer"
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

// Formats job name as specified by the Prometheus config
func formatJobName(jobName string) string {
	return fmt.Sprintf("%s: %s\n", constants.PrometheusJobNameKey, jobName)
}
