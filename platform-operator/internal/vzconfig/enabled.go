// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
package vzconfig

import (
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
)

//IsPrometheusEnabled - Returns false only if explicitly disabled in the CR
func IsPrometheusEnabled(vz *vzapi.Verrazzano) bool {
	if vz != nil && vz.Spec.Components.Prometheus != nil && vz.Spec.Components.Prometheus.Enabled != nil {
		return *vz.Spec.Components.Prometheus.Enabled
	}
	return true
}

//IsKibanaEnabled - Returns false only if explicitly disabled in the CR
func IsKibanaEnabled(vz *vzapi.Verrazzano) bool {
	if vz != nil && vz.Spec.Components.Kibana != nil && vz.Spec.Components.Kibana.Enabled != nil {
		return *vz.Spec.Components.Kibana.Enabled
	}
	return true
}

//IsNGINXEnabled - Always true, for now
func IsNGINXEnabled(_ *vzapi.Verrazzano) bool {
	return true
}

//IsIstioEnabled - Always true, for now
func IsIstioEnabled(cr *vzapi.Verrazzano) bool {
	return true
}

//IsKialiEnabled - Returns false only if explicitly disabled in the CR
func IsKialiEnabled(vz *vzapi.Verrazzano) bool {
	if vz != nil && vz.Spec.Components.Kiali != nil && vz.Spec.Components.Kiali.Enabled != nil {
		return *vz.Spec.Components.Kiali.Enabled
	}
	return true
}

//IsElasticsearchEnabled - Returns false only if explicitly disabled in the CR
func IsElasticsearchEnabled(vz *vzapi.Verrazzano) bool {
	if vz != nil && vz.Spec.Components.Elasticsearch != nil && vz.Spec.Components.Elasticsearch.Enabled != nil {
		return *vz.Spec.Components.Elasticsearch.Enabled
	}
	return true
}

//IsGrafanaEnabled - Returns false only if explicitly disabled in the CR
func IsGrafanaEnabled(vz *vzapi.Verrazzano) bool {
	if vz != nil && vz.Spec.Components.Grafana != nil && vz.Spec.Components.Grafana.Enabled != nil {
		return *vz.Spec.Components.Grafana.Enabled
	}
	return true
}

//IsFluentdEnabled - Returns false only if explicitly disabled in the CR
func IsFluentdEnabled(vz *vzapi.Verrazzano) bool {
	if vz != nil && vz.Spec.Components.Fluentd != nil && vz.Spec.Components.Fluentd.Enabled != nil {
		return *vz.Spec.Components.Fluentd.Enabled
	}
	return true
}

//IsConsoleEnabled - Returns false only if explicitly disabled in the CR
func IsConsoleEnabled(vz *vzapi.Verrazzano) bool {
	if vz != nil && vz.Spec.Components.Console != nil && vz.Spec.Components.Console.Enabled != nil {
		return *vz.Spec.Components.Console.Enabled
	}
	return true
}

//IsKeycloakEnabled - Returns false only if explicitly disabled in the CR
func IsKeycloakEnabled(vz *vzapi.Verrazzano) bool {
	if vz != nil && vz.Spec.Components.Keycloak != nil && vz.Spec.Components.Keycloak.Enabled != nil {
		return *vz.Spec.Components.Keycloak.Enabled
	}
	return true
}

//IsRancherEnabled - Returns false only if explicitly disabled in the CR
func IsRancherEnabled(vz *vzapi.Verrazzano) bool {
	if vz != nil && vz.Spec.Components.Rancher != nil && vz.Spec.Components.Rancher.Enabled != nil {
		return *vz.Spec.Components.Rancher.Enabled
	}
	return true
}
