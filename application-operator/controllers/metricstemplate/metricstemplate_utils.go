// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package metricstemplate

import (
	"fmt"
	"github.com/Jeffail/gabs/v2"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/yaml"
)

const (
	prometheusConfigKey          = "prometheus.yml"
	prometheusScrapeConfigsLabel = "scrape_configs"
	prometheusJobNameLabel       = "job_name"
)

// Creates a job name in the format <namespace>_<name>_<uid>
func createJobName(namespacedName types.NamespacedName, resourceUID types.UID) string {
	return fmt.Sprintf("%s_%s_%s",namespacedName.Namespace, namespacedName.Name, resourceUID)
}

// returns a container of the Prometheus config data from the configmap
func getConfigData(configMap *v1.ConfigMap) (*gabs.Container, error) {
	oldPromConfigData := configMap.Data[prometheusConfigKey]
	promConfigJson, err := yaml.YAMLToJSON([]byte(oldPromConfigData))
	if err != nil {
		return nil, err
	}
	promConfig, err := gabs.ParseJSON(promConfigJson)
	if err != nil {
		return nil, err
	}
	return promConfig, nil
}
