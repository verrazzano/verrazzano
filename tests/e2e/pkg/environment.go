// Copyright (c) 2020, 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package pkg

import (
	"context"
	"fmt"
	"net"
	"os"
	"strings"
	yourl "net/url"

	v1 "k8s.io/api/core/v1"
	certapiv1alpha2 "github.com/jetstack/cert-manager/pkg/apis/certmanager/v1alpha2"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	extensionsv1beta1 "k8s.io/api/extensions/v1beta1"
	"github.com/onsi/ginkgo"

)

const (
	CLUSTER_TYPE_OKE             = "OKE"
	CLUSTER_TYPE_KIND            = "KIND"
	CLUSTER_TYPE_OLCNE           = "OLCNE"
	istioSystemNamespace 		 = "istio-system"
)

// Ingress returns the ingress address
func Ingress() string {
	clusterType, ok := os.LookupEnv("TEST_ENV")
	if !ok {
		clusterType = CLUSTER_TYPE_OKE
	}

	if clusterType == CLUSTER_TYPE_KIND {
		return kindIngress()
	} else if clusterType == CLUSTER_TYPE_OLCNE {
		return olcneIngress()
	} else {
		return okeIngress()
	}
}

// kindIngress returns the ingress address from a KIND cluster
func kindIngress() string {
	fmt.Println("Obtaining KIND control plane address info ...")
	addrHost := ""
	var addrPort int32

	pods, _ := GetKubernetesClientset().CoreV1().Pods(istioSystemNamespace).List(context.TODO(), metav1.ListOptions{})
	for i := range pods.Items {
		if strings.HasPrefix(pods.Items[i].Name, "istio-ingressgateway-") {
			addrHost = pods.Items[i].Status.HostIP
		}
	}

	ingressgateway := findIstioIngressGatewaySvc(false)
	fmt.Println("ingressgateway for KIND cluster is ", ingressgateway)
	for _, eachPort := range ingressgateway.Spec.Ports {
		if eachPort.Port == 80 {
			fmt.Printf("KIND cluster - found ingressgateway port %d with nodeport %d, name %s\n", eachPort.Port, eachPort.NodePort, eachPort.Name)
			addrPort = eachPort.NodePort
		}
	}

	if addrHost == "" {
		fmt.Println("KIND control plane address is empty")
		return ""
	} else {
		ingressAddr := fmt.Sprintf("%s:%d", addrHost, addrPort)
		fmt.Printf("KIND ingress address is %s\n", ingressAddr)
		return ingressAddr
	}
}

// okeIngress returns the ingress address from an OKE cluster
func okeIngress() string {
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

// olcneIngress returns the ingress address from an OLCNE cluster
func olcneIngress() string {
	fmt.Println("Obtaining OLCNE ingressgateway info ...")
	// Test a service for a dynamic address (.status.loadBalancer.ingress[0].ip),
	// 	if that's not present then use .spec.externalIPs[0]
	lb_ingressgateway := findIstioIngressGatewaySvc(true)
	for i := range lb_ingressgateway.Status.LoadBalancer.Ingress {
		ingress := lb_ingressgateway.Status.LoadBalancer.Ingress[i]
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

func Lookup(url string) bool {
	parsed, err := yourl.Parse(url)
	if err != nil {
		Log(Info, fmt.Sprintf("Error parse %v error: %v", url, err))
		return false
	}
	_, err = net.LookupHost(parsed.Host)
	if err != nil {
		Log(Info, fmt.Sprintf("Error LookupHost %v error: %v", url, err))
		return false
	}
	return true
}

// ListCertificates lists certificates in namespace
func ListCertificates(namespace string) (*certapiv1alpha2.CertificateList, error) {
	certs, err := CertManagerClient().Certificates(namespace).List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		ginkgo.Fail(fmt.Sprintf("Could not get list of certificates: %v\n", err.Error()))
	}
	// dump out namespace data to file
	logData := ""
	for i := range certs.Items {
		logData = logData + certs.Items[i].Name + "\n"
	}
	CreateLogFile(fmt.Sprintf("%v-certificates", namespace), logData)
	return certs, err
}

// ListIngress lists ingresses in namespace
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