// Copyright (C) 2020, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package k8s

import (
	"context"
	"fmt"
	"github.com/onsi/ginkgo"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"strings"
)

// DoesCRDExist returns true if the given CRD exists
func (c Client) DoesCRDExist(crdName string) bool {
	crds, err := c.apixClient.CustomResourceDefinitions().List(context.Background(), metav1.ListOptions{})
	if err != nil {
		ginkgo.Fail("ginkgo.Failed to get list of CustomResourceDefinitions")
	}
	for i := range crds.Items {
		if strings.Compare(crds.Items[i].ObjectMeta.Name, crdName) == 0 {
			return true
		}
	}
	return false
}

// DoesClusterRoleExist returns true if the given ClusterRole exists
func (c Client) DoesClusterRoleExist(name string) bool {
	_, err := c.clientset.RbacV1().ClusterRoles().Get(context.TODO(), name, metav1.GetOptions{})
	return procExistsStatus(err, "ClusterRole")
}

// DoesClusterRoleBindingExist returns true if the given ClusterRoleBinding exists
func (c Client) DoesClusterRoleBindingExist(name string) bool {
	_, err := c.clientset.RbacV1().ClusterRoleBindings().Get(context.TODO(), name, metav1.GetOptions{})
	return procExistsStatus(err, "ClusterRoleBinding")
}

// DoesNamespaceExist returns true if the given Namespace exists
func (c Client) DoesNamespaceExist(name string) bool {
	_, err := c.clientset.CoreV1().Namespaces().Get(context.TODO(), name, metav1.GetOptions{})
	return procExistsStatus(err, "Namespace")
}

// DoesSecretExist returns true if the given Secret exists
func (c Client) DoesSecretExist(name string, namespace string) bool {
	_, err := c.clientset.CoreV1().Secrets(namespace).Get(context.TODO(), name, metav1.GetOptions{})
	return procExistsStatus(err, "Secret")
}

// DoesDaemonsetExist returns true if the given DaemonSet exists
func (c Client) DoesDaemonsetExist(name string, namespace string) bool {
	_, err := c.clientset.AppsV1().DaemonSets(namespace).Get(context.TODO(), name, metav1.GetOptions{})
	return procExistsStatus(err, "DaemonSet")
}

// DoesDeploymentExist returns true if the given Deployment exists
func (c Client) DoesDeploymentExist(name string, namespace string) bool {
	_, err := c.clientset.AppsV1().Deployments(namespace).Get(context.TODO(), name, metav1.GetOptions{})
	return procExistsStatus(err, "Deployment")
}

// DoesPodExist returns true if a Pod with the given prefix exists
func (c Client) DoesPodExist(name string, namespace string) bool {
	pods, err := c.clientset.CoreV1().Pods(namespace).List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		ginkgo.Fail("Could not get list of pods" + err.Error())
	}
	for i := range pods.Items {
		if strings.HasPrefix(pods.Items[i].Name, name) {
			return true
		}
	}
	return false
}

// IsPodRunning returns true if a Pod with the given prefix exists and is Running
func (c Client) IsPodRunning(name string, namespace string) bool {
	pods, err := c.clientset.CoreV1().Pods(namespace).List(context.Background(), metav1.ListOptions{})
	if err != nil {
		ginkgo.Fail("Could not get list of pods" + err.Error())
	}
	for i := range pods.Items {
		if strings.HasPrefix(pods.Items[i].Name, name) {
			conditions := pods.Items[i].Status.Conditions
			for j := range conditions {
				if conditions[j].Type == "Ready" {
					if conditions[j].Status == "True" {
						return true
					}
				}
			}
		}
	}
	return false
}

// DoesServiceExist returns true if the given Service exists
func (c Client) DoesServiceExist(name string, namespace string) bool {
	_, err := c.clientset.CoreV1().Services(namespace).Get(context.TODO(), name, metav1.GetOptions{})
	return procExistsStatus(err, "Service")
}

// DoesServiceAccountExist returns true if the given ServiceAccount exists
func (c Client) DoesServiceAccountExist(name string, namespace string) bool {
	_, err := c.clientset.CoreV1().ServiceAccounts(namespace).Get(context.TODO(), name, metav1.GetOptions{})
	return procExistsStatus(err, "ServiceAccount")
}

func procExistsStatus(err error, msg string) bool {
	if err == nil {
		return true
	}
	if !errors.IsNotFound(err) {
		ginkgo.Fail(fmt.Sprintf("ginkgo.Failed calling API to get %s: %v", msg, err))
	}
	return false
}
