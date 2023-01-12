// Copyright (c) 2021, 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
package vzconfig

import (
	"context"
	"fmt"
	"github.com/verrazzano/verrazzano/application-operator/constants"
	k8net "k8s.io/api/networking/v1"
	"k8s.io/apimachinery/pkg/types"

	globalconst "github.com/verrazzano/verrazzano/pkg/constants"
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1beta1"
	vpoconst "github.com/verrazzano/verrazzano/platform-operator/constants"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const defaultWildcardDomain = "nip.io"
const defaultIngressClassName = "verrazzano-nginx"

// GetEnvName Returns the configured environment name, or "default" if not specified in the configuration
func GetEnvName(vz *vzapi.Verrazzano) string {
	envName := vz.Spec.EnvironmentName
	if len(envName) == 0 {
		envName = "default"
	}
	return envName
}

// FindVolumeTemplate Find a named VolumeClaimTemplate in the list for v1beta1.
func FindVolumeTemplate(templateName string, object runtime.Object) (*v1.PersistentVolumeClaimSpec, bool) {
	templates := getVolumeClaimSpecTemplates(object)
	for i, template := range templates {
		if templateName == template.Name {
			return &templates[i].Spec, true
		}
	}
	return nil, false
}

// GetWildcardDomain Get the wildcard domain from the Verrazzano config
func GetWildcardDomain(DNS interface{}) string {
	wildcardDomain := defaultWildcardDomain
	if dnsConfigv1alpha1, ok := DNS.(*vzapi.DNSComponent); ok {
		if dnsConfigv1alpha1 != nil && dnsConfigv1alpha1.Wildcard != nil && len(dnsConfigv1alpha1.Wildcard.Domain) > 0 {
			wildcardDomain = dnsConfigv1alpha1.Wildcard.Domain
		}
	}
	if dnsConfigv1beta1, ok := DNS.(*v1beta1.DNSComponent); ok {
		if dnsConfigv1beta1 != nil && dnsConfigv1beta1.Wildcard != nil && len(dnsConfigv1beta1.Wildcard.Domain) > 0 {
			wildcardDomain = dnsConfigv1beta1.Wildcard.Domain
		}
	}
	return wildcardDomain
}

// GetDNSSuffix Returns the DNS suffix for the Verrazzano installation
// - port of install script function get_dns_suffix from config.sh
func GetDNSSuffix(client client.Client, vz *vzapi.Verrazzano) (string, error) {
	const externalDNSKey = "external-dns.alpha.kubernetes.io/target"
	var dnsSuffix string
	if vz == nil {
		// attempt to get DNS from env
		ingress := k8net.Ingress{}
		err := client.Get(context.TODO(), types.NamespacedName{Name: constants.VzConsoleIngress, Namespace: constants.VerrazzanoSystemNamespace}, &ingress)
		if err != nil {
			return "", err
		}
		externalDNSAnno, ok := ingress.Annotations[externalDNSKey]
		if !ok || len(externalDNSAnno) == 0 {
			return "", fmt.Errorf("Annotation %s missing from Verrazzano ingress, unable to generate DNS name", externalDNSKey)
		}
		return externalDNSAnno[len(constants.VzConsoleIngress)+1:], nil
	}
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
func GetIngressServiceType(cr *vzapi.Verrazzano) (vzapi.IngressType, error) {
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
	serviceType, err := GetIngressServiceType(vz)
	if err != nil {
		return "", err
	}
	return GetExternalIP(client, serviceType, vpoconst.NGINXControllerServiceName, globalconst.IngressNamespace)
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

// GetIngressClassName gets the ingress class name or default of "nginx" if not specified
func GetIngressClassName(vz *vzapi.Verrazzano) string {
	ingressComponent := vz.Spec.Components.Ingress
	if ingressComponent != nil && ingressComponent.IngressClassName != nil && *ingressComponent.IngressClassName != "" {
		return *ingressComponent.IngressClassName
	}
	return defaultIngressClassName
}

// getVolumeClaimSpecTemplates returns the volume claim specs in v1beta1.
func getVolumeClaimSpecTemplates(object runtime.Object) []v1beta1.VolumeClaimSpecTemplate {
	if effectiveCR, ok := object.(*vzapi.Verrazzano); ok {
		return vzapi.ConvertVolumeClaimTemplateTo(effectiveCR.Spec.VolumeClaimSpecTemplates)
	} else if effectiveCR, ok := object.(*v1beta1.Verrazzano); ok {
		return effectiveCR.Spec.VolumeClaimSpecTemplates
	}
	return nil
}
