// Copyright (c) 2021, 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package vzconfig

import (
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
)

// IsPrometheusEnabled - Returns false only if explicitly disabled in the CR
func IsPrometheusEnabled(vz *vzapi.Verrazzano) bool {
	if vz != nil && vz.Spec.Components.Prometheus != nil && vz.Spec.Components.Prometheus.Enabled != nil {
		return *vz.Spec.Components.Prometheus.Enabled
	}
	return true
}

// IsKibanaEnabled - Returns false only if explicitly disabled in the CR
func IsKibanaEnabled(vz *vzapi.Verrazzano) bool {
	if vz != nil && vz.Spec.Components.Kibana != nil && vz.Spec.Components.Kibana.Enabled != nil {
		return *vz.Spec.Components.Kibana.Enabled
	}
	return true
}

// IsNGINXEnabled - Returns false only if explicitly disabled in the CR
func IsNGINXEnabled(vz *vzapi.Verrazzano) bool {
	if vz != nil && vz.Spec.Components.Ingress != nil && vz.Spec.Components.Ingress.Enabled != nil {
		return *vz.Spec.Components.Ingress.Enabled
	}
	return true
}

// IsIstioEnabled - Returns false only if explicitly disabled in the CR
func IsIstioEnabled(vz *vzapi.Verrazzano) bool {
	if vz != nil && vz.Spec.Components.Istio != nil && vz.Spec.Components.Istio.Enabled != nil {
		return *vz.Spec.Components.Istio.Enabled
	}
	return true
}

// IsCertManagerEnabled - Returns false only if CertManager is explictly disabled by the user
func IsCertManagerEnabled(vz *vzapi.Verrazzano) bool {
	if vz != nil && vz.Spec.Components.CertManager != nil && vz.Spec.Components.CertManager.Enabled != nil {
		return *vz.Spec.Components.CertManager.Enabled
	}
	return true
}

// IsKialiEnabled - Returns false only if explicitly disabled in the CR
func IsKialiEnabled(vz *vzapi.Verrazzano) bool {
	if vz != nil && vz.Spec.Components.Kiali != nil && vz.Spec.Components.Kiali.Enabled != nil {
		return *vz.Spec.Components.Kiali.Enabled
	}
	return true
}

// IsElasticsearchEnabled - Returns false only if explicitly disabled in the CR
func IsElasticsearchEnabled(vz *vzapi.Verrazzano) bool {
	if vz != nil && vz.Spec.Components.Elasticsearch != nil && vz.Spec.Components.Elasticsearch.Enabled != nil {
		return *vz.Spec.Components.Elasticsearch.Enabled
	}
	return true
}

// IsGrafanaEnabled - Returns false only if explicitly disabled in the CR
func IsGrafanaEnabled(vz *vzapi.Verrazzano) bool {
	if vz != nil && vz.Spec.Components.Grafana != nil && vz.Spec.Components.Grafana.Enabled != nil {
		return *vz.Spec.Components.Grafana.Enabled
	}
	return true
}

// IsFluentdEnabled - Returns false only if explicitly disabled in the CR
func IsFluentdEnabled(vz *vzapi.Verrazzano) bool {
	if vz != nil && vz.Spec.Components.Fluentd != nil && vz.Spec.Components.Fluentd.Enabled != nil {
		return *vz.Spec.Components.Fluentd.Enabled
	}
	return true
}

// IsConsoleEnabled - Returns false only if explicitly disabled in the CR
func IsConsoleEnabled(vz *vzapi.Verrazzano) bool {
	if vz != nil && vz.Spec.Components.Console != nil && vz.Spec.Components.Console.Enabled != nil {
		return *vz.Spec.Components.Console.Enabled
	}
	return true
}

// IsKeycloakEnabled - Returns false only if explicitly disabled in the CR
func IsKeycloakEnabled(vz *vzapi.Verrazzano) bool {
	if vz != nil && vz.Spec.Components.Keycloak != nil && vz.Spec.Components.Keycloak.Enabled != nil {
		return *vz.Spec.Components.Keycloak.Enabled
	}
	return true
}

// IsRancherEnabled - Returns false only if explicitly disabled in the CR
func IsRancherEnabled(vz *vzapi.Verrazzano) bool {
	if vz != nil && vz.Spec.Components.Rancher != nil && vz.Spec.Components.Rancher.Enabled != nil {
		return *vz.Spec.Components.Rancher.Enabled
	}
	return true
}

// IsExternalDNSEnabled Indicates if the external-dns service is expected to be deployed, true if OCI DNS is configured
func IsExternalDNSEnabled(vz *vzapi.Verrazzano) bool {
	if vz != nil && vz.Spec.Components.DNS != nil && vz.Spec.Components.DNS.OCI != nil {
		return true
	}
	return false
}

// IsVMOEnabled - Returns false if all VMO components are disabled
func IsVMOEnabled(vz *vzapi.Verrazzano) bool {
	return IsPrometheusEnabled(vz) || IsKibanaEnabled(vz) || IsElasticsearchEnabled(vz) || IsGrafanaEnabled(vz)
}

// IsPrometheusOperatorEnabled returns false only if the Prometheus Operator is explicitly disabled in the CR
func IsPrometheusOperatorEnabled(vz *vzapi.Verrazzano) bool {
	if vz != nil && vz.Spec.Components.PrometheusOperator != nil && vz.Spec.Components.PrometheusOperator.Enabled != nil {
		return *vz.Spec.Components.PrometheusOperator.Enabled
	}
	return false
}

// IsPrometheusAdapterEnabled returns false only if the Prometheus Adapter is explicitly disabled in the CR
func IsPrometheusAdapterEnabled(vz *vzapi.Verrazzano) bool {
	if vz != nil && vz.Spec.Components.PrometheusAdapter != nil && vz.Spec.Components.PrometheusAdapter.Enabled != nil {
		return *vz.Spec.Components.PrometheusAdapter.Enabled
	}
	return false
}

// IsKubeStateMetricsEnabled returns false only if Kube State Metrics is explicitly disabled in the CR
func IsKubeStateMetricsEnabled(vz *vzapi.Verrazzano) bool {
	if vz != nil && vz.Spec.Components.KubeStateMetrics != nil && vz.Spec.Components.KubeStateMetrics.Enabled != nil {
		return *vz.Spec.Components.KubeStateMetrics.Enabled
	}
	return false
}

// IsPrometheusPushgatewayEnabled returns false only if the Prometheus Pushgateway is explicitly disabled in the CR
func IsPrometheusPushgatewayEnabled(vz *vzapi.Verrazzano) bool {
	if vz != nil && vz.Spec.Components.PrometheusPushgateway != nil && vz.Spec.Components.PrometheusPushgateway.Enabled != nil {
		return *vz.Spec.Components.PrometheusPushgateway.Enabled
	}
	return false
}

// IsPrometheusNodeExporterEnabled returns false only if the Prometheus Node-Exporter is explicitly disabled in the CR
func IsPrometheusNodeExporterEnabled(vz *vzapi.Verrazzano) bool {
	if vz != nil && vz.Spec.Components.PrometheusNodeExporter != nil && vz.Spec.Components.PrometheusNodeExporter.Enabled != nil {
		return *vz.Spec.Components.PrometheusNodeExporter.Enabled
	}
	return false
}
