// Copyright (c) 2020, 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package pkg

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/google/uuid"
	certmanagerv1 "github.com/jetstack/cert-manager/pkg/apis/certmanager/v1"
	"github.com/onsi/gomega"
	vaoClient "github.com/verrazzano/verrazzano/application-operator/clients/app/clientset/versioned"
	vpClient "github.com/verrazzano/verrazzano/application-operator/clients/clusters/clientset/versioned"
	"github.com/verrazzano/verrazzano/pkg/k8sutil"
	"github.com/verrazzano/verrazzano/pkg/semver"
	"github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	vmcClient "github.com/verrazzano/verrazzano/platform-operator/clients/clusters/clientset/versioned"
	vpoClient "github.com/verrazzano/verrazzano/platform-operator/clients/verrazzano/clientset/versioned"
	istionetv1beta1 "istio.io/client-go/pkg/apis/networking/v1beta1"
	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/authorization/v1"
	corev1 "k8s.io/api/core/v1"
	netv1 "k8s.io/api/networking/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	apiext "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	apiextv1 "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset/typed/apiextensions/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	restclient "k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
	"k8s.io/client-go/tools/remotecommand"
)

const (
	dockerconfigjsonTemplate string = "{\"auths\":{\"%v\":{\"username\":\"%v\",\"password\":\"%v\",\"auth\":\"%v\"}}}"
	verrazzanoErrorTemplate         = "Error Verrazzano Resource: %v"
)

// DoesCRDExist returns whether a CRD with the given name exists for the cluster
func DoesCRDExist(crdName string) (bool, error) {
	crds, err := ListCRDs()
	if err != nil {
		return false, err
	}

	for i := range crds.Items {
		if strings.Compare(crds.Items[i].ObjectMeta.Name, crdName) == 0 {
			return true, nil
		}
	}

	return false, nil
}

// DoesNamespaceExist returns whether a namespace with the given name exists for the cluster set in the environment
func DoesNamespaceExist(name string) (bool, error) {
	kubeconfigPath, err := k8sutil.GetKubeConfigLocation()
	if err != nil {
		Log(Error, fmt.Sprintf("Failed getting kubeconfig: %v", err))
		return false, err
	}

	return DoesNamespaceExistInCluster(name, kubeconfigPath)
}

// DoesNamespaceExistInCluster returns whether a namespace with the given name exists in the specified cluster
func DoesNamespaceExistInCluster(name string, kubeconfigPath string) (bool, error) {
	// Get the Kubernetes clientset
	clientset, err := GetKubernetesClientsetForCluster(kubeconfigPath)
	if err != nil {
		return false, err
	}

	namespace, err := clientset.CoreV1().Namespaces().Get(context.TODO(), name, metav1.GetOptions{})
	if err != nil && !k8serrors.IsNotFound(err) {
		Log(Error, fmt.Sprintf("Failed to get namespace %s: %v", name, err))
		return false, err
	}

	return namespace != nil && len(namespace.Name) > 0, nil
}

// ListCRDs returns the list of CRDs in a cluster
func ListCRDs() (*apiext.CustomResourceDefinitionList, error) {
	// use the current context in the kubeconfig
	config, err := k8sutil.GetKubeConfig()
	if err != nil {
		return nil, err
	}
	apixClient, err := apiextv1.NewForConfig(config)
	if err != nil {
		Log(Error, "Could not get apix client")
		return nil, err
	}

	crds, err := apixClient.CustomResourceDefinitions().List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		Log(Error, fmt.Sprintf("Failed to list CRDS: %v", err))
		return nil, err
	}

	return crds, nil
}

// ListNamespaces returns a namespace list for the given list options
func ListNamespaces(opts metav1.ListOptions) (*corev1.NamespaceList, error) {
	client, err := k8sutil.GetKubernetesClientset()
	if err != nil {
		return nil, err
	}
	return client.CoreV1().Namespaces().List(context.TODO(), opts)
}

// ListPods returns a pod list for the given namespace and list options
func ListPods(namespace string, opts metav1.ListOptions) (*corev1.PodList, error) {
	client, err := k8sutil.GetKubernetesClientset()
	if err != nil {
		return nil, err
	}
	return client.CoreV1().Pods(namespace).List(context.TODO(), opts)
}

// ListDeployments returns the list of deployments in a given namespace for the cluster
func ListDeployments(namespace string) (*appsv1.DeploymentList, error) {
	// Get the Kubernetes clientset
	clientset, err := k8sutil.GetKubernetesClientset()
	if err != nil {
		return nil, err
	}

	deployments, err := clientset.AppsV1().Deployments(namespace).List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		Log(Error, fmt.Sprintf("Failed to list deployments in namespace %s: %v", namespace, err))
		return nil, err
	}
	return deployments, nil
}

// ListStatefulSets returns the list of StatefulSets in a given namespace for the cluster
func ListStatefulSets(namespace string) (*appsv1.StatefulSetList, error) {
	// Get the Kubernetes clientset
	clientset, err := k8sutil.GetKubernetesClientset()
	if err != nil {
		return nil, err
	}

	statefulsets, err := clientset.AppsV1().StatefulSets(namespace).List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		Log(Error, fmt.Sprintf("Failed to list StatefulSets in namespace %s: %v", namespace, err))
		return nil, err
	}
	return statefulsets, nil
}

// GetReplicaCounts Builds a map of pod counts for a list of deployments
// expectedDeployments - a list of namespaced names for deployments to look for
// optsBuilder - a callback func to build the right set of options to select pods for the deployment
func GetReplicaCounts(expectedDeployments []types.NamespacedName, optsBuilder func(name types.NamespacedName) (metav1.ListOptions, error)) (map[string]uint32, error) {
	podCountsMap := map[string]uint32{}
	for _, deployment := range expectedDeployments {
		listOpts, err := optsBuilder(deployment)
		if err != nil {
			return map[string]uint32{}, err
		}
		podList, err := ListPods(deployment.Namespace, listOpts)
		if err != nil {
			return map[string]uint32{}, err
		}
		podCountsMap[deployment.String()] = uint32(len(podList.Items))
	}
	return podCountsMap, nil
}

// DoesDeploymentExist returns whether a deployment with the given name and namespace exists for the cluster
func DoesDeploymentExist(namespace string, name string) (bool, error) {
	deployments, err := ListDeployments(namespace)
	if err != nil {
		Log(Error, fmt.Sprintf("Failed listing deployments in cluster for namespace %s: %v", namespace, err))
		return false, err
	}
	for i := range deployments.Items {
		if strings.HasPrefix(deployments.Items[i].Name, name) {
			return true, nil
		}
	}
	return false, nil
}

// GetDeployment returns a deployment with the given name and namespace
func GetDeployment(namespace string, deploymentName string) (*appsv1.Deployment, error) {
	// Get the Kubernetes clientset
	clientSet, err := k8sutil.GetKubernetesClientset()
	if err != nil {
		return nil, err
	}
	deployment, err := clientSet.AppsV1().Deployments(namespace).Get(context.TODO(), deploymentName, metav1.GetOptions{})
	if err != nil {
		Log(Error, fmt.Sprintf("Failed to get Deployment %s from namespace %s: %v ", deploymentName, namespace, err))
		return nil, err
	}
	return deployment, nil
}

// DoesStatefulSetExist returns whether a StatefulSet with the given name and namespace exists for the cluster
func DoesStatefulSetExist(namespace string, name string) (bool, error) {
	statefulsets, err := ListStatefulSets(namespace)
	if err != nil {
		Log(Error, fmt.Sprintf("Failed to list StatefulSets from namespace %s: %v", namespace, err))
		return false, err
	}
	for i := range statefulsets.Items {
		if strings.HasPrefix(statefulsets.Items[i].Name, name) {
			return true, nil
		}
	}
	return false, nil
}

// GetStatefulSet returns a StatefulSet with the given name and namespace
func GetStatefulSet(namespace string, stsName string) (*appsv1.StatefulSet, error) {
	// Get the Kubernetes clientset
	clientSet, err := k8sutil.GetKubernetesClientset()
	if err != nil {
		return nil, err
	}
	sts, err := clientSet.AppsV1().StatefulSets(namespace).Get(context.TODO(), stsName, metav1.GetOptions{})
	if err != nil {
		Log(Error, fmt.Sprintf("Failed to get StatefulSet %s from namespace %s: %v ", stsName, namespace, err))
		return nil, err
	}
	return sts, nil
}

// GetDaemonSet returns a GetDaemonSet with the given name and namespace
func GetDaemonSet(namespace string, daemonSetName string) (*appsv1.DaemonSet, error) {
	// Get the Kubernetes clientset
	clientSet, err := k8sutil.GetKubernetesClientset()
	if err != nil {
		return nil, err
	}
	daemonset, err := clientSet.AppsV1().DaemonSets(namespace).Get(context.TODO(), daemonSetName, metav1.GetOptions{})
	if err != nil {
		Log(Error, fmt.Sprintf("Failed to get DaemonSet %s from namespace %s: %v ", daemonSetName, namespace, err))
		return nil, err
	}
	return daemonset, nil
}

// GetService returns a Service with the given name and namespace
func GetService(namespace string, serviceName string) (*corev1.Service, error) {
	// Get the Kubernetes clientset
	clientSet, err := k8sutil.GetKubernetesClientset()
	if err != nil {
		return nil, err
	}
	svc, err := clientSet.CoreV1().Services(namespace).Get(context.TODO(), serviceName, metav1.GetOptions{})
	if err != nil {
		Log(Error, fmt.Sprintf("Failed to get Service %s from namespace %s: %v ", serviceName, namespace, err))
		return nil, err
	}
	return svc, nil
}

// GetIngressList returns a list of ingresses in the given namespace
func GetIngressList(namespace string) (*netv1.IngressList, error) {
	// Get the Kubernetes clientset
	clientSet, err := k8sutil.GetKubernetesClientset()
	if err != nil {
		return nil, err
	}
	ingressList, err := clientSet.NetworkingV1().Ingresses(namespace).List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		Log(Error, fmt.Sprintf("Failed to get Ingresses in namespace %s: %v ", namespace, err))
		return nil, err
	}
	return ingressList, nil
}

// GetVirtualServiceList returns a list of virtual services in the given namespace
func GetVirtualServiceList(namespace string) (*istionetv1beta1.VirtualServiceList, error) {
	// Get the Istio clientset
	clientSet, err := k8sutil.GetIstioClientset()
	if err != nil {
		return nil, err
	}
	VirtualServiceList, err := clientSet.NetworkingV1beta1().VirtualServices(namespace).List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		Log(Error, fmt.Sprintf("Failed to get Gateways in namespace %s: %v ", namespace, err))
		return nil, err
	}
	return VirtualServiceList, nil
}

// GetCertificateList returns a list of certificates in the given namespace
func GetCertificateList(namespace string) (*certmanagerv1.CertificateList, error) {
	// Get the Cert-manager clientset
	clientSet, err := k8sutil.GetCertManagerClienset()
	if err != nil {
		return nil, err
	}
	certificateList, err := clientSet.Certificates(namespace).List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		Log(Error, fmt.Sprintf("Failed to get Certificates in namespace %s: %v ", namespace, err))
		return nil, err
	}
	return certificateList, nil
}

// GetClusterIssuerList returns a list of cluster issuers
func GetClusterIssuerList() (*certmanagerv1.ClusterIssuerList, error) {
	// Get the Cert-manager clientset
	clientSet, err := k8sutil.GetCertManagerClienset()
	if err != nil {
		return nil, err
	}
	clusterIssuerList, err := clientSet.ClusterIssuers().List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		Log(Error, fmt.Sprintf("Failed to get Cluster Issuers: %v ", err))
		return nil, err
	}
	return clusterIssuerList, nil
}

// GetIssuerList returns a list of cluster issuers
func GetIssuerList(namespace string) (*certmanagerv1.IssuerList, error) {
	// Get the Cert-manager clientset
	clientSet, err := k8sutil.GetCertManagerClienset()
	if err != nil {
		return nil, err
	}
	issuerList, err := clientSet.Issuers(namespace).List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		Log(Error, fmt.Sprintf("Failed to get Issuers in namespace %s: %v ", namespace, err))
		return nil, err
	}
	return issuerList, nil
}

// ListNodes returns the list of nodes for the cluster
func ListNodes() (*corev1.NodeList, error) {
	// Get the Kubernetes clientset
	clientset, err := k8sutil.GetKubernetesClientset()
	if err != nil {
		return nil, err
	}

	nodes, err := clientset.CoreV1().Nodes().List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		Log(Error, fmt.Sprintf("Failed to list nodes: %v", err))
		return nil, err
	}
	return nodes, nil
}

// GetNodeCount returns the number of nodes for the cluster
func GetNodeCount() (uint32, error) {
	nodes, err := ListNodes()
	if err != nil {
		return 0, err
	}
	if len(nodes.Items) < 1 {
		return 0, fmt.Errorf("can not find node in the cluster")
	}
	return uint32(len(nodes.Items)), nil
}

// GetPodsFromSelector returns a collection of pods for the given namespace and selector
func GetPodsFromSelector(selector *metav1.LabelSelector, namespace string) ([]corev1.Pod, error) {
	var pods *corev1.PodList
	var err error
	if selector == nil {
		pods, err = ListPods(namespace, metav1.ListOptions{})
	} else {
		var labelMap map[string]string
		labelMap, err = metav1.LabelSelectorAsMap(selector)
		if err == nil {
			pods, err = ListPods(namespace, metav1.ListOptions{LabelSelector: labels.SelectorFromSet(labelMap).String()})
		}
	}
	if err != nil {
		return nil, err
	}
	return pods.Items, nil
}

// ListPodsInCluster returns the list of pods in a given namespace for the cluster
func ListPodsInCluster(namespace string, clientset *kubernetes.Clientset) (*corev1.PodList, error) {
	return clientset.CoreV1().Pods(namespace).List(context.TODO(), metav1.ListOptions{})
}

// ListPodsWithLabelsInCluster returns the list of pods in a given namespace that matches a specific label for the cluster
func ListPodsWithLabelsInCluster(namespace, labels string, clientset *kubernetes.Clientset) (*corev1.PodList, error) {
	return clientset.CoreV1().Pods(namespace).List(context.TODO(), metav1.ListOptions{LabelSelector: labels})
}

// DoesPodExist returns whether a pod with the given name and namespace exists for the cluster
func DoesPodExist(namespace string, name string) (bool, error) {
	clientset, err := k8sutil.GetKubernetesClientset()
	if err != nil {
		return false, err
	}
	pods, err := ListPodsInCluster(namespace, clientset)
	if err != nil {
		Log(Error, fmt.Sprintf("Failed listing pods in cluster for namespace: %s: %v", namespace, err))
		return false, err
	}
	for i := range pods.Items {
		if strings.HasPrefix(pods.Items[i].Name, name) {
			return true, nil
		}
	}
	return false, nil
}

// GetKubernetesClientsetForCluster returns the Kubernetes clientset for the cluster whose
// kubeconfig path is specified
func GetKubernetesClientsetForCluster(kubeconfigPath string) (*kubernetes.Clientset, error) {
	// use the current context in the kubeconfig
	config, err := k8sutil.GetKubeConfigGivenPath(kubeconfigPath)
	if err != nil {
		return nil, err
	}
	return createClientset(config)
}

// createClientset Creates Kubernetes Clientset for a given kubeconfig
func createClientset(config *restclient.Config) (*kubernetes.Clientset, error) {
	return kubernetes.NewForConfig(config)
}

// GetVerrazzanoManagedClusterClientset returns the Kubernetes clientset for the VerrazzanoManagedCluster
func GetVerrazzanoManagedClusterClientset() (*vmcClient.Clientset, error) {
	config, err := k8sutil.GetKubeConfig()
	if err != nil {
		return nil, err
	}
	return vmcClient.NewForConfig(config)
}

// GetVerrazzanoClientset returns the Kubernetes clientset for the Verrazzano CRD
func GetVerrazzanoClientset() (*vpoClient.Clientset, error) {
	config, err := k8sutil.GetKubeConfig()
	if err != nil {
		return nil, err
	}
	return vpoClient.NewForConfig(config)
}

// GetVerrazzanoProjectClientsetInCluster returns the Kubernetes clientset for the VerrazzanoProject
func GetVerrazzanoProjectClientsetInCluster(kubeconfigPath string) (*vpClient.Clientset, error) {
	config, err := k8sutil.GetKubeConfigGivenPath(kubeconfigPath)
	if err != nil {
		return nil, err
	}
	return vpClient.NewForConfig(config)
}

// GetVerrazzanoApplicationOperatorClientSet returns the Kubernetes clientset for the Verrazzano Application Operator
func GetVerrazzanoApplicationOperatorClientSet() (*vaoClient.Clientset, error) {
	config, err := k8sutil.GetKubeConfig()
	if err != nil {
		return nil, err
	}
	return vaoClient.NewForConfig(config)
}

// GetDynamicClient returns a dynamic client needed to access Unstructured data
func GetDynamicClient() (dynamic.Interface, error) {
	config, err := k8sutil.GetKubeConfig()
	if err != nil {
		return nil, err
	}
	return dynamic.NewForConfig(config)
}

// GetDynamicClientInCluster returns a dynamic client needed to access Unstructured data
func GetDynamicClientInCluster(kubeconfigPath string) (dynamic.Interface, error) {
	return k8sutil.GetDynamicClientInCluster(kubeconfigPath)
}

// GetVerrazzanoInstallResourceInCluster returns the installed Verrazzano CR in the given cluster
// (there should only be 1 per cluster)
func GetVerrazzanoInstallResourceInCluster(kubeconfigPath string) (*v1alpha1.Verrazzano, error) {
	config, err := k8sutil.GetKubeConfigGivenPath(kubeconfigPath)
	if err != nil {
		return nil, err
	}
	client, err := vpoClient.NewForConfig(config)
	if err != nil {
		return nil, err
	}
	vzClient := client.VerrazzanoV1alpha1().Verrazzanos("")
	vzList, err := vzClient.List(context.TODO(), metav1.ListOptions{})

	if err != nil {
		return nil, fmt.Errorf("error listing out Verrazzano instances: %v", err)
	}
	numVzs := len(vzList.Items)
	if numVzs == 0 {
		return nil, fmt.Errorf("did not find installed Verrazzano instance")
	}
	vz := vzList.Items[0]
	return &vz, nil
}

// IsDevProfile returns true if the deployed resource is a 'dev' profile
func IsDevProfile() bool {
	kubeconfigPath, err := k8sutil.GetKubeConfigLocation()
	if err != nil {
		Log(Error, fmt.Sprintf("Error getting kubeconfig: %v", err))
		return false
	}

	vz, err := GetVerrazzanoInstallResourceInCluster(kubeconfigPath)
	if err != nil {
		return false
	}
	if vz.Spec.Profile == v1alpha1.Dev {
		return true
	}
	return false
}

// GetVerrazzano returns the installed Verrazzano
func GetVerrazzano() (*v1alpha1.Verrazzano, error) {
	kubeconfigPath, err := k8sutil.GetKubeConfigLocation()
	if err != nil {
		Log(Error, fmt.Sprintf("Error getting kubeconfig: %v", err))
		return nil, err
	}
	cr, err := GetVerrazzanoInstallResourceInCluster(kubeconfigPath)
	if err != nil {
		return nil, err
	}
	return cr, nil
}

// GetVerrazzanoVersion returns the Verrazzano Version
func GetVerrazzanoVersion(kubeconfigPath string) (string, error) {
	vz, err := GetVerrazzanoInstallResourceInCluster(kubeconfigPath)
	if err != nil {
		return "", err
	}
	vzVer := vz.Spec.Version
	if vzVer == "" {
		vzVer = vz.Status.Version
	}
	return vzVer, nil
}

// IsVerrazzanoMinVersion returns true if the installed Verrazzano version >= minVersion
func IsVerrazzanoMinVersion(minVersion string, kubeconfigPath string) (bool, error) {
	vzVersion, err := GetVerrazzanoVersion(kubeconfigPath)
	if err != nil {
		return false, err
	}
	if len(vzVersion) == 0 {
		return false, nil
	}
	return IsMinVersion(vzVersion, minVersion)
}

// IsMinVersion returns true if the given version >= minVersion
func IsMinVersion(vzVersion, minVersion string) (bool, error) {
	vzSemver, err := semver.NewSemVersion(vzVersion)
	if err != nil {
		return false, err
	}
	minSemver, err := semver.NewSemVersion(minVersion)
	if err != nil {
		return false, err
	}
	return !vzSemver.IsLessThan(minSemver), nil
}

// IsProdProfile returns true if the deployed resource is a 'prod' profile
func IsProdProfile() bool {
	kubeconfigPath, err := k8sutil.GetKubeConfigLocation()
	if err != nil {
		Log(Error, fmt.Sprintf("Error getting kubeconfig: %v", err))
		return false
	}

	vz, err := GetVerrazzanoInstallResourceInCluster(kubeconfigPath)
	if err != nil {
		return false
	}
	if vz.Spec.Profile == v1alpha1.Prod || vz.Spec.Profile == "" {
		return true
	}
	return false
}

// IsManagedClusterProfile returns true if the deployed resource is a 'managed-cluster' profile
func IsManagedClusterProfile() bool {
	kubeconfigPath, err := k8sutil.GetKubeConfigLocation()
	if err != nil {
		Log(Error, fmt.Sprintf("Error getting kubeconfig: %v", err))
		return false
	}

	vz, err := GetVerrazzanoInstallResourceInCluster(kubeconfigPath)
	if err != nil {
		Log(Error, fmt.Sprintf("Error getting vz install resource: %v", err))
		return false
	}
	if vz.Spec.Profile == v1alpha1.ManagedCluster {
		return true
	}
	return false
}

// GetACMEEnvironment returns true if
func GetACMEEnvironment(kubeconfigPath string) (string, error) {
	vz, err := GetVerrazzanoInstallResourceInCluster(kubeconfigPath)
	if err != nil {
		return "", err
	}
	if vz.Spec.Components.CertManager == nil {
		return "", nil
	}
	return vz.Spec.Components.CertManager.Certificate.Acme.Environment, nil
}

// IsCoherenceOperatorEnabled returns true if the COH operator component is not set, or the value of its Enabled field otherwise
func IsCoherenceOperatorEnabled(kubeconfigPath string) bool {
	vz, err := GetVerrazzanoInstallResourceInCluster(kubeconfigPath)
	if err != nil {
		Log(Error, fmt.Sprintf("Error getting kubeconfig: %v", err))
		return true
	}
	if vz.Spec.Components.CoherenceOperator == nil || vz.Spec.Components.CoherenceOperator.Enabled == nil {
		return true
	}
	return *vz.Spec.Components.CoherenceOperator.Enabled
}

// IsWebLogicOperatorEnabled returns true if the WKO operator component is not set, or the value of its Enabled field otherwise
func IsWebLogicOperatorEnabled(kubeconfigPath string) bool {
	vz, err := GetVerrazzanoInstallResourceInCluster(kubeconfigPath)
	if err != nil {
		Log(Error, fmt.Sprintf(verrazzanoErrorTemplate, err))
		return true
	}
	if vz.Spec.Components.WebLogicOperator == nil || vz.Spec.Components.WebLogicOperator.Enabled == nil {
		return true
	}
	return *vz.Spec.Components.WebLogicOperator.Enabled
}

// IsOpenSearchEnabled returns true if the OpenSearch component is not set, or the value of its Enabled field otherwise
func IsOpenSearchEnabled(kubeconfigPath string) (bool, error) {
	vz, err := GetVerrazzanoInstallResourceInCluster(kubeconfigPath)
	if err != nil {
		Log(Error, fmt.Sprintf("Error Verrazzano Resource: %v", err))
		return false, err
	}
	if vz != nil && vz.Spec.Components.Elasticsearch != nil && vz.Spec.Components.Elasticsearch.Enabled != nil {
		return *vz.Spec.Components.Elasticsearch.Enabled, nil
	}
	return false, nil
}

// IsPrometheusAdapterEnabled returns false if the Prometheus Adapter component is not set, or the value of its Enabled field otherwise
func IsPrometheusAdapterEnabled(kubeconfigPath string) bool {
	vz, err := GetVerrazzanoInstallResourceInCluster(kubeconfigPath)
	if err != nil {
		Log(Error, fmt.Sprintf(verrazzanoErrorTemplate, err))
		return false
	}
	if vz.Spec.Components.PrometheusAdapter == nil || vz.Spec.Components.PrometheusAdapter.Enabled == nil {
		return false
	}
	return *vz.Spec.Components.PrometheusAdapter.Enabled
}

// IsPrometheusOperatorEnabled returns false if the Prometheus Operator component is not set, or the value of its Enabled field otherwise
func IsPrometheusOperatorEnabled(kubeconfigPath string) bool {
	vz, err := GetVerrazzanoInstallResourceInCluster(kubeconfigPath)
	if err != nil {
		Log(Error, fmt.Sprintf(verrazzanoErrorTemplate, err))
		return true
	}
	if vz.Spec.Components.PrometheusOperator == nil || vz.Spec.Components.PrometheusOperator.Enabled == nil {
		return true
	}
	return *vz.Spec.Components.PrometheusOperator.Enabled
}

// IsPrometheusEnabled returns true if the Prometheus component is not set and the Prometheus Operator is enabled, or the value of its Enabled field otherwise
func IsPrometheusEnabled(kubeconfigPath string) bool {
	vz, err := GetVerrazzanoInstallResourceInCluster(kubeconfigPath)
	if err != nil {
		Log(Error, fmt.Sprintf(verrazzanoErrorTemplate, err))
		return false
	}
	if vz.Spec.Components.Prometheus == nil || vz.Spec.Components.Prometheus.Enabled == nil {
		return true
	}
	return *vz.Spec.Components.Prometheus.Enabled
}

// IsKubeStateMetricsEnabled returns false if the Kube State Metrics component is not set, or the value of its Enabled field otherwise
func IsKubeStateMetricsEnabled(kubeconfigPath string) bool {
	vz, err := GetVerrazzanoInstallResourceInCluster(kubeconfigPath)
	if err != nil {
		Log(Error, fmt.Sprintf(verrazzanoErrorTemplate, err))
		return false
	}
	if vz.Spec.Components.KubeStateMetrics == nil || vz.Spec.Components.KubeStateMetrics.Enabled == nil {
		return false
	}
	return *vz.Spec.Components.KubeStateMetrics.Enabled
}

// IsPrometheusPushgatewayEnabled returns false if the Prometheus Pushgateway component is not set, or the value of its Enabled field otherwise
func IsPrometheusPushgatewayEnabled(kubeconfigPath string) bool {
	vz, err := GetVerrazzanoInstallResourceInCluster(kubeconfigPath)
	if err != nil {
		Log(Error, fmt.Sprintf(verrazzanoErrorTemplate, err))
		return false
	}
	if vz.Spec.Components.PrometheusPushgateway == nil || vz.Spec.Components.PrometheusPushgateway.Enabled == nil {
		return false
	}
	return *vz.Spec.Components.PrometheusPushgateway.Enabled
}

// IsPrometheusNodeExporterEnabled returns false if the Prometheus Node Exporter component is not set, or the value of its Enabled field otherwise
func IsPrometheusNodeExporterEnabled(kubeconfigPath string) bool {
	vz, err := GetVerrazzanoInstallResourceInCluster(kubeconfigPath)
	if err != nil {
		Log(Error, fmt.Sprintf("Error Verrazzano Resource: %v", err))
		return false
	}
	if vz.Spec.Components.PrometheusNodeExporter == nil || vz.Spec.Components.PrometheusNodeExporter.Enabled == nil {
		return false
	}
	return *vz.Spec.Components.PrometheusNodeExporter.Enabled
}

// IsOpenSearchDashboardsEnabled returns true if the OpenSearchDashboards component is not set, or the value of its Enabled field otherwise
func IsOpenSearchDashboardsEnabled(kubeconfigPath string) bool {
	vz, err := GetVerrazzanoInstallResourceInCluster(kubeconfigPath)
	if err != nil {
		Log(Error, fmt.Sprintf(verrazzanoErrorTemplate, err))
		return true
	}
	if vz != nil && vz.Spec.Components.Kibana != nil && vz.Spec.Components.Kibana.Enabled != nil {
		return *vz.Spec.Components.Kibana.Enabled
	}
	return true
}

// IsJaegerOperatorEnabled returns false if the Jaeger Operator component is not set, or the value of its Enabled field otherwise
func IsJaegerOperatorEnabled(kubeconfigPath string) bool {
	vz, err := GetVerrazzanoInstallResourceInCluster(kubeconfigPath)
	if err != nil {
		Log(Error, fmt.Sprintf("Error Verrazzano Resource: %v", err))
		return false
	}
	if vz == nil || vz.Spec.Components.JaegerOperator == nil || vz.Spec.Components.JaegerOperator.Enabled == nil {
		return false
	}
	return *vz.Spec.Components.JaegerOperator.Enabled
}

// IsGrafanaEnabled returns false if the Grafana component is not set, or the value of its Enabled field otherwise
func IsGrafanaEnabled(kubeconfigPath string) bool {
	vz, err := GetVerrazzanoInstallResourceInCluster(kubeconfigPath)
	if err != nil {
		Log(Error, fmt.Sprintf("Error Verrazzano Resource: %v", err))
		return false
	}
	if vz.Spec.Components.Grafana == nil || vz.Spec.Components.Grafana.Enabled == nil {
		// Grafana component is enabled by default
		return true
	}
	return *vz.Spec.Components.Grafana.Enabled
}

// IsKeycloakEnabled returns false if the Keycloak component is not set, or the value of its Enabled field otherwise
func IsKeycloakEnabled(kubeconfigPath string) bool {
	vz, err := GetVerrazzanoInstallResourceInCluster(kubeconfigPath)
	if err != nil {
		Log(Error, fmt.Sprintf("Error Verrazzano Resource: %v", err))
		return false
	}
	if vz.Spec.Components.Keycloak == nil || vz.Spec.Components.Keycloak.Enabled == nil {
		// Keycloak component is enabled by default
		return true
	}
	return *vz.Spec.Components.Keycloak.Enabled
}

// IsVeleroEnabled returns false if the Velero component is not set, or the value of its Enabled field otherwise
func IsVeleroEnabled(kubeconfigPath string) bool {
	vz, err := GetVerrazzanoInstallResourceInCluster(kubeconfigPath)
	if err != nil {
		Log(Error, fmt.Sprintf("Error Verrazzano Resource: %v", err))
		return false
	}
	if vz.Spec.Components.Velero == nil || vz.Spec.Components.Velero.Enabled == nil {
		return false
	}
	return *vz.Spec.Components.Velero.Enabled
}

// IsRancherBackupEnabled returns false if the Rancher Backup component is not set, or the value of its Enabled field otherwise
func IsRancherBackupEnabled(kubeconfigPath string) bool {
	vz, err := GetVerrazzanoInstallResourceInCluster(kubeconfigPath)
	if err != nil {
		Log(Error, fmt.Sprintf("Error Verrazzano Resource: %v", err))
		return false
	}
	if vz.Spec.Components.RancherBackup == nil || vz.Spec.Components.RancherBackup.Enabled == nil {
		return false
	}
	return *vz.Spec.Components.RancherBackup.Enabled
}

// APIExtensionsClientSet returns a Kubernetes ClientSet for this cluster.
func APIExtensionsClientSet() (*apiextv1.ApiextensionsV1Client, error) {
	config, err := k8sutil.GetKubeConfig()
	if err != nil {
		return nil, err
	}
	// create the clientset
	return apiextv1.NewForConfig(config)
}

// ListServices returns the list of services in a given namespace for the cluster
func ListServices(namespace string) (*corev1.ServiceList, error) {
	// Get the Kubernetes clientset
	clientset, err := k8sutil.GetKubernetesClientset()
	if err != nil {
		return nil, err
	}

	services, err := clientset.CoreV1().Services(namespace).List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		Log(Error, fmt.Sprintf("Failed to list services in namespace %s: %v", namespace, err))
		return nil, err
	}
	return services, nil
}

// GetNamespace returns a namespace
func GetNamespace(name string) (*corev1.Namespace, error) {
	// Get the Kubernetes clientset
	clientset, err := k8sutil.GetKubernetesClientset()
	if err != nil {
		return nil, err
	}
	return GetNamespaceWithClientSet(name, clientset)
}

// GetNamespaceWithClientSet returns a namespace for the given Clientset
func GetNamespaceWithClientSet(name string, clientset *kubernetes.Clientset) (*corev1.Namespace, error) {
	return clientset.CoreV1().Namespaces().Get(context.TODO(), name, metav1.GetOptions{})
}

// GenerateNamespace takes a string and combines that with a UUID to generate a namespace
func GenerateNamespace(name string) string {
	return name + "-" + uuid.NewString()[:7]
}

// GetEffectiveKeyCloakPersistenceOverride returns the effective PVC override for Keycloak, if it exists
func GetEffectiveKeyCloakPersistenceOverride(kubeconfigPath string) (*v1alpha1.VolumeClaimSpecTemplate, error) {
	verrazzano, err := GetVerrazzanoInstallResourceInCluster(kubeconfigPath)
	if err != nil {
		return nil, err
	}

	mysqlVolSource := verrazzano.Spec.DefaultVolumeSource
	if verrazzano.Spec.Components.Keycloak != nil {
		mysqlVolSource = verrazzano.Spec.Components.Keycloak.MySQL.VolumeSource
	}
	if mysqlVolSource == nil || mysqlVolSource.EmptyDir != nil {
		// no override specified, or its an EmptyDir override
		return nil, nil
	}
	for _, template := range verrazzano.Spec.VolumeClaimSpecTemplates {
		if template.Name == mysqlVolSource.PersistentVolumeClaim.ClaimName {
			return &template, nil
		}
	}
	return nil, fmt.Errorf("did not find matching PVC template for %s", mysqlVolSource.PersistentVolumeClaim.ClaimName)
}

// GetEffectiveVMIPersistenceOverride returns the effective PVC override for the VMI components, if it exists
func GetEffectiveVMIPersistenceOverride(kubeconfigPath string) (*v1alpha1.VolumeClaimSpecTemplate, error) {
	verrazzano, err := GetVerrazzanoInstallResourceInCluster(kubeconfigPath)
	if err != nil {
		return nil, err
	}

	volumeOverride := verrazzano.Spec.DefaultVolumeSource
	if volumeOverride == nil || volumeOverride.EmptyDir != nil {
		// no override specified, or its an EmptyDir override
		return nil, nil
	}
	for _, template := range verrazzano.Spec.VolumeClaimSpecTemplates {
		if template.Name == volumeOverride.PersistentVolumeClaim.ClaimName {
			return &template, nil
		}
	}
	return nil, fmt.Errorf("did not find matching PVC template for %s", volumeOverride.PersistentVolumeClaim.ClaimName)
}

// GetNamespaceInCluster returns a namespace in the cluster whose kubeconfigPath is specified
func GetNamespaceInCluster(name string, kubeconfigPath string) (*corev1.Namespace, error) {
	// Get the Kubernetes clientset
	clientset, err := GetKubernetesClientsetForCluster(kubeconfigPath)
	if err != nil {
		return nil, err
	}

	return clientset.CoreV1().Namespaces().Get(context.TODO(), name, metav1.GetOptions{})
}

// CreateNamespace creates a namespace
func CreateNamespace(name string, labels map[string]string) (*corev1.Namespace, error) {
	return CreateNamespaceWithAnnotations(name, labels, nil)
}

func CreateNamespaceWithAnnotations(name string, labels map[string]string, annotations map[string]string) (*corev1.Namespace, error) {
	// Get the Kubernetes clientset
	clientset, err := k8sutil.GetKubernetesClientset()
	if err != nil {
		return nil, err
	}
	return CreateNamespaceWithClientSet(name, labels, clientset, annotations)
}

// CreateNamespaceWithClientSet creates a namespace using the given Clientset
func CreateNamespaceWithClientSet(name string, labels map[string]string, clientset *kubernetes.Clientset, annotations map[string]string) (*corev1.Namespace, error) {
	if len(os.Getenv(k8sutil.EnvVarTestKubeConfig)) > 0 {
		existingNamespace, err := GetNamespaceWithClientSet(name, clientset)
		if err != nil {
			Log(Error, fmt.Sprintf("CreateNamespace %s, error while getting existing namespace: %v", name, err))
			return nil, err
		}

		if existingNamespace != nil && existingNamespace.Name == name {
			return existingNamespace, nil
		}
		return nil, fmt.Errorf("CreateNamespace %s, test is running with custom service account and namespace must be pre-created", name)
	}
	namespace := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name:        name,
			Labels:      labels,
			Annotations: annotations,
		},
	}
	ns, err := clientset.CoreV1().Namespaces().Create(context.TODO(), namespace, metav1.CreateOptions{})
	if err != nil {
		Log(Error, fmt.Sprintf("CreateNamespace %s error: %v", name, err))
		return nil, err
	}
	return ns, nil
}

func RemoveNamespaceFinalizers(namespace *corev1.Namespace) error {
	clientset, err := k8sutil.GetKubernetesClientset()
	if err != nil {
		return err
	}
	namespace.ObjectMeta.Finalizers = nil
	_, err = clientset.CoreV1().Namespaces().Update(context.TODO(), namespace, metav1.UpdateOptions{})
	return err
}

// DeleteNamespace deletes a namespace in the cluster specified in the environment
func DeleteNamespace(name string) error {
	if len(os.Getenv(k8sutil.EnvVarTestKubeConfig)) > 0 {
		Log(Info, fmt.Sprintf("DeleteNamespace %s, test is running with custom service account and therefore namespace won't be deleted by the test", name))
		return nil
	}

	kubeconfigPath, err := k8sutil.GetKubeConfigLocation()
	if err != nil {
		Log(Error, fmt.Sprintf("Error getting kubeconfig: %v", err))
		return err
	}

	return DeleteNamespaceInCluster(name, kubeconfigPath)
}

func DeleteNamespaceInCluster(name string, kubeconfigPath string) error {
	// Get the Kubernetes clientset
	clientset, err := GetKubernetesClientsetForCluster(kubeconfigPath)
	if err != nil {
		return err
	}
	return DeleteNamespaceWithClientSet(name, clientset)
}

// DeleteNamespaceWithClientSet deletes the namespace using the given Clientset
func DeleteNamespaceWithClientSet(name string, clientset *kubernetes.Clientset) error {
	err := clientset.CoreV1().Namespaces().Delete(context.TODO(), name, metav1.DeleteOptions{})
	if err != nil {
		Log(Error, fmt.Sprintf("Failed to delete namespace %s: %v", name, err))
	}
	return err
}

// DoesClusterRoleExist returns whether a cluster role with the given name exists in the cluster
func DoesClusterRoleExist(name string) (bool, error) {
	// Get the Kubernetes clientset
	clientset, err := k8sutil.GetKubernetesClientset()
	if err != nil {
		return false, err
	}

	clusterrole, err := clientset.RbacV1().ClusterRoles().Get(context.TODO(), name, metav1.GetOptions{})
	if err != nil && !k8serrors.IsNotFound(err) {
		Log(Error, fmt.Sprintf("Failed to get cluster role %s: %v", name, err))
		return false, err
	}

	return clusterrole != nil, nil
}

// GetClusterRole returns the cluster role with the given name
func GetClusterRole(name string) (*rbacv1.ClusterRole, error) {
	// Get the Kubernetes clientset
	clientset, err := k8sutil.GetKubernetesClientset()
	if err != nil {
		return nil, err
	}

	clusterrole, err := clientset.RbacV1().ClusterRoles().Get(context.TODO(), name, metav1.GetOptions{})
	if err != nil {
		Log(Error, fmt.Sprintf("Failed to get cluster role %s: %v", name, err))
		return nil, err
	}

	return clusterrole, nil
}

// DoesServiceAccountExist returns whether a service account with the given name and namespace exists in the cluster
func DoesServiceAccountExist(namespace, name string) (bool, error) {
	sa, err := GetServiceAccount(namespace, name)
	if err != nil {
		return false, err
	}
	return sa != nil, nil
}

// DoesClusterRoleBindingExist returns whether a cluster role with the given name exists in the cluster
func DoesClusterRoleBindingExist(name string) (bool, error) {
	// Get the Kubernetes clientset
	clientset, err := k8sutil.GetKubernetesClientset()
	if err != nil {
		return false, err
	}

	clusterrolebinding, err := clientset.RbacV1().ClusterRoleBindings().Get(context.TODO(), name, metav1.GetOptions{})
	if err != nil && !k8serrors.IsNotFound(err) {
		Log(Error, fmt.Sprintf("Failed to get cluster role binding %s: %v", name, err))
		return false, err
	}

	return clusterrolebinding != nil && len(clusterrolebinding.Name) > 0, nil
}

// GetClusterRoleBinding returns the cluster role with the given name
func GetClusterRoleBinding(name string) (*rbacv1.ClusterRoleBinding, error) {
	// Get the Kubernetes clientset
	clientset, err := k8sutil.GetKubernetesClientset()
	if err != nil {
		return nil, err
	}

	crb, err := clientset.RbacV1().ClusterRoleBindings().Get(context.TODO(), name, metav1.GetOptions{})
	if err != nil {
		Log(Error, fmt.Sprintf("Failed to get cluster role binding %s: %v", name, err))
		return nil, err
	}

	return crb, err
}

// ListClusterRoleBindings returns the list of cluster role bindings for the cluster
func ListClusterRoleBindings() (*rbacv1.ClusterRoleBindingList, error) {
	// Get the Kubernetes clientset
	clientset, err := k8sutil.GetKubernetesClientset()
	if err != nil {
		return nil, err
	}

	bindings, err := clientset.RbacV1().ClusterRoleBindings().List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		Log(Error, fmt.Sprintf("Failed to get cluster role bindings: %v", err))
		return nil, err
	}

	return bindings, err
}

// DoesRoleBindingContainSubject returns true if the RoleBinding exists and it contains the
// specified subject
func DoesRoleBindingContainSubject(namespace, name, subjectKind, subjectName string) (bool, error) {
	clientset, err := k8sutil.GetKubernetesClientset()
	if err != nil {
		return false, err
	}

	rb, err := clientset.RbacV1().RoleBindings(namespace).Get(context.TODO(), name, metav1.GetOptions{})
	if err != nil {
		if !k8serrors.IsNotFound(err) {
			Log(Error, fmt.Sprintf("Failed to get RoleBinding %s in namespace %s: %v", name, namespace, err))
			return false, err
		}
		return false, nil
	}

	for _, s := range rb.Subjects {
		if s.Kind == subjectKind && s.Name == subjectName {
			return true, nil
		}
	}
	return false, nil
}

func CreateRoleBinding(userOCID string, namespace string, rolebindingname string, clusterrolename string) error {

	subject1 := rbacv1.Subject{
		Kind:      "User",
		APIGroup:  "rbac.authorization.k8s.io",
		Name:      userOCID,
		Namespace: "",
	}
	subjects := []rbacv1.Subject{0: subject1}

	rb := rbacv1.RoleBinding{
		TypeMeta: metav1.TypeMeta{
			Kind:       "RoleBinding",
			APIVersion: "rbac.authorization.k8s.io/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: rolebindingname,
		},
		Subjects: subjects,
		RoleRef: rbacv1.RoleRef{
			APIGroup: "rbac.authorization.k8s.io",
			Kind:     "ClusterRole",
			Name:     clusterrolename,
		},
	}

	// Get the Kubernetes clientset
	clientset, err := k8sutil.GetKubernetesClientset()
	if err != nil {
		return err
	}

	_, err = clientset.RbacV1().RoleBindings(namespace).Create(context.TODO(), &rb, metav1.CreateOptions{})
	if err != nil {
		Log(Info, fmt.Sprintf("Failed to create role binding: %v", err))
	}

	return err
}

// DoesRoleBindingExist returns whether a cluster role with the given name exists in the cluster
func DoesRoleBindingExist(name string, namespace string) (bool, error) {
	// Get the Kubernetes clientset
	clientset, err := k8sutil.GetKubernetesClientset()
	if err != nil {
		return false, err
	}

	rolebinding, err := clientset.RbacV1().RoleBindings(namespace).Get(context.TODO(), name, metav1.GetOptions{})
	if err != nil {
		Log(Info, fmt.Sprintf("Failed to verify role binding %s in namespace %s: %v", name, namespace, err))
		return false, err
	}

	return rolebinding != nil, nil
}

// Execute executes the given command on the pod and container identified by the given names and returns the
// resulting stdout and stderr
func Execute(podName, containerName, namespace string, command []string) (string, string, error) {
	clientset, err := k8sutil.GetKubernetesClientset()
	if err != nil {
		return "", "", err
	}
	request := clientset.CoreV1().RESTClient().Post().Resource("pods").Name(podName).
		Namespace(namespace).SubResource("exec")
	request.VersionedParams(
		&corev1.PodExecOptions{
			Command:   command,
			Container: containerName,
			Stdin:     false,
			Stdout:    true,
			Stderr:    true,
			TTY:       false,
		},
		scheme.ParameterCodec,
	)
	client, err := k8sutil.GetKubeConfig()
	if err != nil {
		return "", "", err
	}
	executor, err := remotecommand.NewSPDYExecutor(client, "POST", request.URL())
	if err != nil {
		return "", "", err
	}
	var stdout, stderr bytes.Buffer
	err = executor.Stream(remotecommand.StreamOptions{Stdout: &stdout, Stderr: &stderr})

	return strings.TrimSpace(stdout.String()), strings.TrimSpace(stderr.String()), err
}

// GetConfigMap returns the config map for the passed in ConfigMap Name and Namespace
func GetConfigMap(configMapName string, namespace string) (*corev1.ConfigMap, error) {
	// Get the Kubernetes clientset
	clientset, err := k8sutil.GetKubernetesClientset()
	if err != nil {
		return nil, err
	}
	cmi := clientset.CoreV1().ConfigMaps(namespace)
	configMap, err := cmi.Get(context.TODO(), configMapName, metav1.GetOptions{})
	if err != nil {
		Log(Error, fmt.Sprintf("Failed to get Config Map %s from namespace %s: %v ", configMapName, namespace, err))
		return nil, err
	}
	return configMap, nil
}

/*
The following code adds http headers to the kubernetes client invocations.  This is done to emulate the functionality of
kubectl auth can-i ...

WrapTransport is configured to point to the function
WrapTransport will be invoked for custom HTTP behavior after the underlying transport is initialized
(either the transport created from TLSClientConfig, Transport, or http.DefaultTransport).
The config may layer other RoundTrippers on top of the returned RoundTripper.

WrapperFunc wraps an http.RoundTripper when a new transport is created for a client, allowing per connection behavior to be injected.

RoundTripper is an interface representing the ability to execute a single HTTP transaction, obtaining the Response for a given Request.
*/
// headerAdder is an http.RoundTripper that adds additional headers to the request
type headerAdder struct {
	headers map[string][]string

	rt http.RoundTripper
}

func (h *headerAdder) RoundTrip(req *http.Request) (*http.Response, error) {
	for k, vv := range h.headers {
		for _, v := range vv {
			req.Header.Add(k, v)
		}
	}
	return h.rt.RoundTrip(req)
}

func CanI(userOCID string, namespace string, verb string, resource string) (bool, string, error) {
	return CanIForAPIGroup(userOCID, namespace, verb, resource, "")
}

func CanIForAPIGroup(userOCID string, namespace string, verb string, resource string, group string) (bool, string, error) {
	return CanIForAPIGroupForServiceAccountOrUser(userOCID, namespace, verb, resource, group, false, "")
}

func CanIForAPIGroupForServiceAccountOrUser(saOrUserOCID string, namespace string, verb string, resource string, group string, isServiceAccount bool, saNamespace string) (bool, string, error) {
	canI := &v1.SelfSubjectAccessReview{
		TypeMeta: metav1.TypeMeta{
			Kind:       "SelfSubjectAccessReview",
			APIVersion: "authorization.k8s.io/v1",
		},
		Spec: v1.SelfSubjectAccessReviewSpec{
			ResourceAttributes: &v1.ResourceAttributes{
				Namespace:   namespace,
				Verb:        verb,
				Group:       group,
				Version:     "",
				Resource:    resource,
				Subresource: "",
				Name:        "",
			},
		},
	}

	config, err := k8sutil.GetKubeConfig()
	if err != nil {
		return false, "", err
	}

	wt := config.WrapTransport // Config might already have a transport wrapper
	if isServiceAccount {
		token, err := GetTokenForServiceAccount(saOrUserOCID, saNamespace)
		if err != nil {
			return false, "", err
		}

		kubeconfigPath, err := k8sutil.GetKubeConfigLocation()
		if err != nil {
			Log(Error, fmt.Sprintf("Error getting kubeconfig, error: %v", err))
			return false, "", err
		}

		clientConfig := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
			&clientcmd.ClientConfigLoadingRules{ExplicitPath: kubeconfigPath},
			&clientcmd.ConfigOverrides{ClusterInfo: clientcmdapi.Cluster{Server: ""}})
		rawConfig, err := clientConfig.RawConfig()
		if err != nil {
			return false, "", fmt.Errorf("could not get rawconfig: %v", err)
		}

		rawConfig.AuthInfos["sa-token"] = &clientcmdapi.AuthInfo{Token: string(token)}
		cluster := ""
		if len(rawConfig.Clusters) > 0 {
			for k, v := range rawConfig.Clusters {
				if v != nil {
					cluster = k
					break
				}
			}
		}

		rawConfig.Contexts["sa-context"] = &clientcmdapi.Context{Cluster: cluster, AuthInfo: "sa-token"}
		rawConfig.CurrentContext = "sa-context"
		config, err = clientcmd.NewNonInteractiveClientConfig(rawConfig, "sa-context", &clientcmd.ConfigOverrides{ClusterInfo: clientcmdapi.Cluster{Server: ""}}, clientConfig.ConfigAccess()).ClientConfig()
		if err != nil {
			return false, "", err
		}

	} else {
		config.WrapTransport = func(rt http.RoundTripper) http.RoundTripper {
			if wt != nil {
				rt = wt(rt)
			}
			header := &headerAdder{
				rt:      rt,
				headers: map[string][]string{"Impersonate-User": {saOrUserOCID}},
			}
			return header
		}
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return false, "", err
	}

	auth, err := clientset.AuthorizationV1().SelfSubjectAccessReviews().Create(context.TODO(), canI, metav1.CreateOptions{})
	if err != nil {
		Log(Error, fmt.Sprintf("Failed to check perms: %v", err))
		return false, "", err
	}

	return auth.Status.Allowed, auth.Status.Reason, nil
}

// GetTokenForServiceAccount returns the token associated with service account
func GetTokenForServiceAccount(sa string, namespace string) ([]byte, error) {
	serviceAccount, err := GetServiceAccount(namespace, sa)
	if err != nil {
		return nil, err
	}
	if len(serviceAccount.Secrets) == 0 {
		msg := fmt.Sprintf("no secrets present in service account %s in namespace %s", sa, namespace)
		Log(Error, msg)
		return nil, errors.New(msg)
	}

	secretName := serviceAccount.Secrets[0].Name
	clientset, err := k8sutil.GetKubernetesClientset()
	if err != nil {
		return nil, err
	}
	secret, err := clientset.CoreV1().Secrets(namespace).Get(context.TODO(), secretName, metav1.GetOptions{})
	if err != nil {
		msg := fmt.Sprintf("failed to get secret %s for service account %s in namespace %s: %v", secretName, sa, namespace, err)
		Log(Error, msg)
		return nil, errors.New(msg)
	}

	token, ok := secret.Data["token"]

	if !ok {
		msg := fmt.Sprintf("no token present in secret %s for service account %s in namespace %s: %v", secretName, sa, namespace, err)
		Log(Error, msg)
		return nil, errors.New(msg)
	}

	return token, nil
}

func GetServiceAccount(namespace, name string) (*corev1.ServiceAccount, error) {
	clientset, err := k8sutil.GetKubernetesClientset()
	if err != nil {
		return nil, err
	}
	sa, err := clientset.CoreV1().ServiceAccounts(namespace).Get(context.TODO(), name, metav1.GetOptions{})
	if err != nil {
		Log(Error, fmt.Sprintf("Failed to get service account %s in namespace %s: %v", name, namespace, err))
		return nil, err
	}
	return sa, nil
}

func GetPersistentVolumeClaims(namespace string) (map[string]*corev1.PersistentVolumeClaim, error) {
	clientset, err := k8sutil.GetKubernetesClientset()
	if err != nil {
		return nil, err
	}
	pvcs, err := clientset.CoreV1().PersistentVolumeClaims(namespace).List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	volumeClaims := make(map[string]*corev1.PersistentVolumeClaim)

	for i, pvc := range pvcs.Items {
		volumeClaims[pvc.Name] = &pvcs.Items[i]
	}
	return volumeClaims, nil
}

// DoesVerrazzanoProjectExistInCluster returns whether a VerrazzanoProject with the given name exists in the specified cluster
func DoesVerrazzanoProjectExistInCluster(name string, kubeconfigPath string) (bool, error) {
	// Get the clientset
	clientset, err := GetVerrazzanoProjectClientsetInCluster(kubeconfigPath)
	if err != nil {
		return false, err
	}

	vp, err := clientset.ClustersV1alpha1().VerrazzanoProjects("verrazzano-mc").Get(context.TODO(), name, metav1.GetOptions{})
	if err != nil && !k8serrors.IsNotFound(err) {
		Log(Error, fmt.Sprintf("Failed to get VerrazzanoProject %s: %v", name, err))
		return false, err
	}

	return vp != nil && len(vp.Name) > 0, nil
}

// ContainerHasExpectedArgs returns true if each of the arguments matches a substring of one of the arguments found in the deployment
func ContainerHasExpectedArgs(namespace string, deploymentName string, containerName string, arguments []string) (bool, error) {
	deployment, err := GetDeployment(namespace, deploymentName)
	if err != nil {
		Log(Error, fmt.Sprintf("Deployment %v is not found in the namespace: %v, error: %v", deploymentName, namespace, err))
		return false, nil
	}
	for _, container := range deployment.Spec.Template.Spec.Containers {
		if container.Name == containerName {
			return SlicesContainSubsetSubstring(arguments, container.Args), nil
		}
	}
	return false, nil
}

// UpdateConfigMap updates the config map
func UpdateConfigMap(configMap *corev1.ConfigMap) error {
	// Get the Kubernetes clientset
	clientset, err := k8sutil.GetKubernetesClientset()
	if err != nil {
		return err
	}

	cmi := clientset.CoreV1().ConfigMaps(configMap.GetNamespace())
	_, err = cmi.Update(context.TODO(), configMap, metav1.UpdateOptions{})
	if err != nil {
		Log(Error, fmt.Sprintf("Failed to update Config Map %s from namespace %s: %v ", configMap.GetName(), configMap.GetNamespace(), err))
		return err
	}
	return nil
}

// GetContainerEnv returns an array of environment variables in the specified container for the specified deployment
func GetContainerEnv(namespace string, deploymentName string, containerName string) ([]corev1.EnvVar, error) {
	deployment, err := GetDeployment(namespace, deploymentName)
	if err != nil {
		return nil, fmt.Errorf("deployment %s not found in the namespace: %s, error: %v", deploymentName, namespace, err)
	}
	for _, container := range deployment.Spec.Template.Spec.Containers {
		if container.Name == containerName {
			return container.Env, nil
		}
	}
	return nil, fmt.Errorf("container %s not found in the namespace: %s", containerName, namespace)
}

func GetDeploymentLabelSelector(namespace, deploymentName string) (*metav1.LabelSelector, error) {
	deployment, err := GetDeployment(namespace, deploymentName)
	if err != nil {
		Log(Error, fmt.Sprintf("Deployment %v not found in the namespace: %v, error: %v", deploymentName, namespace, err))
		return nil, fmt.Errorf("deployment %s not found in the namespace: %s, error: %v", deploymentName, namespace, err)
	}
	return deployment.Spec.Selector, err
}

// GetContainerImage returns the image used by the specified container for the specified deployment
func GetContainerImage(namespace string, deploymentName string, containerName string) (string, error) {
	deployment, err := GetDeployment(namespace, deploymentName)
	if err != nil {
		Log(Error, fmt.Sprintf("Deployment %v not found in the namespace: %v, error: %v", deploymentName, namespace, err))
		return "", nil
	}
	for _, container := range deployment.Spec.Template.Spec.Containers {
		if container.Name == containerName {
			return container.Image, nil
		}
	}
	return "", fmt.Errorf("container %v not found in the namespace: %v", containerName, namespace)
}

// WaitForVZCondition waits till the VZ CR reaches the given condition
func WaitForVZCondition(conditionType v1alpha1.ConditionType, pollingInterval, timeout time.Duration) {
	gomega.Eventually(func() bool {
		cr, err := GetVerrazzano()
		if err != nil {
			Log(Error, err.Error())
			return false
		}
		for _, condition := range cr.Status.Conditions {
			Log(Info, fmt.Sprintf("Evaluating condition: [%s - %s]", condition.Type, condition.Status))
			if condition.Type == conditionType && condition.Status == corev1.ConditionTrue {
				return true
			}
		}
		return false
	}).WithPolling(pollingInterval).WithTimeout(timeout).Should(gomega.BeTrue())
}

// DeleteConfigMap to delete the ConfigMap with the given name and namespace
func DeleteConfigMap(namespace string, name string) error {
	clientset, err := k8sutil.GetKubernetesClientset()
	if err != nil {
		return err
	}
	return clientset.CoreV1().ConfigMaps(namespace).Delete(context.TODO(), name, metav1.DeleteOptions{})
}

// CreateConfigMap creates the ConfigMap
func CreateConfigMap(configMap *corev1.ConfigMap) error {
	clientset, err := k8sutil.GetKubernetesClientset()
	if err != nil {
		return err
	}
	_, err = clientset.CoreV1().ConfigMaps(configMap.Namespace).Create(context.TODO(), configMap, metav1.CreateOptions{})
	if err != nil {
		return err
	}
	return nil
}
