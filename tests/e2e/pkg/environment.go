// Copyright (c) 2020, 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package pkg

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/verrazzano/verrazzano/pkg/k8sutil"
	v1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	// ClusterTypeKind represents a Kind cluster
	ClusterTypeKind = "kind"
	// ClusterTypeOlcne represents an OCNE cluster
	ClusterTypeOlcne     = "OCNE"
	istioSystemNamespace = "istio-system"
)

// Ingress returns the ingress address
func Ingress() string {
	clusterType, ok := os.LookupEnv("TEST_ENV")
	if !ok {
		clusterType = ClusterTypeKind
	}

	if clusterType == ClusterTypeKind {
		return loadBalancerIngress()
	} else if clusterType == ClusterTypeOlcne {
		return externalLoadBalancerIngress()
	} else {
		return loadBalancerIngress()
	}
}

// loadBalancerIngress returns the ingress load balancer address
func loadBalancerIngress() string {
	fmt.Println("Obtaining ingressgateway info ...")
	ingressgateway, err := findIstioIngressGatewaySvc(true)
	if err != nil {
		Log(Error, fmt.Sprintf("Error finding Istio ingress gateway service: %v", err))
		return ""
	}
	for i := range ingressgateway.Status.LoadBalancer.Ingress {
		ingress := ingressgateway.Status.LoadBalancer.Ingress[i]
		fmt.Println("Ingress: ", ingress, "hostname: ", ingress.Hostname, "IP: ", ingress.IP)
		if ingress.Hostname != "" {
			fmt.Println("Returning Ingress Hostname: ", ingress.Hostname)
			return ingress.Hostname
		} else if ingress.IP != "" {
			fmt.Println("Returning Ingress IP: ", ingress.IP)
			return ingress.IP
		}
	}
	return ""
}

// externalLoadBalancerIngress returns the ingress external load balancer address
func externalLoadBalancerIngress() string {
	fmt.Println("Obtaining ingressgateway info ...")
	// Test a service for a dynamic address (.status.loadBalancer.ingress[0].ip),
	// 	if that's not present then use .spec.externalIPs[0]
	lbIngressgateway, err := findIstioIngressGatewaySvc(true)
	if err != nil {
		Log(Error, fmt.Sprintf("Error finding Istio ingress gateway service: %v", err))
		return ""
	}
	for i := range lbIngressgateway.Status.LoadBalancer.Ingress {
		ingress := lbIngressgateway.Status.LoadBalancer.Ingress[i]
		if ingress.Hostname != "" {
			fmt.Println("Returning Ingress Hostname: ", ingress.Hostname)
			return ingress.Hostname
		} else if ingress.IP != "" {
			fmt.Println("Returning Ingress IP: ", ingress.IP)
			return ingress.IP
		}
	}
	// Nothing found in .status, check .spec
	ingressgateway, err := findIstioIngressGatewaySvc(false)
	if err != nil {
		Log(Error, fmt.Sprintf("Error finding Istio ingress gateway service: %v", err))
		return ""
	}
	for i := range ingressgateway.Spec.ExternalIPs {
		ingress := ingressgateway.Spec.ExternalIPs[i]
		fmt.Println("Returning Ingress IP: ", ingress)
		return ingress
	}
	return ""
}

// findIstioIngressGatewaySvc retrieves the address of the istio ingress gateway
func findIstioIngressGatewaySvc(requireLoadBalancer bool) (*v1.Service, error) {
	svcList, err := ListServices(istioSystemNamespace)
	if err != nil {
		return nil, err
	}
	var ingressgateway v1.Service
	for i := range svcList.Items {
		svc := svcList.Items[i]
		fmt.Println("Service name: ", svc.Name, ", LoadBalancer: ", svc.Status.LoadBalancer, ", Ingress: ", svc.Status.LoadBalancer.Ingress)
		if strings.Contains(svc.Name, "ingressgateway") {
			if !requireLoadBalancer {
				fmt.Println("Found ingress gateway: ", svc.Name)
				ingressgateway = svc
			} else {
				if svc.Status.LoadBalancer.Ingress != nil {
					fmt.Println("Found ingress gateway: ", svc.Name)
					ingressgateway = svc
				}
			}
		}
	}
	return &ingressgateway, nil
}

// ListIngresses lists ingresses in namespace
func ListIngresses(namespace string) (*networkingv1.IngressList, error) {
	clientset, err := k8sutil.GetKubernetesClientset()
	if err != nil {
		return nil, err
	}
	ingresses, err := clientset.NetworkingV1().Ingresses(namespace).List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	// dump out namespace data to file
	logData := ""
	for i := range ingresses.Items {
		logData = logData + ingresses.Items[i].Name + "\n"
	}
	// this seems to be used for debugging, so if there's an error, just log it but don't bubble it up
	filename := fmt.Sprintf("%v-ingresses", namespace)
	if err := CreateLogFile(filename, logData); err != nil {
		Log(Info, fmt.Sprintf("Could not create output file: %s, error: %v", filename, err))
	}
	return ingresses, nil
}
