// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package cluster_sync_test

import (
	"context"
	"fmt"
	"os"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/verrazzano/verrazzano/pkg/constants"
	"github.com/verrazzano/verrazzano/pkg/log/vzlog"
	"github.com/verrazzano/verrazzano/platform-operator/apis/clusters/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/clientset/versioned"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/clusters"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg/test/framework"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	waitTimeout     = 10 * time.Minute
	pollingInterval = 10 * time.Second
)

var t = framework.NewTestFramework("cluster_sync_test")

var client *versioned.Clientset
var rc *clusters.RancherConfig

var _ = t.BeforeSuite(func() {})
var _ = t.AfterEach(func() {})

var _ = t.AfterSuite(func() {})

var _ = t.Describe("Multi Cluster Rancher Validation", Label("f:platform-lcm.install"), func() {

	// 1. Create the cluster in Rancher
	// 2. Delete the cluster in Rancher
	// Verify that the VMC was created and deleted in sync
	t.Context("When clusters are created and deleted in Rancher", func() {
		const clusterName = "cluster1"
		var clusterID string

		t.BeforeEach(func() {
			client, rc = initializeTestResources()
		})

		t.It("a VMC is automatically created", func() {
			clusterID = testRancherClusterCreation(rc, client, clusterName)
		})

		t.It("a VMC is automatically deleted", func() {
			testRancherClusterDeletion(rc, client, clusterName, clusterID)
		})
	})

	// 1. Create the VMC
	// 2. Delete the VMC
	// Verify the Rancher cluster was created and deleted in sync
	t.Context("When VMCs are created and deleted", func() {
		const clusterName = "cluster2"

		t.BeforeEach(func() {
			client, rc = initializeTestResources()
		})

		t.It("a Rancher cluster is automatically created", func() {
			testVMCCreation(rc, client, clusterName)
		})

		t.It("a Rancher cluster is automatically deleted", func() {
			testVMCDeletion(rc, client, clusterName)
		})
	})

	// 1. Create the VMC
	// 2. Delete the cluster in Rancher
	// Verify the Rancher cluster is created and the VMC is deleted
	t.Context("When VMC is created and deleted in Rancher", func() {
		const clusterName = "cluster3"
		var clusterID string

		t.BeforeEach(func() {
			client, rc = initializeTestResources()
		})

		t.It("a Rancher cluster is automatically created", func() {
			clusterID = testVMCCreation(rc, client, clusterName)
		})

		t.It("a VMC is automatically deleted", func() {
			testRancherClusterDeletion(rc, client, clusterName, clusterID)
		})
	})

	// 1. Create the cluster in Rancher
	// 2. Delete the VMC
	// Verify the VMC is created and the Rancher cluster is deleted
	t.Context("When VMC is created and deleted in Rancher", func() {
		const clusterName = "cluster4"

		t.BeforeEach(func() {
			client, rc = initializeTestResources()
		})

		t.It("a VMC is automatically created", func() {
			testRancherClusterCreation(rc, client, clusterName)
		})

		t.It("a Rancher cluster is automatically deleted", func() {
			testVMCDeletion(rc, client, clusterName)
		})
	})
})

func initializeTestResources() (*versioned.Clientset, *clusters.RancherConfig) {
	adminKubeconfig := os.Getenv("ADMIN_KUBECONFIG")
	Expect(adminKubeconfig).To(Not(BeEmpty()))

	var err error
	client, err = pkg.GetVerrazzanoClientsetInCluster(adminKubeconfig)
	Expect(err).ShouldNot(HaveOccurred())

	// Get Rancher API URL and creds
	rc, err = pkg.CreateNewRancherConfig(t.Logs, adminKubeconfig)
	Expect(err).ShouldNot(HaveOccurred())

	return client, rc
}

// testRancherClusterCreation tests a cluster created in Rancher
func testRancherClusterCreation(rc *clusters.RancherConfig, client *versioned.Clientset, clusterName string) string {
	// GIVEN a Rancher cluster is created using Rancher API/UI
	// WHEN the Rancher cluster is appropriately labeled
	// THEN a VMC is auto-created for that cluster

	// Create cluster in Rancher and label it (when labels are supported)
	var err error
	clusterID, err := clusters.ImportClusterToRancher(rc, clusterName, vzlog.DefaultLogger())
	Expect(err).ShouldNot(HaveOccurred())
	pkg.Log(pkg.Info, fmt.Sprintf("Got cluster id %s from Rancher\n", clusterID))

	// Eventually, a VMC with that cluster name should be created
	Eventually(func() (*v1alpha1.VerrazzanoManagedCluster, error) {
		pkg.Log(pkg.Info, "Waiting for VMC to be created")
		return client.ClustersV1alpha1().VerrazzanoManagedClusters(constants.VerrazzanoMultiClusterNamespace).Get(context.TODO(), clusterName, metav1.GetOptions{})
	}).WithPolling(pollingInterval).WithTimeout(waitTimeout).ShouldNot(BeNil())
	return clusterID
}

// testRancherClusterDeletion tests a cluster deleted in Rancher
func testRancherClusterDeletion(rc *clusters.RancherConfig, client *versioned.Clientset, clusterName, clusterID string) {
	// GIVEN a Rancher cluster is deleted using Rancher API/UI
	// WHEN the Rancher cluster is appropriately labeled
	// THEN the VMC for the cluster is deleted

	// The VMC should have the clusterID field set before we attempt to delete
	Eventually(func() bool {
		return verifyRancherRegistration(clusterName)
	}).WithPolling(pollingInterval).WithTimeout(waitTimeout).Should(BeTrue())

	// Delete cluster using Rancher API
	deleted, err := clusters.DeleteClusterFromRancher(rc, clusterID, vzlog.DefaultLogger())
	Expect(err).ShouldNot(HaveOccurred())
	Expect(deleted).To(BeTrue())

	// Eventually, a VMC with that cluster name should be deleted
	Eventually(func() bool {
		pkg.Log(pkg.Info, "Waiting for VMC to be deleted")
		_, err := client.ClustersV1alpha1().VerrazzanoManagedClusters(constants.VerrazzanoMultiClusterNamespace).Get(context.TODO(), clusterName, metav1.GetOptions{})
		return errors.IsNotFound(err)
	}).WithPolling(pollingInterval).WithTimeout(waitTimeout).Should(BeTrue())
}

// testVMCCreation tests a VMC created for a managed cluster
func testVMCCreation(rc *clusters.RancherConfig, client *versioned.Clientset, clusterName string) string {
	// GIVEN a VMC is created for a cluster
	// WHEN the Rancher clusters are prompted to sync with the VMC
	// THEN a Rancher cluster should be created with the same name

	// Create the VMC resource in the cluster
	Eventually(func() (*v1alpha1.VerrazzanoManagedCluster, error) {
		pkg.Log(pkg.Info, fmt.Sprintf("Attempting to create VMC %s", clusterName))
		vmc := v1alpha1.VerrazzanoManagedCluster{
			ObjectMeta: metav1.ObjectMeta{
				Name: clusterName,
			},
		}
		return client.ClustersV1alpha1().VerrazzanoManagedClusters(constants.VerrazzanoMultiClusterNamespace).Create(context.TODO(), &vmc, metav1.CreateOptions{})
	}).WithPolling(pollingInterval).WithTimeout(waitTimeout).ShouldNot(BeNil())

	// Verify the cluster is created in Rancher
	Eventually(func() bool {
		return clusterExistsInRancher(rc, clusterName)
	}).WithPolling(pollingInterval).WithTimeout(waitTimeout).Should(BeTrue())

	clusterID, err := clusters.GetClusterIDFromRancher(rc, clusterName, vzlog.DefaultLogger())
	Expect(err).ShouldNot(HaveOccurred())
	return clusterID
}

// testVMCDeletion tests a VMC deleted for a managed cluster
func testVMCDeletion(rc *clusters.RancherConfig, client *versioned.Clientset, clusterName string) {
	// GIVEN a VMC is deleted from the admin cluster
	// WHEN the Rancher sync process runs
	// THEN a Rancher cluster with that name should be deleted

	// The VMC should have the clusterID field set before we attempt to delete
	Eventually(func() bool {
		return verifyRancherRegistration(clusterName)
	}).WithPolling(pollingInterval).WithTimeout(waitTimeout).Should(BeTrue())

	// Delete the VMC resource in the cluster
	Eventually(func() error {
		pkg.Log(pkg.Info, fmt.Sprintf("Attempting to delete VMC %s", clusterName))
		return client.ClustersV1alpha1().VerrazzanoManagedClusters(constants.VerrazzanoMultiClusterNamespace).Delete(context.TODO(), clusterName, metav1.DeleteOptions{})
	}).WithPolling(pollingInterval).WithTimeout(waitTimeout).Should(BeNil())

	Eventually(func() bool {
		return clusterExistsInRancher(rc, clusterName)
	}).WithPolling(pollingInterval).WithTimeout(waitTimeout).Should(BeFalse())
}

func verifyRancherRegistration(clusterName string) bool {
	pkg.Log(pkg.Info, fmt.Sprintf("Waiting for Rancher registration to occur for VMC %s", clusterName))
	vmc, err := client.ClustersV1alpha1().VerrazzanoManagedClusters(constants.VerrazzanoMultiClusterNamespace).Get(context.TODO(), clusterName, metav1.GetOptions{})
	if err != nil {
		pkg.Log(pkg.Error, fmt.Sprintf("Failed to get VMC %s from the cluster", clusterName))
		return false
	}
	if vmc.Status.RancherRegistration.ClusterID == "" {
		pkg.Log(pkg.Info, fmt.Sprintf("Cluster ID was empty for VMC %s, waiting until it is set to delete", clusterName))
		return false
	}
	return true
}

// clusterExistsInRancher returns true if the cluster is listed by the Rancher API
func clusterExistsInRancher(rc *clusters.RancherConfig, clusterName string) bool {
	ranchClusters, _, err := clusters.GetAllClustersInRancher(rc, vzlog.DefaultLogger())
	if err != nil {
		pkg.Log(pkg.Error, fmt.Sprintf("Failed to get all clusters in Rancher: %v", err))
		return false
	}
	pkg.Log(pkg.Info, fmt.Sprintf("Looking for cluster %s in Rancher", clusterName))
	for _, cluster := range ranchClusters {
		if cluster.Name == clusterName {
			return true
		}
	}
	return false
}
