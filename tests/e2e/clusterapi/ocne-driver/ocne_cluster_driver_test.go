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
	waitTimeout          = 150 * time.Minute
	pollingInterval      = 30 * time.Second
)

var (
	t                            = framework.NewTestFramework("capi-ocne-driver")
	clusterNameSingleNode        string
	clusterNameNewName           string
	clusterNameNodePool          string
	clusterNameSingleNodeInvalid string
	addedPoolName                string
	poolName                     string

	// Keep track of which clusters to clean up at the end by ID, since cluster names may change.
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
	clusterNameNewName = fmt.Sprintf("strudel-updated-name-%s", ocneClusterNameSuffix)
	clusterNameNodePool = fmt.Sprintf("strudel-pool-%s", ocneClusterNameSuffix)
	clusterNameSingleNodeInvalid = fmt.Sprintf("strudel-single-invalid-k8s-%s", ocneClusterNameSuffix)

	addedPoolName = fmt.Sprintf("added-pool-%s", ocneClusterNameSuffix)
	poolName = fmt.Sprintf("pool-%s", ocneClusterNameSuffix)
}

var _ = t.SynchronizedBeforeSuite(synchronizedBeforeSuiteProcess1Func, synchronizedBeforeSuiteAllProcessesFunc)

// Part of SynchronizedAfterSuite, run by all processes
func synchronizedAfterSuiteAllProcessesFunc() {
	// Delete the clusters concurrently
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
		// clusterConfig specifies the parameters passed into the cluster creation
		// and is updated as update requests are made
		var clusterConfig RancherOCNECluster
		expectedNodeCount := 1

		t.It("create OCNE cluster", func() {
			// Create the cluster
			Eventually(func() error {
				return createSingleNodeCluster(clusterNameSingleNode, &clusterConfig, t.Logs, nil)
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
				return verifyCluster(clusterNameSingleNode, "", expectedNodeCount, activeClusterState, transitioningFlagNo, t.Logs)
			}, shortWaitTimeout, shortPollingInterval).Should(BeNil(), fmt.Sprintf("could not verify cluster %s", clusterNameSingleNode))
		})

		// Update - scale up number of nodes, update node shapes, edit name and description
		description := "A new description for the OCNE cluster."
		t.It("update the OCNE cluster 1", func() { // FIXME: better description
			Eventually(func() error {
				return updateNameShapeAndScaleUp(clusterNameNewName, description, addedPoolName, &clusterConfig, t.Logs)
			}, shortWaitTimeout, shortPollingInterval).Should(BeNil())
			// This update added 2 control plane node, 1 worker node
			expectedNodeCount += 3
		})
		t.It("check the OCNE cluster updated successfully", func() {
			Eventually(func() (bool, error) { return isClusterActive(clusterNameNewName, t.Logs) }, waitTimeout, pollingInterval).Should(
				BeTrue(), fmt.Sprintf("cluster %s is not active", clusterNameNewName))
			Eventually(func() error {
				return verifyCluster(clusterNameSingleNode, description, expectedNodeCount, activeClusterState, transitioningFlagNo, t.Logs)
			}, waitTimeout, pollingInterval).Should(BeNil(), fmt.Sprintf("could not verify cluster %s", clusterNameNewName))
		})

		// Update - increase resource usage
		poolReplicas := 2                          // FIXME: make this useful
		t.It("update the OCNE cluster 2", func() { // FIXME: better description
			Eventually(func() error {
				return updateResourceIncrease(addedPoolName, poolReplicas, &clusterConfig, t.Logs)
			}, shortWaitTimeout, shortPollingInterval).Should(BeNil())
			// This update added 1 worker node
			expectedNodeCount++
		})
		t.It("check the OCNE cluster updated successfully", func() {
			Eventually(func() (bool, error) { return isClusterActive(clusterNameNewName, t.Logs) }, waitTimeout, pollingInterval).Should(
				BeTrue(), fmt.Sprintf("cluster %s is not active", clusterNameNewName))
			Eventually(func() error {
				return verifyCluster(clusterNameSingleNode, description, expectedNodeCount, activeClusterState, transitioningFlagNo, t.Logs)
			}, waitTimeout, pollingInterval).Should(BeNil(), fmt.Sprintf("could not verify cluster %s", clusterNameNewName))
		})
	})

	// Cluster 2. Create with a node pool, then perform some updates.
	t.Context("OCNE cluster creation with node pools", Ordered, func() {
		var clusterConfig RancherOCNECluster
		// expected number of nodes is number of worker nodes + 3 control plane node
		poolReplicas := 2
		expectedNodeCount := poolReplicas + 3

		// Create the cluster and verify it comes up
		t.It("create OCNE cluster", func() {
			Eventually(func() error {
				return createNodePoolCluster(clusterNameNodePool, poolName, poolReplicas, &clusterConfig, t.Logs)
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
		t.It("update the OCNE cluster 3", func() {
			poolReplicas = 1
			Eventually(func() error {
				return updateResourceDecrease(poolName, poolReplicas, &clusterConfig, t.Logs)
			}, shortWaitTimeout, shortPollingInterval).Should(BeNil())
			// expect 3 control plane nodes, 1 worker nodes
			expectedNodeCount = 4
		})
		t.It("check the OCNE cluster updated successfully", func() {
			Eventually(func() (bool, error) { return isClusterActive(clusterNameNodePool, t.Logs) }, waitTimeout, pollingInterval).Should(
				BeTrue(), fmt.Sprintf("cluster %s is not active", clusterNameNodePool))
			Eventually(func() error {
				return verifyCluster(clusterNameNodePool, "", expectedNodeCount, activeClusterState, transitioningFlagNo, t.Logs)
			}, waitTimeout, pollingInterval).Should(BeNil(), fmt.Sprintf("could not verify cluster %s", clusterNameNodePool))
		})

		// Update - scale down to single node
		t.It("update the OCNE cluster 4", func() {
			poolReplicas = 1
			Eventually(func() error {
				return updateScaleDown(&clusterConfig, t.Logs)
			}, shortWaitTimeout, shortPollingInterval).Should(BeNil())
			// expect only 1 control plane node
			expectedNodeCount = 1
		})
		t.It("check the OCNE cluster updated successfully", func() {
			Eventually(func() (bool, error) { return isClusterActive(clusterNameNodePool, t.Logs) }, waitTimeout, pollingInterval).Should(
				BeTrue(), fmt.Sprintf("cluster %s is not active", clusterNameNodePool))
			Eventually(func() error {
				return verifyCluster(clusterNameNodePool, "", expectedNodeCount, activeClusterState, transitioningFlagNo, t.Logs)
			}, waitTimeout, pollingInterval).Should(BeNil(), fmt.Sprintf("could not verify cluster %s", clusterNameNodePool))
		})
	})

	// Cluster 3. Pass in invalid parameters to a cluster creation.
	t.Context("OCNE cluster creation with single node invalid kubernetes version", Ordered, func() {
		var clusterConfig RancherOCNECluster
		t.It("create OCNE cluster", func() {
			// Create the cluster
			Eventually(func() error {
				mutateFn := func(config *RancherOCNECluster) {
					// setting an invalid kubernetes version
					config.OciocneEngineConfig.KubernetesVersion = "v1.22.7"
				}
				return createSingleNodeCluster(clusterNameSingleNodeInvalid, &clusterConfig, t.Logs, mutateFn)
			}, shortWaitTimeout, shortPollingInterval).Should(BeNil())
			// Track this cluster's ID for deletion later
			clusterID, err := getClusterIDFromName(clusterNameSingleNodeInvalid, t.Logs)
			Expect(err).ShouldNot(HaveOccurred())
			clusterIDsToDelete = append(clusterIDsToDelete, clusterID)
			t.Logs.Infof("the cluster ID of %s is %s", clusterNameSingleNodeInvalid, clusterID)
		})

		t.It("check OCNE cluster is not active", func() {
			// Verify the cluster is not active
			Eventually(func() (bool, error) { return isClusterActive(clusterNameSingleNodeInvalid, t.Logs) }, waitTimeout, pollingInterval).Should(
				BeFalse(), fmt.Sprintf("cluster %s is active", clusterNameSingleNodeInvalid))

			// Verify that the cluster is configured correctly
			Eventually(func() error {
				return verifyCluster(clusterNameSingleNodeInvalid, "", 0, provisioningClusterState, transitioningFlagError, t.Logs)
			}, shortWaitTimeout, shortPollingInterval).Should(BeNil(), fmt.Sprintf("could not verify cluster %s", clusterNameSingleNodeInvalid))
		})
	})
})

/*
This function takes in the cluster config of an existing cluster, and changes the fields required to make the update.
Then, this triggers an update for the OCNE cluster:

	Gives the cluster a new name `newName`.
	Adds a description `newDescription`.
	Changes the node shapes to VM.Standard.E3.Flex for both control plane and worker nodes.
	Changes the node image.
	Sets 1 node pool with 1 worker node.
	Adds 2 control plane nodes.
*/
func updateNameShapeAndScaleUp(newName, newDescription, nodePoolName string, config *RancherOCNECluster, log *zap.SugaredLogger) error {
	oldClusterName := config.Name

	// FIXME: lots of "magic" values
	newShape := "VM.Standard.E3.Flex"
	nodePools := []string{getNodePoolSpec(nodePoolName, newShape, 1, 16, 2, 100)}

	// Change values for the update cluster API request's body
	config.OciocneEngineConfig.DisplayName = newName
	config.OciocneEngineConfig.NodePools = nodePools
	config.Name = newName
	config.Description = newDescription
	// FIXME: magic
	config.OciocneEngineConfig.ImageDisplayName = "Oracle-Linux-8.7-2023.05.24-0"
	config.OciocneEngineConfig.NodeShape = newShape
	config.OciocneEngineConfig.ControlPlaneShape = newShape
	config.OciocneEngineConfig.NumControlPlaneNodes += 2

	return updateCluster(oldClusterName, *config, log)
}

/*
This function takes in the cluster config of an existing cluster, and changes the fields required to make the update.
Then, this triggers an update for the OCNE cluster:

	Edit the existing node pool to 2 replicas.
	Change number of OCPUs from 2 on control plane and worker nodes.
	Change memory to 32 Gbs for all nodes.
	Change boot volume size to 150 Gbs for all nodes.
*/
func updateResourceIncrease(nodePoolName string, poolReplicas int, config *RancherOCNECluster, log *zap.SugaredLogger) error {
	clusterName := config.Name

	// FIXME: lots of "magic" values
	newMemory := 32
	numOCPUs := 3
	volumeSize := 150
	// FIXME: edit node pools more intelligently. Can unmarshal the string into JSON.
	nodePools := []string{getNodePoolSpec(nodePoolName, nodeShape, poolReplicas, newMemory, numOCPUs, volumeSize)}

	// Change the values for the update cluster API request's body
	// FIXME: magic
	config.OciocneEngineConfig.ControlPlaneOcpus = numOCPUs
	config.OciocneEngineConfig.ControlPlaneMemoryGbs = newMemory
	config.OciocneEngineConfig.ControlPlaneVolumeGbs = volumeSize
	config.OciocneEngineConfig.NodePools = nodePools

	return updateCluster(clusterName, *config, log)
}

/*
This function takes in the cluster config of an existing cluster, and changes the fields required to make the update.
Then, this triggers an update for the OCNE cluster:

	Edit the existing node pool to 1 replica.
	Change number of OCPUs to 1 on control plane and worker nodes.
	Change memory to 16 Gbs for all nodes.
	Change boot volume size to 100 Gbs for all nodes.
*/
func updateResourceDecrease(nodePoolName string, poolReplicas int, config *RancherOCNECluster, log *zap.SugaredLogger) error {
	clusterName := config.Name

	// FIXME: lots of "magic" values
	newMemory := 16
	numOCPUs := 1
	volumeSize := 100
	// FIXME: edit node pools more intelligently. Can unmarshal the string into JSON.
	nodePools := []string{getNodePoolSpec(nodePoolName, nodeShape, poolReplicas, newMemory, numOCPUs, volumeSize)}

	// Change the values for the update cluster API request's body
	// FIXME: magic
	config.OciocneEngineConfig.ControlPlaneOcpus = numOCPUs
	config.OciocneEngineConfig.ControlPlaneMemoryGbs = newMemory
	config.OciocneEngineConfig.ControlPlaneVolumeGbs = volumeSize
	config.OciocneEngineConfig.NodePools = nodePools

	return updateCluster(clusterName, *config, log)
}

/*
This function takes in the cluster config of an existing cluster, and changes the fields required to make the update.
Then, this triggers an update for the OCNE cluster:

	Removes all node pools.
	Change number of control plane nodes to 1.
*/
func updateScaleDown(config *RancherOCNECluster, log *zap.SugaredLogger) error {
	clusterName := config.Name
	nodePools := []string{}

	// Change the values for the update cluster API request's body
	config.OciocneEngineConfig.NodePools = nodePools
	config.OciocneEngineConfig.NumControlPlaneNodes = 1

	return updateCluster(clusterName, *config, log)
}

type mutateRancherOCNEClusterFunc func(config *RancherOCNECluster)

// Creates a single node OCNE Cluster through CAPI, and returns an error if not successful.
// `config` is expected to point to an empty RancherOCNECluster, which is populated with values by this function.
func createSingleNodeCluster(clusterName string, config *RancherOCNECluster, log *zap.SugaredLogger, mutateFn mutateRancherOCNEClusterFunc) error {
	nodePublicKeyContents, err := getFileContents(nodePublicKeyPath, log)
	if err != nil {
		log.Errorf("error reading node public key file: %v", err)
		return err
	}

	nodePools := []string{}

	// Fill in the values for the create cluster API request body
	config.fillCommonValues()
	config.OciocneEngineConfig.CloudCredentialID = cloudCredentialID
	config.OciocneEngineConfig.DisplayName = clusterName
	config.OciocneEngineConfig.NodePublicKeyContents = nodePublicKeyContents
	config.OciocneEngineConfig.NodePools = nodePools
	config.Name = clusterName
	config.CloudCredentialID = cloudCredentialID

	if mutateFn != nil {
		mutateFn(config)
	}

	return createCluster(clusterName, *config, log)
}

// Creates a OCNE Cluster with node pools through CAPI
// `config` is expected to point to an empty RancherOCNECluster, which is populated with values by this function.
func createNodePoolCluster(clusterName, nodePoolName string, poolReplicas int, config *RancherOCNECluster, log *zap.SugaredLogger) error {
	nodePublicKeyContents, err := getFileContents(nodePublicKeyPath, log)
	if err != nil {
		log.Errorf("error reading node public key file: %v", err)
		return err
	}

	// FIXME: magic
	volumeSize := 150
	ocpus := 2
	memory := 32
	nodePools := []string{getNodePoolSpec(nodePoolName, nodeShape, poolReplicas, memory, ocpus, volumeSize)}

	// Fill in the values for the create cluster API request body
	config.fillCommonValues()
	config.OciocneEngineConfig.CloudCredentialID = cloudCredentialID
	config.OciocneEngineConfig.DisplayName = clusterName
	config.OciocneEngineConfig.NodePublicKeyContents = nodePublicKeyContents
	config.OciocneEngineConfig.NodePools = nodePools
	config.Name = clusterName
	config.CloudCredentialID = cloudCredentialID
	// FIXME: magic
	config.OciocneEngineConfig.NumControlPlaneNodes = 3
	config.OciocneEngineConfig.ControlPlaneMemoryGbs = memory
	config.OciocneEngineConfig.ControlPlaneOcpus = ocpus
	config.OciocneEngineConfig.ControlPlaneVolumeGbs = volumeSize

	return createCluster(clusterName, *config, log)
}
