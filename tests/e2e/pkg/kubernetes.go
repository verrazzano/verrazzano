// Copyright (c) 2020, 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package pkg

import (
	"context"
	"fmt"
	"k8s.io/api/authorization/v1beta1"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	vmcClient "github.com/verrazzano/verrazzano/platform-operator/clients/clusters/clientset/versioned"
	vpoClient "github.com/verrazzano/verrazzano/platform-operator/clients/verrazzano/clientset/versioned"

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

var config *restclient.Config
var clientset *kubernetes.Clientset

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
	// Get the Kubernetes clientset
	clientset := GetKubernetesClientset()

	namespace, err := clientset.CoreV1().Namespaces().Get(context.TODO(), name, metav1.GetOptions{})
	if err != nil {
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

// ListPods returns the list of pods in a given namespace for the cluster
func ListPods(namespace string) *corev1.PodList {
	// Get the Kubernetes clientset
	clientset := GetKubernetesClientset()

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

// GetPlatformOperatorClientset returns the Kubernetes clientset for the Verrazzano Platform Operator
func GetPlatformOperatorClientset() *vpoClient.Clientset {
	client, err := vpoClient.NewForConfig(GetKubeConfig())
	if err != nil {
		ginkgo.Fail("Could not get Verrazzano Platform Operator clientset")
	}
	return client
}

// GetVerrazzanoInstallResource returns the installed Verrazzano CR (there should only be 1 per cluster)
func GetVerrazzanoInstallResource() *v1alpha1.Verrazzano {
	vzClient := GetPlatformOperatorClientset().VerrazzanoV1alpha1().Verrazzanos("")
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
	return GetVerrazzanoInstallResource().Spec.Profile == v1alpha1.Dev
}

// IsProdProfile returns true if the deployed resource is a 'prod' profile
func IsProdProfile() bool {
	return GetVerrazzanoInstallResource().Spec.Profile == v1alpha1.Prod
}

// IsManagedClusterProfile returns true if the deployed resource is a 'managed-cluster' profile
func IsManagedClusterProfile() bool {
	return GetVerrazzanoInstallResource().Spec.Profile == v1alpha1.ManagedCluster
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

// DeleteNamespace deletes a namespace
func DeleteNamespace(name string) error {
	if len(os.Getenv("TEST_KUBECONFIG")) > 0 {
		Log(Info, fmt.Sprintf("DeleteNamespace %s, test is running with custom service account and therefore namespace won't be deleted by the test", name))
		return nil
	}

	// Get the Kubernetes clientset
	clientset := GetKubernetesClientset()
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
	clientset := GetKubernetesClientset()

	sa, err := clientset.CoreV1().ServiceAccounts(namespace).Get(context.TODO(), name, metav1.GetOptions{})

	if err != nil {
		ginkgo.Fail(fmt.Sprintf("Failed to get service account %s in namespace %s with error: %v", name, namespace, err))
	}

	return sa != nil

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
			Name:            rolebindingname,
			GenerateName:    "",
			Namespace:       "",
			SelfLink:        "",
			UID:             "",
			ResourceVersion: "",
			Generation:      0,
			CreationTimestamp: metav1.Time{
				Time: time.Time{},
			},
			DeletionTimestamp: &metav1.Time{
				Time: time.Time{},
			},
			DeletionGracePeriodSeconds: nil,
			Labels:                     nil,
			Annotations:                nil,
			OwnerReferences:            nil,
			Finalizers:                 nil,
			ClusterName:                "",
			ManagedFields:              nil,
		},
		Subjects: subjects,
		RoleRef: rbacv1.RoleRef{
			APIGroup: "rbac.authorization.k8s.io",
			Kind:     "ClusterRole",
			Name:     clusterrolename,
		},
	}

	newrb, err := clientset.RbacV1().RoleBindings(namespace).Create(context.TODO(), &rb, metav1.CreateOptions{})
	if err != nil {
		fmt.Sprintf("Failed to create role binding: %v", err)
	}
	log.Printf("%+v", &newrb)
	return err
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

func CanI(userOCID string, namespace string, verb string, resource string) (bool, string) {
	return CanIGroup(userOCID, namespace, verb, resource, "")
}
func CanIGroup(userOCID string, namespace string, verb string, resource string, group string) (bool, string) {

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
	config.WrapTransport = func(rt http.RoundTripper) http.RoundTripper {
		if wt != nil {
			rt = wt(rt)
		}
		return &headerAdder{
			headers: map[string][]string{"Impersonate-User": []string{userOCID}},
			rt:      rt,
		}
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		ginkgo.Fail("Could not get Kubernetes clientset")
	}

	auth, err := clientset.AuthorizationV1beta1().SelfSubjectAccessReviews().Create(context.TODO(), canI, metav1.CreateOptions{})
	if err != nil {
		fmt.Sprintf("Failed to check perms: %v", err)
	}
	log.Printf("%+v", &auth)
	return auth.Status.Allowed, auth.Status.Reason

}
