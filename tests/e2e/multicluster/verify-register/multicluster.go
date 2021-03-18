// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package register

import (
    "context"
    "fmt"
    "github.com/onsi/ginkgo"
    corev1 "k8s.io/api/core/v1"
    "k8s.io/apimachinery/pkg/api/errors"
    metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
    "k8s.io/client-go/kubernetes"
	restclient "k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

// GetClusterKubeConfig will get the kubeconfig from the specified kubeconfig
func GetClusterKubeConfig(kubeconfig string) *restclient.Config {
	config, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
	if err != nil {
		ginkgo.Fail("Could not get kubeconfig " + kubeconfig)
	}
	return config
}

// GetClusterKubernetesClientset returns the Kubernetes clienset for the cluster
func GetClusterKubernetesClientset(kubeconfig string) *kubernetes.Clientset {
	clientset, err := kubernetes.NewForConfig(GetClusterKubeConfig(kubeconfig))
	if err != nil {
		ginkgo.Fail("Could not get Kubernetes clientset")
	}
	return clientset
}

// GetClusterSecret returns the a secret in a given namespace for the cluster
func GetClusterSecret(kubeconfig string, namespace string, name string) (*corev1.Secret, error) {
    // Get the kubernetes clientset
    clientset := GetClusterKubernetesClientset(kubeconfig)

    secret, err := clientset.CoreV1().Secrets(namespace).Get(context.TODO(), name, metav1.GetOptions{})
    if err != nil && !errors.IsNotFound(err) {
        ginkgo.Fail(fmt.Sprintf("Failed to get secrets %s in namespace %s with error: %v", name, namespace, err))
    }
    return secret, err
}
