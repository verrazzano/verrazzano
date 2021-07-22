// Copyright (c) 2020, 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package pkg

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/tools/remotecommand"

	"github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	vmcClient "github.com/verrazzano/verrazzano/platform-operator/clients/clusters/clientset/versioned"
	vpoClient "github.com/verrazzano/verrazzano/platform-operator/clients/verrazzano/clientset/versioned"

	"k8s.io/api/authorization/v1beta1"
	rbacv1 "k8s.io/api/rbac/v1"

	"github.com/onsi/ginkgo"
	istioClient "istio.io/client-go/pkg/clientset/versioned"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apixv1beta1client "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset/typed/apiextensions/v1beta1"

	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	restclient "k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
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
		ginkgo.Fail("Could not find kube config")
	}
	return kubeconfig
}

// DoesCRDExist returns whether a CRD with the given name exists for the cluster
func DoesCRDExist(crdName string) (bool, error) {
	// use the current context in the kubeconfig
	config := GetKubeConfig()

	apixClient, err := apixv1beta1client.NewForConfig(config)
	if err != nil {
		Log(Error, "Could not get apix client")
		return false, err
	}

	crds, err := apixClient.CustomResourceDefinitions().List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		Log(Error, fmt.Sprintf("Failed to get CRDS with error: %v", err))
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
	return DoesNamespaceExistInCluster(name, GetKubeConfigPathFromEnv())
}

// DoesNamespaceExistInCluster returns whether a namespace with the given name exists in the specified cluster
func DoesNamespaceExistInCluster(name string, kubeconfigPath string) (bool, error) {
	// Get the Kubernetes clientset
	clientset := GetKubernetesClientsetForCluster(kubeconfigPath)

	namespace, err := clientset.CoreV1().Namespaces().Get(context.TODO(), name, metav1.GetOptions{})
	if err != nil && !k8serrors.IsNotFound(err) {
		Log(Error, fmt.Sprintf("Failed to get namespace %s with error: %v", name, err))
		return false, err
	}

	return namespace != nil && len(namespace.Name) > 0, nil
}

// ListNamespaces returns a namespace list for the given list options
func ListNamespaces(opts metav1.ListOptions) (*corev1.NamespaceList, error) {
	return GetKubernetesClientset().CoreV1().Namespaces().List(context.TODO(), opts)
}

// ListPods returns a pod list for the given namespace and list options
func ListPods(namespace string, opts metav1.ListOptions) (*corev1.PodList, error) {
	return GetKubernetesClientset().CoreV1().Pods(namespace).List(context.TODO(), opts)
}

// ListDeployments returns the list of deployments in a given namespace for the cluster
func ListDeployments(namespace string) (*appsv1.DeploymentList, error) {
	// Get the Kubernetes clientset
	clientset := GetKubernetesClientset()

	deployments, err := clientset.AppsV1().Deployments(namespace).List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		Log(Error, fmt.Sprintf("Failed to list deployments in namespace %s with error: %v", namespace, err))
		return nil, err
	}
	return deployments, nil
}

// ListNodes returns the list of nodes for the cluster
func ListNodes() (*corev1.NodeList, error) {
	// Get the Kubernetes clientset
	clientset := GetKubernetesClientset()

	nodes, err := clientset.CoreV1().Nodes().List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		Log(Error, fmt.Sprintf("Failed to list nodes with error: %v", err))
		return nil, err
	}
	return nodes, nil
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

// DoesPodExist returns whether a pod with the given name and namespace exists for the cluster
func DoesPodExist(namespace string, name string) bool {
	clientset := GetKubernetesClientset()
	pods, err := ListPodsInCluster(namespace, clientset)
	if err != nil {
		Log(Error, fmt.Sprintf("Error listing pods in cluster for namespace: %s, error: %v", namespace, err))
		return false
	}
	for i := range pods.Items {
		if strings.HasPrefix(pods.Items[i].Name, name) {
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

// GetVerrazzanoManagedClusterClientset returns the Kubernetes clientset for the VerrazzanoManagedCluster
func GetVerrazzanoManagedClusterClientset() (*vmcClient.Clientset, error) {
	return vmcClient.NewForConfig(GetKubeConfig())
}

// GetDynamicClient returns a dynamic client needed to access Unstructured data
func GetDynamicClient() dynamic.Interface {
	config := GetKubeConfig()
	if config == nil {
		ginkgo.Fail("Could not get an KubeConfig")
	}
	client, err := dynamic.NewForConfig(config)
	if err != nil {
		ginkgo.Fail("Could not get an Dynamic client")
	}
	return client
}

// GetVerrazzanoInstallResourceInCluster returns the installed Verrazzano CR in the given cluster
// (there should only be 1 per cluster)
func GetVerrazzanoInstallResourceInCluster(kubeconfigPath string) (*v1alpha1.Verrazzano, error) {
	client, err := vpoClient.NewForConfig(GetKubeConfigGivenPath(kubeconfigPath))
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

// GetVerrazzanoProfile returns the profile specified in the verrazzano install resource
func GetVerrazzanoProfile() (*v1alpha1.ProfileType, error) {
	vz, err := GetVerrazzanoInstallResourceInCluster(GetKubeConfigPathFromEnv())
	if err != nil {
		return nil, err
	}
	return &vz.Spec.Profile, nil
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

// ListServices returns the list of services in a given namespace for the cluster
func ListServices(namespace string) (*corev1.ServiceList, error) {
	// Get the Kubernetes clientset
	clientset := GetKubernetesClientset()

	services, err := clientset.CoreV1().Services(namespace).List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		Log(Error, fmt.Sprintf("Failed to list services in namespace %s with error: %v", namespace, err))
		return nil, err
	}
	return services, nil
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
func DoesClusterRoleExist(name string) (bool, error) {
	// Get the Kubernetes clientset
	clientset := GetKubernetesClientset()

	clusterrole, err := clientset.RbacV1().ClusterRoles().Get(context.TODO(), name, metav1.GetOptions{})
	if err != nil && !k8serrors.IsNotFound(err) {
		Log(Error, fmt.Sprintf("Failed to get cluster role %s with error: %v", name, err))
		return false, err
	}

	return clusterrole != nil, nil
}

// GetClusterRole returns the cluster role with the given name
func GetClusterRole(name string) (*rbacv1.ClusterRole, error) {
	// Get the Kubernetes clientset
	clientset := GetKubernetesClientset()

	clusterrole, err := clientset.RbacV1().ClusterRoles().Get(context.TODO(), name, metav1.GetOptions{})
	if err != nil {
		Log(Error, fmt.Sprintf("Failed to get cluster role %s with error: %v", name, err))
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
	clientset := GetKubernetesClientset()

	clusterrolebinding, err := clientset.RbacV1().ClusterRoleBindings().Get(context.TODO(), name, metav1.GetOptions{})
	if err != nil && !k8serrors.IsNotFound(err) {
		Log(Error, fmt.Sprintf("Failed to get cluster role binding %s with error: %v", name, err))
		return false, err
	}

	return clusterrolebinding != nil, nil
}

// GetClusterRoleBinding returns the cluster role with the given name
func GetClusterRoleBinding(name string) (*rbacv1.ClusterRoleBinding, error) {
	// Get the Kubernetes clientset
	clientset := GetKubernetesClientset()

	crb, err := clientset.RbacV1().ClusterRoleBindings().Get(context.TODO(), name, metav1.GetOptions{})
	if err != nil {
		Log(Error, fmt.Sprintf("Failed to get cluster role binding %s with error: %v", name, err))
		return nil, err
	}

	return crb, err
}

// ListClusterRoleBindings returns the list of cluster role bindings for the cluster
func ListClusterRoleBindings() (*rbacv1.ClusterRoleBindingList, error) {
	// Get the Kubernetes clientset
	clientset := GetKubernetesClientset()

	bindings, err := clientset.RbacV1().ClusterRoleBindings().List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		Log(Error, fmt.Sprintf("Failed to get cluster role bindings with error: %v", err))
		return nil, err
	}

	return bindings, err
}

// DoesRoleBindingContainSubject returns true if the RoleBinding exists and it contains the
// specified subject
func DoesRoleBindingContainSubject(namespace, name, subjectKind, subjectName string) (bool, error) {
	clientset := GetKubernetesClientset()

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

// Execute executes the given command on the pod and container identified by the given names and returns the
// resulting stdout and stderr
func Execute(podName, containerName, namespace string, command []string) (string, string, error) {
	request := GetKubernetesClientset().CoreV1().RESTClient().Post().Resource("pods").Name(podName).
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
	executor, err := remotecommand.NewSPDYExecutor(GetKubeConfig(), "POST", request.URL())
	if err != nil {
		return "", "", err
	}
	var stdout, stderr bytes.Buffer
	err = executor.Stream(remotecommand.StreamOptions{Stdout: &stdout, Stderr: &stderr})

	return strings.TrimSpace(stdout.String()), strings.TrimSpace(stderr.String()), err
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
func GetConfigMap(configMapName string, namespace string) (*corev1.ConfigMap, error) {
	// Get the Kubernetes clientset
	clientset := GetKubernetesClientset()
	cmi := clientset.CoreV1().ConfigMaps(namespace)
	configMap, err := cmi.Get(context.TODO(), configMapName, metav1.GetOptions{})
	if err != nil {
		Log(Error, fmt.Sprintf("Failed to get Config Map %v from namespace %v:  Error = %v ", configMapName, namespace, err))
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
	if isServiceAccount {
		token, err := GetTokenForServiceAccount(saOrUserOCID, saNamespace)
		if err != nil {
			return false, "", err
		}
		clientConfig := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
			&clientcmd.ClientConfigLoadingRules{ExplicitPath: GetKubeConfigPathFromEnv()},
			&clientcmd.ConfigOverrides{ClusterInfo: clientcmdapi.Cluster{Server: ""}})
		rawConfig, err := clientConfig.RawConfig()
		if err != nil {
			ginkgo.Fail(fmt.Sprintf("Could not get rawconfig, error %v", err))
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
			ginkgo.Fail(fmt.Sprintf("Could not get config for sa, error %v", err))
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
		ginkgo.Fail("Could not get Kubernetes clientset")
	}

	auth, err := clientset.AuthorizationV1beta1().SelfSubjectAccessReviews().Create(context.TODO(), canI, metav1.CreateOptions{})
	if err != nil {
		Log(Error, fmt.Sprintf("Failed to check perms: %v", err))
		return false, "", err
	}

	return auth.Status.Allowed, auth.Status.Reason, nil
}

//GetTokenForServiceAccount returns the token associated with service account
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
	clientset := GetKubernetesClientset()
	secret, err := clientset.CoreV1().Secrets(namespace).Get(context.TODO(), secretName, metav1.GetOptions{})
	if err != nil {
		msg := fmt.Sprintf("failed to get secret %s for service account %s in namespace %s with error: %v", secretName, sa, namespace, err)
		Log(Error, msg)
		return nil, errors.New(msg)
	}

	token, ok := secret.Data["token"]

	if !ok {
		msg := fmt.Sprintf("no token present in secret %s for service account %s in namespace %s with error: %v", secretName, sa, namespace, err)
		Log(Error, msg)
		return nil, errors.New(msg)
	}

	return token, nil
}

func GetServiceAccount(namespace, name string) (*corev1.ServiceAccount, error) {
	clientset := GetKubernetesClientset()
	sa, err := clientset.CoreV1().ServiceAccounts(namespace).Get(context.TODO(), name, metav1.GetOptions{})
	if err != nil {
		Log(Error, fmt.Sprintf("Failed to get service account %s in namespace %s with error: %v", name, namespace, err))
		return nil, err
	}
	return sa, nil
}

func GetPersistentVolumes(namespace string) (map[string]*corev1.PersistentVolumeClaim, error) {
	pvcs, err := GetKubernetesClientset().CoreV1().PersistentVolumeClaims(namespace).List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	volumeClaims := make(map[string]*corev1.PersistentVolumeClaim)

	for _, pvc := range pvcs.Items {
		volumeClaims[pvc.Name] = &pvc
	}
	return volumeClaims, nil
}
