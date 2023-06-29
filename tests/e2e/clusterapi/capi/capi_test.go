// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package capi

import (
	"encoding/base64"
	"fmt"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg/test/framework/metrics"
	"go.uber.org/zap"
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

	// file paths
	clusterTemplate              = "templates/cluster-template-addons-new-vcn.yaml"
	clusterResourceSetTemplate   = "templates/cluster-template-addons.yaml"
	clusterResourceSetTemplateVZ = "templates/cluster-template-addons-verrazzano.yaml"
)

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

// 'It' Wrapper to only run spec if the Velero is supported on the current Verrazzano version
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

func ensurePodsAreRunning(clusterName, namespace, podNamePrefix string, log *zap.SugaredLogger) bool {
	k8sclient, err := getCapiClusterK8sClient(clusterName, log)
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

	err = displayWorkloadClusterResources(ClusterName, t.Logs)
	if err != nil {
		log.Errorf("Unable to display resources from workload cluster ", zap.Error(err))
		return false
	}

	return result
}

func ensureCSIPodsAreRunning(clusterName, namespace string, log *zap.SugaredLogger) bool {
	k8sclient, err := getCapiClusterK8sClient(clusterName, log)
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
	err = displayWorkloadClusterResources(ClusterName, t.Logs)
	if err != nil {
		log.Errorf("Unable to display resources from workload cluster ", zap.Error(err))
		return false
	}

	return result1 && result2
}

func ensureCalicoPodsAreRunning(clusterName string, log *zap.SugaredLogger) bool {
	k8sclient, err := getCapiClusterK8sClient(clusterName, log)
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

	err = displayWorkloadClusterResources(ClusterName, t.Logs)
	if err != nil {
		log.Errorf("Unable to display resources from workload cluster ", zap.Error(err))
		return false
	}

	return result1 && result2 && result3 && result4
}

func ensureVPOPodsAreRunning(clusterName, namespace string, log *zap.SugaredLogger) bool {
	k8sclient, err := getCapiClusterK8sClient(clusterName, log)
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

	err = displayWorkloadClusterResources(ClusterName, t.Logs)
	if err != nil {
		log.Errorf("Unable to display resources from workload cluster ", zap.Error(err))
		return false
	}

	return result1 && result2
}

func envSet() error {
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

	t.Logs.Infof("Setup CAPI base64 env values '%s'", ClusterName)
	Eventually(func() error {
		return envSet()
	}, shortWaitTimeout, shortPollingInterval).Should(BeNil())

	t.Logs.Infof("Process and set OCI private key '%s'", ClusterName)
	Eventually(func() error {
		return processOCIPrivateKeysSingleLine(OCIPrivateKeyPath, OCICredsKey, t.Logs)
	}, shortWaitTimeout, shortPollingInterval).Should(BeNil())

	t.Logs.Infof("Fetch and set OCI Image ID for cluster '%s'", ClusterName)
	Eventually(func() error {
		return setImageID(OCIImageIDKey, t.Logs)
	}, shortWaitTimeout, shortPollingInterval).Should(BeNil())

	t.Logs.Infof("Process and set OCI node ssh key '%s'", ClusterName)
	Eventually(func() error {
		return processOCIPrivateKeysBase64(OCIPrivateKeyPath, OCIPrivateCredsKeyBase64, t.Logs)
	}, shortWaitTimeout, shortPollingInterval).Should(BeNil())

	t.Logs.Infof("Process and set OCI node ssh key '%s'", ClusterName)
	Eventually(func() error {
		return processOCISSHKeys(OCISSHKeyPath, CAPINodeSSHKey, t.Logs)
	}, shortWaitTimeout, shortPollingInterval).Should(BeNil())

	//t.Logs.Infof("Create namespace for capi objects '%s'", ClusterName)
	//Eventually(func() error {
	//	return createNamespace(OCNENamespace, t.Logs)
	//}, shortWaitTimeout, shortPollingInterval).Should(BeNil())
}

func ensureNSG() error {
	calicoWorkerRule := SecurityRuleDetails{
		Protocol:    "6",
		Description: "Added by Jenkins to allow communication for Typha from control plane nodes",
		Source:      "10.0.0.0/29",
		IsStateless: false,
		TcpPortMax:  5473,
		TcpPortMin:  5473,
	}

	err := updateOCINSG(ClusterName, "worker", "calico typha", &calicoWorkerRule, t.Logs)
	if err != nil {
		return err
	}

	calicoControlPlaneRule := SecurityRuleDetails{
		Protocol:    "6",
		Description: "Added by Jenkins to allow communication for Typha from worker nodes",
		Source:      "10.0.64.0/20",
		IsStateless: false,
		TcpPortMax:  5473,
		TcpPortMin:  5473,
	}

	err = updateOCINSG(ClusterName, "control-plane", "calico typha", &calicoControlPlaneRule, t.Logs)
	if err != nil {
		return err
	}

	udpWorkerRule := SecurityRuleDetails{
		Protocol:    "17",
		Description: "Added by Jenkins to allow UDP communication from control plane node",
		Source:      "10.0.0.0/29",
		IsStateless: false,
	}

	err = updateOCINSG(ClusterName, "worker", "udp", &udpWorkerRule, t.Logs)
	if err != nil {
		return err
	}

	udpControlPlaneRule := SecurityRuleDetails{
		Protocol:    "17",
		Description: "Added by Jenkins to allow UDP communication from worker node",
		Source:      "10.0.64.0/20",
		IsStateless: false,
	}

	return updateOCINSG(ClusterName, "control-plane", "udp", &udpControlPlaneRule, t.Logs)

}

func capiCleanup() {
	t.Logs.Infof("Deleting namespace for capi objects '%s'", ClusterName)
	Eventually(func() error {
		return deleteNamespace(OCNENamespace, t.Logs)
	}, shortWaitTimeout, shortPollingInterval).Should(BeNil())

}

var _ = t.Describe("CAPI e2e tests ,", Label("f:platform-verrazzano.capi-e2e-tests"), Serial, func() {

	t.Context(fmt.Sprintf("Create CAPI cluster '%s'", ClusterName), func() {

		WhenClusterAPIInstalledIt("Create CAPI cluster", func() {
			Eventually(func() error {
				return TriggerCapiClusterCreation(OCNENamespace, clusterTemplate, t.Logs)
			}, waitTimeout, pollingInterval).Should(BeNil(), "Create CAPI cluster")
		})

		WhenClusterAPIInstalledIt("Monitor Cluster Creation", func() {
			Eventually(func() error {
				return MonitorCapiClusterCreation(ClusterName, t.Logs)
			}, capiClusterCreationWaitTimeout, pollingInterval).Should(BeNil(), "Monitor Cluster Creation")
		})

		WhenClusterAPIInstalledIt("Create ClusterResourceSets on CAPI cluster", func() {
			Eventually(func() error {
				return DeployClusterResourceSets(ClusterName, clusterResourceSetTemplate, t.Logs)
			}, waitTimeout, pollingInterval).Should(BeNil(), "Create CAPI cluster")
		})

		WhenClusterAPIInstalledIt("Update NSG for new VCN created by CAPi cluster", func() {
			Eventually(func() error {
				return ensureNSG()
			}, waitTimeout, pollingInterval).Should(BeNil(), "ensure nsg update")
		})

		WhenClusterAPIInstalledIt("Ensure ETCD pods in kube-system of CAPI workload cluster are running", func() {
			Eventually(func() bool {
				return ensurePodsAreRunning(ClusterName, "kube-system", "etcd", t.Logs)
			}, waitTimeout, pollingInterval).Should(BeTrue(), "Check if pods are running")
		})

		WhenClusterAPIInstalledIt("Ensure kube API Server pods in kube-system of CAPI workload cluster are running", func() {
			Eventually(func() bool {
				return ensurePodsAreRunning(ClusterName, "kube-system", "apiserver", t.Logs)
			}, waitTimeout, pollingInterval).Should(BeTrue(), "Check if pods are running")
		})

		WhenClusterAPIInstalledIt("Ensure kube controller pods in kube-system of CAPI workload cluster are running", func() {
			Eventually(func() bool {
				return ensurePodsAreRunning(ClusterName, "kube-system", "controller-manager", t.Logs)
			}, waitTimeout, pollingInterval).Should(BeTrue(), "Check if pods are running")
		})

		WhenClusterAPIInstalledIt("Ensure kube scheduler pods in kube-system of CAPI workload cluster are running", func() {
			Eventually(func() bool {
				return ensurePodsAreRunning(ClusterName, "kube-system", "scheduler", t.Logs)
			}, waitTimeout, pollingInterval).Should(BeTrue(), "Check if pods are running")
		})

		WhenClusterAPIInstalledIt("Ensure kube proxy pods in kube-system of CAPI workload cluster are running", func() {
			Eventually(func() bool {
				return ensurePodsAreRunning(ClusterName, "kube-system", "proxy", t.Logs)
			}, waitTimeout, pollingInterval).Should(BeTrue(), "Check if pods are running")
		})

		WhenClusterAPIInstalledIt("Ensure CCM pods in kube-system of CAPI workload cluster are running", func() {
			Eventually(func() bool {
				return ensurePodsAreRunning(ClusterName, "kube-system", "ccm", t.Logs)
			}, waitTimeout, pollingInterval).Should(BeTrue(), "Check if pods are running")
		})

		WhenClusterAPIInstalledIt("Ensure CSI pods in kube-system of CAPI workload cluster are running", func() {
			Eventually(func() bool {
				return ensureCSIPodsAreRunning(ClusterName, "kube-system", t.Logs)
			}, waitTimeout, pollingInterval).Should(BeTrue(), "Check if pods are running")
		})

		WhenClusterAPIInstalledIt("Display objects from CAPI workload cluster", func() {
			Eventually(func() error {
				return displayWorkloadClusterResources(ClusterName, t.Logs)
			}, waitTimeout, pollingInterval).Should(BeNil(), "Display objects from CAPI workload cluster")
		})

		WhenClusterAPIInstalledIt("Ensure Calico pods in kube-system of CAPI workload cluster are running", func() {
			Eventually(func() bool {
				return ensureCalicoPodsAreRunning(ClusterName, t.Logs)
			}, waitTimeout, pollingInterval).Should(BeTrue(), "Check if pods are running")
		})

		WhenClusterAPIInstalledIt("Ensure Module operator pods in verrazzano-module-operator of CAPI workload cluster are running", func() {
			Eventually(func() bool {
				return ensurePodsAreRunning(ClusterName, "verrazzano-module-operator", "module", t.Logs)
			}, waitTimeout, pollingInterval).Should(BeTrue(), "Check if pods are running")
		})

		WhenClusterAPIInstalledIt("Ensure Verrazzano Platform operator pods in verrazzano-install of CAPI workload cluster are running", func() {
			Eventually(func() bool {
				return ensureVPOPodsAreRunning(ClusterName, "verrazzano-install", t.Logs)
			}, waitTimeout, pollingInterval).Should(BeTrue(), "Check if pods are running")
		})
	})

	t.Context(fmt.Sprintf("Verrazzano installation monitoring '%s'", ClusterName), func() {
		WhenClusterAPIInstalledIt("Ensure Verrazzano install is completed on workload cluster", func() {
			Eventually(func() error {
				return ensureVerrazzano(ClusterName, t.Logs)
			}, waitTimeout, pollingInterval).Should(BeNil(), "verify verrazzano is installed")
		})
	})

	//t.Context(fmt.Sprintf("Verrazzano cleanup and monitoring '%s'", ClusterName), func() {
	//	WhenClusterAPIInstalledIt("Ensure Verrazzano is uninstalled from workload cluster", func() {
	//		Eventually(func() error {
	//			return deleteVerrazzano(ClusterName, t.Logs)
	//		}, waitTimeout, pollingInterval).Should(BeNil(), "start verrazzano uninstall")
	//	})
	//	WhenClusterAPIInstalledIt("Ensure Verrazzano is removed from workload cluster", func() {
	//		Eventually(func() error {
	//			return getVerrazzano(ClusterName, t.Logs)
	//		}, waitTimeout, pollingInterval).Should(BeNil(), "check verrazzano")
	//	})
	//})

	//t.Context(fmt.Sprintf("Delete CAPI cluster '%s'", ClusterName), func() {
	//	WhenClusterAPIInstalledIt("Delete CAPI cluster", func() {
	//		Eventually(func() error {
	//			return TriggerCapiClusterDeletion(ClusterName, OCNENamespace, t.Logs)
	//		}, waitTimeout, pollingInterval).Should(BeNil(), "Delete CAPI cluster")
	//	})
	//
	//	WhenClusterAPIInstalledIt("Monitor Cluster Deletion", func() {
	//		Eventually(func() error {
	//			return MonitorCapiClusterDeletion(ClusterName, t.Logs)
	//		}, waitTimeout, pollingInterval).Should(BeNil(), "Monitor Cluster Deletion")
	//	})
	//
	//})

})
