// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package helpers

import (
	"errors"
	projectclientset "github.com/verrazzano/verrazzano/application-operator/clients/clusters/clientset/versioned"
	clientset "github.com/verrazzano/verrazzano/platform-operator/clients/clusters/clientset/versioned"
	verrazzanoclientset "github.com/verrazzano/verrazzano/platform-operator/clients/verrazzano/clientset/versioned"
	istioClient "istio.io/client-go/pkg/clientset/versioned"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	restclient "k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
	"os"
	"path/filepath"
)

// Helper function to obtain the default kubeConfig location
func GetKubeConfigLocation() (string, error) {

	var kubeConfig string
	kubeConfigEnvVar := os.Getenv("KUBECONFIG")

	if len(kubeConfigEnvVar) > 0 {
		// Find using environment variables
		kubeConfig = kubeConfigEnvVar
	} else if home := homedir.HomeDir(); home != "" {
		// Find in the ~/.kube/ directory
		kubeConfig = filepath.Join(home, ".kube", "config")
	} else {
		// give up
		return kubeConfig, errors.New("Unable to find kubeconfig")
	}
	return kubeConfig, nil
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

// Returns kubeconfig from KUBECONFIG env var if set
// Else from default location ~/.kube/config
func GetKubeConfig() (*restclient.Config, error) {
	var config *restclient.Config
	kubeConfigLoc, err := GetKubeConfigLocation()
	if err != nil {
		return config, err
	}
	config, err = clientcmd.BuildConfigFromFlags("", kubeConfigLoc)
	return config, err
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

type Kubernetes interface {
	GetKubeConfig() (*rest.Config, error)
	NewClustersClientSet() (clientset.Interface, error)
	NewProjectClientSet() (projectclientset.Interface, error)
	NewVerrazzanoClientSet() (verrazzanoclientset.Interface, error)
	NewClientSet() (kubernetes.Interface, error)
}
