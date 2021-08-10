// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package k8sutil

import (
	"errors"
	"os"
	"path/filepath"

	"context"
	"fmt"

	istiov1alpha3 "istio.io/client-go/pkg/apis/networking/v1alpha3"
	istioClient "istio.io/client-go/pkg/clientset/versioned"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
)

// Name of Environment Variable for KUBECONFIG
const ENV_VAR_KUBECONFIG = "KUBECONFIG"

// Name of Environment Variable for test KUBECONFIG
const ENV_VAR_TEST_KUBECONFIG = "TEST_KUBECONFIG"

// Helper function to obtain the default kubeConfig location
func GetKubeConfigLocation() (string, error) {
	if testKubeConfig := os.Getenv(ENV_VAR_TEST_KUBECONFIG); len(testKubeConfig) > 0 {
		return testKubeConfig, nil
	}

	if kubeConfig := os.Getenv(ENV_VAR_KUBECONFIG); len(kubeConfig) > 0 {
		return kubeConfig, nil
	}

	if home := homedir.HomeDir(); home != "" {
		return filepath.Join(home, ".kube", "config"), nil
	}

	return "", errors.New("unable to find kubeconfig")

}

// Returns kubeconfig from KUBECONFIG env var if set
// Else from default location ~/.kube/config
func GetKubeConfig() (*rest.Config, error) {
	var config *rest.Config
	kubeConfigLoc, err := GetKubeConfigLocation()
	if err != nil {
		return config, err
	}
	config, err = clientcmd.BuildConfigFromFlags("", kubeConfigLoc)
	return config, err
}

// GetKubernetesClientset returns the Kubernetes clientset for the cluster set in the environment
func GetKubernetesClientset() (*kubernetes.Clientset, error) {
	// use the current context in the kubeconfig
	var clientset *kubernetes.Clientset
	config, err := GetKubeConfig()
	if err != nil {
		return clientset, err
	}
	clientset, err = kubernetes.NewForConfig(config)
	return clientset, err
}

// GetIstioClientset returns the clientset object for Istio
func GetIstioClientset() (*istioClient.Clientset, error) {
	var cs *istioClient.Clientset
	kubeConfig, err := GetKubeConfig()
	if err != nil {
		return cs, err
	}
	cs, err = istioClient.NewForConfig(kubeConfig)
	return cs, err
}

// GetHostnameFromGateway returns the host name from the application gateway that was
// created for the ApplicationConfiguration with name appConfigName from list of input gateways. If
// the input list of gateways is not provided, it is fetched from the kubernetes cluster
func GetHostnameFromGateway(namespace string, appConfigName string, gateways ...istiov1alpha3.Gateway) (string, error) {
	if len(gateways) == 0 {
		cs, err := GetIstioClientset()
		if err != nil {
			fmt.Printf("Could not get istio clientset: %v", err)
			return "", err
		}

		gatewayList, err := cs.NetworkingV1alpha3().Gateways(namespace).List(context.TODO(), metav1.ListOptions{})
		if err != nil {
			fmt.Printf("Could not list application ingress gateways: %v", err)
			return "", err
		}

		gateways = gatewayList.Items
	}

	// if an optional appConfigName is provided, construct the gateway name from the namespace and
	// appConfigName and look for that specific gateway, otherwise just use the first gateway
	gatewayName := ""
	if len(appConfigName) > 0 {
		gatewayName = fmt.Sprintf("%s-%s-gw", namespace, appConfigName)
	}

	for _, gateway := range gateways {
		if len(gatewayName) > 0 && gatewayName != gateway.ObjectMeta.Name {
			continue
		}

		fmt.Printf("Found an app ingress gateway with name: %s\n", gateway.ObjectMeta.Name)
		if len(gateway.Spec.Servers) > 0 && len(gateway.Spec.Servers[0].Hosts) > 0 {
			return gateway.Spec.Servers[0].Hosts[0], nil
		}
	}

	// this can happen if the app gateway has not been created yet, the caller should
	// keep retrying and eventually we should get a gateway with a host
	fmt.Printf("Could not find host in application ingress gateways in namespace: %s\n", namespace)
	return "", nil
}
