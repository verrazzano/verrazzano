// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package ingresstrait

import (
	"context"
	"fmt"
	vzapi "github.com/verrazzano/verrazzano/application-operator/apis/oam/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	k8net "k8s.io/api/networking/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
	"strings"
)


func CoallateAllHostsForTrait(trait *vzapi.IngressTrait, appName string, appNamespace string) ([]string, error) {
	allHosts := []string{}
	var err error
	for _, rule := range trait.Spec.Rules {
		if allHosts, err = CreateHostsFromIngressTraitRule(rule, trait, appName, appNamespace, allHosts...); err != nil {
			print(err)
			return nil, err
		}
	}
	return allHosts, nil
}

func CreateHostsFromIngressTraitRule(rule vzapi.IngressRule, trait *vzapi.IngressTrait, appName string, appNamespace string, toList ...string) ([]string, error) {
	validHosts := toList
	useDefaultHost := true
	for _, h := range rule.Hosts {
		h = strings.TrimSpace(h)
		if _, hostAlreadyPresent := findHost(validHosts, h); hostAlreadyPresent {
			// Avoid duplicates
			useDefaultHost = false
			continue
		}
		// Ignore empty or wildcard hostname
		if len(h) == 0 || strings.Contains(h, "*") {
			continue
		}
		h = strings.ToLower(strings.TrimSpace(h))
		validHosts = append(validHosts, h)
		useDefaultHost = false
	}
	// Add done if a host was added to the host list
	if !useDefaultHost {
		return validHosts, nil
	}

	// Generate a default hostname

	hostName, err := buildAppFullyQualifiedHostName(trait, appName, appNamespace)
	if err != nil {
		return nil, err
	}
	// Only add the generated hostname if it doesn't exist in hte list
	if _, hostAlreadyPresent := findHost(validHosts, hostName); !hostAlreadyPresent {
		validHosts = append(validHosts, hostName)
	}
	return validHosts, nil
}

// buildAppFullyQualifiedHostName generates a DNS host name for the application using the following structure:
// <app>.<namespace>.<dns-subdomain>  where
//
//	app is the OAM application name
//	namespace is the namespace of the OAM application
//	dns-subdomain is The DNS subdomain name

func buildAppFullyQualifiedHostName(trait *vzapi.IngressTrait, appName string, appNamespace string) (string, error) {

	domainName, err := buildNamespacedDomainName(trait, appNamespace)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%s.%s", appName, domainName), nil

}

// buildNamespacedDomainName generates a domain name for the application using the following structure:
// <namespace>.<dns-subdomain>  where
//
//	namespace is the namespace of the OAM application
//	dns-subdomain is The DNS subdomain name

func buildNamespacedDomainName(trait *vzapi.IngressTrait, appNamespace string) (string, error) {
	const externalDNSKey = "external-dns.alpha.kubernetes.io/target"
	const wildcardDomainKey = "verrazzano.io/dns.wildcard.domain"
	cfg, _ := config.GetConfig()
	cli, _ := client.New(cfg, client.Options{})

	// Extract the domain name from the Verrazzano ingress
	ingress := k8net.Ingress{}
	err := cli.Get(context.TODO(), types.NamespacedName{Name: "verrazzano-ingress", Namespace: "verrazzano-system"}, &ingress)
	if err != nil {
		return "", err
	}
	externalDNSAnno, ok := ingress.Annotations[externalDNSKey]
	if !ok || len(externalDNSAnno) == 0 {
		return "", fmt.Errorf("Annotation %s missing from Verrazzano ingress, unable to generate DNS name", externalDNSKey)
	}

	domain := externalDNSAnno[len("verrazzano-ingress")+1:]

	// Get the DNS wildcard domain from the annotation if it exist.  This annotation is only available
	// when the install is using DNS type wildcard (nip.io, sslip.io, etc.)
	suffix := ""
	wildcardDomainAnno, ok := ingress.Annotations[wildcardDomainKey]
	if ok {
		suffix = wildcardDomainAnno
	}

	// Build the domain name using Istio info
	if len(suffix) != 0 {
		domain, err = buildDomainNameForWildcard(cli, suffix)
		if err != nil {
			return "", err
		}
	}

	return fmt.Sprintf("%s.%s", appNamespace, domain), nil

}

// findHost searches for a host in the provided list. If found it will
// return it's key, otherwise it will return -1 and a bool of false.
func findHost(hosts []string, newHost string) (int, bool) {
	for i, host := range hosts {
		if strings.EqualFold(host, newHost) {
			return i, true
		}
	}
	return -1, false
}

// buildDomainNameForWildcard generates a domain name in the format of "<IP>.<wildcard-domain>"
// Get the IP from Istio resources
func buildDomainNameForWildcard(cli client.Reader, suffix string) (string, error) {
	istioIngressGateway := "istio-ingressgateway"
	istioSystemNamespace := "istio-system"
	istio := corev1.Service{}
	err := cli.Get(context.TODO(), types.NamespacedName{Name: istioIngressGateway, Namespace: istioSystemNamespace}, &istio)
	if err != nil {
		return "", err
	}
	var IP string
	if istio.Spec.Type == corev1.ServiceTypeLoadBalancer || istio.Spec.Type == corev1.ServiceTypeNodePort {
		if len(istio.Spec.ExternalIPs) > 0 {
			IP = istio.Spec.ExternalIPs[0]
		} else if len(istio.Status.LoadBalancer.Ingress) > 0 {
			IP = istio.Status.LoadBalancer.Ingress[0].IP
		} else {
			return "", fmt.Errorf("%s is missing loadbalancer IP", istioIngressGateway)
		}
	} else {
		return "", fmt.Errorf("unsupported service type %s for istio_ingress", string(istio.Spec.Type))
	}
	domain := IP + "." + suffix
	return domain, nil
}
