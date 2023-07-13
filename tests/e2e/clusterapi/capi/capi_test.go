// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package capi

import (
	"context"
	"encoding/base64"
	"fmt"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg/test/framework/metrics"
	"go.uber.org/zap"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"os"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/verrazzano/verrazzano/pkg/k8sutil"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg/test/framework"
)

const (
	shortWaitTimeout               = 10 * time.Minute
	shortPollingInterval           = 30 * time.Second
	waitTimeout                    = 30 * time.Minute
	capiClusterCreationWaitTimeout = 60 * time.Minute
	pollingInterval                = 30 * time.Second
	vzPollingInterval              = 60 * time.Second

	// file paths
	clusterTemplate              = "templates/cluster-template-addons-new-vcn.yaml"
	clusterResourceSetTemplate   = "templates/cluster-template-addons.yaml"
	clusterSecurityListTemplate  = "templates/cluster-template-addons-new-vcn-securitylist.yaml"
	clusterResourceSetTemplateVZ = "templates/cluster-template-verrazzano-resource.yaml"
	ocnecpmoduldisabled          = "templates/ocnecontrolplanemoduledisabled.yaml"
	ocnecpmodulenabled           = "templates/ocnecontrolplanemoduleenabled.yaml"
	moduleOperatorStatusType     = "ModuleOperatorDeployed"
	moduleUninstalled            = "ModuleOperatorUninstalled"
	vzOperatorStatusType         = "VerrazzanoPlatformOperatorDeployed"
	vzOperatorUninstalled        = "VerrazzanoPlatformOperatorUninstalled"
)

var verrazzanoPlatformOperatorPods = []string{"verrazzano-platform-operator", "verrazzano-platform-operator-webhook"}
var verrazzanoModuleOperatorPod = []string{"verrazzano-module-operator"}

func init() {
	ensureCAPIVarsInitialized()
}

var beforeSuite = t.BeforeSuiteFunc(func() {
	start := time.Now()
	capiPrerequisites()
	metrics.Emit(t.Metrics.With("deployment_elapsed_time", time.Since(start).Milliseconds()))
})

var _ = BeforeSuite(beforeSuite)

var afterSuite = t.AfterSuiteFunc(func() {
	start := time.Now()
	//capiCleanup()
	metrics.Emit(t.Metrics.With("undeployment_elapsed_time", time.Since(start).Milliseconds()))
})

var _ = AfterSuite(afterSuite)

var t = framework.NewTestFramework("cluster-api")

var capiTest = NewCapiTestClient()

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

func EnsurePodsAreRunning(clusterName, namespace, podNamePrefix string, log *zap.SugaredLogger) bool {
	k8sclient, err := capiTest.GetCapiClusterK8sClient(clusterName, log)
	if err != nil {
		t.Logs.Info("Failed to get k8s client for workload cluster")
		return false
	}

	var podLabel string
	switch podNamePrefix {
	case "etcd":
		podLabel = "component=etcd"
	case "dns":
		podLabel = "k8s-app=kube-dns"
	case "apiserver":
		podLabel = "component=kube-apiserver"
	case "controller-manager":
		podLabel = "component=kube-controller-manager"
	case "proxy":
		podLabel = "k8s-app=kube-proxy"
	case "scheduler":
		podLabel = "component=kube-scheduler"
	case "ccm":
		podLabel = "component=oci-cloud-controller-manager"
	case "module":
		podLabel = "app.kubernetes.io/instance=verrazzano-module-operator"
	}
	result, err := pkg.SpecificPodsRunningInClusterWithClient(namespace, podLabel, k8sclient)
	if err != nil {
		AbortSuite(fmt.Sprintf("One or more pods are not running in the namespace: %v, error: %v", namespace, err))
	}

	err = capiTest.DisplayWorkloadClusterResources(ClusterName, t.Logs)
	if err != nil {
		log.Errorf("Unable to display resources from workload cluster ", zap.Error(err))
		return false
	}

	return result
}

func EnsureCSIPodsAreRunning(clusterName, namespace string, log *zap.SugaredLogger) bool {
	k8sclient, err := capiTest.GetCapiClusterK8sClient(clusterName, log)
	if err != nil {
		t.Logs.Info("Failed to get k8s client for workload cluster")
		return false
	}
	result1, err := pkg.SpecificPodsRunningInClusterWithClient(namespace, "app=csi-oci-node", k8sclient)
	if err != nil {
		AbortSuite(fmt.Sprintf("One or more pods are not running in the namespace: %v, error: %v", namespace, err))
	}
	result2, err := pkg.SpecificPodsRunningInClusterWithClient(namespace, "app=csi-oci-controller", k8sclient)
	if err != nil {
		AbortSuite(fmt.Sprintf("One or more pods are not running in the namespace: %v, error: %v", namespace, err))
	}
	err = capiTest.DisplayWorkloadClusterResources(ClusterName, t.Logs)
	if err != nil {
		log.Errorf("Unable to display resources from workload cluster ", zap.Error(err))
		return false
	}

	return result1 && result2
}

func EnsureCalicoPodsAreRunning(clusterName string, log *zap.SugaredLogger) bool {
	k8sclient, err := capiTest.GetCapiClusterK8sClient(clusterName, log)
	if err != nil {
		t.Logs.Info("Failed to get k8s client for workload cluster")
		return false
	}
	result1, err := pkg.SpecificPodsRunningInClusterWithClient("calico-system", "app.kubernetes.io/name=calico-node", k8sclient)
	if err != nil {
		AbortSuite(fmt.Sprintf("One or more pods are not running in the namespace: calico-system, error: %v", err))
	}
	result2, err := pkg.SpecificPodsRunningInClusterWithClient("calico-system", "app.kubernetes.io/name=calico-kube-controllers", k8sclient)
	if err != nil {
		AbortSuite(fmt.Sprintf("One or more pods are not running in the namespace: calico-system, error: %v", err))
	}
	result3, err := pkg.SpecificPodsRunningInClusterWithClient("calico-apiserver", "app.kubernetes.io/name=calico-apiserver", k8sclient)
	if err != nil {
		AbortSuite(fmt.Sprintf("One or more pods are not running in the namespace: calico-apiserver, error: %v", err))
	}
	result4, err := pkg.SpecificPodsRunningInClusterWithClient("calico-system", "app.kubernetes.io/name=calico-typha", k8sclient)
	if err != nil {
		AbortSuite(fmt.Sprintf("One or more pods are not running in the namespace: calico-system, error: %v", err))
	}

	err = capiTest.DisplayWorkloadClusterResources(ClusterName, t.Logs)
	if err != nil {
		log.Errorf("Unable to display resources from workload cluster ", zap.Error(err))
		return false
	}

	return result1 && result2 && result3 && result4
}

func EnsureVPOPodsAreRunning(clusterName, namespace string, log *zap.SugaredLogger) bool {
	k8sclient, err := capiTest.GetCapiClusterK8sClient(clusterName, log)
	if err != nil {
		t.Logs.Info("Failed to get k8s client for workload cluster")
		return false
	}
	result1, err := pkg.SpecificPodsRunningInClusterWithClient(namespace, "app=verrazzano-platform-operator", k8sclient)
	if err != nil {
		AbortSuite(fmt.Sprintf("One or more pods are not running in the namespace: %v, error: %v", namespace, err))
	}
	result2, err := pkg.SpecificPodsRunningInClusterWithClient(namespace, "app=verrazzano-platform-operator-webhook", k8sclient)
	if err != nil {
		AbortSuite(fmt.Sprintf("One or more pods are not running in the namespace: %v, error: %v", namespace, err))
	}

	err = capiTest.DisplayWorkloadClusterResources(ClusterName, t.Logs)
	if err != nil {
		log.Errorf("Unable to display resources from workload cluster ", zap.Error(err))
		return false
	}

	return result1 && result2
}

func EnsurePodsAreNotRunning(clusterName, namespace string, nameprefix []string, log *zap.SugaredLogger) bool {
	k8sclient, err := capiTest.GetCapiClusterK8sClient(clusterName, log)
	if err != nil {
		t.Logs.Info("Failed to get k8s client for workload cluster")
		return false
	}
	result, err := pkg.SpecificPodsPodsNotRunningInClusterWithClient(namespace, k8sclient, nameprefix)
	if err != nil {
		AbortSuite(fmt.Sprintf("One or more pods are not running in the namespace: %v, error: %v", namespace, err))
	}

	return result
}

func EnsureSecret(clusterName, namespace, secretName string, log *zap.SugaredLogger) bool {
	k8sclient, err := capiTest.GetCapiClusterK8sClient(clusterName, log)
	if err != nil {
		t.Logs.Info("Failed to get k8s client for workload cluster")
		return false
	}
	secret, err := k8sclient.CoreV1().Secrets(namespace).Get(context.TODO(), secretName, metav1.GetOptions{})
	if err != nil {
		t.Logs.Info("Failed to get secret from workload cluster: %v", zap.Error(err))
		return false
	}

	if secret != nil {
		return true
	}
	return false
}

func EnsureConfigMap(clusterName, namespace, configMapName string, log *zap.SugaredLogger) bool {
	k8sclient, err := capiTest.GetCapiClusterK8sClient(clusterName, log)
	if err != nil {
		t.Logs.Info("Failed to get k8s client for workload cluster")
		return false
	}
	cm, err := k8sclient.CoreV1().ConfigMaps(namespace).Get(context.TODO(), configMapName, metav1.GetOptions{})
	if err != nil {
		t.Logs.Info("Failed to get configmap from workload cluster: %v", zap.Error(err))
		return false
	}

	if cm != nil {
		return true
	}
	return false
}

func EnvSet() error {
	if err := os.Setenv(OCITenancyIDKeyBase64, base64.StdEncoding.EncodeToString([]byte(OCITenancyID))); err != nil {
		return err
	}
	if err := os.Setenv(OCICredsFingerprintKeyBase64, base64.StdEncoding.EncodeToString([]byte(OCIFingerprint))); err != nil {
		return err
	}
	if err := os.Setenv(OCIUserIDKeyBase64, base64.StdEncoding.EncodeToString([]byte(OCIUserID))); err != nil {
		return err
	}
	return os.Setenv(OCIRegionKeyBase64, base64.StdEncoding.EncodeToString([]byte(OCIRegion)))
}

// Run as part of BeforeSuite
func capiPrerequisites() {
	t.Logs.Infof("Start capi pre-requisites for cluster '%s'", ClusterName)

	t.Logs.Infof("Setup CAPI base64 encoded env values '%s'", ClusterName)
	Eventually(func() error {
		return EnvSet()
	}, shortWaitTimeout, shortPollingInterval).Should(BeNil())

	t.Logs.Infof("Process and set OCI private keys base64 encoded '%s'", ClusterName)
	Eventually(func() error {
		return capiTest.ProcessOCIPrivateKeysBase64(OCIPrivateKeyPath, OCIPrivateCredsKeyBase64, t.Logs)
	}, shortWaitTimeout, shortPollingInterval).Should(BeNil())

	t.Logs.Infof("Process and set OCI private key '%s'", ClusterName)
	Eventually(func() error {
		return capiTest.ProcessOCIPrivateKeysSingleLine(OCIPrivateKeyPath, OCICredsKey, t.Logs)
	}, shortWaitTimeout, shortPollingInterval).Should(BeNil())

	t.Logs.Infof("Process and set OCI node ssh key '%s'", ClusterName)
	Eventually(func() error {
		return capiTest.ProcessOCISSHKeys(OCISSHKeyPath, CAPINodeSSHKey, t.Logs)
	}, shortWaitTimeout, shortPollingInterval).Should(BeNil())

	t.Logs.Infof("Fetch and set OCI Image ID for cluster '%s'", ClusterName)
	Eventually(func() error {
		return capiTest.SetImageID(OCIImageIDKey, t.Logs)
	}, shortWaitTimeout, shortPollingInterval).Should(BeNil())

	//t.Logs.Infof("Create namespace for capi objects '%s'", ClusterName)
	//Eventually(func() error {
	//	return capiTest.CreateNamespace(OCNENamespace, t.Logs)
	//}, shortWaitTimeout, shortPollingInterval).Should(BeNil())
}

var _ = t.Describe("CAPI e2e tests ,", Label("f:platform-verrazzano.capi-e2e-tests"), Serial, func() {

	t.Context(fmt.Sprintf("Create CAPI cluster '%s'", ClusterName), func() {

		WhenClusterAPIInstalledIt("Create CAPI cluster", func() {
			Eventually(func() error {
				return capiTest.TriggerCapiClusterCreation(OCNENamespace, clusterSecurityListTemplate, t.Logs)
			}, capiClusterCreationWaitTimeout, pollingInterval).Should(BeNil(), "Create CAPI cluster")
		})

		WhenClusterAPIInstalledIt("Monitor Cluster Creation", func() {
			Eventually(func() error {
				return capiTest.MonitorCapiClusterCreation(ClusterName, t.Logs)
			}, capiClusterCreationWaitTimeout, pollingInterval).Should(BeNil(), "Monitor Cluster Creation")
		})

		WhenClusterAPIInstalledIt("Create ClusterResourceSets on CAPI cluster", func() {
			Eventually(func() error {
				return capiTest.DeployClusterInfraClusterResourceSets(ClusterName, clusterResourceSetTemplate, t.Logs)
			}, capiClusterCreationWaitTimeout, pollingInterval).Should(BeNil(), "Create CAPI cluster")
		})

		WhenClusterAPIInstalledIt("Ensure ETCD pods in kube-system of CAPI workload cluster are running", func() {
			Eventually(func() bool {
				return EnsurePodsAreRunning(ClusterName, "kube-system", "etcd", t.Logs)
			}, capiClusterCreationWaitTimeout, pollingInterval).Should(BeTrue(), "Check if pods are running")
		})

		WhenClusterAPIInstalledIt("Ensure kube API Server pods in kube-system of CAPI workload cluster are running", func() {
			Eventually(func() bool {
				return EnsurePodsAreRunning(ClusterName, "kube-system", "apiserver", t.Logs)
			}, capiClusterCreationWaitTimeout, pollingInterval).Should(BeTrue(), "Check if pods are running")
		})

		WhenClusterAPIInstalledIt("Ensure kube controller pods in kube-system of CAPI workload cluster are running", func() {
			Eventually(func() bool {
				return EnsurePodsAreRunning(ClusterName, "kube-system", "controller-manager", t.Logs)
			}, capiClusterCreationWaitTimeout, pollingInterval).Should(BeTrue(), "Check if pods are running")
		})

		WhenClusterAPIInstalledIt("Ensure kube scheduler pods in kube-system of CAPI workload cluster are running", func() {
			Eventually(func() bool {
				return EnsurePodsAreRunning(ClusterName, "kube-system", "scheduler", t.Logs)
			}, waitTimeout, pollingInterval).Should(BeTrue(), "Check if pods are running")
		})

		WhenClusterAPIInstalledIt("Ensure kube proxy pods in kube-system of CAPI workload cluster are running", func() {
			Eventually(func() bool {
				return EnsurePodsAreRunning(ClusterName, "kube-system", "proxy", t.Logs)
			}, capiClusterCreationWaitTimeout, pollingInterval).Should(BeTrue(), "Check if pods are running")
		})

		WhenClusterAPIInstalledIt("Ensure CCM pods in kube-system of CAPI workload cluster are running", func() {
			Eventually(func() bool {
				return EnsurePodsAreRunning(ClusterName, "kube-system", "ccm", t.Logs)
			}, capiClusterCreationWaitTimeout, pollingInterval).Should(BeTrue(), "Check if pods are running")
		})

		WhenClusterAPIInstalledIt("Ensure CSI pods in kube-system of CAPI workload cluster are running", func() {
			Eventually(func() bool {
				return EnsureCSIPodsAreRunning(ClusterName, "kube-system", t.Logs)
			}, capiClusterCreationWaitTimeout, pollingInterval).Should(BeTrue(), "Check if pods are running")
		})

		WhenClusterAPIInstalledIt("Ensure Calico pods in kube-system of CAPI workload cluster are running", func() {
			Eventually(func() bool {
				return EnsureCalicoPodsAreRunning(ClusterName, t.Logs)
			}, capiClusterCreationWaitTimeout, pollingInterval).Should(BeTrue(), "Check if pods are running")
		})

		WhenClusterAPIInstalledIt("Ensure Module operator pods in verrazzano-module-operator of CAPI workload cluster are running", func() {
			Eventually(func() bool {
				return EnsurePodsAreRunning(ClusterName, "verrazzano-module-operator", "module", t.Logs)
			}, capiClusterCreationWaitTimeout, pollingInterval).Should(BeTrue(), "Check if pods are running")
		})

		WhenClusterAPIInstalledIt("Ensure Verrazzano Platform operator pods in verrazzano-install of CAPI workload cluster are running", func() {
			Eventually(func() bool {
				return EnsureVPOPodsAreRunning(ClusterName, "verrazzano-install", t.Logs)
			}, capiClusterCreationWaitTimeout, pollingInterval).Should(BeTrue(), "Check if pods are running")
		})

		WhenClusterAPIInstalledIt("Ensure secret 'test-overrides' is created on CAPI workload cluster", func() {
			Eventually(func() bool {
				return EnsureSecret(ClusterName, "default", "test-overrides", t.Logs)
			}, capiClusterCreationWaitTimeout, pollingInterval).Should(BeTrue(), "Check if secret exists")
		})

		WhenClusterAPIInstalledIt("Ensure configmap 'test-overrides' is created on CAPI workload cluster", func() {
			Eventually(func() bool {
				return EnsureConfigMap(ClusterName, "default", "test-overrides", t.Logs)
			}, capiClusterCreationWaitTimeout, pollingInterval).Should(BeTrue(), "Check if secret exists")
		})

	})

	t.Context(fmt.Sprintf("Disable Module and VPO '%s'", ClusterName), func() {
		WhenClusterAPIInstalledIt("Disable Module operator and VPO from CAPI cluster", func() {
			Eventually(func() error {
				return capiTest.DeployAnyClusterResourceSets(ClusterName, ocnecpmoduldisabled, t.Logs)
			}, capiClusterCreationWaitTimeout, pollingInterval).Should(BeNil(), "disable module and vpo")
		})

		WhenClusterAPIInstalledIt("Ensure Verrazzano Module operator pods in verrazzano-module-operator of CAPI workload cluster are not running", func() {
			Eventually(func() bool {
				return EnsurePodsAreNotRunning(ClusterName, "verrazzano-module-operator", verrazzanoModuleOperatorPod, t.Logs)
			}, capiClusterCreationWaitTimeout, pollingInterval).Should(BeTrue(), "Check if module operator pods are not running")
		})

		WhenClusterAPIInstalledIt("Ensure Verrazzano Platform operator pods in verrazzano-install of CAPI workload cluster are not running", func() {
			Eventually(func() bool {
				return EnsurePodsAreNotRunning(ClusterName, "verrazzano-install", verrazzanoPlatformOperatorPods, t.Logs)
			}, capiClusterCreationWaitTimeout, pollingInterval).Should(BeTrue(), "Check if vpo pods are not running")
		})

		WhenClusterAPIInstalledIt("Check OCNE Control plane status for module operator info", func() {
			Eventually(func() bool {
				return capiTest.CheckOCNEControlPlaneStatus(ClusterName, moduleOperatorStatusType, "False", moduleUninstalled, t.Logs)
			}, capiClusterCreationWaitTimeout, pollingInterval).Should(BeTrue(), "check ocne status for module info")
		})

		WhenClusterAPIInstalledIt("Check OCNE Control plane status for verrazzano platform operator info", func() {
			Eventually(func() bool {
				return capiTest.CheckOCNEControlPlaneStatus(ClusterName, vzOperatorStatusType, "False", vzOperatorUninstalled, t.Logs)
			}, capiClusterCreationWaitTimeout, pollingInterval).Should(BeTrue(), "check ocne status for vpo info")
		})

		WhenClusterAPIInstalledIt("Display objects from CAPI workload cluster", func() {
			Eventually(func() error {
				return capiTest.DisplayWorkloadClusterResources(ClusterName, t.Logs)
			}, capiClusterCreationWaitTimeout, pollingInterval).Should(BeNil(), "Display objects from CAPI workload cluster")
		})

	})

	t.Context(fmt.Sprintf("Enable back Module and VPO '%s'", ClusterName), func() {
		WhenClusterAPIInstalledIt("Enable Module operator and VPO from CAPI cluster", func() {
			Eventually(func() error {
				return capiTest.DeployAnyClusterResourceSets(ClusterName, ocnecpmodulenabled, t.Logs)
			}, capiClusterCreationWaitTimeout, pollingInterval).Should(BeNil(), "enable module and vpo")
		})

		WhenClusterAPIInstalledIt("Check OCNE Control plane status for module operator info", func() {
			Eventually(func() bool {
				return capiTest.CheckOCNEControlPlaneStatus(ClusterName, moduleOperatorStatusType, "True", "", t.Logs)
			}, capiClusterCreationWaitTimeout, pollingInterval).Should(BeTrue(), "check ocne status for module info")
		})

		WhenClusterAPIInstalledIt("Check OCNE Control plane status for verrazzano platform operator info", func() {
			Eventually(func() bool {
				return capiTest.CheckOCNEControlPlaneStatus(ClusterName, vzOperatorStatusType, "True", "", t.Logs)
			}, capiClusterCreationWaitTimeout, pollingInterval).Should(BeTrue(), "check ocne status for vpo info")
		})

		WhenClusterAPIInstalledIt("Ensure Module operator pods in verrazzano-module-operator of CAPI workload cluster are running", func() {
			Eventually(func() bool {
				return EnsurePodsAreRunning(ClusterName, "verrazzano-module-operator", "module", t.Logs)
			}, capiClusterCreationWaitTimeout, pollingInterval).Should(BeTrue(), "Check if pods are running")
		})

		WhenClusterAPIInstalledIt("Ensure Verrazzano Platform operator pods in verrazzano-install of CAPI workload cluster are running", func() {
			Eventually(func() bool {
				return EnsureVPOPodsAreRunning(ClusterName, "verrazzano-install", t.Logs)
			}, capiClusterCreationWaitTimeout, pollingInterval).Should(BeTrue(), "Check if pods are running")
		})

	})

	t.Context(fmt.Sprintf("Deploy Verrazzano and monitor vz install status  '%s'", ClusterName), func() {
		WhenClusterAPIInstalledIt("Create ClusterResourceSets on CAPI cluster", func() {
			Eventually(func() error {
				return capiTest.DeployAnyClusterResourceSets(ClusterName, clusterResourceSetTemplateVZ, t.Logs)
			}, capiClusterCreationWaitTimeout, pollingInterval).Should(BeNil(), "Create CAPI cluster")
		})

		t.Context(fmt.Sprintf("Verrazzano installation monitoring '%s'", ClusterName), func() {
			WhenClusterAPIInstalledIt("Ensure Verrazzano install is completed on workload cluster", func() {
				Eventually(func() error {
					return capiTest.EnsureVerrazzano(ClusterName, t.Logs)
				}, capiClusterCreationWaitTimeout, vzPollingInterval).Should(BeNil(), "verify verrazzano is installed")
			})
		})
	})

})
