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
	"github.com/verrazzano/verrazzano/tests/e2e/pkg"
	"go.uber.org/zap"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"os"
	"sigs.k8s.io/cluster-api/api/v1beta1"
	clipkg "sigs.k8s.io/controller-runtime/pkg/client"
	"time"
)

const (
	minimumVersion  = "2.0.0"
	waitTimeOut     = 30 * time.Minute
	pollingInterval = 30 * time.Second

	shortWaitTimeout            = 4 * time.Minute
	shortPollingInterval        = 60 * time.Second
	vzPollingInterval           = 60 * time.Second
	AddonControllerPodNamespace = "caapv-system"
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

var _ = t.Describe("using the quick create api", func() {
	t.Context("with a kubeconfig", func() {
		kcpath, err := k8sutil.GetKubeConfigLocation()
		if err != nil {
			t.Fail(fmt.Sprintf("Failed to get default kubeconfig path: %s", err.Error()))
		}
		t.ItMinimumVersion("creates a usuable cluster", minimumVersion, kcpath, createCluster)
	})
})

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
	supported, err := pkg.IsVerrazzanoMinVersion("1.7.0", kubeconfigPath)
	if err != nil {
		t.It(description, func() {
			Fail(fmt.Sprintf("Failed to check Verrazzano version 1.7.0: %s", err.Error()))
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
	tmpFile, err := os.CreateTemp(os.TempDir(), clusterName)
	if err != nil {
		return nil, errors.Wrap(err, "Failed to create temporary file")
	}

	if err := os.WriteFile(tmpFile.Name(), capiK8sConfig, 0600); err != nil {
		return nil, errors.Wrap(err, "failed to write to destination file")
	}

	k8sRestConfig, err := k8sutil.GetKubeConfigGivenPathAndContext(tmpFile.Name(), fmt.Sprintf("%s-admin@%s", clusterName, clusterName))
	if err != nil {
		return nil, errors.Wrap(err, "Failed to get k8s rest config")
	}

	return k8sutil.GetKubernetesClientsetWithConfig(k8sRestConfig)
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

	return dclient.Resource(gvr).Namespace(okeClusterNamespace).Get(context.TODO(), okeClusterName, metav1.GetOptions{})
}

func getCapiClusterDynamicClient(clusterName string, log *zap.SugaredLogger) (dynamic.Interface, error) {
	capiK8sConfig, err := getCapiClusterKubeConfig(clusterName, log)
	if err != nil {
		return nil, err
	}
	tmpFile, err := os.CreateTemp(os.TempDir(), fmt.Sprintf("%s-kubeconfig", clusterName))
	if err != nil {
		log.Errorf("Failed to create temporary file : %v", zap.Error(err))
		return nil, err
	}

	if err := os.WriteFile(tmpFile.Name(), capiK8sConfig, 0600); err != nil {
		log.Errorf("failed to write to destination file : %v", zap.Error(err))
		return nil, err
	}

	k8sRestConfig, err := k8sutil.GetKubeConfigGivenPathAndContext(tmpFile.Name(), fmt.Sprintf("%s-admin@%s", clusterName, clusterName))
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

/*var _ = t.Describe("addon e2e tests ,", Label("f:addon-provider-verrazzano-e2e-tests"), Serial, func() {
	t.Context("Deploy and verify addon controller pod", func() {
		WhenClusterAPIInstalledIt("Deploy addon controller on admin luster", func() {
			Eventually(func() error {
				file, err := pkg.FindTestDataFile("templates/addon-components.yaml")
				if err != nil {
					return err
				}
				return resource.CreateOrUpdateResourceFromFile(file, t.Logs)
			}, shortPollingInterval, shortWaitTimeout).Should(BeTrue(), "Deploy addon controller")
		})

		WhenClusterAPIInstalledIt("Verify addon controller pod is running on admin cluster", func() {
			Eventually(func() bool {
				return isAddonControllerPodRunning()
			}, shortPollingInterval, shortWaitTimeout).Should(BeTrue(), "Verify addon controller pod is running")
		})
		t.Context(fmt.Sprintf("Deploy VerrazzanoFleet resource '%s'", okeClusterName), func() {

			t.Context(fmt.Sprintf("Create VerrazzanoFleet resource  '%s'", okeClusterName), func() {
				WhenClusterAPIInstalledIt("Create verrrazanoFleet", func() {
					Eventually(func() error {
						return ctx.applyVerrazzanoFleet()
					}, shortWaitTimeout, shortPollingInterval).ShouldNot(HaveOccurred(), "Create verrazzanoFleet resource")
				})

				WhenClusterAPIInstalledIt("Verify if VerrazzanoFleetBinding resource created", func() {
					Eventually(func() error {
						return ensureVerrazzanoFleetBindingExists(okeClusterName, t.Logs)
					}, shortWaitTimeout, vzPollingInterval).Should(BeNil(), "verify VerrazzanoFleetBinding resource")
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
			})
		})

	})
})*/
