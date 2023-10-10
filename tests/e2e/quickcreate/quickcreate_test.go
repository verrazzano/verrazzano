// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package quickcreate

import (
	"context"
	"encoding/json"
	"fmt"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/pkg/errors"
	"github.com/verrazzano/verrazzano/pkg/k8sutil"
	"github.com/verrazzano/verrazzano/tests/e2e/backup/helpers"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg/update"
	"go.uber.org/zap"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"os"
	"sigs.k8s.io/cluster-api/api/v1beta1"
	clipkg "sigs.k8s.io/controller-runtime/pkg/client"
	"strings"
	"text/tabwriter"
	"time"
)

const (
	minimumVersion  = "2.0.0"
	waitTimeOut     = 40 * time.Minute
	pollingInterval = 30 * time.Second

	shortWaitTimeout            = 15 * time.Minute
	shortPollingInterval        = 60 * time.Second
	vzPollingInterval           = 60 * time.Second
	addonControllerPodNamespace = "verrazzano-capi"
	nodeLabel                   = "node-role.kubernetes.io/node"
	controlPlaneLabel           = "node-role.kubernetes.io/control-plane"
	addonControllerPodLabel     = "cluster.x-k8s.io/provider"
)

var (
	client              clipkg.Client
	ctx                 *QCContext
	ImagePullSecret     = os.Getenv("IMAGE_PULL_SECRET")
	DockerRepo          = os.Getenv("DOCKER_REPO")
	DockerCredsUser     = os.Getenv("DOCKER_CREDS_USR")
	DockerCredsPassword = os.Getenv("DOCKER_CREDS_PSW")
)

var beforeSuite = t.BeforeSuiteFunc(func() {
	// Get Kubeconfig information and create clients
	kubeconfigPath, err := k8sutil.GetKubeConfigLocation()
	Expect(err).To(BeNil())
	cfg, err := k8sutil.GetKubeConfigGivenPath(kubeconfigPath)
	Expect(err).To(BeNil())
	scheme := runtime.NewScheme()
	_ = v1beta1.AddToScheme(scheme)
	_ = corev1.AddToScheme(scheme)
	c, err := clipkg.New(cfg, clipkg.Options{
		Scheme: scheme,
	})
	Expect(err).To(BeNil())
	client = c

	// Create test context and setup
	ctx, err = newContext(client, clusterType)
	Expect(err).To(BeNil())
	err = ctx.setup()
	Expect(err).To(BeNil())

	t.Logs.Infof("Creating Cluster of type [%s] - parameters [%s] = namespace [%s] - clustername [%s] - clusternamespace [%s]", ctx.ClusterType, ctx.Parameters, ctx.Namespace, clusterName, clusterNamespace)
	if err != nil {
		t.Fail(fmt.Sprintf("Failed to get default kubeconfig path: %s", err.Error()))
	}
	supported, err := pkg.IsVerrazzanoMinVersion(minimumVersion, kubeconfigPath)
	if err != nil {
		t.Logs.Errorf("Error getting Verrazzano version: %v", err)
	}
	if !supported {
		t.Logs.Infof("Skipping test because Verrazzano version is less than %s", minimumVersion)
		//return
	}
	createCluster()
	t.Logs.Infof("Wait for 30 seconds before verification")
	time.Sleep(30 * time.Second)
	err = CreateImagePullSecrets(clusterName, t.Logs)
	if err != nil {
		t.Logs.Errorf("Error creating image pull secrets")
	}
})
var afterSuite = t.AfterSuiteFunc(func() {
	if ctx == nil {
		return
	}
	Eventually(func() error {
		err := ctx.cleanupCAPICluster()
		if err != nil {
			t.Logs.Info(err)
		}
		return err
	}).WithPolling(pollingInterval).WithTimeout(waitTimeOut).ShouldNot(HaveOccurred())
	Eventually(func() error {
		err := ctx.deleteObject(ctx.namespaceObject())
		if err != nil {
			t.Logs.Info(err)
		}
		return err
	}).WithPolling(pollingInterval).WithTimeout(waitTimeOut).ShouldNot(HaveOccurred())
})
var _ = BeforeSuite(beforeSuite)
var _ = AfterSuite(afterSuite)

func createCluster() {
	err := ctx.applyCluster()
	Expect(err).To(BeNil())
	Eventually(func() error {
		err := ctx.isClusterReady()
		if err != nil {
			t.Logs.Info(err)
		}
		return err
	}).WithPolling(pollingInterval).WithTimeout(waitTimeOut).ShouldNot(HaveOccurred())
}

// 'It' Wrapper to only run spec if the ClusterAPI is supported on the current Verrazzano version
func WhenClusterAPIInstalledIt(description string, f func()) {
	kubeconfigPath, err := k8sutil.GetKubeConfigLocation()
	if err != nil {
		t.It(description, func() {
			Fail(fmt.Sprintf("Failed to get default kubeconfig path: %s", err.Error()))
		})
	}
	supported, err := pkg.IsVerrazzanoMinVersion("1.6.0", kubeconfigPath)
	if err != nil {
		t.It(description, func() {
			Fail(fmt.Sprintf("Failed to check Verrazzano version 1.6.0: %s", err.Error()))
		})
	}
	if supported {
		t.It(description, f)
	} else {
		t.Logs.Infof("Skipping check '%v',clusterAPI is not supported", description)
	}
}

func ensureVPOPodsAreRunningOnWorkloadCluster(clusterName, namespace string, log *zap.SugaredLogger) bool {
	k8sclient, err := getCapiClusterK8sClient(clusterName, log)
	if err != nil {
		t.Logs.Info("Failed to get k8s client for workload cluster")
		return false
	}
	vpo, err := pkg.SpecificPodsRunningInClusterWithClient(namespace, "app=verrazzano-platform-operator", k8sclient)
	if err != nil {
		t.Logs.Error(fmt.Sprintf("One or more pods are not running in the namespace: %v, error: %v", namespace, err))
	}
	vpoWebhook, err := pkg.SpecificPodsRunningInClusterWithClient(namespace, "app=verrazzano-platform-operator-webhook", k8sclient)
	if err != nil {
		t.Logs.Error(fmt.Sprintf("One or more pods are not running in the namespace: %v, error: %v", namespace, err))
	}

	if err != nil {
		log.Errorf("Unable to display resources from workload cluster ", zap.Error(err))
		return false
	}

	return vpo && vpoWebhook
}

func getCapiClusterKubeConfig(clusterName string, log *zap.SugaredLogger) ([]byte, error) {
	clientset, err := k8sutil.GetKubernetesClientset()
	if err != nil {
		log.Errorf("Failed to get clientset with error: %v", err)
		return nil, err
	}

	secret, err := clientset.CoreV1().Secrets(clusterNamespace).Get(context.TODO(), fmt.Sprintf("%s-kubeconfig", clusterName), metav1.GetOptions{})
	if err != nil {
		log.Infof("Error fetching secret ", zap.Error(err))
		return nil, err
	}

	return secret.Data["value"], nil
}

func getCapiClusterK8sClient(clusterName string, log *zap.SugaredLogger) (client *kubernetes.Clientset, err error) {
	capiK8sConfig, err := getCapiClusterKubeConfig(clusterName, log)
	if err != nil {
		return nil, err
	}
	k8sRestConfig, err := GetRESTConfigGivenString(capiK8sConfig)
	if err != nil {
		return nil, errors.Wrap(err, "Failed to get k8s rest config")
	}

	return k8sutil.GetKubernetesClientsetWithConfig(k8sRestConfig)
}

func GetRESTConfigGivenString(kubeconfig []byte) (*rest.Config, error) {
	config, err := clientcmd.RESTConfigFromKubeConfig(kubeconfig)
	if err != nil {
		return nil, err
	}
	setConfigQPSBurst(config)
	return config, nil
}

func setConfigQPSBurst(config *rest.Config) {
	config.Burst = 150
	config.QPS = 100
}

func CreateImagePullSecrets(clusterName string, log *zap.SugaredLogger) error {
	log.Infof("Creating image pull secrets on workload cluster ...")

	capiK8sConfig, err := getCapiClusterKubeConfig(clusterName, log)
	if err != nil {
		return err
	}
	tmpFile, err := os.CreateTemp(os.TempDir(), clusterName)
	if err != nil {
		log.Error("Failed to create temporary file ", zap.Error(err))
		return err
	}

	if err = os.WriteFile(tmpFile.Name(), capiK8sConfig, 0600); err != nil {
		log.Error("failed to write to destination file ", zap.Error(err))
		return err
	}

	var cmdArgs []string
	var bcmd helpers.BashCommand
	dockerSecretCommand := fmt.Sprintf("kubectl --kubeconfig %s create secret docker-registry %s --docker-server=%s --docker-username=%s --docker-password=%s", tmpFile.Name(), ImagePullSecret, DockerRepo, DockerCredsUser, DockerCredsPassword)
	cmdArgs = append(cmdArgs, "/bin/bash", "-c", dockerSecretCommand)
	bcmd.CommandArgs = cmdArgs
	secretCreateResponse := helpers.Runner(&bcmd, log)
	if secretCreateResponse.CommandError != nil {
		return secretCreateResponse.CommandError
	}

	cmdArgs = []string{}
	dockerSecretCommand = fmt.Sprintf("kubectl --kubeconfig %s  create ns verrazzano-install", tmpFile.Name())
	cmdArgs = append(cmdArgs, "/bin/bash", "-c", dockerSecretCommand)
	bcmd.CommandArgs = cmdArgs
	secretCreateResponse = helpers.Runner(&bcmd, log)
	if secretCreateResponse.CommandError != nil {
		return secretCreateResponse.CommandError
	}

	cmdArgs = []string{}
	dockerSecretCommand = fmt.Sprintf("kubectl --kubeconfig %s create secret docker-registry %s --docker-server=%s --docker-username=%s --docker-password=%s -n verrazzano-install", tmpFile.Name(), ImagePullSecret, DockerRepo, DockerCredsUser, DockerCredsPassword)
	cmdArgs = append(cmdArgs, "/bin/bash", "-c", dockerSecretCommand)
	bcmd.CommandArgs = cmdArgs
	secretCreateResponse = helpers.Runner(&bcmd, log)
	if secretCreateResponse.CommandError != nil {
		return secretCreateResponse.CommandError
	}

	cmdArgs = []string{}
	dockerSecretCommand = fmt.Sprintf("kubectl --kubeconfig %s create secret docker-registry github-packages --docker-server=%s --docker-username=%s --docker-password=%s", tmpFile.Name(), DockerRepo, DockerCredsUser, DockerCredsPassword)
	cmdArgs = append(cmdArgs, "/bin/bash", "-c", dockerSecretCommand)
	bcmd.CommandArgs = cmdArgs
	secretCreateResponse = helpers.Runner(&bcmd, log)
	if secretCreateResponse.CommandError != nil {
		return secretCreateResponse.CommandError
	}

	cmdArgs = []string{}
	dockerSecretCommand = fmt.Sprintf("kubectl --kubeconfig %s create secret docker-registry ocr --docker-server=%s --docker-username=%s --docker-password=%s", tmpFile.Name(), DockerRepo, DockerCredsUser, DockerCredsPassword)
	cmdArgs = append(cmdArgs, "/bin/bash", "-c", dockerSecretCommand)
	bcmd.CommandArgs = cmdArgs
	secretCreateResponse = helpers.Runner(&bcmd, log)
	if secretCreateResponse.CommandError != nil {
		return secretCreateResponse.CommandError
	}
	return nil
}

func ensureVerrazzano(clusterName string, log *zap.SugaredLogger) error {

	vzFetched, err := getVerrazzano(clusterName, "default", "verrazzano", log)
	if err != nil {
		log.Errorf("unable to fetch vz resource from %s due to '%v'", clusterName, zap.Error(err))
		return err
	}
	var vz Verrazzano
	modBinaryData, err := json.Marshal(vzFetched)
	if err != nil {
		log.Error("json marshalling error ", zap.Error(err))
		return err
	}

	err = json.Unmarshal(modBinaryData, &vz)
	if err != nil {
		log.Error("json unmarshalling error ", zap.Error(err))
		return err
	}

	curState := "InstallStarted"
	for _, cond := range vz.Status.Conditions {
		if cond.Type == "InstallComplete" {
			curState = cond.Type
		}
	}

	if curState == "InstallComplete" {
		return nil
	}
	return fmt.Errorf("All components are not ready: Current State = %v", curState)
}

func ensureVerrazzanoFleetBindingExists(clusterName string, log *zap.SugaredLogger) error {
	log.Infof("Wait for 30 seconds before verification")
	time.Sleep(30 * time.Second)

	vfbFetched, err := getVerrazzanoFleetBinding(log)
	if err != nil {
		log.Errorf("unable to fetch verrazzanofleetbinding resource from %s due to '%v'", clusterName, zap.Error(err))
		return err
	}
	var vfb VerrazzanoFleetBinding
	modBinaryData, err := json.Marshal(vfbFetched)
	if err != nil {
		log.Error("json marshalling error ", zap.Error(err))
		return err
	}

	err = json.Unmarshal(modBinaryData, &vfb)
	if err != nil {
		log.Error("json unmarshalling error ", zap.Error(err))
		return err
	}

	curState := "Ready"
	for _, cond := range vfb.Status.Conditions {
		if cond.Type == "Ready" {
			curState = cond.Type
		}
	}

	if curState == "Ready" {
		return nil
	}
	return fmt.Errorf("All components are not ready: Current State = %v", curState)
}

func ensureVerrazzanoFleetExists(clusterName string, log *zap.SugaredLogger) error {
	log.Infof("Wait for 30 seconds before verification")
	time.Sleep(30 * time.Second)

	vfFetched, err := getVerrazzanoFleet(log)
	if err != nil {
		log.Errorf("unable to fetch verrazzanofleetbinding resource from %s due to '%v'", clusterName, zap.Error(err))
		return err
	}
	var vf VerrazzanoFleet
	modBinaryData, err := json.Marshal(vfFetched)
	if err != nil {
		log.Error("json marshalling error ", zap.Error(err))
		return err
	}

	err = json.Unmarshal(modBinaryData, &vf)
	if err != nil {
		log.Error("json unmarshalling error ", zap.Error(err))
		return err
	}

	curState := "Ready"
	for _, cond := range vf.Status.Conditions {
		if cond.Type == "Ready" {
			curState = cond.Type
		}
	}

	if curState == "Ready" {
		return nil
	}
	return fmt.Errorf("VerrazzanoFleet is not ready: Current State = %v", curState)
}

func createFleetForUnknownCluster(clusterName string, log *zap.SugaredLogger) error {
	dclient, err := k8sutil.GetDynamicClient()
	if err != nil {
		log.Errorf("unable to get workload kubeconfig ", zap.Error(err))
		return err
	}

	gvr := getVerrazzanoFleetGVR()
	vfFetched, err := getVerrazzanoFleet(log)
	if err != nil {
		log.Errorf("unable to fetch verrazzanofleetbinding resource from %s due to '%v'", clusterName, zap.Error(err))
		return err
	}
	vfDeepCopy := vfFetched.DeepCopy()
	vfDeepCopy.Object["spec"].(map[string]interface{})["clusterSelector"].(map[string]interface{})["name"] = "test-clustername"
	vfDeepCopy.Object["metadata"].(map[string]interface{})["name"] = "new-fleet-name"
	delete(vfDeepCopy.Object["metadata"].(map[string]interface{}), "resourceVersion")

	_, err = dclient.Resource(gvr).Namespace(clusterNamespace).Create(context.TODO(), vfDeepCopy, metav1.CreateOptions{})
	if err != nil {
		log.Errorf("Unable to create the verrazzanofleet resource", err)
		return err
	}
	return nil
}

func updateVerrazzanoFleet(clusterName string, log *zap.SugaredLogger) error {
	dclient, err := k8sutil.GetDynamicClient()
	if err != nil {
		log.Errorf("unable to get workload kubeconfig ", zap.Error(err))
		return err
	}

	gvr := getVerrazzanoFleetGVR()
	vfFetched, err := getVerrazzanoFleet(log)
	if err != nil {
		log.Errorf("unable to fetch verrazzanofleet resource from %s due to '%v'", clusterName, zap.Error(err))
		return err
	}
	if vfFetched.Object["spec"].(map[string]interface{})["verrazzano"].(map[string]interface{})["spec"].(map[string]interface{})["components"] == nil {
		vfFetched.Object["spec"].(map[string]interface{})["verrazzano"].(map[string]interface{})["spec"].(map[string]interface{})["components"] = make(map[string]interface{})
	}
	vfFetched.Object["spec"].(map[string]interface{})["verrazzano"].(map[string]interface{})["spec"].(map[string]interface{})["components"].(map[string]interface{})["console"] = map[string]interface{}{"enabled": true}

	_, err = dclient.Resource(gvr).Namespace(clusterNamespace).Update(context.TODO(), vfFetched, metav1.UpdateOptions{})
	if err != nil {
		log.Errorf("Unable to update the verrazzanofleet resource", err)
		return err
	}
	return nil
}

func createMultipleFleetForSameCluster(clusterName string, log *zap.SugaredLogger) error {
	dclient, err := k8sutil.GetDynamicClient()
	if err != nil {
		log.Errorf("unable to get workload kubeconfig ", zap.Error(err))
		return err
	}

	gvr := getVerrazzanoFleetGVR()
	vfFetched, err := getVerrazzanoFleet(log)
	if err != nil {
		log.Errorf("unable to fetch verrazzanofleetbinding resource from %s due to '%v'", clusterName, zap.Error(err))
		return err
	}
	vfDeepCopy := vfFetched.DeepCopy()
	vfDeepCopy.Object["metadata"].(map[string]interface{})["name"] = "duplicate-fleet-name"
	delete(vfDeepCopy.Object["metadata"].(map[string]interface{}), "resourceVersion")
	_, err = dclient.Resource(gvr).Namespace(clusterNamespace).Create(context.TODO(), vfDeepCopy, metav1.CreateOptions{})
	if err != nil {
		log.Errorf("Unable to create the verrazzanofleet resource", err)
		return err
	}
	return nil
}

func getVerrazzano(clusterName, namespace, vzinstallname string, log *zap.SugaredLogger) (*unstructured.Unstructured, error) {
	dclient, err := getCapiClusterDynamicClient(clusterName, log)
	if err != nil {
		log.Errorf("unable to get workload kubeconfig ", zap.Error(err))
		return nil, err
	}

	gvr := schema.GroupVersionResource{
		Group:    "install.verrazzano.io",
		Version:  "v1beta1",
		Resource: "verrazzanos",
	}

	return dclient.Resource(gvr).Namespace(namespace).Get(context.TODO(), vzinstallname, metav1.GetOptions{})
}

// DisplayWorkloadClusterResources displays the pods of workload OCNE cluster as a formatted table.
func displayWorkloadClusterResources(clusterName string, log *zap.SugaredLogger) error {
	client, err := getCapiClusterK8sClient(clusterName, log)
	if err != nil {
		return errors.Wrap(err, "Failed to get k8s client for workload cluster")
	}

	log.Infof("----------- Node in workload cluster ---------------------")
	err = showNodeInfo(client, clusterName, log)
	if err != nil {
		return err
	}

	log.Infof("----------- Pods running on workload cluster ---------------------")
	return showPodInfo(client, clusterName, log)
}

// ShowPodInfo displays the pods of workload OCNE cluster as a formatted table.
func showPodInfo(client *kubernetes.Clientset, clusterName string, log *zap.SugaredLogger) error {
	nsList, err := client.CoreV1().Namespaces().List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return errors.Wrap(err, fmt.Sprintf("failed to get list of namespaces from cluster '%s'", clusterName))
	}
	writer := tabwriter.NewWriter(os.Stdout, 0, 8, 1, '\t', tabwriter.AlignRight)
	fmt.Fprintln(writer, "Name\tNamespace\tStatus\tIP\tNode\tAge")
	//var dnsPod, ccmPod, calicokubePod *v1.Pod
	for _, ns := range nsList.Items {
		podList, err := client.CoreV1().Pods(ns.Name).List(context.TODO(), metav1.ListOptions{})
		if err != nil {
			return errors.Wrap(err, fmt.Sprintf("failed to get list of pods from cluster '%s'", clusterName))
		}
		for _, pod := range podList.Items {
			podData, err := client.CoreV1().Pods(ns.Name).Get(context.TODO(), pod.Name, metav1.GetOptions{})
			if err != nil {
				if apierrors.IsNotFound(err) {
					log.Infof("No pods in namespace '%s'", ns.Name)
				} else {
					return errors.Wrap(err, fmt.Sprintf("failed to get pod '%s' from cluster '%s'", pod.Name, clusterName))
				}
			}

			fmt.Fprintf(writer, "%v\n", fmt.Sprintf("%v\t%v\t%v\t%v\t%v\t%v",
				podData.GetName(), podData.GetNamespace(), podData.Status.Phase, podData.Status.PodIP, podData.Spec.NodeName,
				time.Until(podData.GetCreationTimestamp().Time).Abs()))
		}
	}
	writer.Flush()
	return nil
}

// ShowNodeInfo displays the nodes of workload OCNE cluster as a formatted table.
func showNodeInfo(client *kubernetes.Clientset, clustername string, log *zap.SugaredLogger) error {
	writer := tabwriter.NewWriter(os.Stdout, 0, 8, 1, '\t', tabwriter.AlignRight)
	fmt.Fprintln(writer, "Name\tRole\tVersion\tInternalIP\tExternalIP\tOSImage\tKernelVersion\tContainerRuntime\tAge")
	nodeList, err := client.CoreV1().Nodes().List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return errors.Wrap(err, fmt.Sprintf("failed to get list of nodes from cluster '%s'", clustername))
	}
	for _, node := range nodeList.Items {
		labels := node.GetLabels()
		_, nodeOK := labels[nodeLabel]
		_, controlPlaneOK := labels[nodeLabel]
		var role, internalIP string
		if nodeOK {
			role = strings.Split(nodeLabel, "/")[len(strings.Split(nodeLabel, "/"))-1]
		}
		if controlPlaneOK {
			role = strings.Split(controlPlaneLabel, "/")[len(strings.Split(controlPlaneLabel, "/"))-1]
		}

		addresses := node.Status.Addresses
		for _, address := range addresses {
			if address.Type == "InternalIP" {
				internalIP = address.Address
				break
			}
		}
		fmt.Fprintf(writer, "%v\n", fmt.Sprintf("%v\t%v\t%v\t%v\t%v\t%v\t%v\t%v\t%v",
			node.GetName(), role, node.Status.NodeInfo.KubeletVersion, internalIP, "None", node.Status.NodeInfo.OSImage, node.Status.NodeInfo.KernelVersion,
			node.Status.NodeInfo.ContainerRuntimeVersion, time.Until(node.GetCreationTimestamp().Time).Abs()))
	}
	writer.Flush()
	return nil
}

func getVerrazzanoFleetBinding(log *zap.SugaredLogger) (*unstructured.Unstructured, error) {
	dclient, err := k8sutil.GetDynamicClient()
	if err != nil {
		log.Errorf("unable to get workload kubeconfig ", zap.Error(err))
		return nil, err
	}

	gvr := schema.GroupVersionResource{
		Group:    "addons.cluster.x-k8s.io",
		Version:  "v1alpha1",
		Resource: "verrazzanofleetbindings",
	}
	return dclient.Resource(gvr).Namespace(clusterNamespace).Get(context.TODO(), clusterName, metav1.GetOptions{})
}

func getVerrazzanoFleet(log *zap.SugaredLogger) (*unstructured.Unstructured, error) {
	dclient, err := k8sutil.GetDynamicClient()
	if err != nil {
		log.Errorf("unable to get workload kubeconfig ", zap.Error(err))
		return nil, err
	}

	gvr := getVerrazzanoFleetGVR()
	return dclient.Resource(gvr).Namespace(clusterNamespace).Get(context.TODO(), vzFleetName, metav1.GetOptions{})
}

func getVerrazzanoFleetGVR() schema.GroupVersionResource {
	gvr := schema.GroupVersionResource{
		Group:    "addons.cluster.x-k8s.io",
		Version:  "v1alpha1",
		Resource: "verrazzanofleets",
	}
	return gvr
}
func deleteVerrazzanoFleet(log *zap.SugaredLogger) error {
	dclient, err := k8sutil.GetDynamicClient()
	if err != nil {
		log.Errorf("unable to get workload kubeconfig ", zap.Error(err))
		return err
	}

	gvr := getVerrazzanoFleetGVR()

	return dclient.Resource(gvr).Namespace(clusterNamespace).Delete(context.TODO(), vzFleetName, metav1.DeleteOptions{})
}

func getCapiClusterDynamicClient(clusterName string, log *zap.SugaredLogger) (dynamic.Interface, error) {

	capiK8sConfig, err := getCapiClusterKubeConfig(clusterName, log)
	if err != nil {
		return nil, err
	}

	k8sRestConfig, err := GetRESTConfigGivenString(capiK8sConfig)
	if err != nil {
		log.Errorf("failed to obtain k8s rest config : %v", zap.Error(err))
		return nil, err
	}

	dclient, err := dynamic.NewForConfig(k8sRestConfig)
	if err != nil {
		log.Errorf("unable to create dynamic client for workload cluster %v", zap.Error(err))
		return nil, err
	}
	return dclient, nil

}

var _ = t.Describe("addon e2e tests ,", Label("f:addon-provider-verrazzano-e2e-tests"), Serial, func() {
	WhenClusterAPIInstalledIt("Verify  addon controller running", func() {
		update.ValidatePods("verrazzano-fleet", addonControllerPodLabel, addonControllerPodNamespace, 1, false)
	})
	t.Context(fmt.Sprintf("Create VerrazzanoFleet resource  '%s'", clusterName), func() {
		WhenClusterAPIInstalledIt("Create verrrazanoFleet", func() {
			Eventually(func() error {
				return ctx.applyVerrazzanoFleet()
			}, shortWaitTimeout, shortPollingInterval).ShouldNot(HaveOccurred(), "Create verrazzanoFleet resource")
		})
		WhenClusterAPIInstalledIt("Verify if VerrazzanoFleetBinding resource created", func() {
			Eventually(func() error {
				return ensureVerrazzanoFleetBindingExists(clusterName, t.Logs)
			}, shortWaitTimeout, shortPollingInterval).Should(BeNil(), "verify VerrazzanoFleetBinding resource")
		})

		WhenClusterAPIInstalledIt("Display objects from CAPI workload cluster", func() {
			Eventually(func() error {
				return displayWorkloadClusterResources(clusterName, t.Logs)
			}, shortWaitTimeout, pollingInterval).Should(BeNil(), "Display objects from CAPI workload cluster")
		})

		WhenClusterAPIInstalledIt("Verify VPO on the workload cluster", func() {
			Eventually(func() bool {
				return ensureVPOPodsAreRunningOnWorkloadCluster(clusterName, "verrazzano-install", t.Logs)
			}, shortWaitTimeout, vzPollingInterval).Should(BeTrue(), "verify VPO")
		})

		WhenClusterAPIInstalledIt("Verify verrazzano CR resource", func() {
			Eventually(func() error {
				return ensureVerrazzano(clusterName, t.Logs)
			}, waitTimeOut, vzPollingInterval).Should(BeNil(), "verify verrazzano resource")
		})

		WhenClusterAPIInstalledIt("create verrazzanofleet for unknown workload cluster", func() {
			Eventually(func() error {
				return createFleetForUnknownCluster(clusterName, t.Logs)
			}, waitTimeOut, vzPollingInterval).Should(HaveOccurred(), "create verrazzanofleet for unknown workload cluster")
		})

		WhenClusterAPIInstalledIt("verify create multiple verrazzanofleet for the same workload cluster", func() {
			Eventually(func() error {
				return createMultipleFleetForSameCluster(clusterName, t.Logs)
			}, waitTimeOut, vzPollingInterval).Should(HaveOccurred(), "verify create multiple verrazzanofleet for the same workload cluster")
		})

		WhenClusterAPIInstalledIt("Display objects from CAPI workload cluster", func() {
			Eventually(func() error {
				return displayWorkloadClusterResources(clusterName, t.Logs)
			}, shortWaitTimeout, pollingInterval).Should(BeNil(), "Display objects from CAPI workload cluster")
		})

		WhenClusterAPIInstalledIt("verify update verrazzano spec in verrazzanofleet", func() {
			Eventually(func() error {
				return updateVerrazzanoFleet(clusterName, t.Logs)
			}, waitTimeOut, vzPollingInterval).Should(BeNil(), "verify update verrazzano spec in verrazzanofleet")
		})

		WhenClusterAPIInstalledIt("Verify update to verrazzano CR resource ", func() {
			Eventually(func() error {
				return ensureVerrazzano(clusterName, t.Logs)
			}, waitTimeOut, vzPollingInterval).Should(BeNil(), "verify update to verrazzano resource")
		})

		WhenClusterAPIInstalledIt("Display objects from CAPI workload cluster", func() {
			Eventually(func() error {
				return displayWorkloadClusterResources(clusterName, t.Logs)
			}, shortWaitTimeout, pollingInterval).Should(BeNil(), "Display objects from CAPI workload cluster")
		})

		WhenClusterAPIInstalledIt("Delete VerrazzanoFleet from admin cluster", func() {
			Eventually(func() error {
				return deleteVerrazzanoFleet(t.Logs)
			}, shortWaitTimeout, pollingInterval).Should(BeNil(), "Delete VerrazzanoFleet resource from admin cluster")
		})

		WhenClusterAPIInstalledIt("Verify VerrazzanoFleet resource does not exist on adminc luster", func() {
			Eventually(func() error {
				return ensureVerrazzanoFleetExists(clusterName, t.Logs)
			}, shortWaitTimeout, shortPollingInterval).Should(HaveOccurred(), "verify VerrazzanoFleetBinding resource does not exist on admin cluster")
		})

		WhenClusterAPIInstalledIt("Verify VerrazzanoFleetBinding resource does not exist on admin cluster", func() {
			Eventually(func() error {
				return ensureVerrazzanoFleetBindingExists(clusterName, t.Logs)
			}, shortWaitTimeout, shortPollingInterval).Should(HaveOccurred(), "verify VerrazzanoFleetBinding resource does not exist on admin cluster")
		})

		WhenClusterAPIInstalledIt("Verify VPO does not exist on the workload cluster", func() {
			Eventually(func() bool {
				return ensureVPOPodsAreRunningOnWorkloadCluster(clusterName, "verrazzano-install", t.Logs)
			}, shortWaitTimeout, vzPollingInterval).Should(BeFalse(), "verify VPO does not exist")
		})

		WhenClusterAPIInstalledIt("Verify verrazzano CR resource on workload cluster after deleting VerrazzanoFleet resource", func() {
			Eventually(func() error {
				return ensureVerrazzano(clusterName, t.Logs)
			}, waitTimeOut, vzPollingInterval).Should(HaveOccurred(), "verify verrazzano resource on workload cluster after deleting VerrazzanoFleet resource")
		})

		WhenClusterAPIInstalledIt("Display objects from CAPI workload cluster", func() {
			Eventually(func() error {
				return displayWorkloadClusterResources(clusterName, t.Logs)
			}, shortWaitTimeout, pollingInterval).Should(BeNil(), "Display objects from CAPI workload cluster")
		})
	})
})
