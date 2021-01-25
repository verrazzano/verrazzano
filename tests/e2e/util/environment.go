package util

import (
	"fmt"
	"strings"

	corev1 "k8s.io/api/core/v1"
)

const istioSystemNamespace string = "istio-system"

func GetKindIngress() string {
	fmt.Println("Obtaining KIND control plane address info ...")
	addrHost := ""
	var addrPort int32

	pods := ListPods(istioSystemNamespace)
	//pods, _ := clientset.CoreV1().Pods(istioSystemNamespace).List(context.TODO, metav1.ListOptions{})
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

func findIstioIngressGatewaySvc(requireLoadBalancer bool) corev1.Service {
	svcList := ListServices(istioSystemNamespace)
	//svcList, _ := clientset.Namespace(istioSystemNamespace).ListServices()
	var ingressgateway corev1.Service
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



