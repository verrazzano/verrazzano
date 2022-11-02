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
	waitTimeout     = 5 * time.Minute
	pollingInterval = 10 * time.Second
)

var t = framework.NewTestFramework("cluster_sync_test")

var _ = t.BeforeSuite(func() {})
var _ = t.AfterEach(func() {})

var _ = t.AfterSuite(func() {})

var _ = t.Describe("Multi Cluster Rancher Validation", Label("f:platform-lcm.install"), func() {
	t.Context("When clusters are created and deleted in Rancher", func() {
		const rancherClusterName = "cluster1"

		var client *versioned.Clientset
		var rc *clusters.RancherConfig
		var clusterID string

		BeforeEach(func() {
			adminKubeconfig := os.Getenv("ADMIN_KUBECONFIG")
			Expect(adminKubeconfig).To(Not(BeEmpty()))

			var err error
			client, err = pkg.GetVerrazzanoClientsetInCluster(adminKubeconfig)
			Expect(err).ShouldNot(HaveOccurred())

			// Get Rancher API URL and creds
			rc, err = pkg.CreateNewRancherConfig(t.Logs, adminKubeconfig)
			Expect(err).ShouldNot(HaveOccurred())
		})

		t.It("a VMC is automatically created", func() {
			// GIVEN a Rancher cluster is created using Rancher API/UI
			// WHEN the Rancher cluster is appropriately labeled
			// THEN a VMC is auto-created for that cluster

			// Create cluster in Rancher and label it (when labels are supported)
			var err error
			clusterID, err = clusters.ImportClusterToRancher(rc, rancherClusterName, vzlog.DefaultLogger())
			Expect(err).ShouldNot(HaveOccurred())
			pkg.Log(pkg.Info, fmt.Sprintf("Got cluster id %s from Rancher\n", clusterID))

			// Eventually, a VMC with that cluster name should be created
			Eventually(func() (*v1alpha1.VerrazzanoManagedCluster, error) {
				pkg.Log(pkg.Info, "Waiting for VMC to be created")
				return client.ClustersV1alpha1().VerrazzanoManagedClusters(constants.VerrazzanoMultiClusterNamespace).Get(context.TODO(), rancherClusterName, metav1.GetOptions{})
			}).WithPolling(pollingInterval).WithTimeout(waitTimeout).ShouldNot(BeNil())
		})
		t.It("a VMC is automatically deleted", func() {
			// GIVEN a Rancher cluster is deleted using Rancher API/UI
			// WHEN the Rancher cluster is appropriately labeled
			// THEN the VMC for the cluster is deleted

			// Delete cluster using Rancher API
			deleted, err := clusters.DeleteClusterFromRancher(rc, clusterID, vzlog.DefaultLogger())
			Expect(err).ShouldNot(HaveOccurred())
			Expect(deleted).To(BeTrue())

			// Eventually, a VMC with that cluster name should be deleted
			Eventually(func() bool {
				pkg.Log(pkg.Info, "Waiting for VMC to be deleted")
				_, err := client.ClustersV1alpha1().VerrazzanoManagedClusters(constants.VerrazzanoMultiClusterNamespace).Get(context.TODO(), rancherClusterName, metav1.GetOptions{})
				return errors.IsNotFound(err)
			}).WithPolling(pollingInterval).WithTimeout(waitTimeout).Should(BeTrue())
		})
	})

	t.Context("When VMCs are created and deleted", func() {
		const vmcClusterName = "cluster2"

		var client *versioned.Clientset
		var rc *clusters.RancherConfig

		BeforeEach(func() {
			adminKubeconfig := os.Getenv("ADMIN_KUBECONFIG")
			Expect(adminKubeconfig).To(Not(BeEmpty()))

			var err error
			client, err = pkg.GetVerrazzanoClientsetInCluster(adminKubeconfig)
			Expect(err).ShouldNot(HaveOccurred())

			// Get Rancher API URL and creds
			rc, err = pkg.CreateNewRancherConfig(t.Logs, adminKubeconfig)
			Expect(err).ShouldNot(HaveOccurred())
		})

		t.It("a Rancher cluster is automatically created", func() {
			// GIVEN a VMC is created for a cluster
			// WHEN the Rancher clusters are prompted to sync with the VMC
			// THEN a Rancher cluster should be created with the same name

			// Create the VMC resource in the cluster
			Eventually(func() (*v1alpha1.VerrazzanoManagedCluster, error) {
				pkg.Log(pkg.Info, fmt.Sprintf("Attempting to create VMC %s", vmcClusterName))
				vmc := v1alpha1.VerrazzanoManagedCluster{
					ObjectMeta: metav1.ObjectMeta{
						Name: vmcClusterName,
					},
				}
				return client.ClustersV1alpha1().VerrazzanoManagedClusters(constants.VerrazzanoMultiClusterNamespace).Create(context.TODO(), &vmc, metav1.CreateOptions{})
			}).WithPolling(pollingInterval).WithTimeout(waitTimeout).ShouldNot(BeNil())

			// Verify the cluster is created in Rancher
			Eventually(func() bool {
				return clusterExistsInRancher(rc, vmcClusterName)
			}).WithPolling(pollingInterval).WithTimeout(waitTimeout).Should(BeTrue())
		})

		// Delete the VMC resource and check if it is deleted in Rancher
		t.It("a Rancher cluster is automatically deleted", func() {

			// Delete the VMC resource in the cluster
			Eventually(func() error {
				pkg.Log(pkg.Info, fmt.Sprintf("Attempting to delete VMC %s", vmcClusterName))
				return client.ClustersV1alpha1().VerrazzanoManagedClusters(constants.VerrazzanoMultiClusterNamespace).Delete(context.TODO(), vmcClusterName, metav1.DeleteOptions{})
			}).WithPolling(pollingInterval).WithTimeout(waitTimeout).Should(BeNil())

			Eventually(func() bool {
				return clusterExistsInRancher(rc, vmcClusterName)
			}).WithPolling(pollingInterval).WithTimeout(waitTimeout).Should(BeFalse())
		})
	})
})

// clusterExistsInRancher returns true if the cluster is listed by the Rancher API
func clusterExistsInRancher(rc *clusters.RancherConfig, clusterName string) bool {
	ranchClusters, _, err := clusters.GetAllClustersInRancher(rc, vzlog.DefaultLogger())
	if err != nil {
		pkg.Log(pkg.Error, fmt.Sprintf("Failed to get all clusters in Rancher: %v", err))
		return false
	}
	pkg.Log(pkg.Info, "Waiting for the cluster to exist in Rancher")
	for _, cluster := range ranchClusters {
		if cluster.Name == clusterName {
			return true
		}
	}
	return false
}
