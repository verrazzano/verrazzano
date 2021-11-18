// Copyright (c) 2020, 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
package vzconfig

import (
	"context"
	"fmt"
	globalconst "github.com/verrazzano/verrazzano/pkg/constants"
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	vpoconst "github.com/verrazzano/verrazzano/platform-operator/constants"
	"k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
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
	//if vz != nil && vz.Spec.Components.Ingress != nil && vz.Spec.Components.Ingress.Enabled != nil {
	//	return *vz.Spec.Components.Ingress.Enabled
	//}
	return true
}

//IsIstioEnabled - Always true, for now
func IsIstioEnabled(cr *vzapi.Verrazzano) bool {
	//if cr != nil && cr.Spec.Components.Istio != nil && cr.Spec.Components.Istio.Enabled != nil {
	//	return *cr.Spec.Components.Istio.Enabled
	//}
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

// GetEnvName Returns the configured environment name, or "default" if not specified in the configuration
func GetEnvName(vz *vzapi.Verrazzano) string {
	envName := vz.Spec.EnvironmentName
	if len(envName) == 0 {
		envName = "default"
	}
	return envName
}

// FindVolumeTemplate Find a named VolumeClaimTemplate in the list
func FindVolumeTemplate(templateName string, templates []vzapi.VolumeClaimSpecTemplate) (*v1.PersistentVolumeClaimSpec, bool) {
	for i, template := range templates {
		if templateName == template.Name {
			return &templates[i].Spec, true
		}
	}
	return nil, false
}

// IsExternalDNSEnabled Indicates if the external-dns service is expected to be deployed, true if OCI DNS is configured
func IsExternalDNSEnabled(vz *vzapi.Verrazzano) bool {
	if vz.Spec.Components.DNS != nil && vz.Spec.Components.DNS.OCI != nil {
		return true
	}
	return false
}

// GetWildcardDomain Get the wildcard domain from the Verrazzano config
func GetWildcardDomain(dnsConfig *vzapi.DNSComponent) string {
	wildcardDomain := "nip.io"
	if dnsConfig != nil && dnsConfig.Wildcard != nil && len(dnsConfig.Wildcard.Domain) > 0 {
		wildcardDomain = dnsConfig.Wildcard.Domain
	}
	return wildcardDomain
}

// GetDNSSuffix Returns the DNS suffix for the Verrazzano installation
// - port of install script function get_dns_suffix from config.sh
func GetDNSSuffix(client client.Client, vz *vzapi.Verrazzano) (string, error) {
	var dnsSuffix string
	dnsConfig := vz.Spec.Components.DNS
	if dnsConfig == nil || dnsConfig.Wildcard != nil {
		ingressIP, err := GetIngressIP(client, vz)
		if err != nil {
			return "", err
		}
		dnsSuffix = fmt.Sprintf("%s.%s", ingressIP, GetWildcardDomain(dnsConfig))
	} else if dnsConfig.OCI != nil {
		dnsSuffix = dnsConfig.OCI.DNSZoneName
	} else if dnsConfig.External != nil {
		dnsSuffix = dnsConfig.External.Suffix
	}
	if len(dnsSuffix) == 0 {
		return "", fmt.Errorf("Invalid OCI DNS configuration, no zone name specified")
	}
	return dnsSuffix, nil
}

// Identify the service type, LB vs NodePort
func GetServiceType(cr *vzapi.Verrazzano) (vzapi.IngressType, error) {
	ingressConfig := cr.Spec.Components.Ingress
	if ingressConfig == nil || len(ingressConfig.Type) == 0 {
		return vzapi.LoadBalancer, nil
	}
	switch ingressConfig.Type {
	case vzapi.NodePort, vzapi.LoadBalancer:
		return ingressConfig.Type, nil
	default:
		return "", fmt.Errorf("Unrecognized ingress type %s", ingressConfig.Type)
	}
}

// GetIngressIP Returns the ingress IP of the LoadBalancer
// - port of install scripts function get_verrazzano_ingress_ip in config.sh
func GetIngressIP(client client.Client, vz *vzapi.Verrazzano) (string, error) {
	// Default for NodePort services
	// - On MAC and Windows, container IP is not accessible.  Port forwarding from 127.0.0.1 to container IP is needed.
	ingressIP := "127.0.0.1"
	serviceType, err := GetServiceType(vz)
	if err != nil {
		return "", err
	}
	if serviceType == vzapi.LoadBalancer {
		svc := v1.Service{}
		if err := client.Get(context.TODO(), types.NamespacedName{Name: vpoconst.NGINXControllerServiceName, Namespace: globalconst.IngressNamespace}, &svc); err != nil {
			return "", err
		}
		// Test for IP from status, if that is not present then assume an on premises installation and use the externalIPs hint
		if len(svc.Status.LoadBalancer.Ingress) > 0 {
			ingressIP = svc.Status.LoadBalancer.Ingress[0].IP
		} else if len(svc.Spec.ExternalIPs) > 0 {
			// In case of OLCNE, the Status.LoadBalancer.Ingress field will be empty, so use the external IP if present
			ingressIP = svc.Spec.ExternalIPs[0]
		} else {
			return "", fmt.Errorf("No IP found for LoadBalancer service type")
		}
	}
	return ingressIP, nil
}

// BuildDNSDomain Constructs the full DNS subdomain for the deployment
func BuildDNSDomain(client client.Client, vz *vzapi.Verrazzano) (string, error) {
	dnsSuffix, err := GetDNSSuffix(client, vz)
	if err != nil {
		return "", err
	}
	envName := GetEnvName(vz)
	dnsDomain := fmt.Sprintf("%s.%s", envName, dnsSuffix)
	return dnsDomain, nil
}
