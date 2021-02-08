// Copyright (C) 2020, 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package k8s

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	oamv1 "github.com/crossplane/oam-kubernetes-runtime/apis/core/v1alpha2"

	"github.com/onsi/ginkgo"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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

// IsDeploymentUpdated returns true if the given Deployment has been updated with sidecar container
func (c Client) IsDeploymentUpdated(name string, namespace string) bool {
	dep, err := c.clientset.AppsV1().Deployments(namespace).Get(context.TODO(), name, metav1.GetOptions{})
	if err != nil {
		return false
	}
	return len(dep.Spec.Template.Spec.Containers) > 1
}

// DoesPodExist returns true if a Pod with the given prefix exists
func (c Client) DoesPodExist(name string, namespace string) bool {
	return (c.getPod(name, namespace) != nil)
}

// DoesContainerExist returns true if a container with the given name exists in the pod
func (c Client) DoesContainerExist(namespace, podName, containerName string) bool {
	pods, err := c.clientset.CoreV1().Pods(namespace).List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		ginkgo.Fail("Could not get list of pods" + err.Error())
		return false
	}
	for _, pod := range pods.Items {
		if strings.HasPrefix(pod.Name, podName) {
			for _, container := range pod.Status.ContainerStatuses {
				if container.Name == containerName && container.Ready {
					return true
				}
			}
		}
	}
	return false
}

// IsPodRunning returns true if a Pod with the given prefix is running
func (c Client) IsPodRunning(name string, namespace string) bool {
	pod := c.getPod(name, namespace)
	if pod != nil {
		if pod.Status.Phase == corev1.PodRunning {
			for _, c := range pod.Status.ContainerStatuses {
				if !c.Ready {
					return false
				}
			}
			return len(pod.Status.ContainerStatuses) != 0
		}
	}
	return false
}

func (c Client) getPod(name string, namespace string) *corev1.Pod {
	pods, err := c.clientset.CoreV1().Pods(namespace).List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		ginkgo.Fail("Could not get list of pods" + err.Error())
		return nil
	}
	for i := range pods.Items {
		if strings.HasPrefix(pods.Items[i].Name, name) {
			return &pods.Items[i]
		}
	}
	return nil
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

//GetAppConfig gets OAM custom-resource ApplicationConfiguration
func (c Client) GetAppConfig(namespace, name string) (*oamv1.ApplicationConfiguration, error) {
	bytes, err := c.clientset.RESTClient().
		Get().
		AbsPath("/apis/core.oam.dev/v1alpha2").
		Namespace(namespace).
		Resource("applicationconfigurations").
		Name(name).
		DoRaw(context.TODO())
	if err != nil {
		return nil, err
	}
	var appConfig oamv1.ApplicationConfiguration
	err = json.Unmarshal(bytes, &appConfig)
	return &appConfig, err
}
