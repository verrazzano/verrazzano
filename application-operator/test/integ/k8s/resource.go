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
	clustersv1alpha1 "github.com/verrazzano/verrazzano/application-operator/apis/clusters/v1alpha1"
	"github.com/verrazzano/verrazzano/application-operator/apis/oam/v1alpha1"
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

// DoesRoleBindingExist returns true if the given RoleBinding exists
func (c Client) DoesRoleBindingExist(name string, namespace string) bool {
	_, err := c.clientset.RbacV1().RoleBindings(namespace).Get(context.TODO(), name, metav1.GetOptions{})
	return procExistsStatus(err, "RoleBinding")
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

// GetAppConfig gets OAM custom-resource ApplicationConfiguration
func (c Client) GetAppConfig(namespace, name string) (*oamv1.ApplicationConfiguration, error) {
	bytes, err := c.getRaw("/apis/core.oam.dev/v1alpha2", "applicationconfigurations", namespace, name)
	if err != nil {
		return nil, err
	}
	var appConfig oamv1.ApplicationConfiguration
	err = json.Unmarshal(bytes, &appConfig)
	return &appConfig, err
}

// GetMultiClusterSecret gets the specified MultiClusterSecret resource
func (c Client) GetMultiClusterSecret(namespace, name string) (*clustersv1alpha1.MultiClusterSecret, error) {
	bytes, err := c.getRaw("/apis/clusters.verrazzano.io/v1alpha1", "multiclustersecrets", namespace, name)
	if err != nil {
		return nil, err
	}
	var mcSecret clustersv1alpha1.MultiClusterSecret
	err = json.Unmarshal(bytes, &mcSecret)
	return &mcSecret, err
}

// GetSecret gets the specified K8S secret
func (c Client) GetSecret(namespace, name string) (*corev1.Secret, error) {
	return c.clientset.CoreV1().Secrets(namespace).Get(context.TODO(), name, metav1.GetOptions{})
}

// GetNamespace gets the specified K8S namespace
func (c Client) GetNamespace(name string) (*corev1.Namespace, error) {
	return c.clientset.CoreV1().Namespaces().Get(context.TODO(), name, metav1.GetOptions{})
}

// GetMultiClusterComponent gets the specified MultiClusterComponent
func (c Client) GetMultiClusterComponent(namespace string, name string) (*clustersv1alpha1.MultiClusterComponent, error) {
	bytes, err := c.getRaw("/apis/clusters.verrazzano.io/v1alpha1", "multiclustercomponents", namespace, name)
	if err != nil {
		return nil, err
	}
	var mcComp clustersv1alpha1.MultiClusterComponent
	err = json.Unmarshal(bytes, &mcComp)
	return &mcComp, err
}

// GetOAMComponent gets the specified OAM Component
func (c Client) GetOAMComponent(namespace string, name string) (*oamv1.Component, error) {
	bytes, err := c.getRaw("/apis/core.oam.dev/v1alpha2", "components", namespace, name)
	if err != nil {
		return nil, err
	}
	var comp oamv1.Component
	err = json.Unmarshal(bytes, &comp)
	return &comp, err
}

// GetMultiClusterConfigMap gets the specified MultiClusterConfigMap
func (c Client) GetMultiClusterConfigMap(namespace string, name string) (*clustersv1alpha1.MultiClusterConfigMap, error) {
	bytes, err := c.getRaw("/apis/clusters.verrazzano.io/v1alpha1", "multiclusterconfigmaps", namespace, name)
	if err != nil {
		return nil, err
	}
	var mcConfigMap clustersv1alpha1.MultiClusterConfigMap
	err = json.Unmarshal(bytes, &mcConfigMap)
	return &mcConfigMap, err
}

// GetConfigMap gets the specified K8S ConfigMap
func (c Client) GetConfigMap(namespace string, name string) (*corev1.ConfigMap, error) {
	return c.clientset.CoreV1().ConfigMaps(namespace).Get(context.TODO(), name, metav1.GetOptions{})
}

// GetMultiClusterLoggingScope gets the specified MultiClusterLoggingScope
func (c Client) GetMultiClusterLoggingScope(namespace string, name string) (*clustersv1alpha1.MultiClusterLoggingScope, error) {
	bytes, err := c.getRaw("/apis/clusters.verrazzano.io/v1alpha1", "multiclusterloggingscopes", namespace, name)
	if err != nil {
		return nil, err
	}
	var mcLogScope clustersv1alpha1.MultiClusterLoggingScope
	err = json.Unmarshal(bytes, &mcLogScope)
	return &mcLogScope, err
}

// GetLoggingScope gets the specified LoggingScope
func (c Client) GetLoggingScope(namespace string, name string) (*v1alpha1.LoggingScope, error) {
	bytes, err := c.getRaw("/apis/oam.verrazzano.io/v1alpha1", "loggingscopes", namespace, name)
	if err != nil {
		return nil, err
	}
	var logScope v1alpha1.LoggingScope
	err = json.Unmarshal(bytes, &logScope)
	return &logScope, err
}

// GetMultiClusterAppConfig gets the specified MultiClusterApplicationConfiguration
func (c Client) GetMultiClusterAppConfig(namespace string, name string) (*clustersv1alpha1.MultiClusterApplicationConfiguration, error) {
	bytes, err := c.getRaw("/apis/clusters.verrazzano.io/v1alpha1", "multiclusterapplicationconfigurations", namespace, name)
	if err != nil {
		return nil, err
	}
	var mcAppConf clustersv1alpha1.MultiClusterApplicationConfiguration
	err = json.Unmarshal(bytes, &mcAppConf)
	return &mcAppConf, err
}

// GetOAMAppConfig gets the specified OAM ApplicationConfiguration
func (c Client) GetOAMAppConfig(namespace string, name string) (*oamv1.ApplicationConfiguration, error) {
	bytes, err := c.getRaw("/apis/core.oam.dev/v1alpha2", "applicationconfigurations", namespace, name)
	if err != nil {
		return nil, err
	}
	var appConf oamv1.ApplicationConfiguration
	err = json.Unmarshal(bytes, &appConf)
	return &appConf, err
}

func (c Client) getRaw(absPath, resource, namespace, name string) ([]byte, error) {
	return c.clientset.RESTClient().
		Get().
		AbsPath(absPath).
		Namespace(namespace).
		Resource(resource).
		Name(name).
		DoRaw(context.TODO())
}
