// Copyright (c) 2021, 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package metricstemplate

import (
	"fmt"

	"github.com/Jeffail/gabs/v2"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/yaml"
)

const (
	prometheusConfigKey          = "prometheus.yml"
	prometheusScrapeConfigsLabel = "scrape_configs"
	prometheusJobNameLabel       = "job_name"

	configMapKind       = "ConfigMap"
	configMapAPIVersion = "v1"

	metricsTemplateKind       = "MetricsTemplate"
	metricsTemplateAPIVersion = "app.verrazzano.io/v1alpha1"

	finalizerName = "metricstemplate.finalizers.verrazzano.io/finalizer"
)

// Creates a job name in the format <namespace>_<name>_<uid>
func createJobName(namespacedName types.NamespacedName, resourceUID types.UID) string {
	return fmt.Sprintf("%s_%s_%s", namespacedName.Namespace, namespacedName.Name, resourceUID)
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
	return "job_name: " + jobName + "\n"
}
