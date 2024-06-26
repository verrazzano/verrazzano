// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package ocnedriver

import (
	"fmt"
	"sync"
	"time"

	"github.com/verrazzano/verrazzano/pkg/k8sutil"
	"github.com/verrazzano/verrazzano/tests/e2e/backup/helpers"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg/test/framework"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

const (
	shortWaitTimeout       = 10 * time.Minute
	shortPollingInterval   = 10 * time.Second
	waitTimeout            = 45 * time.Minute
	pollingInterval        = 30 * time.Second
	skipOCNEUpgradeMessage = "Skipping test since the kubernetes version is same for install and update operations for OCNE cluster upgrade"
	skipRunAllTestsMessage = "Skipping test since the runAllTests flag was set to false"
)

var (
	t = framework.NewTestFramework("capi-ocne-driver")

	clusterNameSingleNode            string
	clusterNameNodePool              string
	clusterNameSingleNodeOCNEUpgrade string
	clusterNameSingleNodeInvalid     string
)

// Part of SynchronizedBeforeSuite, run by only one process
func synchronizedBeforeSuiteProcess1Func() []byte {
	kubeconfigPath, err := k8sutil.GetKubeConfigLocation()
	Expect(err).ShouldNot(HaveOccurred())
	if !pkg.IsRancherEnabled(kubeconfigPath) || !pkg.IsClusterAPIEnabled(kubeconfigPath) {
		AbortSuite("skipping ocne cluster driver test suite since either of rancher and capi components are not enabled")
	}

	httpClient, err = pkg.GetVerrazzanoHTTPClient(kubeconfigPath)
	if err != nil {
		AbortSuite(fmt.Sprintf("failed getting http client: %v", err))
	}

	rancherURL, err = helpers.GetRancherURL(t.Logs)
	if err != nil {
		AbortSuite(fmt.Sprintf("failed getting rancherURL: %v", err))
	}

	verifyRequiredEnvironmentVariables()

	cloudCredentialName := fmt.Sprintf("strudel-cred-%s", ocneClusterNameSuffix)
	// Create the cloud credential to be used for all tests
	var credentialID string
	Eventually(func() error {
		var err error
		credentialID, err = createCloudCredential(cloudCredentialName, t.Logs)
		return err
	}, shortWaitTimeout, shortPollingInterval).Should(BeNil())

	Eventually(func() error {
		return validateCloudCredential(credentialID, t.Logs)
	}, shortWaitTimeout, shortPollingInterval).Should(BeNil())

	// Return byte encoded cloud credential ID to be shared across all processes
	return []byte(credentialID)
}

// Part of SynchronizedBeforeSuite, run by all processes
func synchronizedBeforeSuiteAllProcessesFunc(credentialIDBytes []byte) {
	// Define global variables for all processes
	cloudCredentialID = string(credentialIDBytes)

	kubeconfigPath, err := k8sutil.GetKubeConfigLocation()
	Expect(err).ShouldNot(HaveOccurred())

	httpClient, err = pkg.GetVerrazzanoHTTPClient(kubeconfigPath)
	if err != nil {
		AbortSuite(fmt.Sprintf("failed getting http client: %v", err))
	}

	rancherURL, err = helpers.GetRancherURL(t.Logs)
	if err != nil {
		AbortSuite(fmt.Sprintf("failed getting rancherURL: %v", err))
	}

	// Calling this method again so that all processes have the variables initialized
	verifyRequiredEnvironmentVariables()

	err = ensureOCNEDriverVarsInitialized(t.Logs)
	Expect(err).ShouldNot(HaveOccurred())

	t.Logs.Infof("Min k8s version: %s", ocneMetadataItemToInstall.KubernetesVersion.Original())
	t.Logs.Infof("Max k8s version: %s", ocneMetadataItemToUpgrade.KubernetesVersion.Original())

	clusterNameSingleNode = fmt.Sprintf("strudel-single-%s", ocneClusterNameSuffix)
	clusterNameNodePool = fmt.Sprintf("strudel-pool-%s", ocneClusterNameSuffix)
	clusterNameSingleNodeInvalid = fmt.Sprintf("strudel-single-invalid-k8s-%s", ocneClusterNameSuffix)
	clusterNameSingleNodeOCNEUpgrade = fmt.Sprintf("strudel-single-ocne-upgrade-%s", ocneClusterNameSuffix)
}

var _ = t.SynchronizedBeforeSuite(synchronizedBeforeSuiteProcess1Func, synchronizedBeforeSuiteAllProcessesFunc)

// Part of SynchronizedAfterSuite, run by only one process
func synchronizedAfterSuiteProcess1Func() {
	// Delete the clusters concurrently
	clusterNames := []string{clusterNameSingleNode}
	if runAllTests {
		clusterNames = append(clusterNames, clusterNameNodePool, clusterNameSingleNodeInvalid, clusterNameSingleNodeOCNEUpgrade)
	}
	var wg sync.WaitGroup
	for _, clusterName := range clusterNames {
		if clusterName != "" {
			wg.Add(1)
			go func(name string) {
				defer wg.Done()
				// Delete the OCNE cluster
				Eventually(func() error {
					return deleteCluster(name, t.Logs)
				}, shortWaitTimeout, shortPollingInterval).Should(BeNil())

				// Verify the cluster is deleted
				Eventually(func() (bool, error) { return isClusterDeleted(name, t.Logs) }, waitTimeout, pollingInterval).Should(
					BeTrue(), fmt.Sprintf("cluster %s is not deleted", name))
			}(clusterName)
		}
	}
	wg.Wait()

	// Delete the credential
	deleteCredential(cloudCredentialID, t.Logs)

	// Verify the credential is deleted
	Eventually(func() (bool, error) { return isCredentialDeleted(cloudCredentialID, t.Logs) }, waitTimeout, pollingInterval).Should(
		BeTrue(), fmt.Sprintf("cloud credential %s is not deleted", cloudCredentialID))
}

var _ = t.SynchronizedAfterSuite(func() {}, synchronizedAfterSuiteProcess1Func)

var _ = t.Describe("OCNE Cluster Driver", Label("f:rancher-capi:ocne-cluster-driver"), func() {
	// Cluster 1. Create with a single node.
	t.Context("OCNE cluster creation with single node", Ordered, func() {
		var clusterConfig RancherOCNECluster

		t.It("create OCNE cluster", func() {
			// Create the cluster
			Eventually(func() error {
				return createClusterAndFillConfig(clusterNameSingleNode, &clusterConfig, t.Logs, nil)
			}, shortWaitTimeout, shortPollingInterval).Should(BeNil())
		})

		t.It("check OCNE cluster is active", func() {
			// Verify the cluster is active
			Eventually(func() (bool, error) { return isClusterActive(clusterNameSingleNode, t.Logs) }, waitTimeout, pollingInterval).Should(
				BeTrue(), fmt.Sprintf("cluster %s is not active", clusterNameSingleNode))
			// Verify that the cluster is configured correctly
			Eventually(func() error {
				return verifyCluster(clusterNameSingleNode, 1, activeClusterState, transitioningFlagNo, t.Logs)
			}, shortWaitTimeout, shortPollingInterval).Should(BeNil(), fmt.Sprintf("could not verify cluster %s", clusterNameSingleNode))
		})
	})

	// Cluster 2. Create with a node pool, then perform an update.
	t.Context("OCNE cluster creation with node pools", Ordered, func() {
		var poolName string
		var poolReplicas int
		var expectedNodeCount int

		// clusterConfig specifies the parameters passed into the cluster creation
		// and is updated as update requests are made
		var clusterConfig RancherOCNECluster

		t.BeforeAll(func() {
			if !runAllTests {
				Skip(skipRunAllTestsMessage)
			}

			poolName = fmt.Sprintf("pool-%s", ocneClusterNameSuffix)
			poolReplicas = 2
			expectedNodeCount = poolReplicas + numControlPlaneNodes
		})

		// Create the cluster and verify it comes up
		t.It("create OCNE cluster", func() {
			Eventually(func() error {
				volumeSize, ocpus, memory := 150, 2, 32
				mutateFn := getMutateFnNodePoolsAndResourceUsage(poolName, poolReplicas, volumeSize, ocpus, memory)
				return createClusterAndFillConfig(clusterNameNodePool, &clusterConfig, t.Logs, mutateFn)
			}, shortWaitTimeout, shortPollingInterval).Should(BeNil())
		})
		t.It("check OCNE cluster is active", func() {
			Eventually(func() (bool, error) { return isClusterActive(clusterNameNodePool, t.Logs) }, waitTimeout, pollingInterval).Should(
				BeTrue(), fmt.Sprintf("cluster %s is not active", clusterNameNodePool))
			Eventually(func() error {
				return verifyCluster(clusterNameNodePool, expectedNodeCount, activeClusterState, transitioningFlagNo, t.Logs)
			}, shortWaitTimeout, shortPollingInterval).Should(BeNil(), fmt.Sprintf("could not verify cluster %s", clusterNameNodePool))
		})

		// Update - decrease resource usage
		t.It("update OCNE cluster to decrease resource usage", func() {
			poolReplicas--
			expectedNodeCount--

			Eventually(func() error {
				volumeSize, ocpus, memory := 100, 1, 16
				mutateFn := getMutateFnNodePoolsAndResourceUsage(poolName, poolReplicas, volumeSize, ocpus, memory)
				return updateConfigAndCluster(&clusterConfig, mutateFn, t.Logs)
			}, shortWaitTimeout, shortPollingInterval).Should(BeNil())
		})
		t.It("check the OCNE cluster updated", func() {
			Eventually(func() (bool, error) { return isClusterActive(clusterNameNodePool, t.Logs) }, waitTimeout, pollingInterval).Should(
				BeTrue(), fmt.Sprintf("cluster %s is not active", clusterNameNodePool))
			Eventually(func() error {
				return verifyCluster(clusterNameNodePool, expectedNodeCount, activeClusterState, transitioningFlagNo, t.Logs)
			}, waitTimeout, pollingInterval).Should(BeNil(), fmt.Sprintf("could not verify cluster %s", clusterNameNodePool))
		})
	})

	// Cluster 3. Pass in invalid parameters when creating a cluster.
	t.Context("OCNE cluster creation with single node invalid kubernetes version", Ordered, func() {
		var clusterConfig RancherOCNECluster

		t.BeforeAll(func() {
			if !runAllTests {
				Skip(skipRunAllTestsMessage)
			}
		})

		t.It("create OCNE cluster", func() {
			// Create the cluster
			Eventually(func() error {
				mutateFn := func(config *RancherOCNECluster) {
					// setting an invalid kubernetes version
					config.OciocneEngineConfig.KubernetesVersion = "v1.22.7"
				}
				return createClusterAndFillConfig(clusterNameSingleNodeInvalid, &clusterConfig, t.Logs, mutateFn)
			}, shortWaitTimeout, shortPollingInterval).Should(BeNil())
		})

		t.It("check OCNE cluster is not active", func() {
			// Verify the cluster is not active
			waitTimeoutNegative := 20 * time.Minute
			Eventually(func() (bool, error) { return isClusterActive(clusterNameSingleNodeInvalid, t.Logs) }, waitTimeoutNegative, pollingInterval).Should(
				BeFalse(), fmt.Sprintf("cluster %s is active", clusterNameSingleNodeInvalid))

			// Verify that the cluster is configured correctly
			Eventually(func() error {
				return verifyCluster(clusterNameSingleNodeInvalid, 0, provisioningClusterState, transitioningFlagError, t.Logs)
			}, shortWaitTimeout, shortPollingInterval).Should(BeNil(), fmt.Sprintf("could not verify cluster %s", clusterNameSingleNodeInvalid))
		})
	})

	// Cluster 4. Create with a single node with the minimum OCNE supported Kubernetes version and related info.
	// Later, update the cluster with the maximum OCNE supported Kubernetes version and related info.
	t.Context("OCNE cluster creation with single node with OCNE cluster upgrade", Ordered, func() {
		var clusterConfig RancherOCNECluster

		t.BeforeAll(func() {
			if !runAllTests {
				Skip(skipRunAllTestsMessage)
			}
			if !ocneMetadataItemToInstall.KubernetesVersion.LessThan(ocneMetadataItemToUpgrade.KubernetesVersion) {
				Skip(skipOCNEUpgradeMessage)
			}
		})

		// Create the cluster
		t.It("create OCNE cluster with the minimum OCNE supported Kubernetes version and related info", func() {
			mutateFn := func(config *RancherOCNECluster) {
				config.OciocneEngineConfig.KubernetesVersion = ocneMetadataItemToInstall.KubernetesVersion.Original()
				config.OciocneEngineConfig.OcneVersion = ocneMetadataItemToInstall.Release
				config.OciocneEngineConfig.EtcdImageTag = ocneMetadataItemToInstall.ContainerImages.Etcd
				config.OciocneEngineConfig.CorednsImageTag = ocneMetadataItemToInstall.ContainerImages.Coredns
				config.OciocneEngineConfig.TigeraImageTag = ocneMetadataItemToInstall.ContainerImages.TigeraOperator
				config.OciocneEngineConfig.CorednsImageTag = ocneMetadataItemToInstall.ContainerImages.Coredns
			}
			Eventually(func() error {
				return createClusterAndFillConfig(clusterNameSingleNodeOCNEUpgrade, &clusterConfig, t.Logs, mutateFn)
			}, shortWaitTimeout, shortPollingInterval).Should(BeNil())
		})

		t.It("check OCNE cluster is active with the minimum OCNE supported Kubernetes version and related info", func() {
			// Verify the cluster is active
			Eventually(func() (bool, error) { return isClusterActive(clusterNameSingleNodeOCNEUpgrade, t.Logs) }, waitTimeout, pollingInterval).Should(
				BeTrue(), fmt.Sprintf("cluster %s is not active", clusterNameSingleNodeOCNEUpgrade))
			// Verify that the cluster is configured correctly
			Eventually(func() error {
				return verifyCluster(clusterNameSingleNodeOCNEUpgrade, 1, activeClusterState, transitioningFlagNo, t.Logs)
			}, shortWaitTimeout, shortPollingInterval).Should(BeNil(), fmt.Sprintf("could not verify cluster %s", clusterNameSingleNodeOCNEUpgrade))
		})

		// Update the cluster
		t.It("update OCNE cluster with the maximum OCNE supported Kubernetes version and related info", func() {
			Eventually(func() error {
				mutateFn := func(config *RancherOCNECluster) {
					config.OciocneEngineConfig.KubernetesVersion = ocneMetadataItemToUpgrade.KubernetesVersion.Original()
					config.OciocneEngineConfig.OcneVersion = ocneMetadataItemToUpgrade.Release
					config.OciocneEngineConfig.EtcdImageTag = ocneMetadataItemToUpgrade.ContainerImages.Etcd
					config.OciocneEngineConfig.CorednsImageTag = ocneMetadataItemToUpgrade.ContainerImages.Coredns
					config.OciocneEngineConfig.TigeraImageTag = ocneMetadataItemToUpgrade.ContainerImages.TigeraOperator
					config.OciocneEngineConfig.CorednsImageTag = ocneMetadataItemToUpgrade.ContainerImages.Coredns
				}
				return updateConfigAndCluster(&clusterConfig, mutateFn, t.Logs)
			}, shortWaitTimeout, shortPollingInterval).Should(BeNil())
		})

		t.It("check the OCNE cluster updated with the maximum OCNE supported Kubernetes version and related info", func() {
			Eventually(func() (bool, error) { return isClusterActive(clusterNameSingleNodeOCNEUpgrade, t.Logs) }, waitTimeout, pollingInterval).Should(
				BeTrue(), fmt.Sprintf("cluster %s is not active", clusterNameSingleNodeOCNEUpgrade))
			Eventually(func() error {
				return verifyCluster(clusterNameSingleNodeOCNEUpgrade, 1, activeClusterState, transitioningFlagNo, t.Logs)
			}, waitTimeout, pollingInterval).Should(BeNil(), fmt.Sprintf("could not verify cluster %s", clusterNameSingleNodeOCNEUpgrade))
		})
	})
})
