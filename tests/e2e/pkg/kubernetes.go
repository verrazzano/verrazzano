// Copyright (c) 2020, 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package pkg

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	vmcClient "github.com/verrazzano/verrazzano/platform-operator/clients/clusters/clientset/versioned"
	vpoClient "github.com/verrazzano/verrazzano/platform-operator/clients/verrazzano/clientset/versioned"

	"k8s.io/api/authorization/v1beta1"
	rbacv1 "k8s.io/api/rbac/v1"

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

// GetKubeConfig will get the kubeconfig from the given kubeconfigPath
func GetKubeConfigGivenPath(kubeconfigPath string) *restclient.Config {
	return buildKubeConfig(kubeconfigPath)
}

// GetKubeConfig will get the kubeconfig from the TEST_KUBECONFIG env var if set, then the env var KUBECONFIG, if set,
// or else from $HOME/.kube/config
func GetKubeConfig() *restclient.Config {
	kubeconfig := GetKubeConfigPathFromEnv()

	return buildKubeConfig(kubeconfig)
}

func buildKubeConfig(kubeconfig string) *restclient.Config {
	var err error
	config, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
	if err != nil {
		ginkgo.Fail("Could not get current context from kubeconfig " + kubeconfig)
	}

	return config
}

// GetKubeConfigPathFromEnv returns the path to the default kubernetes config file in use (from
// the TEST_KUBECONFIG env var if set, then the env var KUBECONFIG, if set, or else from $HOME/.kube/config
func GetKubeConfigPathFromEnv() string {
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
	return kubeconfig
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

// DoesNamespaceExist returns whether a namespace with the given name exists for the cluster set in the environment
func DoesNamespaceExist(name string) bool {
	return DoesNamespaceExistInCluster(name, GetKubeConfigPathFromEnv())
}

// DoesNamespaceExistInCluster returns whether a namespace with the given name exists in the specified cluster
func DoesNamespaceExistInCluster(name string, kubeconfigPath string) bool {
	// Get the Kubernetes clientset
	clientset := GetKubernetesClientsetForCluster(kubeconfigPath)

	namespace, err := clientset.CoreV1().Namespaces().Get(context.TODO(), name, metav1.GetOptions{})
	if err != nil && !errors.IsNotFound(err) {
		ginkgo.Fail(fmt.Sprintf("Failed to get namespace %s with error: %v", name, err))
	}

	return namespace != nil
}

// ListNamespaces returns whether a namespace with the given name exists for the cluster
func ListNamespaces() *corev1.NamespaceList {
	// Get the Kubernetes clientset
	clientset := GetKubernetesClientset()

	namespaces, err := clientset.CoreV1().Namespaces().List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		ginkgo.Fail(fmt.Sprintf("Failed to get namespaces with error: %v", err))
	}

	return namespaces
}

// DoesJobExist returns whether a job with the given name and namespace exists for the cluster
func DoesJobExist(namespace string, name string) bool {
	// Get the Kubernetes clientset
	clientset := GetKubernetesClientset()

	job, err := clientset.BatchV1().Jobs(namespace).Get(context.TODO(), name, metav1.GetOptions{})
	if err != nil {
		ginkgo.Fail(fmt.Sprintf("Failed to get job %s in namespace %s with error: %v", name, namespace, err))
	}

	return job != nil
}

// ListDeployments returns the list of deployments in a given namespace for the cluster
func ListDeployments(namespace string) *appsv1.DeploymentList {
	// Get the Kubernetes clientset
	clientset := GetKubernetesClientset()

	deployments, err := clientset.AppsV1().Deployments(namespace).List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		ginkgo.Fail(fmt.Sprintf("Failed to list deployments in namespace %s with error: %v", namespace, err))
	}
	return deployments
}

// ListNodes returns the list of nodes for the cluster
func ListNodes() *corev1.NodeList {
	// Get the Kubernetes clientset
	clientset := GetKubernetesClientset()

	nodes, err := clientset.CoreV1().Nodes().List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		ginkgo.Fail(fmt.Sprintf("Failed to list nodes with error: %v", err))
	}
	return nodes
}

// ListPodsInCluster returns the list of pods in a given namespace for the cluster
func ListPodsInCluster(namespace string, clientset *kubernetes.Clientset) *corev1.PodList {
	pods, err := clientset.CoreV1().Pods(namespace).List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		ginkgo.Fail(fmt.Sprintf("Failed to list pods in namespace %s with error: %v", namespace, err))
	}
	return pods
}

// GetVerrazzanoMonitoringInstance returns the a Verrazzano monitoring instance in a given namespace for the cluster
func GetVerrazzanoMonitoringInstance(namespace string, name string) (*vmov1.VerrazzanoMonitoringInstance, error) {
	// Get the Kubernetes clientset
	clientset := GetVMOClientset()

	vmi, err := clientset.VerrazzanoV1().VerrazzanoMonitoringInstances(namespace).Get(context.TODO(), name, metav1.GetOptions{})
	if err != nil && !errors.IsNotFound(err) {
		ginkgo.Fail(fmt.Sprintf("Failed to get Verrazzano monitoring instance %s in namespace %s with error: %v", name, namespace, err))
	}
	return vmi, err
}

// DoesPodExist returns whether a pod with the given name and namespace exists for the cluster
func DoesPodExist(namespace string, name string) bool {
	clientset := GetKubernetesClientset()
	pods := ListPodsInCluster(namespace, clientset)
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

// GetKubernetesClientset returns the Kubernetes clientset for the cluster set in the environment
func GetKubernetesClientset() *kubernetes.Clientset {
	// use the current context in the kubeconfig
	config := GetKubeConfig()

	return createClientset(config)
}

// GetKubernetesClientsetForCluster returns the Kubernetes clientset for the cluster whose
// kubeconfig path is specified
func GetKubernetesClientsetForCluster(kubeconfigPath string) *kubernetes.Clientset {
	// use the current context in the kubeconfig
	config := GetKubeConfigGivenPath(kubeconfigPath)
	return createClientset(config)
}

// createClientset Creates Kubernetes Clientset for a given kubeconfig
func createClientset(config *restclient.Config) *kubernetes.Clientset {
	// create the clientset once and cache it
	var err error
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		ginkgo.Fail("Could not get Kubernetes clientset")
	}

	return clientset
}

// GetVMOClientset returns the Kubernetes clientset for the Verrazzano Monitoring Operator
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

// GetVerrazzanoManagedClusterClientset returns the Kubernetes clientset for the VerrazzanoManagedCluster
func GetVerrazzanoManagedClusterClientset() *vmcClient.Clientset {
	client, err := vmcClient.NewForConfig(GetKubeConfig())
	if err != nil {
		ginkgo.Fail("Could not get Verrazzano Platform Operator clientset")
	}
	return client
}

// getPlatformOperatorClientsetForCluster returns the Kubernetes clientset for the Verrazzano Platform Operator
func getPlatformOperatorClientsetForCluster(kubeconfigPath string) *vpoClient.Clientset {
	client, err := vpoClient.NewForConfig(GetKubeConfigGivenPath(kubeconfigPath))
	if err != nil {
		ginkgo.Fail("Could not get Verrazzano Platform Operator clientset")
	}
	return client
}

// GetVerrazzanoInstallResourceInCluster returns the installed Verrazzano CR in the given cluster
// (there should only be 1 per cluster)
func GetVerrazzanoInstallResourceInCluster(kubeconfigPath string) *v1alpha1.Verrazzano {
	vzClient := getPlatformOperatorClientsetForCluster(kubeconfigPath).VerrazzanoV1alpha1().Verrazzanos("")
	vzList, err := vzClient.List(context.TODO(), metav1.ListOptions{})

	if err != nil {
		ginkgo.Fail(fmt.Sprintf("Error listing out Verrazzano instances: %v", err))
	}
	numVzs := len(vzList.Items)
	if numVzs == 0 {
		ginkgo.Fail("Did not find installed verrazzano instance")
	}
	if numVzs > 1 {
		ginkgo.Fail(fmt.Sprintf("Found more than one Verrazzano instance installed: %v", numVzs))
	}
	vz := vzList.Items[0]
	return &vz
}

// IsDevProfile returns true if the deployed resource is a Dev profile
func IsDevProfile() bool {
	return GetVerrazzanoInstallResourceInCluster(GetKubeConfigPathFromEnv()).Spec.Profile == v1alpha1.Dev
}

// IsProdProfile returns true if the deployed resource is a 'prod' profile
func IsProdProfile() bool {
	return GetVerrazzanoInstallResourceInCluster(GetKubeConfigPathFromEnv()).Spec.Profile == v1alpha1.Prod
}

// IsManagedClusterProfile returns true if the deployed resource is a 'managed-cluster' profile
func IsManagedClusterProfile() bool {
	return GetVerrazzanoInstallResourceInCluster(GetKubeConfigPathFromEnv()).Spec.Profile == v1alpha1.ManagedCluster
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
	// Get the Kubernetes clientset
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
	// Get the Kubernetes clientset
	clientset := GetKubernetesClientset()

	return clientset.CoreV1().Namespaces().Get(context.TODO(), name, metav1.GetOptions{})
}

// GetNamespaceInCluster returns a namespace in the cluster whose kubeconfigPath is specified
func GetNamespaceInCluster(name string, kubeconfigPath string) (*corev1.Namespace, error) {
	// Get the Kubernetes clientset
	clientset := GetKubernetesClientsetForCluster(kubeconfigPath)

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

	// Get the Kubernetes clientset
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

// DeleteNamespace deletes a namespace in the cluster specified in the environment
func DeleteNamespace(name string) error {
	if len(os.Getenv("TEST_KUBECONFIG")) > 0 {
		Log(Info, fmt.Sprintf("DeleteNamespace %s, test is running with custom service account and therefore namespace won't be deleted by the test", name))
		return nil
	}

	return DeleteNamespaceInCluster(name, GetKubeConfigPathFromEnv())
}

func DeleteNamespaceInCluster(name string, kubeconfigPath string) error {
	// Get the Kubernetes clientset
	clientset := GetKubernetesClientsetForCluster(kubeconfigPath)
	err := clientset.CoreV1().Namespaces().Delete(context.TODO(), name, metav1.DeleteOptions{})
	if err != nil {
		Log(Error, fmt.Sprintf("DeleteNamespace %s error: %v", name, err))
	}

	return err
}

// DoesClusterRoleExist returns whether a cluster role with the given name exists in the cluster
func DoesClusterRoleExist(name string) bool {
	// Get the Kubernetes clientset
	clientset := GetKubernetesClientset()

	clusterrole, err := clientset.RbacV1().ClusterRoles().Get(context.TODO(), name, metav1.GetOptions{})
	if err != nil {
		ginkgo.Fail(fmt.Sprintf("Failed to get cluster role %s with error: %v", name, err))
	}

	return clusterrole != nil
}

// GetClusterRole returns the cluster role with the given name
func GetClusterRole(name string) *rbacv1.ClusterRole {
	// Get the Kubernetes clientset
	clientset := GetKubernetesClientset()

	clusterrole, err := clientset.RbacV1().ClusterRoles().Get(context.TODO(), name, metav1.GetOptions{})
	if err != nil {
		ginkgo.Fail(fmt.Sprintf("Failed to get cluster role %s with error: %v", name, err))
	}

	return clusterrole
}

//DoesServiceAccountExist returns whether a service account with the given name and namespace exists in the cluster
func DoesServiceAccountExist(namespace, name string) bool {
	sa := GetServiceAccount(namespace, name)
	return sa != nil
}

//GetServiceAccount returns a service account with the given name and namespace
func GetServiceAccount(namespace, name string) *corev1.ServiceAccount {
	clientset := GetKubernetesClientset()

	sa, err := clientset.CoreV1().ServiceAccounts(namespace).Get(context.TODO(), name, metav1.GetOptions{})

	if err != nil {
		ginkgo.Fail(fmt.Sprintf("Failed to get service account %s in namespace %s with error: %v", name, namespace, err))
	}

	return sa

}

// DoesClusterRoleBindingExist returns whether a cluster role with the given name exists in the cluster
func DoesClusterRoleBindingExist(name string) bool {
	// Get the Kubernetes clientset
	clientset := GetKubernetesClientset()

	clusterrolebinding, err := clientset.RbacV1().ClusterRoleBindings().Get(context.TODO(), name, metav1.GetOptions{})
	if err != nil {
		ginkgo.Fail(fmt.Sprintf("Failed to get cluster role binding %s with error: %v", name, err))
	}

	return clusterrolebinding != nil
}

// GetClusterRoleBinding returns the cluster role with the given name
func GetClusterRoleBinding(name string) *rbacv1.ClusterRoleBinding {
	// Get the Kubernetes clientset
	clientset := GetKubernetesClientset()

	crb, err := clientset.RbacV1().ClusterRoleBindings().Get(context.TODO(), name, metav1.GetOptions{})
	if err != nil {
		ginkgo.Fail(fmt.Sprintf("Failed to get cluster role binding %s with error: %v", name, err))
	}

	return crb
}

// DoesRoleBindingContainSubject returns true if the RoleBinding exists and it contains the
// specified subject
func DoesRoleBindingContainSubject(namespace, name, subjectKind, subjectName string) bool {
	clientset := GetKubernetesClientset()

	rb, err := clientset.RbacV1().RoleBindings(namespace).Get(context.TODO(), name, metav1.GetOptions{})
	if err != nil {
		if !errors.IsNotFound(err) {
			ginkgo.Fail(fmt.Sprintf("Failed to get RoleBinding %s in namespace %s: %v", name, namespace, err))
		}
		return false
	}

	for _, s := range rb.Subjects {
		if s.Kind == subjectKind && s.Name == subjectName {
			return true
		}
	}
	return false
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
	clientset := GetKubernetesClientset()

	_, err := clientset.RbacV1().RoleBindings(namespace).Create(context.TODO(), &rb, metav1.CreateOptions{})
	if err != nil {
		Log(Info, fmt.Sprintf("Failed to create role binding: %v", err))
	}

	return err
}

// DoesClusterRoleBindingExist returns whether a cluster role with the given name exists in the cluster
func DoesRoleBindingExist(name string, namespace string) bool {
	// Get the Kubernetes clientset
	clientset := GetKubernetesClientset()

	rolebinding, err := clientset.RbacV1().RoleBindings(namespace).Get(context.TODO(), name, metav1.GetOptions{})
	if err != nil {
		Log(Info, fmt.Sprintf("Failed to verify role binding %s in namespace %s with error: %v", name, namespace, err))
	}

	return rolebinding != nil
}

// GetIstioClientset returns the clientset object for Istio
func GetIstioClientset() *istioClient.Clientset {
	cs, err := istioClient.NewForConfig(GetKubeConfig())
	if err != nil {
		ginkgo.Fail(fmt.Sprintf("Failed to get Istio clientset: %v", err))
	}
	return cs
}

// GetConfigMap returns the config map for the passed in ConfigMap Name and Namespace
func GetConfigMap(configMapName string, namespace string) *corev1.ConfigMap {
	// Get the Kubernetes clientset
	clientset := GetKubernetesClientset()
	cmi := clientset.CoreV1().ConfigMaps(namespace)
	configMap, err := cmi.Get(context.TODO(), configMapName, metav1.GetOptions{})
	if err != nil {
		ginkgo.Fail(fmt.Sprintf("Failed to get Config Map %v from namespace %v:  Error = %v ", configMapName, namespace, err))
	}
	return configMap
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

func CanI(userOCID string, namespace string, verb string, resource string) (bool, string) {
	return CanIForAPIGroup(userOCID, namespace, verb, resource, "")
}

func CanIForAPIGroup(userOCID string, namespace string, verb string, resource string, group string) (bool, string) {
	return CanIForAPIGroupForServiceAccountOrUser(userOCID, namespace, verb, resource, group, false, "")
}

func CanIForAPIGroupForServiceAccountOrUser(saOrUserOCID string, namespace string, verb string, resource string, group string, isServiceAccount bool, saNamespace string) (bool, string) {
	canI := &v1beta1.SelfSubjectAccessReview{
		TypeMeta: metav1.TypeMeta{
			Kind:       "SelfSubjectAccessReview",
			APIVersion: "authorization.k8s.io/v1",
		},
		Spec: v1beta1.SelfSubjectAccessReviewSpec{
			ResourceAttributes: &v1beta1.ResourceAttributes{
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

	config := GetKubeConfig()

	wt := config.WrapTransport // Config might already have a transport wrapper
	var token []byte
	if isServiceAccount {
		token = GetTokenForServiceAccount(saOrUserOCID, saNamespace)
		config.BearerToken = string(token)
	}

	config.WrapTransport = func(rt http.RoundTripper) http.RoundTripper {
		if wt != nil {
			rt = wt(rt)
		}
		header := &headerAdder{
			rt: rt,
		}
		if !isServiceAccount {
			header.headers = map[string][]string{"Impersonate-User": {saOrUserOCID}}
		}

		return header
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		ginkgo.Fail("Could not get Kubernetes clientset")
	}

	auth, err := clientset.AuthorizationV1beta1().SelfSubjectAccessReviews().Create(context.TODO(), canI, metav1.CreateOptions{})
	if err != nil {
		ginkgo.Fail(fmt.Sprintf("Failed to check perms: %v", err))
	}

	return auth.Status.Allowed, auth.Status.Reason
}

//GetTokenForServiceAccount returns the token associated with service account
func GetTokenForServiceAccount(sa string, namespace string) []byte {
	serviceAccount := GetServiceAccount(namespace, sa)
	if len(serviceAccount.Secrets) == 0 {
		ginkgo.Fail(fmt.Sprintf("No secrets present in service account %s in namespace %s", sa, namespace))
	}

	secretName := serviceAccount.Secrets[0].Name
	clientset := GetKubernetesClientset()
	secret, err := clientset.CoreV1().Secrets(namespace).Get(context.TODO(), secretName, metav1.GetOptions{})
	if err != nil {
		ginkgo.Fail(fmt.Sprintf("Failed to get secret %s for service account %s in namespace %s with error: %v", secretName, sa, namespace, err))
	}

	token, ok := secret.Data["token"]

	if !ok {
		ginkgo.Fail(fmt.Sprintf("No token present in secret %s for service account %s in namespace %s with error: %v", secretName, sa, namespace, err))
	}

	return token
}

//CanIForAPIGroupForServiceAccountREST verifies servcieaccount privs using REST interface of K8S api directly
func CanIForAPIGroupForServiceAccountREST(sa string, namespace string, verb string, resource string, group string, saNamespace string) (bool, string) {
	canI := &v1beta1.SelfSubjectAccessReview{
		TypeMeta: metav1.TypeMeta{
			Kind:       "SelfSubjectAccessReview",
			APIVersion: "authorization.k8s.io/v1",
		},
		Spec: v1beta1.SelfSubjectAccessReviewSpec{
			ResourceAttributes: &v1beta1.ResourceAttributes{
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

	config := GetKubeConfig()

	token := GetTokenForServiceAccount(sa, saNamespace)
	config.BearerToken = string(token)

	k8sApi := &APIEndpoint{
		AccessToken: string(token),
		APIURL:      config.Host,
		HTTPClient:  GetVerrazzanoHTTPClient(),
	}

	buff, err := json.Marshal(canI)
	if err != nil {
		ginkgo.Fail(fmt.Sprintf("Failed to marshal selfsubjectaccessreview: %v", err))
	}

	res, err := k8sApi.Post("apis/authorization.k8s.io/v1/selfsubjectaccessreviews", bytes.NewBuffer(buff))
	if err != nil {
		ginkgo.Fail(fmt.Sprintf("Failed to check perms: %v", err))
	}

	auth := v1beta1.SelfSubjectAccessReview{}
	err = json.Unmarshal(res.Body, &auth)
	if err != nil {
		ginkgo.Fail(fmt.Sprintf("Failed to unmarshal selfsubjectaccessreview: %v", err))
	}

	return auth.Status.Allowed, auth.Status.Reason
}
