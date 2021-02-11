// Copyright (C) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package instance

import (
	"github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"

	"strings"
)

// GetInstanceInfo returns the instance info for the local install.
func GetInstanceInfo(vzURI string) *v1alpha1.InstanceInfo {
	return &v1alpha1.InstanceInfo{
		Console:       deriveURL(vzURI, "verrazzano"),
		VzAPIURL:      deriveURL(vzURI, "api"),
		RancherURL:    deriveURL(vzURI, "rancher"),
		ElasticURL:    deriveURL(vzURI, "elasticsearch.vmi.system"),
		KibanaURL:     deriveURL(vzURI, "kibana.vmi.system"),
		GrafanaURL:    deriveURL(vzURI, "grafana.vmi.system"),
		PrometheusURL: deriveURL(vzURI, "prometheus.vmi.system"),
		KeyCloakURL:   deriveURL(vzURI, "keycloak"),
	}
}

// Derive the URL from the verrazzano URI by prefixing with the given URL segment
func deriveURL(verrazzanoURI string, component string) *string {
	if len(strings.TrimSpace(verrazzanoURI)) > 0 {
		url := "https://" + component + "." + verrazzanoURI
		return &url
	}
	return nil
}
