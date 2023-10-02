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
	"github.com/verrazzano/verrazzano/pkg/k8s/resource"
	"github.com/verrazzano/verrazzano/pkg/k8sutil"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg"
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
	waitTimeOut     = 30 * time.Minute
	pollingInterval = 30 * time.Second

	shortWaitTimeout            = 10 * time.Minute
	shortPollingInterval        = 60 * time.Second
	vzPollingInterval           = 60 * time.Second
	AddonControllerPodNamespace = "caapv-system"
	nodeLabel                   = "node-role.kubernetes.io/node"
	controlPlaneLabel           = "node-role.kubernetes.io/control-plane"
	AddonComponentsYamlPath     = "tests/e2e/quickcreate/templates/verrazzanofleet-none-profile.goyaml"
)

var (
	client                         clipkg.Client
	ctx                            *QCContext
	verrazzanoPlatformOperatorPods = []string{"verrazzano-platform-operator", "verrazzano-platform-operator-webhook"}
	//verrazzanoModuleOperatorPod    = []string{"verrazzano-module-operator"}
	addonControllerPod = []string{"caapv-controller-manager"}
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

	t.Logs.Infof("Creating Cluster of type [%s] - parameters [%s] = namespace [%s] - okeclustername [%s] - okeclusternamespace [%s]", ctx.ClusterType, ctx.Parameters, ctx.Namespace, okeClusterName, okeClusterNamespace)
	if err != nil {
		t.Fail(fmt.Sprintf("Failed to get default kubeconfig path: %s", err.Error()))
	}
	supported, err := pkg.IsVerrazzanoMinVersion(minimumVersion, kubeconfigPath)
	if err != nil {
		t.Logs.Errorf("Error getting Verrazzano version: %v", err)
	}
	if !supported {
		t.Logs.Infof("Skipping test because Verrazzano version is less than %s", minimumVersion)
		return
	}
	//t.ItMinimumVersion("creates a usuable cluster", minimumVersion, kcpath, createCluster)
	createCluster()

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

/*func isVerrazzanoPodRunning(clusterName, namespace string, log *zap.SugaredLogger) bool {
	k8sclient, err := getCapiClusterK8sClient(clusterName, log)
	if err != nil {
		t.Logs.Info("Failed to get k8s client for workload cluster")
		return false
	}
	vpo, err := pkg.SpecificPodsRunningInClusterWithClient(namespace, "app=verrazzano-platform-operator", k8sclient)
	if err != nil {
		AbortSuite(fmt.Sprintf("One or more pods are not running in the namespace: %v, error: %v", namespace, err))
	}
	vpoWebhook, err := pkg.SpecificPodsRunningInClusterWithClient(namespace, "app=verrazzano-platform-operator-webhook", k8sclient)
	if err != nil {
		AbortSuite(fmt.Sprintf("One or more pods are not running in the namespace: %v, error: %v", namespace, err))
	}

	if err != nil {
		log.Errorf("Unable to display resources from workload cluster ", zap.Error(err))
		return false
	}

	return vpo && vpoWebhook
}*/

func isAddonControllerPodRunning() bool {
	result, err := pkg.PodsRunning(AddonControllerPodNamespace, addonControllerPod)
	if err != nil {
		AbortSuite(fmt.Sprintf("One or more pods are not running in the namespace: %v, error: %v", AddonControllerPodNamespace, err))
	}
	return result
}

func ensureVPOPodsAreRunningOnWorkloadCluster(clusterName, namespace string, log *zap.SugaredLogger) bool {
	k8sclient, err := getCapiClusterK8sClient(clusterName, log)
	if err != nil {
		t.Logs.Info("Failed to get k8s client for workload cluster")
		return false
	}
	vpo, err := pkg.SpecificPodsRunningInClusterWithClient(namespace, "app=verrazzano-platform-operator", k8sclient)
	if err != nil {
		AbortSuite(fmt.Sprintf("One or more pods are not running in the namespace: %v, error: %v", namespace, err))
	}
	vpoWebhook, err := pkg.SpecificPodsRunningInClusterWithClient(namespace, "app=verrazzano-platform-operator-webhook", k8sclient)
	if err != nil {
		AbortSuite(fmt.Sprintf("One or more pods are not running in the namespace: %v, error: %v", namespace, err))
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
	secrets, err := clientset.CoreV1().Secrets(okeClusterNamespace).List(context.TODO(), metav1.ListOptions{})
	log.Infof("-----------------------Secrets" + secrets.String())
	secret, err := clientset.CoreV1().Secrets(okeClusterNamespace).Get(context.TODO(), fmt.Sprintf("%s-kubeconfig", clusterName), metav1.GetOptions{})
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
	/*	tmpFile, err := os.CreateTemp(os.TempDir(), clusterName)
		if err != nil {
			return nil, errors.Wrap(err, "Failed to create temporary file")
		}

		if err := os.WriteFile(tmpFile.Name(), capiK8sConfig, 0600); err != nil {
			return nil, errors.Wrap(err, "failed to write to destination file")
		}
	*/
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
	list, err := dclient.Resource(gvr).List(context.TODO(), metav1.ListOptions{})
	log.Infof("----------ALL FLEET BINDINGS------%v", list.Items)
	return dclient.Resource(gvr).Namespace(okeClusterNamespace).Get(context.TODO(), okeClusterName, metav1.GetOptions{})
}

func getCapiClusterDynamicClient(clusterName string, log *zap.SugaredLogger) (dynamic.Interface, error) {
	capiK8sConfig, err := getCapiClusterKubeConfig(clusterName, log)
	if err != nil {
		return nil, err
	}
	/*	tmpFile, err := os.CreateTemp(os.TempDir(), fmt.Sprintf("%s-kubeconfig", clusterName))
		if err != nil {
			log.Errorf("Failed to create temporary file : %v", zap.Error(err))
			return nil, err
		}

		if err := os.WriteFile(tmpFile.Name(), capiK8sConfig, 0600); err != nil {
			log.Errorf("failed to write to destination file : %v", zap.Error(err))
			return nil, err
		}*/

	//k8sRestConfig, err := k8sutil.GetKubeConfigGivenPathAndContext(tmpFile.Name(), fmt.Sprintf("%s-admin@%s", clusterName, clusterName))
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

	WhenClusterAPIInstalledIt("Deploy addon component", func() {
		Eventually(func() bool {
			pwd, _ := os.Getwd()
			t.Logs.Infof("----------Finding the addon components template, %v", pwd)
			file, err := pkg.FindTestDataFile("templates/addon-components.yaml")
			if err != nil {
				t.Logs.Infof("----------Finding the addon components template - COULD NOT FIND THE TEMPLATE PATH")
			}
			err = resource.CreateOrUpdateResourceFromFile(file, t.Logs)
			if err != nil {
				return false
			}
			return true
		}, shortPollingInterval, shortWaitTimeout).Should(BeTrue(), "Deploy addon controller")
	})
	WhenClusterAPIInstalledIt("Verify  addon controller running", func() {
		Eventually(func() bool {
			return isAddonControllerPodRunning()
		}, shortPollingInterval, shortWaitTimeout).Should(BeTrue(), "Verify addon controller pod is running")
	})

	t.Context(fmt.Sprintf("Create VerrazzanoFleet resource  '%s'", okeClusterName), func() {
		WhenClusterAPIInstalledIt("Create verrrazanoFleet", func() {
			Eventually(func() error {
				return ctx.applyVerrazzanoFleet()
			}, shortWaitTimeout, shortPollingInterval).ShouldNot(HaveOccurred(), "Create verrazzanoFleet resource")
		})

		WhenClusterAPIInstalledIt("Verify if VerrazzanoFleetBinding resource created", func() {
			Eventually(func() error {
				return ensureVerrazzanoFleetBindingExists(okeClusterName, t.Logs)
			}, shortWaitTimeout, shortPollingInterval).Should(BeNil(), "verify VerrazzanoFleetBinding resource")
		})

		WhenClusterAPIInstalledIt("Verify VPO on the workload cluster", func() {
			Eventually(func() bool {
				return ensureVPOPodsAreRunningOnWorkloadCluster(okeClusterName, "verrazzano-install", t.Logs)
			}, shortWaitTimeout, vzPollingInterval).Should(BeTrue(), "verify VPO")
		})

		WhenClusterAPIInstalledIt("Verify verrazzano CR resource", func() {
			Eventually(func() error {
				return ensureVerrazzano(okeClusterName, t.Logs)
			}, shortWaitTimeout, vzPollingInterval).Should(BeNil(), "verify verrazzano resource")
		})

		WhenClusterAPIInstalledIt("Display objects from CAPI workload cluster", func() {
			Eventually(func() error {
				return displayWorkloadClusterResources(okeClusterName, t.Logs)
			}, shortWaitTimeout, pollingInterval).Should(BeNil(), "Display objects from CAPI workload cluster")
		})
	})
})
