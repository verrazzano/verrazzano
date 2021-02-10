// Copyright (C) 2020, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package instance

import (
	"fmt"
	"github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"

	"strings"
)

// GetInstanceInfo returns a single instance identified by the secret Kubernetes UID.
func GetInstanceInfo(envName string, dnsSuffix string) *v1alpha1.InstanceInfo {
	vzURI := getVerrazzanoURI(envName, dnsSuffix)
	return &v1alpha1.InstanceInfo{
		Console:       getConsoleURL(vzURI),
		VzAPIURL:      deriveURL(vzURI, "api"),
		RancherURL:    deriveURL(vzURI, "rancher"),
		ElasticURL:    getElasticURL(vzURI),
		KibanaURL:     getKibanaURL(vzURI),
		GrafanaURL:    getGrafanaURL(vzURI),
		PrometheusURL: getPrometheusURL(vzURI),
		KeyCloakURL:   getKeyCloakURL(vzURI),
	}
}

func getVerrazzanoURI(name string, suffix string) string {
	return fmt.Sprintf("%s.%s", name, suffix)
}

// Derive the URL from the verrazzano URI by prefixing with the given URL segment
func deriveURL(verrazzanoURI string, component string) *string {
	if len(strings.TrimSpace(verrazzanoURI)) > 0 {
		url := "https://" + component + "." + verrazzanoURI
		return &url
	}
	return nil
}

// GetKeyCloakURL returns Keycloak URL
func getKeyCloakURL(verrazzanoURI string) *string {
	return deriveURL(verrazzanoURI, "keycloak")
}

// GetKibanaURL returns Kibana URL
func getKibanaURL(verrazzanoURI string) *string {
	return deriveURL(verrazzanoURI, "kibana.vmi.system")
}

// GetGrafanaURL returns Grafana URL
func getGrafanaURL(verrazzanoURI string) *string {
	return deriveURL(verrazzanoURI, "grafana.vmi.system")
}

// GetPrometheusURL returns Prometheus URL
func getPrometheusURL(verrazzanoURI string) *string {
	return deriveURL(verrazzanoURI, "prometheus.vmi.system")
}

// GetElasticURL returns Elasticsearch URL
func getElasticURL(verrazzanoURI string) *string {
	return deriveURL(verrazzanoURI, "elasticsearch.vmi.system")
}

// GetConsoleURL returns the Verrazzano Console URL
func getConsoleURL(verrazzanoURI string) *string {
	return deriveURL(verrazzanoURI, "verrazzano")
}
