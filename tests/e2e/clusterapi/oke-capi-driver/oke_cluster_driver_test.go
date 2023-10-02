// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package okecapidriver

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
	skipOKEUpgradeMessage  = "Skipping test since the kubernetes version is same for install and update operations for OKE cluster upgrade"
	skipRunAllTestsMessage = "Skipping test since the runAllTests flag was set to false"
)

var (
	t = framework.NewTestFramework("okecapi-driver")

	clusterNameNodePool       string
	clusterNameNodePoolUpdate string
	clusterNameOKEUpgrade     string
	clusterNameInvalid        string
)

// Part of SynchronizedBeforeSuite, run by only one process
func synchronizedBeforeSuiteProcess1Func() []byte {
	kubeconfigPath, err := k8sutil.GetKubeConfigLocation()
	Expect(err).ShouldNot(HaveOccurred())
	if !pkg.IsRancherEnabled(kubeconfigPath) || !pkg.IsClusterAPIEnabled(kubeconfigPath) {
		AbortSuite("skipping OKE CAPI cluster driver test suite since either of rancher and capi components are not enabled")
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

	cloudCredentialName := fmt.Sprintf("strudel-cred-%s", okeCapiClusterNameSuffix)
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

	err = ensureOKEDriverVarsInitialized(t.Logs)
	Expect(err).ShouldNot(HaveOccurred())

	t.Logs.Infof("Min k8s version: %s", okeMetadataItemToInstall.KubernetesVersion.Original())
	t.Logs.Infof("Max k8s version: %s", okeMetadataItemToUpgrade.KubernetesVersion.Original())

	clusterNameNodePool = fmt.Sprintf("strudel-pool-%s", okeCapiClusterNameSuffix)
	clusterNameNodePoolUpdate = fmt.Sprintf("strudel-pool-update-%s", okeCapiClusterNameSuffix)
	clusterNameInvalid = fmt.Sprintf("strudel-invalid-k8s-%s", okeCapiClusterNameSuffix)
	clusterNameOKEUpgrade = fmt.Sprintf("strudel-oke-upgrade-%s", okeCapiClusterNameSuffix)
}

var _ = t.SynchronizedBeforeSuite(synchronizedBeforeSuiteProcess1Func, synchronizedBeforeSuiteAllProcessesFunc)

// Part of SynchronizedAfterSuite, run by only one process
func synchronizedAfterSuiteProcess1Func() {
	// Delete the clusters concurrently
	clusterNames := []string{clusterNameNodePool}
	if runAllTests {
		clusterNames = append(clusterNames, clusterNameNodePoolUpdate, clusterNameInvalid, clusterNameOKEUpgrade)
	}
	var wg sync.WaitGroup
	for _, clusterName := range clusterNames {
		if clusterName != "" {
			wg.Add(1)
			go func(name string) {
				defer wg.Done()
				// Delete the OKE cluster
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

var _ = t.Describe("OKE CAPI Cluster Driver", Label("f:rancher-capi:okecapi-cluster-driver"), func() {
	// Cluster 1. Create with a node pool.
	t.Context("OKE cluster creation with a node pool", Ordered, func() {
		var poolName string
		var poolReplicas int
		var expectedNodeCount int

		// clusterConfig specifies the parameters passed into the cluster creation
		var clusterConfig RancherOKECluster

		t.BeforeAll(func() {
			poolName = fmt.Sprintf("pool-%s", okeCapiClusterNameSuffix)
			poolReplicas = 1
			expectedNodeCount = poolReplicas
		})

		// Create the cluster and verify it comes up
		t.It("create OKE CAPI cluster", func() {
			Eventually(func() error {
				volumeSize, ocpus, memory := 150, 2, 32
				version := kubernetesVersion
				mutateFn := getMutateFnNodePoolsAndResourceUsage(poolName, version, poolReplicas, volumeSize, ocpus, memory)
				return createClusterAndFillConfig(clusterNameNodePool, &clusterConfig, t.Logs, mutateFn)
			}, shortWaitTimeout, shortPollingInterval).Should(BeNil())
		})
		t.It("check OKE cluster is active", func() {
			Eventually(func() (bool, error) { return isClusterActive(clusterNameNodePool, t.Logs) }, waitTimeout, pollingInterval).Should(
				BeTrue(), fmt.Sprintf("cluster %s is not active", clusterNameNodePool))
			Eventually(func() error {
				return verifyCluster(clusterNameNodePool, expectedNodeCount, activeClusterState, transitioningFlagNo, t.Logs)
			}, shortWaitTimeout, shortPollingInterval).Should(BeNil(), fmt.Sprintf("could not verify cluster %s", clusterNameNodePool))
		})
	})

	// Cluster 2. Create with a node pool, then perform an update.
	t.Context("OKE cluster creation with a node pool update", Ordered, func() {
		var poolName string
		var poolReplicas int
		var expectedNodeCount int

		// clusterConfig specifies the parameters passed into the cluster creation
		// and is updated as update requests are made
		var clusterConfig RancherOKECluster

		t.BeforeAll(func() {
			if !runAllTests {
				Skip(skipRunAllTestsMessage)
			}
			poolName = fmt.Sprintf("pool-%s", okeCapiClusterNameSuffix)
			poolReplicas = 1
			expectedNodeCount = poolReplicas
		})

		// Create the cluster and verify it comes up
		t.It("create OKE CAPI cluster", func() {
			Eventually(func() error {
				volumeSize, ocpus, memory := 150, 2, 32
				version := kubernetesVersion
				mutateFn := getMutateFnNodePoolsAndResourceUsage(poolName, version, poolReplicas, volumeSize, ocpus, memory)
				return createClusterAndFillConfig(clusterNameNodePoolUpdate, &clusterConfig, t.Logs, mutateFn)
			}, shortWaitTimeout, shortPollingInterval).Should(BeNil())
		})
		t.It("check OKE cluster is active", func() {
			Eventually(func() (bool, error) { return isClusterActive(clusterNameNodePoolUpdate, t.Logs) }, waitTimeout, pollingInterval).Should(
				BeTrue(), fmt.Sprintf("cluster %s is not active", clusterNameNodePoolUpdate))
			Eventually(func() error {
				return verifyCluster(clusterNameNodePoolUpdate, expectedNodeCount, activeClusterState, transitioningFlagNo, t.Logs)
			}, shortWaitTimeout, shortPollingInterval).Should(BeNil(), fmt.Sprintf("could not verify cluster %s", clusterNameNodePoolUpdate))
		})

		// Update - increase node usage
		t.It("update OKE cluster to increase node usage", func() {
			poolReplicas++
			expectedNodeCount++

			Eventually(func() error {
				volumeSize, ocpus, memory := 150, 2, 32
				version := kubernetesVersion
				mutateFn := getMutateFnNodePoolsAndResourceUsage(poolName, version, poolReplicas, volumeSize, ocpus, memory)
				return updateConfigAndCluster(&clusterConfig, mutateFn, t.Logs)
			}, shortWaitTimeout, shortPollingInterval).Should(BeNil())
		})
		t.It("check the OKE cluster updated", func() {
			Eventually(func() (bool, error) { return isClusterActive(clusterNameNodePoolUpdate, t.Logs) }, waitTimeout, pollingInterval).Should(
				BeTrue(), fmt.Sprintf("cluster %s is not active", clusterNameNodePoolUpdate))
			Eventually(func() error {
				return verifyCluster(clusterNameNodePoolUpdate, expectedNodeCount, activeClusterState, transitioningFlagNo, t.Logs)
			}, waitTimeout, pollingInterval).Should(BeNil(), fmt.Sprintf("could not verify cluster %s", clusterNameNodePoolUpdate))
		})
	})

	// Cluster 3. Pass in invalid parameters when creating a cluster.
	t.Context("OKE cluster creation with invalid kubernetes version", Ordered, func() {
		var clusterConfig RancherOKECluster

		t.BeforeAll(func() {
			if !runAllTests {
				Skip(skipRunAllTestsMessage)
			}
		})

		t.It("create OKE cluster", func() {
			// Create the cluster
			Eventually(func() error {
				mutateFn := func(config *RancherOKECluster) {
					// setting an invalid kubernetes version
					config.OKECAPIEngineConfig.KubernetesVersion = "v1.22.7"
				}
				return createClusterAndFillConfig(clusterNameInvalid, &clusterConfig, t.Logs, mutateFn)
			}, shortWaitTimeout, shortPollingInterval).Should(BeNil())
		})

		t.It("check OKE cluster is not active", func() {
			// Verify the cluster is not active
			waitTimeoutNegative := 20 * time.Minute
			Eventually(func() (bool, error) { return isClusterActive(clusterNameInvalid, t.Logs) }, waitTimeoutNegative, pollingInterval).Should(
				BeFalse(), fmt.Sprintf("cluster %s is active", clusterNameInvalid))

			// Verify that the cluster is configured correctly
			Eventually(func() error {
				return verifyCluster(clusterNameInvalid, 0, provisioningClusterState, transitioningFlagError, t.Logs)
			}, shortWaitTimeout, shortPollingInterval).Should(BeNil(), fmt.Sprintf("could not verify cluster %s", clusterNameInvalid))
		})
	})

	// Cluster 4. Create with a single node with the minimum OKE supported Kubernetes version and related info.
	// Later, update the cluster with the maximum OKE supported Kubernetes version and related info.
	t.Context("OKE cluster creation with single node with OKE cluster upgrade", Ordered, func() {
		var poolName string
		var poolReplicas, volumeSize, ocpus, memory int
		var expectedNodeCount int
		var clusterConfig RancherOKECluster

		t.BeforeAll(func() {
			if !runAllTests {
				Skip(skipRunAllTestsMessage)
			}
			if !okeMetadataItemToInstall.KubernetesVersion.LessThan(okeMetadataItemToUpgrade.KubernetesVersion) {
				Skip(skipOKEUpgradeMessage)
			}
			poolName = fmt.Sprintf("pool-%s", okeCapiClusterNameSuffix)
			poolReplicas = 1
			expectedNodeCount = poolReplicas
			volumeSize, ocpus, memory = 150, 2, 32
		})

		// Create the cluster
		t.It("create OKE cluster with the minimum OKE supported Kubernetes version and related info", func() {

			mutateFn := func(config *RancherOKECluster) {
				config.OKECAPIEngineConfig.KubernetesVersion = okeMetadataItemToInstall.KubernetesVersion.Original()
				config.OKECAPIEngineConfig.NodePools = []string{
					getNodePoolSpec(poolName, okeMetadataItemToInstall.KubernetesVersion.Original(), nodeShape,
						expectedNodeCount, memory, ocpus, volumeSize),
				}
			}
			Eventually(func() error {
				return createClusterAndFillConfig(clusterNameOKEUpgrade, &clusterConfig, t.Logs, mutateFn)
			}, shortWaitTimeout, shortPollingInterval).Should(BeNil())
		})

		t.It("check OKE cluster is active with the minimum OKE supported Kubernetes version and related info", func() {
			// Verify the cluster is active
			Eventually(func() (bool, error) { return isClusterActive(clusterNameOKEUpgrade, t.Logs) }, waitTimeout, pollingInterval).Should(
				BeTrue(), fmt.Sprintf("cluster %s is not active", clusterNameOKEUpgrade))
			// Verify that the cluster is configured correctly
			Eventually(func() error {
				return verifyCluster(clusterNameOKEUpgrade, expectedNodeCount, activeClusterState, transitioningFlagNo, t.Logs)
			}, shortWaitTimeout, shortPollingInterval).Should(BeNil(), fmt.Sprintf("could not verify cluster %s", clusterNameOKEUpgrade))
		})

		// Update the cluster
		t.It("update OKE cluster with the maximum OKE supported Kubernetes version and related info", func() {
			Eventually(func() error {
				mutateFn := func(config *RancherOKECluster) {
					config.OKECAPIEngineConfig.KubernetesVersion = okeMetadataItemToUpgrade.KubernetesVersion.Original()
					config.OKECAPIEngineConfig.NodePools = []string{
						getNodePoolSpec(poolName, okeMetadataItemToUpgrade.KubernetesVersion.Original(), nodeShape,
							expectedNodeCount, memory, ocpus, volumeSize),
					}
				}
				return updateConfigAndCluster(&clusterConfig, mutateFn, t.Logs)
			}, shortWaitTimeout, shortPollingInterval).Should(BeNil())
		})

		t.It("check the OKE cluster updated with the maximum OKE supported Kubernetes version and related info", func() {
			Eventually(func() (bool, error) { return isClusterActive(clusterNameOKEUpgrade, t.Logs) }, waitTimeout, pollingInterval).Should(
				BeTrue(), fmt.Sprintf("cluster %s is not active", clusterNameOKEUpgrade))
			Eventually(func() error {
				return verifyCluster(clusterNameOKEUpgrade, expectedNodeCount, activeClusterState, transitioningFlagNo, t.Logs)
			}, waitTimeout, pollingInterval).Should(BeNil(), fmt.Sprintf("could not verify cluster %s", clusterNameOKEUpgrade))
		})
	})
})
