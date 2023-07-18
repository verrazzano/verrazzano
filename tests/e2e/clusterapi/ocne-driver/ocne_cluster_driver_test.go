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
	shortWaitTimeout     = 10 * time.Minute
	shortPollingInterval = 10 * time.Second
	waitTimeout          = 45 * time.Minute
	pollingInterval      = 30 * time.Second
)

var (
	t                     = framework.NewTestFramework("capi-ocne-driver")
	clusterNameSingleNode string
	clusterNameNodePool   string
)

// Part of SynchronizedBeforeSuite, run by only one process
func sbsProcess1Func() []byte {
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
func sbsAllProcessesFunc(credentialIDBytes []byte) {
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

	err = ensureOCNEDriverVarsInitialized(t.Logs)
	Expect(err).ShouldNot(HaveOccurred())

	clusterNameSingleNode = fmt.Sprintf("strudel-single-%s", ocneClusterNameSuffix)
	clusterNameNodePool = fmt.Sprintf("strudel-pool-%s", ocneClusterNameSuffix)
}

var _ = t.SynchronizedBeforeSuite(sbsProcess1Func, sbsAllProcessesFunc)

// Part of SynchronizedAfterSuite, run by only one process
func sasProcess1Func() {
	// Delete the clusters concurrently
	clusterNames := [...]string{clusterNameSingleNode, clusterNameNodePool}
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

var _ = t.SynchronizedAfterSuite(func() {}, sasProcess1Func)

var _ = t.Describe("OCNE Cluster Driver", Label("f:rancher-capi:ocne-cluster-driver"), func() {
	t.Context("OCNE cluster creation with single node", Ordered, func() {
		t.It("create OCNE cluster", func() {
			// Create the cluster
			Eventually(func() error {
				return createSingleNodeCluster(clusterNameSingleNode, t.Logs)
			}, shortWaitTimeout, shortPollingInterval).Should(BeNil())
		})

		t.It("check OCNE cluster is active", func() {
			// Verify the cluster is active
			Eventually(func() (bool, error) { return isClusterActive(clusterNameSingleNode, t.Logs) }, waitTimeout, pollingInterval).Should(
				BeTrue(), fmt.Sprintf("cluster %s is not active", clusterNameSingleNode))

			// Verify that the cluster is configured correctly
			Eventually(func() error {
				return verifyCluster(clusterNameSingleNode, 1, t.Logs)
			}, shortWaitTimeout, shortPollingInterval).Should(BeNil(), fmt.Sprintf("could not verify cluster %s", clusterNameSingleNode))
		})
	})

	t.Context("OCNE cluster creation with node pools", Ordered, func() {
		t.It("create OCNE cluster", func() {
			nodePoolName := fmt.Sprintf("pool-%s", ocneClusterNameSuffix)
			// Create the cluster
			Eventually(func() error {
				return createNodePoolCluster(clusterNameNodePool, nodePoolName, t.Logs)
			}, shortWaitTimeout, shortPollingInterval).Should(BeNil())
		})

		t.It("check OCNE cluster is active", func() {
			// Verify the cluster is active
			Eventually(func() (bool, error) { return isClusterActive(clusterNameNodePool, t.Logs) }, waitTimeout, pollingInterval).Should(
				BeTrue(), fmt.Sprintf("cluster %s is not active", clusterNameNodePool))

			// Verify that the cluster is configured correctly
			Eventually(func() error {
				return verifyCluster(clusterNameNodePool, 2, t.Logs)
			}, shortWaitTimeout, shortPollingInterval).Should(BeNil(), fmt.Sprintf("could not verify cluster %s", clusterNameNodePool))
		})
	})
})
