// Copyright (c) 2021, 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package vzcr

import (
	"fmt"
	installv1alpha1 "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	installv1beta1 "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1beta1"
	"k8s.io/apimachinery/pkg/runtime"
)

// IsPrometheusEnabled - Returns false only if explicitly disabled in the CR
func IsPrometheusEnabled(cr runtime.Object) bool {
	if vzv1alpha1, ok := cr.(*installv1alpha1.Verrazzano); ok {
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
	if vzv1alpha1, ok := cr.(*installv1alpha1.Verrazzano); ok {
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
	if vzv1alpha1, ok := cr.(*installv1alpha1.Verrazzano); ok {
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
	if vzv1alpha1, ok := cr.(*installv1alpha1.Verrazzano); ok {
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

// IsIstioInjectionEnabled - Returns false if either:
//
//	Istio is explicitly disabled in the CR OR
//	Istio is enabled but injection is explicitly disabled in the CR
func IsIstioInjectionEnabled(cr runtime.Object) bool {
	if vzv1alpha1, ok := cr.(*installv1alpha1.Verrazzano); ok {
		if vzv1alpha1 != nil && vzv1alpha1.Spec.Components.Istio != nil && vzv1alpha1.Spec.Components.Istio.Enabled != nil {
			return *vzv1alpha1.Spec.Components.Istio.Enabled && (vzv1alpha1.Spec.Components.Istio.InjectionEnabled == nil || *vzv1alpha1.Spec.Components.Istio.InjectionEnabled)
		}
	} else if vzv1beta1, ok := cr.(*installv1beta1.Verrazzano); ok {
		if vzv1beta1 != nil && vzv1beta1.Spec.Components.Istio != nil && vzv1beta1.Spec.Components.Istio.Enabled != nil {
			return *vzv1beta1.Spec.Components.Istio.Enabled && (vzv1beta1.Spec.Components.Istio.InjectionEnabled == nil || *vzv1beta1.Spec.Components.Istio.InjectionEnabled)
		}
	}

	return true
}

// IsCAPIEnabled - Returns false only if CAPI is explicitly disabled by the user
func IsCAPIEnabled(cr runtime.Object) bool {
	if vzv1alpha1, ok := cr.(*installv1alpha1.Verrazzano); ok {
		if vzv1alpha1 != nil && vzv1alpha1.Spec.Components.CAPI != nil && vzv1alpha1.Spec.Components.CAPI.Enabled != nil {
			return *vzv1alpha1.Spec.Components.CAPI.Enabled
		}
	} else if vzv1beta1, ok := cr.(*installv1beta1.Verrazzano); ok {
		if vzv1beta1 != nil && vzv1beta1.Spec.Components.CAPI != nil && vzv1beta1.Spec.Components.CAPI.Enabled != nil {
			return *vzv1beta1.Spec.Components.CAPI.Enabled
		}
	}
	return true
}

// IsCertManagerEnabled - Returns false only if CertManager is explicitly disabled by the user
func IsCertManagerEnabled(cr runtime.Object) bool {
	if vzv1alpha1, ok := cr.(*installv1alpha1.Verrazzano); ok {
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

// IsClusterIssuerEnabled - Returns false only if the ClusterIssuerComponent is explicitly disabled
func IsClusterIssuerEnabled(cr runtime.Object) bool {
	if vzv1alpha1, ok := cr.(*installv1alpha1.Verrazzano); ok {
		if vzv1alpha1 != nil && vzv1alpha1.Spec.Components.ClusterIssuer != nil &&
			vzv1alpha1.Spec.Components.ClusterIssuer.Enabled != nil {
			return *vzv1alpha1.Spec.Components.ClusterIssuer.Enabled
		}
	} else if vzv1beta1, ok := cr.(*installv1beta1.Verrazzano); ok {
		if vzv1beta1 != nil && vzv1beta1.Spec.Components.ClusterIssuer != nil &&
			vzv1beta1.Spec.Components.ClusterIssuer.Enabled != nil {
			return *vzv1beta1.Spec.Components.ClusterIssuer.Enabled
		}
	}
	return true
}

// IsCertManagerWebhookOCIEnabled - Returns true IFF the ExternalCertManager component is explicitly enabled
func IsCertManagerWebhookOCIEnabled(cr runtime.Object) bool {
	if vzv1alpha1, ok := cr.(*installv1alpha1.Verrazzano); ok {
		if vzv1alpha1 != nil && vzv1alpha1.Spec.Components.CertManagerWebhookOCI != nil &&
			vzv1alpha1.Spec.Components.CertManagerWebhookOCI.Enabled != nil {
			return *vzv1alpha1.Spec.Components.CertManagerWebhookOCI.Enabled
		}
	} else if vzv1beta1, ok := cr.(*installv1beta1.Verrazzano); ok {
		if vzv1beta1 != nil && vzv1beta1.Spec.Components.CertManagerWebhookOCI != nil &&
			vzv1beta1.Spec.Components.CertManagerWebhookOCI.Enabled != nil {
			return *vzv1beta1.Spec.Components.CertManagerWebhookOCI.Enabled
		}
	}
	return false
}

// IsCertManagerWebhookOCIRequired - Returns true if the ExternalCertManager component is explicitly enabled, OR
// if all of the following is true:
// - ACME/LetsEncrypt certificates are configured
// - OCI DNS is enabled
// - the ClusterIssuerComponent is enabled
//
// This behavior is to allow backwards compatibility with earlier releases where the behavior was not implemented in
// a separate component, and was implicitly enabled by the other conditions
func IsCertManagerWebhookOCIRequired(cr runtime.Object) bool {
	isLetsEncryptConfig, _ := IsLetsEncryptConfig(cr)
	return IsCertManagerWebhookOCIEnabled(cr) || IsOCIDNSEnabled(cr) && isLetsEncryptConfig && IsClusterIssuerEnabled(cr)
}

// IsLetsEncryptConfig - Check if cert-type is LetsEncrypt
func IsLetsEncryptConfig(cr runtime.Object) (bool, error) {
	if cr == nil {
		return false, fmt.Errorf("Nil CR passed in")
	}
	if vzv1alpha1, ok := cr.(*installv1alpha1.Verrazzano); ok {
		componentSpec := vzv1alpha1.Spec.Components
		if componentSpec.ClusterIssuer == nil {
			return false, nil
		}
		return componentSpec.ClusterIssuer.IsLetsEncryptIssuer()
	} else if vzv1beta1, ok := cr.(*installv1beta1.Verrazzano); ok {
		componentSpec := vzv1beta1.Spec.Components
		if componentSpec.ClusterIssuer == nil {
			return false, nil
		}
		return componentSpec.ClusterIssuer.IsLetsEncryptIssuer()
	}
	return false, fmt.Errorf("Illegal configuration state, unable to resolve ClusterIssuerComponent type: %v", cr)
}

// IsCAConfig - Check if cert-type is CA, if not it is assumed to be Acme
func IsCAConfig(cr runtime.Object) (bool, error) {
	if cr == nil {
		return false, fmt.Errorf("Nil CR passed in")
	}
	if vzv1alpha1, ok := cr.(*installv1alpha1.Verrazzano); ok {
		componentSpec := vzv1alpha1.Spec.Components
		if componentSpec.ClusterIssuer == nil {
			return true, nil
		}
		return componentSpec.ClusterIssuer.IsCAIssuer()
	} else if vzv1beta1, ok := cr.(*installv1beta1.Verrazzano); ok {
		componentSpec := vzv1beta1.Spec.Components
		if componentSpec.ClusterIssuer == nil {
			return true, nil
		}
		return componentSpec.ClusterIssuer.IsCAIssuer()
	}
	return false, fmt.Errorf("Illegal configuration state, unable to resolve ClusterIssuerComponent type: %v", cr)
}

// IsKialiEnabled - Returns false only if explicitly disabled in the CR
func IsKialiEnabled(cr runtime.Object) bool {
	if vzv1alpha1, ok := cr.(*installv1alpha1.Verrazzano); ok {
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
	if vzv1alpha1, ok := cr.(*installv1alpha1.Verrazzano); ok {
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
	if vzv1alpha1, ok := cr.(*installv1alpha1.Verrazzano); ok {
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
	if vzv1alpha1, ok := cr.(*installv1alpha1.Verrazzano); ok {
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
	if vzv1alpha1, ok := cr.(*installv1alpha1.Verrazzano); ok {
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
	if vzv1alpha1, ok := cr.(*installv1alpha1.Verrazzano); ok {
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
	if vzv1alpha1, ok := cr.(*installv1alpha1.Verrazzano); ok {
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
	return IsOCIDNSEnabled(cr)
}

// IsOCIDNSEnabled Returns true if OCI DNS is configured
func IsOCIDNSEnabled(cr runtime.Object) bool {
	if vzv1alpha1, ok := cr.(*installv1alpha1.Verrazzano); ok {
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
	if vzv1alpha1, ok := cr.(*installv1alpha1.Verrazzano); ok {
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
	if vzv1alpha1, ok := cr.(*installv1alpha1.Verrazzano); ok {
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
	if vzv1alpha1, ok := cr.(*installv1alpha1.Verrazzano); ok {
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
	if vzv1alpha1, ok := cr.(*installv1alpha1.Verrazzano); ok {
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
	if vzv1alpha1, ok := cr.(*installv1alpha1.Verrazzano); ok {
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
	if vzv1alpha1, ok := cr.(*installv1alpha1.Verrazzano); ok {
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
	if vzv1alpha1, ok := cr.(*installv1alpha1.Verrazzano); ok {
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
	if vzv1alpha1, ok := cr.(*installv1alpha1.Verrazzano); ok {
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
	if vzv1alpha1, ok := cr.(*installv1alpha1.Verrazzano); ok {
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
	if vzv1alpha1, ok := cr.(*installv1alpha1.Verrazzano); ok {
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
	if vzv1alpha1, ok := cr.(*installv1alpha1.Verrazzano); ok {
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
	if vzv1alpha1, ok := cr.(*installv1alpha1.Verrazzano); ok {
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

// IsClusterOperatorEnabled returns false only if Cluster Operator is explicitly disabled in the CR
func IsClusterOperatorEnabled(cr runtime.Object) bool {
	if vzv1alpha1, ok := cr.(*installv1alpha1.Verrazzano); ok {
		if vzv1alpha1 != nil && vzv1alpha1.Spec.Components.ClusterOperator != nil && vzv1alpha1.Spec.Components.ClusterOperator.Enabled != nil {
			return *vzv1alpha1.Spec.Components.ClusterOperator.Enabled
		}
	} else if vzv1beta1, ok := cr.(*installv1beta1.Verrazzano); ok {
		if vzv1beta1 != nil && vzv1beta1.Spec.Components.ClusterOperator != nil && vzv1beta1.Spec.Components.ClusterOperator.Enabled != nil {
			return *vzv1beta1.Spec.Components.ClusterOperator.Enabled
		}
	}
	return true
}

// IsWebLogicOperatorEnabled returns false if WebLogicOperator is explicitly disabled in the CR
func IsWebLogicOperatorEnabled(cr runtime.Object) bool {
	if vzv1alpha1, ok := cr.(*installv1alpha1.Verrazzano); ok {
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
	if vzv1alpha1, ok := cr.(*installv1alpha1.Verrazzano); ok {
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
	if vzv1alpha1, ok := cr.(*installv1alpha1.Verrazzano); ok {
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
	if vzv1alpha1, ok := cr.(*installv1alpha1.Verrazzano); ok {
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

// IsArgoCDEnabled returns false unless ArgoCD is explicitly enabled in the CR
func IsArgoCDEnabled(cr runtime.Object) bool {
	if vzv1alpha1, ok := cr.(*installv1alpha1.Verrazzano); ok {
		if vzv1alpha1 != nil && vzv1alpha1.Spec.Components.ArgoCD != nil && vzv1alpha1.Spec.Components.ArgoCD.Enabled != nil {
			return *vzv1alpha1.Spec.Components.ArgoCD.Enabled
		}
	} else if vzv1beta1, ok := cr.(*installv1beta1.Verrazzano); ok {
		if vzv1beta1 != nil && vzv1beta1.Spec.Components.ArgoCD != nil && vzv1beta1.Spec.Components.ArgoCD.Enabled != nil {
			return *vzv1beta1.Spec.Components.ArgoCD.Enabled
		}
	}
	return false
}

// IsThanosEnabled returns true only if Thanos is explicitly enabled in the CR
func IsThanosEnabled(cr runtime.Object) bool {
	if vzv1alpha1, ok := cr.(*installv1alpha1.Verrazzano); ok {
		if vzv1alpha1 != nil && vzv1alpha1.Spec.Components.Thanos != nil && vzv1alpha1.Spec.Components.Thanos.Enabled != nil {
			return *vzv1alpha1.Spec.Components.Thanos.Enabled
		}
	} else if vzv1beta1, ok := cr.(*installv1beta1.Verrazzano); ok {
		if vzv1beta1 != nil && vzv1beta1.Spec.Components.Thanos != nil && vzv1beta1.Spec.Components.Thanos.Enabled != nil {
			return *vzv1beta1.Spec.Components.Thanos.Enabled
		}
	}
	return false
}

// IsComponentStatusEnabled checks if the component is enabled by looking at the component status State field
func IsComponentStatusEnabled(cr runtime.Object, componentName string) bool {
	if vzv1alpha1, ok := cr.(*installv1alpha1.Verrazzano); ok {
		if vzv1alpha1 != nil &&
			vzv1alpha1.Status.Components[componentName] != nil &&
			!(vzv1alpha1.Status.Components[componentName].State == installv1alpha1.CompStateDisabled ||
				vzv1alpha1.Status.Components[componentName].State == installv1alpha1.CompStateUninstalled ||
				vzv1alpha1.Status.Components[componentName].State == installv1alpha1.CompStateUninstalling) {
			return true
		}
	} else if vzv1beta1, ok := cr.(*installv1beta1.Verrazzano); ok {
		if vzv1beta1 != nil &&
			vzv1beta1.Status.Components[componentName] != nil &&
			!(vzv1beta1.Status.Components[componentName].State == installv1beta1.CompStateDisabled ||
				vzv1beta1.Status.Components[componentName].State == installv1beta1.CompStateUninstalled ||
				vzv1beta1.Status.Components[componentName].State == installv1beta1.CompStateUninstalling) {
			return true
		}
	}
	return false
}
