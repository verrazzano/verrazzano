// Copyright (c) 2021, 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package pkg

import (
	"context"
	"fmt"
	"github.com/verrazzano/verrazzano/pkg/k8sutil"
	"github.com/verrazzano/verrazzano/platform-operator/constants"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"reflect"
	"strings"

	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
)

const (
	NipDomain   = "nip.io"
	SslipDomain = "sslip.io"
)

// GetDNS gets the DNS configured in the CR
func GetDNS(cr *vzapi.Verrazzano) string {
	if cr.Spec.Components.DNS != nil {
		if cr.Spec.Components.DNS.Wildcard != nil {
			return cr.Spec.Components.DNS.Wildcard.Domain
		}
		if cr.Spec.Components.DNS.OCI != nil {
			return cr.Spec.Components.DNS.OCI.DNSZoneName
		}
		if cr.Spec.Components.DNS.External != nil {
			return cr.Spec.Components.DNS.External.Suffix
		}
	}
	return NipDomain
}

// Returns well-known wildcard DNS name is used
func GetWildcardDNS(s string) string {
	wildcards := []string{NipDomain, SslipDomain}
	for _, w := range wildcards {
		if strings.Contains(s, w) {
			return w
		}
	}
	return ""
}

// Returns true if string has DNS wildcard name
func HasWildcardDNS(s string) bool {
	return GetWildcardDNS(s) != ""
}

func IsDefaultDNS(dns *vzapi.DNSComponent) bool {
	return dns == nil ||
		reflect.DeepEqual(*dns, vzapi.DNSComponent{}) ||
		reflect.DeepEqual(*dns, vzapi.DNSComponent{Wildcard: &vzapi.Wildcard{Domain: NipDomain}})
}

// GetEnvironmentName returns the name of the Verrazzano install environment
func GetEnvironmentName(cr *vzapi.Verrazzano) string {
	if cr.Spec.EnvironmentName != "" {
		return cr.Spec.EnvironmentName
	}

	return constants.DefaultEnvironmentName
}

func GetIngressIP() string {
	var externalIP string
	kubeconfigPath, err := k8sutil.GetKubeConfigLocation()
	if err != nil {
		Log(Error, fmt.Sprintf("Error getting kubeconfig: %v", err))
		return ""
	}
	clientset, err := GetKubernetesClientsetForCluster(kubeconfigPath)
	if err != nil {
		Log(Error, fmt.Sprintf("Failed to get clientset for cluster %v", err))
		return ""
	}
	svc, err := clientset.CoreV1().Services(constants.IngressNginxNamespace).Get(context.TODO(), constants.NGINXControllerServiceName, metav1.GetOptions{})
	if err != nil {
		Log(Info, fmt.Sprintf("Could not get services quickstart-es-http in sockshop: %v\n", err.Error()))
		return ""
	}
	if len(svc.Spec.ExternalIPs) > 0 {
		// In case of OLCNE, the Status.LoadBalancer.Ingress field will be empty, so use the external IP if present
		externalIP = svc.Spec.ExternalIPs[0]
	} else if len(svc.Status.LoadBalancer.Ingress) > 0 {
		externalIP = svc.Status.LoadBalancer.Ingress[0].IP
	}

	return externalIP
}
