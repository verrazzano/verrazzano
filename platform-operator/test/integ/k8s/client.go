// Copyright (C) 2020, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package k8s

import (
	apixv1beta1client "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset/typed/apiextensions/v1beta1"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

// Client to access the Kubernetes API objects needed for the integration test
type Client struct {
	// Client to access the Kubernetes API
	Clientset *kubernetes.Clientset

	// Client to access the Kubernetes API for extensions
	ApixClient *apixv1beta1client.ApiextensionsV1beta1Client
}

// NewClient gets a new client that calls the Kubernetes API server to access the Verrazzano API Objects
func NewClient(kubeconfig string) (Client, error) {
	// use the current context in the kubeconfig
	config, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
	if err != nil {
		return Client{}, err
	}

	// Client to access the Kubernetes API
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return Client{}, err
	}

	// Client to access the Kubernetes API for extensions
	apixcli, err := apixv1beta1client.NewForConfig(config)
	if err != nil {
		return Client{}, err
	}

	return Client{
		Clientset:  clientset,
		ApixClient: apixcli,
	}, err
}
