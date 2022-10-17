// Copyright (c) 2021, 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package vzconfig

import (
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	installv1beta1 "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1beta1"
	"k8s.io/apimachinery/pkg/runtime"
)

// IsPrometheusEnabled - Returns false only if explicitly disabled in the CR
func IsPrometheusEnabled(cr runtime.Object) bool {
	if vzv1alpha1, ok := cr.(*vzapi.Verrazzano); ok {
		if vzv1alpha1 != nil && vzv1alpha1.Spec.Components.Prometheus != nil && vzv1alpha1.Spec.Components.Prometheus.Enabled != nil {
			return *vzv1alpha1.Spec.Components.Prometheus.Enabled
		}
	} else if vzv1beta1, ok := cr.(*installv1beta1.Verrazzano); ok {
		if vzv1beta1 != nil && vzv1beta1.Spec.Components.Prometheus != nil && vzv1beta1.Spec.Components.Prometheus.Enabled != nil {
			return *vzv1beta1.Spec.Components.Prometheus.Enabled
		}
	}
	return true
}

// IsOpenSearchDashboardsEnabled - Returns false only if explicitly disabled in the CR
func IsOpenSearchDashboardsEnabled(cr runtime.Object) bool {
	if vzv1alpha1, ok := cr.(*vzapi.Verrazzano); ok {
		if vzv1alpha1 != nil && vzv1alpha1.Spec.Components.Kibana != nil && vzv1alpha1.Spec.Components.Kibana.Enabled != nil {
			return *vzv1alpha1.Spec.Components.Kibana.Enabled
		}
	} else if vzv1beta1, ok := cr.(*installv1beta1.Verrazzano); ok {
		if vzv1beta1 != nil && vzv1beta1.Spec.Components.OpenSearchDashboards != nil && vzv1beta1.Spec.Components.OpenSearchDashboards.Enabled != nil {
			return *vzv1beta1.Spec.Components.OpenSearchDashboards.Enabled
		}
	}
	return true
}

// IsNGINXEnabled - Returns false only if explicitly disabled in the CR
func IsNGINXEnabled(cr runtime.Object) bool {
	if vzv1alpha1, ok := cr.(*vzapi.Verrazzano); ok {
		if vzv1alpha1 != nil && vzv1alpha1.Spec.Components.Ingress != nil && vzv1alpha1.Spec.Components.Ingress.Enabled != nil {
			return *vzv1alpha1.Spec.Components.Ingress.Enabled
		}
	} else if vzv1beta1, ok := cr.(*installv1beta1.Verrazzano); ok {
		if vzv1beta1 != nil && vzv1beta1.Spec.Components.IngressNGINX != nil && vzv1beta1.Spec.Components.IngressNGINX.Enabled != nil {
			return *vzv1beta1.Spec.Components.IngressNGINX.Enabled
		}
	}
	return true
}

// IsIstioEnabled - Returns false only if explicitly disabled in the CR
func IsIstioEnabled(cr runtime.Object) bool {
	if vzv1alpha1, ok := cr.(*vzapi.Verrazzano); ok {
		if vzv1alpha1 != nil && vzv1alpha1.Spec.Components.Istio != nil && vzv1alpha1.Spec.Components.Istio.Enabled != nil {
			return *vzv1alpha1.Spec.Components.Istio.Enabled
		}
	} else if vzv1beta1, ok := cr.(*installv1beta1.Verrazzano); ok {
		if vzv1beta1 != nil && vzv1beta1.Spec.Components.Istio != nil && vzv1beta1.Spec.Components.Istio.Enabled != nil {
			return *vzv1beta1.Spec.Components.Istio.Enabled
		}
	}

	return true
}

// IsCertManagerEnabled - Returns false only if CertManager is explicitly disabled by the user
func IsCertManagerEnabled(cr runtime.Object) bool {
	if vzv1alpha1, ok := cr.(*vzapi.Verrazzano); ok {
		if vzv1alpha1 != nil && vzv1alpha1.Spec.Components.CertManager != nil && vzv1alpha1.Spec.Components.CertManager.Enabled != nil {
			return *vzv1alpha1.Spec.Components.CertManager.Enabled
		}
	} else if vzv1beta1, ok := cr.(*installv1beta1.Verrazzano); ok {
		if vzv1beta1 != nil && vzv1beta1.Spec.Components.CertManager != nil && vzv1beta1.Spec.Components.CertManager.Enabled != nil {
			return *vzv1beta1.Spec.Components.CertManager.Enabled
		}
	}
	return true
}

// IsKialiEnabled - Returns false only if explicitly disabled in the CR
func IsKialiEnabled(cr runtime.Object) bool {
	if vzv1alpha1, ok := cr.(*vzapi.Verrazzano); ok {
		if vzv1alpha1 != nil && vzv1alpha1.Spec.Components.Kiali != nil && vzv1alpha1.Spec.Components.Kiali.Enabled != nil {
			return *vzv1alpha1.Spec.Components.Kiali.Enabled
		}
	} else if vzv1beta1, ok := cr.(*installv1beta1.Verrazzano); ok {
		if vzv1beta1 != nil && vzv1beta1.Spec.Components.Kiali != nil && vzv1beta1.Spec.Components.Kiali.Enabled != nil {
			return *vzv1beta1.Spec.Components.Kiali.Enabled
		}
	}
	return true
}

// IsOpenSearchEnabled - Returns false only if explicitly disabled in the CR
func IsOpenSearchEnabled(cr runtime.Object) bool {
	if vzv1alpha1, ok := cr.(*vzapi.Verrazzano); ok {
		if vzv1alpha1 != nil && vzv1alpha1.Spec.Components.Elasticsearch != nil && vzv1alpha1.Spec.Components.Elasticsearch.Enabled != nil {
			return *vzv1alpha1.Spec.Components.Elasticsearch.Enabled
		}
	} else if vzv1beta1, ok := cr.(*installv1beta1.Verrazzano); ok {
		if vzv1beta1 != nil && vzv1beta1.Spec.Components.OpenSearch != nil && vzv1beta1.Spec.Components.OpenSearch.Enabled != nil {
			return *vzv1beta1.Spec.Components.OpenSearch.Enabled
		}
	}
	return true
}

// IsGrafanaEnabled - Returns false only if explicitly disabled in the CR
func IsGrafanaEnabled(cr runtime.Object) bool {
	if vzv1alpha1, ok := cr.(*vzapi.Verrazzano); ok {
		if vzv1alpha1 != nil && vzv1alpha1.Spec.Components.Grafana != nil && vzv1alpha1.Spec.Components.Grafana.Enabled != nil {
			return *vzv1alpha1.Spec.Components.Grafana.Enabled
		}
	} else if vzv1beta1, ok := cr.(*installv1beta1.Verrazzano); ok {
		if vzv1beta1 != nil && vzv1beta1.Spec.Components.Grafana != nil && vzv1beta1.Spec.Components.Grafana.Enabled != nil {
			return *vzv1beta1.Spec.Components.Grafana.Enabled
		}
	}
	return true
}

// IsFluentdEnabled - Returns false only if explicitly disabled in the CR
func IsFluentdEnabled(cr runtime.Object) bool {
	if vzv1alpha1, ok := cr.(*vzapi.Verrazzano); ok {
		if vzv1alpha1 != nil && vzv1alpha1.Spec.Components.Fluentd != nil && vzv1alpha1.Spec.Components.Fluentd.Enabled != nil {
			return *vzv1alpha1.Spec.Components.Fluentd.Enabled
		}
	} else if vzv1beta1, ok := cr.(*installv1beta1.Verrazzano); ok {
		if vzv1beta1 != nil && vzv1beta1.Spec.Components.Fluentd != nil && vzv1beta1.Spec.Components.Fluentd.Enabled != nil {
			return *vzv1beta1.Spec.Components.Fluentd.Enabled
		}
	}
	return true
}

// IsConsoleEnabled - Returns false only if explicitly disabled in the CR
func IsConsoleEnabled(cr runtime.Object) bool {
	if vzv1alpha1, ok := cr.(*vzapi.Verrazzano); ok {
		if vzv1alpha1 != nil && vzv1alpha1.Spec.Components.Console != nil && vzv1alpha1.Spec.Components.Console.Enabled != nil {
			return *vzv1alpha1.Spec.Components.Console.Enabled
		}
	} else if vzv1beta1, ok := cr.(*installv1beta1.Verrazzano); ok {
		if vzv1beta1 != nil && vzv1beta1.Spec.Components.Console != nil && vzv1beta1.Spec.Components.Console.Enabled != nil {
			return *vzv1beta1.Spec.Components.Console.Enabled
		}
	}
	return true
}

// IsKeycloakEnabled - Returns false only if explicitly disabled in the CR
func IsKeycloakEnabled(cr runtime.Object) bool {
	if vzv1alpha1, ok := cr.(*vzapi.Verrazzano); ok {
		if vzv1alpha1 != nil && vzv1alpha1.Spec.Components.Keycloak != nil && vzv1alpha1.Spec.Components.Keycloak.Enabled != nil {
			return *vzv1alpha1.Spec.Components.Keycloak.Enabled
		}
	} else if vzv1beta1, ok := cr.(*installv1beta1.Verrazzano); ok {
		if vzv1beta1 != nil && vzv1beta1.Spec.Components.Keycloak != nil && vzv1beta1.Spec.Components.Keycloak.Enabled != nil {
			return *vzv1beta1.Spec.Components.Keycloak.Enabled
		}
	}
	return true
}

// IsRancherEnabled - Returns false only if explicitly disabled in the CR
func IsRancherEnabled(cr runtime.Object) bool {
	if vzv1alpha1, ok := cr.(*vzapi.Verrazzano); ok {
		if vzv1alpha1 != nil && vzv1alpha1.Spec.Components.Rancher != nil && vzv1alpha1.Spec.Components.Rancher.Enabled != nil {
			return *vzv1alpha1.Spec.Components.Rancher.Enabled
		}
	} else if vzv1beta1, ok := cr.(*installv1beta1.Verrazzano); ok {
		if vzv1beta1 != nil && vzv1beta1.Spec.Components.Rancher != nil && vzv1beta1.Spec.Components.Rancher.Enabled != nil {
			return *vzv1beta1.Spec.Components.Rancher.Enabled
		}
	}
	return true
}

// IsExternalDNSEnabled Indicates if the external-dns service is expected to be deployed, true if OCI DNS is configured
func IsExternalDNSEnabled(cr runtime.Object) bool {
	if vzv1alpha1, ok := cr.(*vzapi.Verrazzano); ok {
		if vzv1alpha1 != nil && vzv1alpha1.Spec.Components.DNS != nil && vzv1alpha1.Spec.Components.DNS.OCI != nil {
			return true
		}
	} else if vzv1beta1, ok := cr.(*installv1beta1.Verrazzano); ok {
		if vzv1beta1 != nil && vzv1beta1.Spec.Components.DNS != nil && vzv1beta1.Spec.Components.DNS.OCI != nil {
			return true
		}
	}
	return false
}

// IsVMOEnabled - Returns false if all VMO components are disabled
func IsVMOEnabled(vz runtime.Object) bool {
	return IsPrometheusEnabled(vz) || IsOpenSearchDashboardsEnabled(vz) || IsOpenSearchEnabled(vz) || IsGrafanaEnabled(vz)
}

// IsPrometheusOperatorEnabled returns false only if the Prometheus Operator is explicitly disabled in the CR
func IsPrometheusOperatorEnabled(cr runtime.Object) bool {
	if vzv1alpha1, ok := cr.(*vzapi.Verrazzano); ok {
		if vzv1alpha1 != nil && vzv1alpha1.Spec.Components.PrometheusOperator != nil && vzv1alpha1.Spec.Components.PrometheusOperator.Enabled != nil {
			return *vzv1alpha1.Spec.Components.PrometheusOperator.Enabled
		}
	} else if vzv1beta1, ok := cr.(*installv1beta1.Verrazzano); ok {
		if vzv1beta1 != nil && vzv1beta1.Spec.Components.PrometheusOperator != nil && vzv1beta1.Spec.Components.PrometheusOperator.Enabled != nil {
			return *vzv1beta1.Spec.Components.PrometheusOperator.Enabled
		}
	}
	return true
}

// IsPrometheusAdapterEnabled returns true only if the Prometheus Adapter is explicitly enabled in the CR
func IsPrometheusAdapterEnabled(cr runtime.Object) bool {
	if vzv1alpha1, ok := cr.(*vzapi.Verrazzano); ok {
		if vzv1alpha1 != nil && vzv1alpha1.Spec.Components.PrometheusAdapter != nil && vzv1alpha1.Spec.Components.PrometheusAdapter.Enabled != nil {
			return *vzv1alpha1.Spec.Components.PrometheusAdapter.Enabled
		}
	} else if vzv1beta1, ok := cr.(*installv1beta1.Verrazzano); ok {
		if vzv1beta1 != nil && vzv1beta1.Spec.Components.PrometheusAdapter != nil && vzv1beta1.Spec.Components.PrometheusAdapter.Enabled != nil {
			return *vzv1beta1.Spec.Components.PrometheusAdapter.Enabled
		}
	}
	return false
}

// IsKubeStateMetricsEnabled returns true only if Kube State Metrics is explicitly enabled in the CR
func IsKubeStateMetricsEnabled(cr runtime.Object) bool {
	if vzv1alpha1, ok := cr.(*vzapi.Verrazzano); ok {
		if vzv1alpha1 != nil && vzv1alpha1.Spec.Components.KubeStateMetrics != nil && vzv1alpha1.Spec.Components.KubeStateMetrics.Enabled != nil {
			return *vzv1alpha1.Spec.Components.KubeStateMetrics.Enabled
		}
	} else if vzv1beta1, ok := cr.(*installv1beta1.Verrazzano); ok {
		return *vzv1beta1.Spec.Components.KubeStateMetrics.Enabled
	}
	return false
}

// IsPrometheusPushgatewayEnabled returns true only if the Prometheus Pushgateway is explicitly enabled in the CR
func IsPrometheusPushgatewayEnabled(cr runtime.Object) bool {
	if vzv1alpha1, ok := cr.(*vzapi.Verrazzano); ok {
		if vzv1alpha1 != nil && vzv1alpha1.Spec.Components.PrometheusPushgateway != nil && vzv1alpha1.Spec.Components.PrometheusPushgateway.Enabled != nil {
			return *vzv1alpha1.Spec.Components.PrometheusPushgateway.Enabled
		}
	} else if vzv1beta1, ok := cr.(*installv1beta1.Verrazzano); ok {
		if vzv1beta1 != nil && vzv1beta1.Spec.Components.PrometheusPushgateway != nil && vzv1beta1.Spec.Components.PrometheusPushgateway.Enabled != nil {
			return *vzv1beta1.Spec.Components.PrometheusPushgateway.Enabled
		}
	}
	return false
}

// IsPrometheusNodeExporterEnabled returns false only if the Prometheus Node-Exporter is explicitly disabled in the CR
func IsPrometheusNodeExporterEnabled(cr runtime.Object) bool {
	if vzv1alpha1, ok := cr.(*vzapi.Verrazzano); ok {
		if vzv1alpha1 != nil && vzv1alpha1.Spec.Components.PrometheusNodeExporter != nil && vzv1alpha1.Spec.Components.PrometheusNodeExporter.Enabled != nil {
			return *vzv1alpha1.Spec.Components.PrometheusNodeExporter.Enabled
		}
	} else if vzv1beta1, ok := cr.(*installv1beta1.Verrazzano); ok {
		if vzv1beta1 != nil && vzv1beta1.Spec.Components.PrometheusNodeExporter != nil && vzv1beta1.Spec.Components.PrometheusNodeExporter.Enabled != nil {
			return *vzv1beta1.Spec.Components.PrometheusNodeExporter.Enabled
		}
	}
	return true
}

// IsJaegerOperatorEnabled returns true only if the Jaeger Operator is explicitly enabled in the CR
func IsJaegerOperatorEnabled(cr runtime.Object) bool {
	if vzv1alpha1, ok := cr.(*vzapi.Verrazzano); ok {
		if vzv1alpha1 != nil && vzv1alpha1.Spec.Components.JaegerOperator != nil && vzv1alpha1.Spec.Components.JaegerOperator.Enabled != nil {
			return *vzv1alpha1.Spec.Components.JaegerOperator.Enabled
		}
	} else if vzv1beta1, ok := cr.(*installv1beta1.Verrazzano); ok {
		if vzv1beta1 != nil && vzv1beta1.Spec.Components.JaegerOperator != nil && vzv1beta1.Spec.Components.JaegerOperator.Enabled != nil {
			return *vzv1beta1.Spec.Components.JaegerOperator.Enabled
		}
	}
	return false
}

// IsAuthProxyEnabled returns false only if Auth Proxy is explicitly disabled in the CR
func IsAuthProxyEnabled(cr runtime.Object) bool {
	if vzv1alpha1, ok := cr.(*vzapi.Verrazzano); ok {
		if vzv1alpha1 != nil && vzv1alpha1.Spec.Components.AuthProxy != nil && vzv1alpha1.Spec.Components.AuthProxy.Enabled != nil {
			return *vzv1alpha1.Spec.Components.AuthProxy.Enabled
		}
	} else if vzv1beta1, ok := cr.(*installv1beta1.Verrazzano); ok {
		if vzv1beta1 != nil && vzv1beta1.Spec.Components.AuthProxy != nil && vzv1beta1.Spec.Components.AuthProxy.Enabled != nil {
			return *vzv1beta1.Spec.Components.AuthProxy.Enabled
		}
	}
	return true
}

// IsApplicationOperatorEnabled returns false only if Application Operator is explicitly disabled in the CR
func IsApplicationOperatorEnabled(cr runtime.Object) bool {
	if vzv1alpha1, ok := cr.(*vzapi.Verrazzano); ok {
		if vzv1alpha1 != nil && vzv1alpha1.Spec.Components.ApplicationOperator != nil && vzv1alpha1.Spec.Components.ApplicationOperator.Enabled != nil {
			return *vzv1alpha1.Spec.Components.ApplicationOperator.Enabled
		}
	} else if vzv1beta1, ok := cr.(*installv1beta1.Verrazzano); ok {
		if vzv1beta1 != nil && vzv1beta1.Spec.Components.ApplicationOperator != nil && vzv1beta1.Spec.Components.ApplicationOperator.Enabled != nil {
			return *vzv1beta1.Spec.Components.ApplicationOperator.Enabled
		}
	}
	return true
}

// IsVeleroEnabled returns false unless Velero is not explicitly enabled in the CR
func IsVeleroEnabled(cr runtime.Object) bool {
	if vzv1alpha1, ok := cr.(*vzapi.Verrazzano); ok {
		if vzv1alpha1 != nil && vzv1alpha1.Spec.Components.Velero != nil && vzv1alpha1.Spec.Components.Velero.Enabled != nil {
			return *vzv1alpha1.Spec.Components.Velero.Enabled
		}
	} else if vzv1beta1, ok := cr.(*installv1beta1.Verrazzano); ok {
		if vzv1beta1 != nil && vzv1beta1.Spec.Components.Velero != nil && vzv1beta1.Spec.Components.Velero.Enabled != nil {
			return *vzv1beta1.Spec.Components.Velero.Enabled
		}
	}
	return false
}

// IsMySQLOperatorEnabled returns false if MySqlOperator is explicitly disabled in the CR
func IsMySQLOperatorEnabled(cr runtime.Object) bool {
	if vzv1alpha1, ok := cr.(*vzapi.Verrazzano); ok {
		if vzv1alpha1 != nil && vzv1alpha1.Spec.Components.MySQLOperator != nil && vzv1alpha1.Spec.Components.MySQLOperator.Enabled != nil {
			return *vzv1alpha1.Spec.Components.MySQLOperator.Enabled
		}
	} else if vzv1beta1, ok := cr.(*installv1beta1.Verrazzano); ok {
		if vzv1beta1 != nil && vzv1beta1.Spec.Components.MySQLOperator != nil && vzv1beta1.Spec.Components.MySQLOperator.Enabled != nil {
			return *vzv1beta1.Spec.Components.MySQLOperator.Enabled
		}
	}
	return true
}

// IsOAMEnabled returns false if OAM is explicitly disabled in the CR
func IsOAMEnabled(cr runtime.Object) bool {
	if vzv1alpha1, ok := cr.(*vzapi.Verrazzano); ok {
		if vzv1alpha1 != nil && vzv1alpha1.Spec.Components.OAM != nil && vzv1alpha1.Spec.Components.OAM.Enabled != nil {
			return *vzv1alpha1.Spec.Components.OAM.Enabled
		}
	} else if vzv1beta1, ok := cr.(*installv1beta1.Verrazzano); ok {
		if vzv1beta1 != nil && vzv1beta1.Spec.Components.OAM != nil && vzv1beta1.Spec.Components.OAM.Enabled != nil {
			return *vzv1beta1.Spec.Components.OAM.Enabled
		}
	}
	return true
}

// IsVerrazzanoComponentEnabled returns false if Verrazzano is explicitly disabled in the CR
func IsVerrazzanoComponentEnabled(cr runtime.Object) bool {
	if vzv1alpha1, ok := cr.(*vzapi.Verrazzano); ok {
		if vzv1alpha1 != nil && vzv1alpha1.Spec.Components.Verrazzano != nil && vzv1alpha1.Spec.Components.Verrazzano.Enabled != nil {
			return *vzv1alpha1.Spec.Components.Verrazzano.Enabled
		}
	} else if vzv1beta1, ok := cr.(*installv1beta1.Verrazzano); ok {
		if vzv1beta1 != nil && vzv1beta1.Spec.Components.Verrazzano != nil && vzv1beta1.Spec.Components.Verrazzano.Enabled != nil {
			return *vzv1beta1.Spec.Components.Verrazzano.Enabled
		}
	}
	return true
}

// IsWebLogicOperatorEnabled returns false if WebLogicOperator is explicitly disabled in the CR
func IsWebLogicOperatorEnabled(cr runtime.Object) bool {
	if vzv1alpha1, ok := cr.(*vzapi.Verrazzano); ok {
		if vzv1alpha1 != nil && vzv1alpha1.Spec.Components.WebLogicOperator != nil && vzv1alpha1.Spec.Components.WebLogicOperator.Enabled != nil {
			return *vzv1alpha1.Spec.Components.WebLogicOperator.Enabled
		}
	} else if vzv1beta1, ok := cr.(*installv1beta1.Verrazzano); ok {
		if vzv1beta1 != nil && vzv1beta1.Spec.Components.WebLogicOperator != nil && vzv1beta1.Spec.Components.WebLogicOperator.Enabled != nil {
			return *vzv1beta1.Spec.Components.WebLogicOperator.Enabled
		}
	}
	return true
}

// IsCoherenceOperatorEnabled returns false if CoherenceOperator is explicitly disabled in the CR
func IsCoherenceOperatorEnabled(cr runtime.Object) bool {
	if vzv1alpha1, ok := cr.(*vzapi.Verrazzano); ok {
		if vzv1alpha1 != nil && vzv1alpha1.Spec.Components.CoherenceOperator != nil && vzv1alpha1.Spec.Components.CoherenceOperator.Enabled != nil {
			return *vzv1alpha1.Spec.Components.CoherenceOperator.Enabled
		}
	} else if vzv1beta1, ok := cr.(*installv1beta1.Verrazzano); ok {
		if vzv1beta1 != nil && vzv1beta1.Spec.Components.CoherenceOperator != nil && vzv1beta1.Spec.Components.CoherenceOperator.Enabled != nil {
			return *vzv1beta1.Spec.Components.CoherenceOperator.Enabled
		}
	}
	return true
}

// IsNodeExporterEnabled returns false if NodeExporter is explicitly disabled in the CR, or Prometheus is disabled in the CR
func IsNodeExporterEnabled(cr runtime.Object) bool {
	if vzv1alpha1, ok := cr.(*vzapi.Verrazzano); ok {
		if vzv1alpha1 != nil && vzv1alpha1.Spec.Components.PrometheusNodeExporter != nil && vzv1alpha1.Spec.Components.PrometheusNodeExporter.Enabled != nil {
			return *vzv1alpha1.Spec.Components.PrometheusNodeExporter.Enabled
		}
	} else if vzv1beta1, ok := cr.(*installv1beta1.Verrazzano); ok {
		if vzv1beta1 != nil && vzv1beta1.Spec.Components.PrometheusNodeExporter != nil && vzv1beta1.Spec.Components.PrometheusNodeExporter.Enabled != nil {
			return *vzv1beta1.Spec.Components.PrometheusNodeExporter.Enabled
		}
	}
	return IsPrometheusEnabled(cr)
}

// IsRancherBackupEnabled returns false unless RancherBackup is explicitly enabled in the CR
func IsRancherBackupEnabled(cr runtime.Object) bool {
	if vzv1alpha1, ok := cr.(*vzapi.Verrazzano); ok {
		if vzv1alpha1 != nil && vzv1alpha1.Spec.Components.RancherBackup != nil && vzv1alpha1.Spec.Components.RancherBackup.Enabled != nil {
			return *vzv1alpha1.Spec.Components.RancherBackup.Enabled
		}
	} else if vzv1beta1, ok := cr.(*installv1beta1.Verrazzano); ok {
		if vzv1beta1 != nil && vzv1beta1.Spec.Components.RancherBackup != nil && vzv1beta1.Spec.Components.RancherBackup.Enabled != nil {
			return *vzv1beta1.Spec.Components.RancherBackup.Enabled
		}
	}
	return false
}

// IsArgoCDEnabled returns false if ArgoCD is explicitly disabled in the CR
func IsArgoCDEnabled(cr runtime.Object) bool {
	if vzv1alpha1, ok := cr.(*vzapi.Verrazzano); ok {
		if vzv1alpha1 != nil && vzv1alpha1.Spec.Components.ArgoCD != nil && vzv1alpha1.Spec.Components.ArgoCD.Enabled != nil {
			return *vzv1alpha1.Spec.Components.PrometheusNodeExporter.Enabled
		}
	} else if vzv1beta1, ok := cr.(*installv1beta1.Verrazzano); ok {
		if vzv1beta1 != nil && vzv1beta1.Spec.Components.ArgoCD != nil && vzv1beta1.Spec.Components.ArgoCD.Enabled != nil {
			return *vzv1beta1.Spec.Components.PrometheusNodeExporter.Enabled
		}
	}
	return IsPrometheusEnabled(cr)
}
