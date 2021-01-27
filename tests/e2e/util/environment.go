// Copyright (c) 2020, 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package util

import (
	"context"
	"fmt"
	"github.com/onsi/ginkgo"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"os"
	"strings"
)

const (
	CLUSTER_TYPE_OKE             = "OKE"
	CLUSTER_TYPE_KIND            = "KIND"
	CLUSTER_TYPE_OLCNE           = "OLCNE"
	istioSystemNamespace 		 = "istio-system"
)

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

func QueryMetric(metricsName string) string {
	metricsURL := fmt.Sprintf("https://%s/api/v1/query?query=%s", getPrometheusIngressHost(), metricsName)
	status, content := GetWebPageWithBasicAuth(metricsURL, "", "verrazzano", getVerrazzanoPassword())
	if status != 200 {
		ginkgo.Fail(fmt.Sprintf("Error retrieving metric %s", metricsName))
	}
	return content
}

func getPrometheusIngressHost() string {
	ingressList, _ := GetKubernetesClientset().ExtensionsV1beta1().Ingresses("verrazzano-system").List(context.TODO(), metav1.ListOptions{})
	for _, ingress := range ingressList.Items {
		if ingress.Name == "vmi-system-prometheus" {
			Log(Info, fmt.Sprintf("Found Ingress %v", ingress.Name))
			return ingress.Spec.Rules[0].Host
		}
	}
	return ""
}

func getVerrazzanoPassword() string {
	secret, _ := GetKubernetesClientset().CoreV1().Secrets("verrazzano-system").Get(context.TODO(),"verrazzano", metav1.GetOptions{})
	return string(secret.Data["password"])
}



