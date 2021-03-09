// Copyright (c) 2020, 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package pkg

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/onsi/ginkgo"
	v1 "k8s.io/api/core/v1"
	extensionsv1beta1 "k8s.io/api/extensions/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	// ClusterTypeKind represents a Kind cluster
	ClusterTypeKind = "kind"
	// ClusterTypeOlcne represents an OLCNE cluster
	ClusterTypeOlcne     = "OLCNE"
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

// nodePortIngress returns the ingress node port address
func nodePortIngress() string {
	fmt.Println("Obtaining node address info ...")
	addrHost := ""
	var addrPort int32

	pods, _ := GetKubernetesClientset().CoreV1().Pods(istioSystemNamespace).List(context.TODO(), metav1.ListOptions{})
	for i := range pods.Items {
		if strings.HasPrefix(pods.Items[i].Name, "istio-ingressgateway-") {
			addrHost = pods.Items[i].Status.HostIP
		}
	}

	ingressgateway := findIstioIngressGatewaySvc(false)
	fmt.Println("ingressgateway for cluster is ", ingressgateway)
	for _, eachPort := range ingressgateway.Spec.Ports {
		if eachPort.Port == 80 {
			fmt.Printf("cluster - found ingressgateway port %d with nodeport %d, name %s\n", eachPort.Port, eachPort.NodePort, eachPort.Name)
			fmt.Printf("cluster - found ingressgateway port %d with nodeport %d, name %s\n", eachPort.Port, eachPort.NodePort, eachPort.Name)
			addrPort = eachPort.NodePort
		}
	}

	if addrHost == "" {
		fmt.Println("node address is empty")
		return ""
	}

	ingressAddr := fmt.Sprintf("%s:%d", addrHost, addrPort)
	fmt.Printf("ingress address is %s\n", ingressAddr)
	return ingressAddr

}

// loadBalancerIngress returns the ingress load balancer address
func loadBalancerIngress() string {
	fmt.Println("Obtaining ingressgateway info ...")
	ingressgateway := findIstioIngressGatewaySvc(true)
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
	lbIngressgateway := findIstioIngressGatewaySvc(true)
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
	ingressgateway := findIstioIngressGatewaySvc(false)
	for i := range ingressgateway.Spec.ExternalIPs {
		ingress := ingressgateway.Spec.ExternalIPs[i]
		fmt.Println("Returning Ingress IP: ", ingress)
		return ingress
	}
	return ""
}

// findIstioIngressGatewaySvc retrieves the address of the istio ingress gateway
func findIstioIngressGatewaySvc(requireLoadBalancer bool) v1.Service {
	svcList := ListServices(istioSystemNamespace)
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
	return ingressgateway
}

//// ListCertificates lists certificates in namespace
//func ListCertificates(namespace string) (*certapiv1alpha2.CertificateList, error) {
//	certs, err := CertManagerClient().Certificates(namespace).List(metav1.ListOptions{})
//	if err != nil {
//		ginkgo.Fail(fmt.Sprintf("Could not get list of certificates: %v\n", err.Error()))
//	}
//	// dump out namespace data to file
//	logData := ""
//	for i := range certs.Items {
//		logData = logData + certs.Items[i].Name + "\n"
//	}
//	CreateLogFile(fmt.Sprintf("%v-certificates", namespace), logData)
//	return certs, err
//}

// ListIngresses lists ingresses in namespace
func ListIngresses(namespace string) (*extensionsv1beta1.IngressList, error) {
	ingresses, err := GetKubernetesClientset().ExtensionsV1beta1().Ingresses(namespace).List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		ginkgo.Fail(fmt.Sprintf("Could not get list of ingresses: %v\n", err.Error()))
	}
	// dump out namespace data to file
	logData := ""
	for i := range ingresses.Items {
		logData = logData + ingresses.Items[i].Name + "\n"
	}
	CreateLogFile(fmt.Sprintf("%v-ingresses", namespace), logData)
	return ingresses, err
}

// GetHostnameFromGateway returns the host name from the application gateway that was
// created by the ingress trait controller
func GetHostnameFromGateway(namespace string, appConfigName string) string {
	gateways, err := GetIstioClientset().NetworkingV1alpha3().Gateways(namespace).List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		ginkgo.Fail(fmt.Sprintf("Could not list application ingress gateways: %v\n", err.Error()))
	}

	// if an optional appConfigName is provided, construct the gateway name from the namespace and
	// appConfigName and look for that specific gateway, otherwise just use the first gateway
	gatewayName := ""
	if len(appConfigName) > 0 {
		gatewayName = fmt.Sprintf("%s-%s-gw", namespace, appConfigName)
	}

	for _, gateway := range gateways.Items {
		fmt.Printf("Found an app ingress gateway with name: %s\n", gateway.ObjectMeta.Name)

		if len(gatewayName) > 0 && gatewayName != gateway.ObjectMeta.Name {
			continue
		}
		if len(gateway.Spec.Servers) > 0 && len(gateway.Spec.Servers[0].Hosts) > 0 {
			return gateway.Spec.Servers[0].Hosts[0]
		}
	}

	// this can happen if the app gateway has not been created yet, the caller should
	// keep retrying and eventually we should get a gateway with a host
	fmt.Printf("Could not find host in application ingress gateways in namespace: %s\n", namespace)
	return ""
}
