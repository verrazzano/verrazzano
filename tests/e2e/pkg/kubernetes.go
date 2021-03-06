// Copyright (c) 2020, 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package pkg

import (
	"context"
	"fmt"
	v1 "k8s.io/api/rbac/v1"
	"os"
	"path/filepath"
	"strings"

	"github.com/onsi/ginkgo"
	vmov1 "github.com/verrazzano/verrazzano-monitoring-operator/pkg/apis/vmcontroller/v1"
	vmoclient "github.com/verrazzano/verrazzano-monitoring-operator/pkg/client/clientset/versioned"
	istioClient "istio.io/client-go/pkg/clientset/versioned"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apixv1beta1client "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset/typed/apiextensions/v1beta1"

	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	restclient "k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
)

const dockerconfigjsonTemplate string = "{\"auths\":{\"%v\":{\"username\":\"%v\",\"password\":\"%v\",\"auth\":\"%v\"}}}"

var config *restclient.Config
var clientset *kubernetes.Clientset

// GetKubeConfig will get the kubeconfig from the environment variable KUBECONFIG, if set, or else from $HOME/.kube/config
func GetKubeConfig() *restclient.Config {
	if config == nil {
		kubeconfig := ""
		kubeconfigEnvVar := ""
		testKubeConfigEnvVar := os.Getenv("TEST_KUBECONFIG")
		if len(testKubeConfigEnvVar) > 0 {
			kubeconfigEnvVar = testKubeConfigEnvVar
		}

		if kubeconfigEnvVar == "" {
			// if the KUBECONFIG environment variable is set, use that
			kubeconfigEnvVar = os.Getenv("KUBECONFIG")
		}

		if len(kubeconfigEnvVar) > 0 {
			kubeconfig = kubeconfigEnvVar
		} else if home := homedir.HomeDir(); home != "" {
			// next look for $HOME/.kube/config
			kubeconfig = filepath.Join(home, ".kube", "config")
		} else {
			// give up
			ginkgo.Fail("Could not find kube")
		}

		var err error
		config, err = clientcmd.BuildConfigFromFlags("", kubeconfig)
		if err != nil {
			ginkgo.Fail("Could not get current context from kubeconfig " + kubeconfig)
		}
	}
	return config
}

// DoesCRDExist returns whether a CRD with the given name exists for the cluster
func DoesCRDExist(crdName string) bool {
	// use the current context in the kubeconfig
	config := GetKubeConfig()

	apixClient, err := apixv1beta1client.NewForConfig(config)
	if err != nil {
		ginkgo.Fail("Could not get apix client")
	}

	crds, err := apixClient.CustomResourceDefinitions().List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		ginkgo.Fail(fmt.Sprintf("Failed to get CRDS with error: %v", err))
	}

	for i := range crds.Items {
		if strings.Compare(crds.Items[i].ObjectMeta.Name, crdName) == 0 {
			return true
		}
	}

	return false
}

// DoesNamespaceExist returns whether a namespace with the given name exists for the cluster
func DoesNamespaceExist(name string) bool {
	// Get the kubernetes clientset
	clientset := GetKubernetesClientset()

	namespace, err := clientset.CoreV1().Namespaces().Get(context.TODO(), name, metav1.GetOptions{})
	if err != nil {
		ginkgo.Fail(fmt.Sprintf("Failed to get namespace %s with error: %v", name, err))
	}

	return namespace != nil
}

// ListNamespaces returns whether a namespace with the given name exists for the cluster
func ListNamespaces() *corev1.NamespaceList {
	// Get the kubernetes clientset
	clientset := GetKubernetesClientset()

	namespaces, err := clientset.CoreV1().Namespaces().List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		ginkgo.Fail(fmt.Sprintf("Failed to get namespaces with error: %v", err))
	}

	return namespaces
}

// DoesJobExist returns whether a job with the given name and namespace exists for the cluster
func DoesJobExist(namespace string, name string) bool {
	// Get the kubernetes clientset
	clientset := GetKubernetesClientset()

	job, err := clientset.BatchV1().Jobs(namespace).Get(context.TODO(), name, metav1.GetOptions{})
	if err != nil {
		ginkgo.Fail(fmt.Sprintf("Failed to get job %s in namespace %s with error: %v", name, namespace, err))
	}

	return job != nil
}

// ListDeployments returns the list of deployments in a given namespace for the cluster
func ListDeployments(namespace string) *appsv1.DeploymentList {
	// Get the kubernetes clientset
	clientset := GetKubernetesClientset()

	deployments, err := clientset.AppsV1().Deployments(namespace).List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		ginkgo.Fail(fmt.Sprintf("Failed to list deployments in namespace %s with error: %v", namespace, err))
	}
	return deployments
}

// ListNodes returns the list of nodes for the cluster
func ListNodes() *corev1.NodeList {
	// Get the kubernetes clientset
	clientset := GetKubernetesClientset()

	nodes, err := clientset.CoreV1().Nodes().List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		ginkgo.Fail(fmt.Sprintf("Failed to list nodes with error: %v", err))
	}
	return nodes
}

// ListPods returns the list of pods in a given namespace for the cluster
func ListPods(namespace string) *corev1.PodList {
	// Get the kubernetes clientset
	clientset := GetKubernetesClientset()

	pods, err := clientset.CoreV1().Pods(namespace).List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		ginkgo.Fail(fmt.Sprintf("Failed to list pods in namespace %s with error: %v", namespace, err))
	}
	return pods
}

// GetVerrazzanoMonitoringInstance returns the a Verrazzano monitoring instance in a given namespace for the cluster
func GetVerrazzanoMonitoringInstance(namespace string, name string) (*vmov1.VerrazzanoMonitoringInstance, error) {
	// Get the kubernetes clientset
	clientset := GetVMOClientset()

	vmi, err := clientset.VerrazzanoV1().VerrazzanoMonitoringInstances(namespace).Get(context.TODO(), name, metav1.GetOptions{})
	if err != nil && !errors.IsNotFound(err) {
		ginkgo.Fail(fmt.Sprintf("Failed to get Verrazzano monitoring instance %s in namespace %s with error: %v", name, namespace, err))
	}
	return vmi, err
}

// DoesPodExist returns whether a pod with the given name and namespace exists for the cluster
func DoesPodExist(namespace string, name string) bool {
	pods := ListPods(namespace)
	for i := range pods.Items {
		if strings.HasPrefix(pods.Items[i].Name, name) {
			return true
		}
	}
	return false
}

// DoesServiceExist returns whether a service with the given name and namespace exists for the cluster
func DoesServiceExist(namespace string, name string) bool {
	services := ListServices(namespace)
	for i := range services.Items {
		if strings.HasPrefix(services.Items[i].Name, name) {
			return true
		}
	}
	return false
}

// GetKubernetesClientset returns the Kubernetes clienset for the cluster
func GetKubernetesClientset() *kubernetes.Clientset {
	// use the current context in the kubeconfig
	if clientset == nil {
		config := GetKubeConfig()

		// create the clientset once and cache it
		var err error
		clientset, err = kubernetes.NewForConfig(config)
		if err != nil {
			ginkgo.Fail("Could not get Kubernetes clientset")
		}
	}
	return clientset
}

// GetVMOClientset returns the Kubernetes clienset for the Verrazzano Monitoring Operator
func GetVMOClientset() *vmoclient.Clientset {
	// use the current context in the kubeconfig
	config := GetKubeConfig()

	// create the clientset once and cache it
	clientset, err := vmoclient.NewForConfig(config)
	if err != nil {
		ginkgo.Fail("Could not get Verrazzano Monitoring Operator clientset")
	}

	return clientset
}

// APIExtensionsClientSet returns a Kubernetes ClientSet for this cluster.
func APIExtensionsClientSet() *apixv1beta1client.ApiextensionsV1beta1Client {
	config := GetKubeConfig()

	// create the clientset
	clientset, err := apixv1beta1client.NewForConfig(config)
	if err != nil {
		ginkgo.Fail("Could not get clientset from config")
	}

	return clientset
}

// CertManagerClient returns a CertmanagerV1alpha2Client for this cluster
//func CertManagerClient() *certclientv1alpha2.CertmanagerV1alpha2Client {
//	config := GetKubeConfig()
//
//	client, err := certclientv1alpha2.NewForConfig(config)
//	if err != nil {
//		ginkgo.Fail(fmt.Sprintf("Failed to create cert-manager client: %v", err))
//	}
//
//	return client
//}

// ListServices returns the list of services in a given namespace for the cluster
func ListServices(namespace string) *corev1.ServiceList {
	// Get the kubernetes clientset
	clientset := GetKubernetesClientset()

	services, err := clientset.CoreV1().Services(namespace).List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		ginkgo.Fail(fmt.Sprintf("Failed to list services in namespace %s with error: %v", namespace, err))
	}
	return services
}

// GetService returns a service in a given namespace for the cluster
func GetService(namespace string, name string) *corev1.Service {
	services := ListServices(namespace)
	if services == nil {
		ginkgo.Fail(fmt.Sprintf("No services in namespace %s", namespace))
	}
	for _, service := range services.Items {
		if name == service.Name {
			return &service
		}
	}
	ginkgo.Fail(fmt.Sprintf("No service %s in namespace %s", name, namespace))
	return nil
}

// GetNamespace returns a namespace
func GetNamespace(name string) (*corev1.Namespace, error) {
	// Get the kubernetes clientset
	clientset := GetKubernetesClientset()

	return clientset.CoreV1().Namespaces().Get(context.TODO(), name, metav1.GetOptions{})
}

// CreateNamespace creates a namespace
func CreateNamespace(name string, labels map[string]string) (*corev1.Namespace, error) {
	if len(os.Getenv("TEST_KUBECONFIG")) > 0 {
		existingNamespace, err := GetNamespace(name)
		if err != nil {
			Log(Error, fmt.Sprintf("CreateNamespace %s, error while getting existing namespace: %v", name, err))
			return nil, err
		}

		if existingNamespace != nil && existingNamespace.Name == name {
			return existingNamespace, nil
		}

		return nil, fmt.Errorf("CreateNamespace %s, test is running with custom service account and namespace must be pre-created", name)
	}

	// Get the kubernetes clientset
	clientset := GetKubernetesClientset()

	namespace := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name:   name,
			Labels: labels,
		},
	}
	ns, err := clientset.CoreV1().Namespaces().Create(context.TODO(), namespace, metav1.CreateOptions{})
	if err != nil {
		Log(Error, fmt.Sprintf("CreateNamespace %s error: %v", name, err))
		return nil, err
	}

	return ns, nil
}

// DeleteNamespace deletes a namespace
func DeleteNamespace(name string) error {
	if len(os.Getenv("TEST_KUBECONFIG")) > 0 {
		Log(Info, fmt.Sprintf("DeleteNamespace %s, test is running with custom service account and therefore namespace won't be deleted by the test", name))
		return nil
	}

	// Get the kubernetes clientset
	clientset := GetKubernetesClientset()
	err := clientset.CoreV1().Namespaces().Delete(context.TODO(), name, metav1.DeleteOptions{})
	if err != nil {
		Log(Error, fmt.Sprintf("DeleteNamespace %s error: %v", name, err))
	}

	return err
}

// DoesClusterRoleExist returns whether a cluster role with the given name exists in the cluster
func DoesClusterRoleExist(name string) bool {
	// Get the kubernetes clientset
	clientset := GetKubernetesClientset()

	clusterrole, err := clientset.RbacV1().ClusterRoles().Get(context.TODO(), name, metav1.GetOptions{})
	if err != nil {
		ginkgo.Fail(fmt.Sprintf("Failed to get cluster role %s with error: %v", name, err))
	}

	return clusterrole != nil
}

// GetClusterRole returns the cluster role with the given name
func GetClusterRole(name string) *v1.ClusterRole {
	// Get the kubernetes clientset
	clientset := GetKubernetesClientset()

	clusterrole, err := clientset.RbacV1().ClusterRoles().Get(context.TODO(), name, metav1.GetOptions{})
	if err != nil {
		ginkgo.Fail(fmt.Sprintf("Failed to get cluster role %s with error: %v", name, err))
	}

	return clusterrole
}

// DoesClusterRoleBindingExist returns whether a cluster role with the given name exists in the cluster
func DoesClusterRoleBindingExist(name string) bool {
	// Get the kubernetes clientset
	clientset := GetKubernetesClientset()

	clusterrolebinding, err := clientset.RbacV1().ClusterRoleBindings().Get(context.TODO(), name, metav1.GetOptions{})
	if err != nil {
		ginkgo.Fail(fmt.Sprintf("Failed to get cluster role binding %s with error: %v", name, err))
	}

	return clusterrolebinding != nil
}

// GetClusterRoleBinding returns the cluster role with the given name
func GetClusterRoleBinding(name string) *v1.ClusterRoleBinding {
	// Get the kubernetes clientset
	clientset := GetKubernetesClientset()

	crb, err := clientset.RbacV1().ClusterRoleBindings().Get(context.TODO(), name, metav1.GetOptions{})
	if err != nil {
		ginkgo.Fail(fmt.Sprintf("Failed to get cluster role binding %s with error: %v", name, err))
	}

	return crb
}

// GetIstioClientset returns the clientset object for Istio
func GetIstioClientset() *istioClient.Clientset {
	cs, err := istioClient.NewForConfig(GetKubeConfig())
	if err != nil {
		ginkgo.Fail(fmt.Sprintf("Failed to get Istio clientset: %v", err))
	}
	return cs
}
