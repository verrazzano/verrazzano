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
	"go.uber.org/zap"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

const (
	shortWaitTimeout     = 10 * time.Minute
	shortPollingInterval = 10 * time.Second
	waitTimeout          = 45 * time.Minute
	pollingInterval      = 30 * time.Second
)

var (
	t                            = framework.NewTestFramework("capi-ocne-driver")
	clusterNameSingleNode        string
	clusterNameNodePool          string
	clusterNameSingleNodeInvalid string

	// Keep track of which clusters to clean up at the end by ID, since cluster names can be edited.
	clusterIDsToDelete []string
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

	clusterNameSingleNode = fmt.Sprintf("strudel-single-%s", ocneClusterNameSuffix)
	clusterNameNodePool = fmt.Sprintf("strudel-pool-%s", ocneClusterNameSuffix)
	clusterNameSingleNodeInvalid = fmt.Sprintf("strudel-single-invalid-k8s-%s", ocneClusterNameSuffix)
}

var _ = t.SynchronizedBeforeSuite(synchronizedBeforeSuiteProcess1Func, synchronizedBeforeSuiteAllProcessesFunc)

// Part of SynchronizedAfterSuite, run by all processes
func synchronizedAfterSuiteAllProcessesFunc() {
	// Delete the clusters tracked by this process concurrently.
	// Each clusterID should only be present on one process's clusterIDsToDelete list.
	var wg sync.WaitGroup
	for _, clusterID := range clusterIDsToDelete {
		wg.Add(1)
		go func(id string) {
			defer wg.Done()
			// Delete the OCNE cluster
			Eventually(func() error {
				return deleteClusterFromID(id, t.Logs)
			}, shortWaitTimeout, shortPollingInterval).Should(BeNil())

			// Verify the cluster is deleted
			Eventually(func() (bool, error) { return isClusterDeletedFromID(id, t.Logs) }, waitTimeout, pollingInterval).Should(
				BeTrue(), fmt.Sprintf("cluster %s is not deleted", id))
		}(clusterID)
	}
	wg.Wait()
}

// Part of SynchronizedAfterSuite, run by only one process
func synchronizedAfterSuiteProcess1Func() {
	// Delete the credential
	deleteCredential(cloudCredentialID, t.Logs)

	// Verify the credential is deleted
	Eventually(func() (bool, error) { return isCredentialDeleted(cloudCredentialID, t.Logs) }, waitTimeout, pollingInterval).Should(
		BeTrue(), fmt.Sprintf("cloud credential %s is not deleted", cloudCredentialID))
}

var _ = t.SynchronizedAfterSuite(synchronizedAfterSuiteAllProcessesFunc, synchronizedAfterSuiteProcess1Func)

var _ = t.Describe("OCNE Cluster Driver", Label("f:rancher-capi:ocne-cluster-driver"), func() {
	// Cluster 1. Create with a single node, then perform some updates.
	t.Context("OCNE cluster creation with single node", Ordered, func() {
		var clusterConfig RancherOCNECluster

		t.It("create OCNE cluster", func() {
			// Create the cluster
			Eventually(func() error {
				return createClusterAndFillConfig(clusterNameSingleNode, &clusterConfig, t.Logs, nil)
			}, shortWaitTimeout, shortPollingInterval).Should(BeNil())

			// Track this cluster's ID for deletion later
			clusterID, err := getClusterIDFromName(clusterNameSingleNode, t.Logs)
			Expect(err).ShouldNot(HaveOccurred())
			clusterIDsToDelete = append(clusterIDsToDelete, clusterID)
			t.Logs.Infof("the cluster ID of %s is %s", clusterNameSingleNode, clusterID)
		})

		t.It("check OCNE cluster is active", func() {
			// Verify the cluster is active
			Eventually(func() (bool, error) { return isClusterActive(clusterNameSingleNode, t.Logs) }, waitTimeout, pollingInterval).Should(
				BeTrue(), fmt.Sprintf("cluster %s is not active", clusterNameSingleNode))
			// Verify that the cluster is configured correctly
			Eventually(func() error {
				return verifyCluster(clusterNameSingleNode, "", 1, activeClusterState, transitioningFlagNo, t.Logs)
			}, shortWaitTimeout, shortPollingInterval).Should(BeNil(), fmt.Sprintf("could not verify cluster %s", clusterNameSingleNode))
		})
	})

	// Cluster 2. Create with a node pool, then perform some updates.
	t.Context("OCNE cluster creation with node pools", Ordered, func() {
		var poolName string

		// clusterConfig specifies the parameters passed into the cluster creation
		// and is updated as update requests are made
		var clusterConfig RancherOCNECluster

		// expected number of nodes is number of worker nodes + control plane node
		poolReplicas := 2
		expectedNodeCount := poolReplicas + numControlPlaneNodes

		// Create the cluster and verify it comes up
		t.It("create OCNE cluster", func() {
			poolName = fmt.Sprintf("pool-%s", ocneClusterNameSuffix)

			Eventually(func() error {
				volumeSize, ocpus, memory := 150, 2, 32
				mutateFn := getMutateFnNodePoolsAndResourceUsage(poolName, poolReplicas, volumeSize, ocpus, memory)
				return createClusterAndFillConfig(clusterNameNodePool, &clusterConfig, t.Logs, mutateFn)
			}, shortWaitTimeout, shortPollingInterval).Should(BeNil())

			// Track this cluster's ID for deletion later
			clusterID, err := getClusterIDFromName(clusterNameNodePool, t.Logs)
			Expect(err).ShouldNot(HaveOccurred())
			clusterIDsToDelete = append(clusterIDsToDelete, clusterID)
			t.Logs.Infof("the cluster ID of %s is %s", clusterNameNodePool, clusterID)
		})
		t.It("check OCNE cluster is active", func() {
			Eventually(func() (bool, error) { return isClusterActive(clusterNameNodePool, t.Logs) }, waitTimeout, pollingInterval).Should(
				BeTrue(), fmt.Sprintf("cluster %s is not active", clusterNameNodePool))
			Eventually(func() error {
				return verifyCluster(clusterNameNodePool, "", expectedNodeCount, activeClusterState, transitioningFlagNo, t.Logs)
			}, shortWaitTimeout, shortPollingInterval).Should(BeNil(), fmt.Sprintf("could not verify cluster %s", clusterNameNodePool))
		})

		// Update - decrease resource usage
		t.It("update OCNE cluster to decrease resource usage", func() {
			poolReplicas = 1
			Eventually(func() error {
				volumeSize, ocpus, memory := 100, 1, 16
				mutateFn := getMutateFnNodePoolsAndResourceUsage(poolName, poolReplicas, volumeSize, ocpus, memory)
				return updateConfigAndCluster(&clusterConfig, mutateFn, t.Logs)
			}, shortWaitTimeout, shortPollingInterval).Should(BeNil())
			// expect 3 control plane nodes, 1 worker nodes
			expectedNodeCount = 4
		})
		t.It("check the OCNE cluster updated", func() {
			Eventually(func() (bool, error) { return isClusterActive(clusterNameNodePool, t.Logs) }, waitTimeout, pollingInterval).Should(
				BeTrue(), fmt.Sprintf("cluster %s is not active", clusterNameNodePool))
			Eventually(func() error {
				return verifyCluster(clusterNameNodePool, "", expectedNodeCount, activeClusterState, transitioningFlagNo, t.Logs)
			}, waitTimeout, pollingInterval).Should(BeNil(), fmt.Sprintf("could not verify cluster %s", clusterNameNodePool))
		})
	})

	// Cluster 3. Pass in invalid parameters when creating a cluster.
	t.Context("OCNE cluster creation with single node invalid kubernetes version", Ordered, func() {
		var clusterConfig RancherOCNECluster
		t.It("create OCNE cluster", func() {
			// Create the cluster
			Eventually(func() error {
				mutateFn := func(config *RancherOCNECluster) {
					// setting an invalid kubernetes version
					config.OciocneEngineConfig.KubernetesVersion = "v1.22.7"
				}
				return createClusterAndFillConfig(clusterNameSingleNodeInvalid, &clusterConfig, t.Logs, mutateFn)
			}, shortWaitTimeout, shortPollingInterval).Should(BeNil())
			// Track this cluster's ID for deletion later
			clusterID, err := getClusterIDFromName(clusterNameSingleNodeInvalid, t.Logs)
			Expect(err).ShouldNot(HaveOccurred())
			clusterIDsToDelete = append(clusterIDsToDelete, clusterID)
			t.Logs.Infof("the cluster ID of %s is %s", clusterNameSingleNodeInvalid, clusterID)
		})

		t.It("check OCNE cluster is not active", func() {
			// Verify the cluster is not active
			waitTimeoutNegative := 20 * time.Minute
			Eventually(func() (bool, error) { return isClusterActive(clusterNameSingleNodeInvalid, t.Logs) }, waitTimeoutNegative, pollingInterval).Should(
				BeFalse(), fmt.Sprintf("cluster %s is active", clusterNameSingleNodeInvalid))

			// Verify that the cluster is configured correctly
			Eventually(func() error {
				return verifyCluster(clusterNameSingleNodeInvalid, "", 0, provisioningClusterState, transitioningFlagError, t.Logs)
			}, shortWaitTimeout, shortPollingInterval).Should(BeNil(), fmt.Sprintf("could not verify cluster %s", clusterNameSingleNodeInvalid))
		})
	})
})

type mutateRancherOCNEClusterFunc func(config *RancherOCNECluster)

// Creates an OCNE Cluster, and returns an error if not successful. Creates a single node cluster by default.
// `config` is expected to point to an empty RancherOCNECluster struct, which is populated with values by this function.
// `mutateFn`, if not nil, can be used to make additional changes to the cluster config before the cluster creation request is made.
func createClusterAndFillConfig(clusterName string, config *RancherOCNECluster, log *zap.SugaredLogger, mutateFn mutateRancherOCNEClusterFunc) error {
	nodePublicKeyContents, err := getFileContents(nodePublicKeyPath, log)
	if err != nil {
		log.Errorf("error reading node public key file: %v", err)
		return err
	}

	// Fill in the values for the create cluster API request body
	config.fillCommonValues()
	config.OciocneEngineConfig.CloudCredentialID = cloudCredentialID
	config.OciocneEngineConfig.DisplayName = clusterName
	config.OciocneEngineConfig.NodePublicKeyContents = nodePublicKeyContents
	config.OciocneEngineConfig.NodePools = []string{}
	config.Name = clusterName
	config.CloudCredentialID = cloudCredentialID

	// Make additional changes to the cluster config
	if mutateFn != nil {
		mutateFn(config)
	}

	return createCluster(clusterName, *config, log)
}

// This function takes in the cluster config of an existing cluster, and changes the fields required to make the update.
// Then, this triggers an update for the OCNE cluster.
func updateConfigAndCluster(config *RancherOCNECluster, mutateFn mutateRancherOCNEClusterFunc, log *zap.SugaredLogger) error {
	if mutateFn == nil {
		err := fmt.Errorf("cannot provide a nil mutate function to update the cluster")
		log.Error(err)
		return err
	}

	clusterName := config.Name
	mutateFn(config)
	return updateCluster(clusterName, *config, log)
}

// This returns a mutateRancherOCNEClusterFunc, which edits a cluster config to have a node pool with the specified name and number of replicas.
// Both the control plane and node pool nodes use the specified volume size, number of ocpus, and memory.
func getMutateFnNodePoolsAndResourceUsage(nodePoolName string, poolReplicas, volumeSize, ocpus, memory int) mutateRancherOCNEClusterFunc {
	return func(config *RancherOCNECluster) {
		config.OciocneEngineConfig.ControlPlaneVolumeGbs = volumeSize
		config.OciocneEngineConfig.ControlPlaneOcpus = ocpus
		config.OciocneEngineConfig.ControlPlaneMemoryGbs = memory

		config.OciocneEngineConfig.NodePools = []string{
			getNodePoolSpec(nodePoolName, nodeShape, poolReplicas, memory, ocpus, volumeSize),
		}
	}
}
