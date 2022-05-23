// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package multicluster

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	errs "errors"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"time"

	oamcore "github.com/crossplane/oam-kubernetes-runtime/apis/core/v1alpha2"
	"github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
	clustersv1alpha1 "github.com/verrazzano/verrazzano/application-operator/apis/clusters/v1alpha1"
	"github.com/verrazzano/verrazzano/pkg/constants"
	"github.com/verrazzano/verrazzano/pkg/k8sutil"
	mcapi "github.com/verrazzano/verrazzano/platform-operator/apis/clusters/v1alpha1"
	mcClient "github.com/verrazzano/verrazzano/platform-operator/clients/clusters/clientset/versioned"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg"
	yv2 "gopkg.in/yaml.v2"
	corev1 "k8s.io/api/core/v1"
	rbac "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	utilyaml "k8s.io/apimachinery/pkg/util/yaml"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/discovery/cached/memory"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/restmapper"
	"k8s.io/client-go/tools/clientcmd"
	cmdapi "k8s.io/client-go/tools/clientcmd/api/v1"
	yml "sigs.k8s.io/yaml"
)

const (
	comps        = "components"
	mcAppConfigs = "multiclusterapplicationconfigurations"
	mcNamespace  = "verrazzano-mc"
	projects     = "verrazzanoprojects"
)

// DeployVerrazzanoProject deploys the VerrazzanoProject to the cluster with the given kubeConfig
func DeployVerrazzanoProject(projectConfiguration, kubeConfig string) error {
	if err := pkg.CreateOrUpdateResourceFromFileInCluster(projectConfiguration, kubeConfig); err != nil {
		return fmt.Errorf("failed to create project resource: %v", err)
	}
	return nil
}

// TestNamespaceExists returns true if the test namespace exists in the given cluster
func TestNamespaceExists(kubeConfig string, namespace string) bool {
	_, err := pkg.GetNamespaceInCluster(namespace, kubeConfig)
	return err == nil
}

// DeployCompResource deploys the OAM Component resource to the cluster with the given kubeConfig
func DeployCompResource(compConfiguration, testNamespace, kubeConfig string) error {
	if err := pkg.CreateOrUpdateResourceFromFileInClusterInGeneratedNamespace(compConfiguration, kubeConfig, testNamespace); err != nil {
		return fmt.Errorf("failed to create multi-cluster component resources: %v", err)
	}
	return nil
}

// DeployAppResource deploys the OAM Application resource to the cluster with the given kubeConfig
func DeployAppResource(appConfiguration, testNamespace, kubeConfig string) error {
	if err := pkg.CreateOrUpdateResourceFromFileInClusterInGeneratedNamespace(appConfiguration, kubeConfig, testNamespace); err != nil {
		return fmt.Errorf("failed to create multi-cluster application resource: %v", err)
	}
	return nil
}

// VerifyMCResources verifies that the MC resources are present or absent depending on whether this is an admin
// cluster and whether the resources are placed in the given cluster
func VerifyMCResources(kubeConfig string, isAdminCluster bool, placedInThisCluster bool, namespace string, appConfigName string, expectedComps []string) bool {
	// call both appConfExists and componentExists and store the results, to avoid short-circuiting
	// since we should check both in all cases
	mcAppConfExists := appConfExists(kubeConfig, namespace, appConfigName)

	compExists := true
	// check each component in expectedComps
	for _, comp := range expectedComps {
		compExists = componentExists(kubeConfig, namespace, comp) && compExists
	}

	if isAdminCluster || placedInThisCluster {
		// always expect MC resources on admin cluster - otherwise expect them only if placed here
		return mcAppConfExists && compExists
	}
	// don't expect either
	return !mcAppConfExists && !compExists
}

// VerifyAppResourcesInCluster verifies that the app resources are either present or absent
// depending on whether the app is placed in this cluster
func VerifyAppResourcesInCluster(kubeConfig string, isAdminCluster bool, placedInThisCluster bool, projectName string, namespace string, appPods []string) (bool, error) {
	projectExists := projectExists(kubeConfig, projectName)
	podsRunning, err := checkPodsRunning(kubeConfig, namespace, appPods)
	if err != nil {
		return false, err
	}

	if placedInThisCluster {
		return projectExists && podsRunning, nil
	}
	if isAdminCluster {
		return projectExists && !podsRunning, nil
	}
	return !podsRunning && !projectExists, nil
}

// VerifyDeleteOnAdminCluster verifies that the app resources have been deleted from the admin
// cluster after the application has been deleted
func VerifyDeleteOnAdminCluster(kubeConfig string, placedInCluster bool, namespace string, projectName string, appConfigName string, appPods []string) bool {
	mcResDeleted := verifyMCResourcesDeleted(kubeConfig, namespace, projectName, appConfigName, appPods)
	if !placedInCluster {
		return mcResDeleted
	}
	appDeleted := verifyAppDeleted(kubeConfig, namespace, appPods)
	return mcResDeleted && appDeleted
}

// VerifyDeleteOnManagedCluster verifies that the app resources have been deleted from the managed
// cluster after the application has been deleted
func VerifyDeleteOnManagedCluster(kubeConfig string, namespace string, projectName string, appConfigName string, appPods []string) bool {
	mcResDeleted := verifyMCResourcesDeleted(kubeConfig, namespace, projectName, appConfigName, appPods)
	appDeleted := verifyAppDeleted(kubeConfig, namespace, appPods)

	return mcResDeleted && appDeleted
}

// appConfExists Check if app config exists
func appConfExists(kubeConfig string, namespace string, appConfigName string) bool {
	gvr := schema.GroupVersionResource{
		Group:    clustersv1alpha1.SchemeGroupVersion.Group,
		Version:  clustersv1alpha1.SchemeGroupVersion.Version,
		Resource: mcAppConfigs,
	}
	return resourceExists(gvr, namespace, appConfigName, kubeConfig)
}

// resourceExists Check if given resource exists
func resourceExists(gvr schema.GroupVersionResource, ns string, name string, kubeConfig string) bool {
	config, err := k8sutil.GetKubeConfigGivenPath(kubeConfig)
	if err != nil {
		pkg.Log(pkg.Error, fmt.Sprintf("Could not get kube config: %v\n", err))
		return false
	}
	client, err := dynamic.NewForConfig(config)
	if err != nil {
		pkg.Log(pkg.Error, fmt.Sprintf("Could not create dynamic client: %v\n", err))
		return false
	}

	u, err := client.Resource(gvr).Namespace(ns).Get(context.TODO(), name, metav1.GetOptions{})

	if err != nil {
		if errors.IsNotFound(err) {
			return false
		}
		pkg.Log(pkg.Error, fmt.Sprintf("Could not retrieve resource %s: %v\n", gvr.String(), err))
		return false
	}
	return u != nil
}

// componentExists Check if individual component exists
func componentExists(kubeConfig string, namespace string, component string) bool {
	gvr := schema.GroupVersionResource{
		Group:    oamcore.Group,
		Version:  oamcore.Version,
		Resource: comps,
	}
	return resourceExists(gvr, namespace, component, kubeConfig)
}

// projectExists Check if project with name projectName exists
func projectExists(kubeConfig string, projectName string) bool {
	gvr := schema.GroupVersionResource{
		Group:    clustersv1alpha1.SchemeGroupVersion.Group,
		Version:  clustersv1alpha1.SchemeGroupVersion.Version,
		Resource: projects,
	}
	return resourceExists(gvr, mcNamespace, projectName, kubeConfig)
}

// checkPodsRunning Check if expected pods are running on a given cluster
func checkPodsRunning(kubeConfig string, namespace string, appPods []string) (bool, error) {
	result, err := pkg.PodsRunningInCluster(namespace, appPods, kubeConfig)
	if err != nil {
		pkg.Log(pkg.Error, fmt.Sprintf("One or more pods are not running in the namespace: %v, error: %v", namespace, err))
		return false, err
	}
	return result, nil
}

// verifyAppDeleted verifies that the workload and pods are deleted on the specified cluster
func verifyAppDeleted(kubeConfig string, namespace string, appPods []string) bool {
	podsDeleted := true
	// check that each pod is deleted
	for _, pod := range appPods {
		podsDeleted = checkPodDeleted(namespace, pod, kubeConfig) && podsDeleted
	}
	return podsDeleted
}

// checkPodDeleted Check if expected pods are running on a given cluster
func checkPodDeleted(kubeConfig string, namespace string, pod string) bool {
	deletedPod := []string{pod}
	result, _ := pkg.PodsRunningInCluster(namespace, deletedPod, kubeConfig)
	return !result
}

// verifyMCResourcesDeleted verifies that any resources created by the deployment are deleted on the specified cluster
func verifyMCResourcesDeleted(kubeConfig string, namespace string, projectName string, appConfigName string, appPods []string) bool {
	appConfExists := appConfExists(kubeConfig, namespace, appConfigName)
	projExists := projectExists(kubeConfig, projectName)

	compExists := true
	// check each component in appPods
	for _, comp := range appPods {
		compExists = componentExists(kubeConfig, namespace, comp) && compExists
	}

	return !appConfExists && !compExists && !projExists
}

const (
	oneMinute       = 1 * time.Minute
	fiveMinutes     = 5 * time.Minute
	pollingInterval = 5 * time.Second
	manifestKey     = "yaml"
)

type Cluster struct {
	Name           string
	KubeConfigPath string
	restConfig     *rest.Config
	kubeClient     *kubernetes.Clientset
	server         string
}

func getCluster(name, kcfgDir string, count int) *Cluster {
	kcfgPath := fmt.Sprintf("%s/%v/kube_config", kcfgDir, count)
	if _, err := os.Stat(kcfgPath); errs.Is(err, os.ErrNotExist) {
		return nil
	}
	return newCluster(name, kcfgPath)
}

func ManagedClusters() []*Cluster {
	kcfgDir := os.Getenv("KUBECONFIG_DIR")
	if kcfgDir == "" {
		ginkgo.Fail("KUBECONFIG_DIR is required")
	}
	var clusters []*Cluster
	count := 1
	for {
		name := fmt.Sprintf("managed%v", count)
		count = count + 1
		cluster := getCluster(name, kcfgDir, count)
		if cluster == nil {
			return clusters
		}
		clusters = append(clusters, cluster)
	}
}

func AdminCluster() *Cluster {
	admKubeCfg := os.Getenv("ADMIN_KUBECONFIG")
	if admKubeCfg == "" {
		admKubeCfg = os.Getenv("KUBECONFIG")
	}
	if admKubeCfg != "" {
		return newCluster("admin", admKubeCfg)
	}
	return getCluster("admin", os.Getenv("KUBECONFIG_DIR"), 1)
}

func (c *Cluster) CreateNamespace(ns string) error {
	_, err := c.kubeClient.CoreV1().Namespaces().Get(context.TODO(), ns, metav1.GetOptions{})
	if err != nil && errors.IsNotFound(err) {
		n := &corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name:      ns,
				Namespace: ns,
			},
		}
		_, err = c.kubeClient.CoreV1().Namespaces().Create(context.TODO(), n, metav1.CreateOptions{})
		if err != nil {
			pkg.Log(pkg.Error, fmt.Sprintf("CreateNS %v error: %v", n, err))
		}
	}
	return err
}

func (c *Cluster) UpsertCaSec(managedClusterName string, bytes []byte) error {
	c.CreateNamespace(constants.VerrazzanoMultiClusterNamespace)
	casecName := fmt.Sprintf("ca-secret-%s", managedClusterName)
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      casecName,
			Namespace: constants.VerrazzanoMultiClusterNamespace,
		},
		Type: corev1.SecretTypeOpaque,
		Data: map[string][]byte{"cacrt": bytes},
	}
	_, err := c.kubeClient.CoreV1().Secrets(constants.VerrazzanoMultiClusterNamespace).Get(context.TODO(), casecName, metav1.GetOptions{})
	if err != nil && errors.IsNotFound(err) {
		_, err = c.kubeClient.CoreV1().Secrets(constants.VerrazzanoMultiClusterNamespace).Create(context.TODO(), secret, metav1.CreateOptions{})
	} else {
		_, err = c.kubeClient.CoreV1().Secrets(constants.VerrazzanoMultiClusterNamespace).Update(context.TODO(), secret, metav1.UpdateOptions{})
	}
	if err != nil {
		pkg.Log(pkg.Error, fmt.Sprintf("UpsertCaSec %v error: %v", casecName, err))
	}
	return err
}

func (c *Cluster) CreateCaSecOf(managed *Cluster) error {
	c.CreateNamespace(constants.VerrazzanoMultiClusterNamespace)
	bytes, err := managed.getCacrt()
	if err != nil {
		pkg.Log(pkg.Error, fmt.Sprintf("Error getting %v cacrt: %v", managed.Name, err))
	}
	return c.UpsertCaSec(managed.Name, bytes)
}

func (c *Cluster) ConfigAdminCluster() error {
	name := "verrazzano-admin-cluster"
	cm := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: constants.VerrazzanoMultiClusterNamespace,
		},
		Data: map[string]string{"server": c.server},
	}
	_, err := c.kubeClient.CoreV1().ConfigMaps(constants.VerrazzanoMultiClusterNamespace).Get(context.TODO(), name, metav1.GetOptions{})
	if err != nil && errors.IsNotFound(err) {
		_, err = c.kubeClient.CoreV1().ConfigMaps(constants.VerrazzanoMultiClusterNamespace).Create(context.TODO(), cm, metav1.CreateOptions{})
	} else {
		_, err = c.kubeClient.CoreV1().ConfigMaps(constants.VerrazzanoMultiClusterNamespace).Update(context.TODO(), cm, metav1.UpdateOptions{})
	}
	return err
}

func (c *Cluster) GetSecret(ns, name string) (*corev1.Secret, error) {
	return c.kubeClient.CoreV1().Secrets(ns).Get(context.TODO(), name, metav1.GetOptions{})
}

func (c *Cluster) GetSecretData(ns, name, key string) ([]byte, error) {
	secret, err := c.GetSecret(ns, name)
	if secret == nil || err != nil {
		return []byte{}, err
	}
	data, ok := secret.Data[key]
	if !ok {
		return []byte{}, fmt.Errorf("%s not found in %s", key, name)
	}
	return data, nil
}

func (c *Cluster) getCacrt() ([]byte, error) {
	//cattle-system get secret tls-ca-additional
	data, err := c.GetSecretData(constants.RancherSystemNamespace, "tls-ca-additional", "ca-additional.pem")
	if len(data) != 0 {
		return data, err
	}
	return c.GetSecretData(constants.VerrazzanoSystemNamespace, "verrazzano-tls", "ca.crt")
}

func (c *Cluster) apply(data []byte) {
	gomega.Eventually(func() bool {
		err := apply(data, c.restConfig)
		if err != nil {
			pkg.Log(pkg.Error, fmt.Sprintf("Error applying changes on %s: %v", c.Name, err))
		}
		return err == nil
	}, fiveMinutes, pollingInterval).Should(gomega.BeTrue(), fmt.Sprintf(" %s failed registration", c.Name))
}

func (c *Cluster) UpsertManagedCluster(name string) error {
	casec := fmt.Sprintf("ca-secret-%s", name)
	vmc := &mcapi.VerrazzanoManagedCluster{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: constants.VerrazzanoMultiClusterNamespace,
		},
		Spec: mcapi.VerrazzanoManagedClusterSpec{
			Description: "VerrazzanoManagedCluster object",
			CASecret:    casec,
		},
	}
	mcCli, err := mcClient.NewForConfig(c.restConfig)
	if err != nil {
		return err
	}
	_, err = mcCli.ClustersV1alpha1().
		VerrazzanoManagedClusters(constants.VerrazzanoMultiClusterNamespace).Get(context.TODO(), name, metav1.GetOptions{})
	if err != nil && errors.IsNotFound(err) {
		_, err = mcCli.ClustersV1alpha1().
			VerrazzanoManagedClusters(constants.VerrazzanoMultiClusterNamespace).
			Create(context.TODO(), vmc, metav1.CreateOptions{})
	} else {
		_, err = mcCli.ClustersV1alpha1().
			VerrazzanoManagedClusters(constants.VerrazzanoMultiClusterNamespace).
			Update(context.TODO(), vmc, metav1.UpdateOptions{})
	}
	if err != nil {
		return fmt.Errorf("failed to create or update VerrazzanoManagedCluster %v: %w", name, err)
	}
	gomega.Eventually(func() bool {
		vmcCreated, err := mcCli.ClustersV1alpha1().
			VerrazzanoManagedClusters(constants.VerrazzanoMultiClusterNamespace).
			Get(context.TODO(), vmc.Name, metav1.GetOptions{})
		if err != nil {
			pkg.Log(pkg.Error, fmt.Sprintf("Error getting vmc %s: %v", vmc.Name, err))
		}
		size := len(vmcCreated.Status.Conditions)
		if vmcCreated == nil || size == 0 {
			return false
		}
		return vmcCreated.Status.Conditions[size-1].Type == mcapi.ConditionReady
	}, fiveMinutes, pollingInterval).Should(gomega.BeTrue(), fmt.Sprintf("VerrazzanoManagedCluster %s is not ready", vmc.Name))
	return nil
}

func (c *Cluster) Register(managed *Cluster) error {
	err := c.CreateCaSecOf(managed)
	if err != nil {
		return nil
	}
	err = c.ConfigAdminCluster()
	if err != nil {
		return nil
	}
	err = c.UpsertManagedCluster(managed.Name)
	if err != nil {
		pkg.Log(pkg.Error, fmt.Sprintf("CreateManagedCluster %v error: %v", managed.Name, err))
	}
	reg, err := c.GetManifest(managed.Name)
	if err != nil {
		pkg.Log(pkg.Error, fmt.Sprintf("manifest %v error: %v", managed.Name, err))
	}
	managed.apply(reg)
	return nil
}

func (c *Cluster) GetManifest(name string) ([]byte, error) {
	manifest := fmt.Sprintf("verrazzano-cluster-%s-manifest", name)
	gomega.Eventually(func() bool {
		data, _ := c.GetSecretData(constants.VerrazzanoMultiClusterNamespace, manifest, manifestKey)
		return len(data) > 0
	}, oneMinute, pollingInterval).Should(gomega.BeTrue(), fmt.Sprintf("manifest %s is not ready", manifest))
	return c.GetSecretData(constants.VerrazzanoMultiClusterNamespace, manifest, manifestKey)
}

func (c *Cluster) GetRegistration(name string) (*corev1.Secret, error) {
	reg := fmt.Sprintf("verrazzano-cluster-%s-registration", name)
	return c.GetSecret(constants.VerrazzanoMultiClusterNamespace, reg)
}

func newCluster(name, kubeCfgPath string) *Cluster {
	server := serverFromDockerInspect(name)
	if server == "" {
		server = serverFromKubeConfig(kubeCfgPath, name)
	}
	cnf, err := clientcmd.BuildConfigFromFlags("", kubeCfgPath)
	failOnErr := func(err error) {
		if err != nil {
			ginkgo.Fail(fmt.Sprintf("Error getting Cluster %v: %v", name, err))
		}
	}
	failOnErr(err)
	cli, err := kubernetes.NewForConfig(cnf)
	failOnErr(err)
	return &Cluster{Name: name, KubeConfigPath: kubeCfgPath, kubeClient: cli, server: server, restConfig: cnf}
}

func serverFromDockerInspect(name string) string {
	cmd := exec.Command("docker", "inspect", fmt.Sprintf("%s-control-plane", name)) //nolint:gosec
	out, err := cmd.Output()
	if err == nil {
		var info []map[string]interface{}
		json.Unmarshal(out, &info)
		if len(info) > 0 {
			ipa := yq(info[0], "NetworkSettings", "Networks", "kind", "IPAddress")
			if ipa != nil {
				if addr, ok := ipa.(string); ok && addr != "" {
					return fmt.Sprintf("https://%s:6443", addr)
				}
			}
		}
	}
	return ""
}

func serverFromKubeConfig(kubeCfgPath, name string) string {
	kubeServerConf := cmdapi.Config{}
	cmd := exec.Command("kind", "get", "kubeconfig", "--internal", "--Name", name) //nolint:gosec
	out, err := cmd.Output()
	if err != nil {
		out, _ = ioutil.ReadFile(kubeCfgPath)
	}
	yv2.Unmarshal(out, &kubeServerConf)
	for _, c := range kubeServerConf.Clusters {
		return c.Cluster.Server
	}
	return ""
}

func apply(data []byte, config *rest.Config) error {
	client, err := dynamic.NewForConfig(config)
	if err != nil {
		return fmt.Errorf("failed to create dynamic client: %w", err)
	}
	disco, err := discovery.NewDiscoveryClientForConfig(config)
	if err != nil {
		return fmt.Errorf("failed to create discovery client: %w", err)
	}
	mapper := restmapper.NewDeferredDiscoveryRESTMapper(memory.NewMemCacheClient(disco))
	reader := utilyaml.NewYAMLReader(bufio.NewReader(bytes.NewReader(data)))
	for {
		uns := &unstructured.Unstructured{Object: map[string]interface{}{}}
		unsMap, err := readYaml(reader, mapper, uns)
		if err != nil {
			return fmt.Errorf("failed to read resource from bytes: %w", err)
		}
		if unsMap == nil {
			return nil
		}
		if uns.GetNamespace() == "" {
			_, err = client.Resource(unsMap.Resource).Create(context.TODO(), uns, metav1.CreateOptions{})
		} else {
			_, err = client.Resource(unsMap.Resource).Namespace(uns.GetNamespace()).Create(context.TODO(), uns, metav1.CreateOptions{})
		}
		if err != nil && errors.IsAlreadyExists(err) {
			// Get, read the resource version, and then update the resource.
			resource, err := client.Resource(unsMap.Resource).Namespace(uns.GetNamespace()).Get(context.TODO(), uns.GetName(), metav1.GetOptions{})
			if err != nil {
				return fmt.Errorf("failed to get resource for update: %w", err)
			}
			uns.SetResourceVersion(resource.GetResourceVersion())
			_, err = client.Resource(unsMap.Resource).Namespace(uns.GetNamespace()).Update(context.TODO(), uns, metav1.UpdateOptions{})
			if err != nil && uns.GetKind() == "Service" && uns.GetName() == "cattle-cluster-agent" {
				_ = client.Resource(unsMap.Resource).Namespace(uns.GetNamespace()).Delete(context.TODO(), uns.GetName(), metav1.DeleteOptions{})
				uns.SetResourceVersion("")
				_, err = client.Resource(unsMap.Resource).Namespace(uns.GetNamespace()).Create(context.TODO(), uns, metav1.CreateOptions{})
			}
			if err != nil {
				return fmt.Errorf("failed to update resource: %w", err)
			}
		} else if err != nil {
			if uns.GetKind() == "ClusterRoleBinding" {
				cli, _ := kubernetes.NewForConfig(config)
				crb := toClusterRoleBinding(uns)
				_, err = cli.RbacV1().ClusterRoleBindings().Get(context.TODO(), crb.Name, metav1.GetOptions{})
				if err != nil && errors.IsNotFound(err) {
					_, err = cli.RbacV1().ClusterRoleBindings().Create(context.TODO(), crb, metav1.CreateOptions{})
				} else {
					_, err = cli.RbacV1().ClusterRoleBindings().Update(context.TODO(), crb, metav1.UpdateOptions{})
				}
				if err != nil {
					return fmt.Errorf("failed to create ClusterRoleBinding: %w", err)
				}
			} else {
				msg := "failed to create resource: %v"
				pkg.Log(pkg.Error, fmt.Sprintf(msg, err))
				return fmt.Errorf(msg, err)
			}
		}
	}
}

func toClusterRoleBinding(uns *unstructured.Unstructured) *rbac.ClusterRoleBinding {
	toString := func(node interface{}, path ...string) string {
		val := yq(node, path...)
		if val == nil {
			return ""
		}
		if s, ok := val.(string); ok {
			return s
		}
		return fmt.Sprintf("%v", val)
	}
	rb := &rbac.ClusterRoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name:      uns.GetName(),
			Namespace: uns.GetNamespace(),
			Labels:    uns.GetLabels(),
		},
		Subjects: []rbac.Subject{},
		RoleRef: rbac.RoleRef{
			Kind:     toString(uns.Object, "roleRef", "kind"),
			Name:     toString(uns.Object, "roleRef", "name"),
			APIGroup: toString(uns.Object, "roleRef", "apiGroup"),
		},
	}
	if rb.Name == "" {
		rb.Name = toString(uns.Object, "metadata", "name")
	}
	if rb.Namespace == "" {
		rb.Namespace = toString(uns.Object, "metadata", "namespace")
	}
	if len(rb.Labels) == 0 {
		rb.Labels = map[string]string{}
		labels := yq(uns.Object, "metadata", "labels")
		if labels != nil {
			for k, v := range labels.(map[interface{}]interface{}) {
				rb.Labels[k.(string)] = v.(string)
			}
		}
	}
	if sbj := yq(uns.Object, "subjects"); sbj != nil {
		arr, ok := sbj.([]interface{})
		if len(arr) > 0 && ok {
			for _, i := range arr {
				rb.Subjects = append(rb.Subjects, rbac.Subject{
					Kind:      toString(i, "kind"),
					Name:      toString(i, "name"),
					Namespace: toString(i, "namespace"),
				})
			}
		}
	}
	return rb
}

func readYaml(reader *utilyaml.YAMLReader, mapper *restmapper.DeferredDiscoveryRESTMapper, uns *unstructured.Unstructured) (*meta.RESTMapping, error) {
	buf, err := reader.Read()
	if err == io.EOF {
		return nil, nil
	} else if err != nil {
		return nil, fmt.Errorf("failed to read resource section: %w", err)
	}
	if err = yml.Unmarshal(buf, &uns.Object); err != nil {
		return nil, fmt.Errorf("failed to unmarshal resource: %w", err)
	}
	unsGvk := schema.FromAPIVersionAndKind(uns.GetAPIVersion(), uns.GetKind())
	unsMap, err := mapper.RESTMapping(unsGvk.GroupKind(), unsGvk.Version)
	if err != nil {
		return unsMap, fmt.Errorf("failed to map resource kind: %w", err)
	}
	return unsMap, nil
}

func yq(node interface{}, path ...string) interface{} {
	for _, p := range path {
		if node == nil {
			return nil
		}
		var nodeMap, ok = node.(map[string]interface{})
		if ok {
			node = nodeMap[p]
		} else {
			n, ok := node.(map[interface{}]interface{})
			if ok {
				node = n[p]
			} else {
				return nil
			}
		}
	}
	return node
}
