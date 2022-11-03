// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package rancher_test

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"regexp"
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
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

const (
	waitTimeout           = 5 * time.Minute
	pollingInterval       = 10 * time.Second
	cattleSystemNamespace = "cattle-system"
	searchTimeWindow      = "1h"
)

const (
	agentSecName = "verrazzano-cluster-agent"
	regSecName   = "verrazzano-cluster-registration"
)

var t = framework.NewTestFramework("rancher_test")

var _ = t.BeforeSuite(func() {})
var _ = t.AfterEach(func() {})

var _ = t.AfterSuite(func() {})

var _ = t.Describe("Multi Cluster Rancher Validation", Label("f:platform-lcm.install"), func() {
	t.It("Rancher log records do not contain any websocket bad handshake messages", func() {
		// GIVEN existing system logs
		// WHEN the Elasticsearch index for the cattle-system namespace is retrieved
		// THEN is has a limited number of bad socket messages
		adminKubeconfig := os.Getenv("ADMIN_KUBECONFIG")
		indexName, err := pkg.GetOpenSearchSystemIndexWithKC(cattleSystemNamespace, adminKubeconfig)
		Expect(err).ShouldNot(HaveOccurred())
		Eventually(func() bool {
			return pkg.LogIndexFound(indexName)
		}, waitTimeout, pollingInterval).Should(BeTrue(), "Expected to find Elasticsearch index cattle-system")

		Expect(getNumBadSocketMessages()).To(BeNumerically("<", 25))
	})

	t.Context("When the VMC is updated to the status of the managed cluster", func() {
		var adminClient *versioned.Clientset
		var managedClient *kubernetes.Clientset
		BeforeEach(func() {
			adminKubeconfig := os.Getenv("ADMIN_KUBECONFIG")
			Expect(adminKubeconfig).To(Not(BeEmpty()))
			managedKubeconfig := os.Getenv("MANAGED_KUBECONFIG")
			Expect(managedKubeconfig).To(Not(BeEmpty()))

			var err error

			adminClient, err = pkg.GetVerrazzanoClientsetInCluster(adminKubeconfig)
			Expect(err).ShouldNot(HaveOccurred())
			managedClient, err = pkg.GetKubernetesClientsetForCluster(managedKubeconfig)
			Expect(err).ShouldNot(HaveOccurred())
		})

		t.It("the VMC status is updated that objects have been pushed to the managed cluster", func() {
			// GIVEN the VMC has been registered
			// WHEN the VMC is retrieved
			// THEN the VMC should have a status condition of Type: Manifest Pushed and Status: True
			Eventually(func() error {
				pkg.Log(pkg.Info, "Waiting for all VMC to have status condition ManifestPushed = True")
				vmcList, err := adminClient.ClustersV1alpha1().VerrazzanoManagedClusters(constants.VerrazzanoMultiClusterNamespace).List(context.TODO(), metav1.ListOptions{})
				if err != nil {
					return err
				}

				for _, vmc := range vmcList.Items {
					statusPushedFound := false
					for _, condition := range vmc.Status.Conditions {
						if condition.Type == v1alpha1.ConditionManifestPushed && condition.Status == corev1.ConditionFalse {
							return fmt.Errorf("failed to find successful condition for ManifestPushed, VMC %s/%s has condition: %s = %s",
								vmc.Name, vmc.Namespace, condition.Type, condition.Status)
						}
						if condition.Type == v1alpha1.ConditionManifestPushed && condition.Status == corev1.ConditionTrue {
							statusPushedFound = true
						}
					}
					if !statusPushedFound {
						return fmt.Errorf("failed to find expected condition, VMC %s/%s had no condition of type %s",
							vmc.Name, vmc.Namespace, v1alpha1.ConditionManifestPushed)
					}
				}
				return nil
			}).WithPolling(pollingInterval).WithTimeout(waitTimeout).Should(BeNil())
		})

		t.It("the managed cluster should contain the pushed secrets", func() {
			// GIVEN the VMC has a status of ManifestPushed = True
			// WHEN we search for secrets on a managed cluster
			// THEN we should see that the agent and registration secrets exist in the verrazzano-system namespace
			Eventually(func() error {
				adminSec, err := managedClient.CoreV1().Secrets(constants.VerrazzanoSystemNamespace).Get(context.TODO(), agentSecName, metav1.GetOptions{})
				if err != nil {
					return err
				}
				if adminSec == nil {
					return fmt.Errorf("get admin secret %s returned nil on the managed cluster", agentSecName)
				}

				managedSec, err := managedClient.CoreV1().Secrets(constants.VerrazzanoSystemNamespace).Get(context.TODO(), regSecName, metav1.GetOptions{})
				if err != nil {
					return err
				}
				if managedSec == nil {
					return fmt.Errorf("get registration secret %s returned nil on the managed cluster", regSecName)
				}
				return nil
			}).WithPolling(pollingInterval).WithTimeout(waitTimeout).Should(BeNil())
		})
	})

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
})

func getNumBadSocketMessages() int {
	badSocket := regexp.MustCompile(`websocket: bad handshake`)
	numMessages := 0
	index, err := pkg.GetOpenSearchSystemIndex(cattleSystemNamespace)
	if err != nil {
		pkg.Log(pkg.Error, fmt.Sprintf("Failed to get OpenSearch index: %v", err))
		return -1
	}

	template :=
		`{
			"size": 1000,
			"sort": [{"@timestamp": {"order": "desc"}}],
			"query": {
				"bool": {
					"filter" : [
						{"match_phrase": {"%s": "%s"}},
						{"range": {"@timestamp": {"gte": "now-%s"}}}
					]
				}
			}
		}`
	query := fmt.Sprintf(template, "kubernetes.labels.app.keyword", "rancher", searchTimeWindow)
	resp, err := pkg.PostElasticsearch(fmt.Sprintf("%s/_search", index), query)
	if err != nil {
		pkg.Log(pkg.Error, fmt.Sprintf("Failed to query Elasticsearch: %v", err))
		return -1
	}
	if resp.StatusCode != 200 {
		pkg.Log(pkg.Error, fmt.Sprintf("Failed to query Elasticsearch: status=%d: body=%s", resp.StatusCode, string(resp.Body)))
		return -1
	}
	var result map[string]interface{}
	json.Unmarshal(resp.Body, &result)

	hits := pkg.Jq(result, "hits", "hits")
	if hits == nil {
		pkg.Log(pkg.Info, "Expected to find hits in log record query results")
		return -1
	}
	pkg.Log(pkg.Info, fmt.Sprintf("Found %d records", len(hits.([]interface{}))))
	if len(hits.([]interface{})) == 0 {
		pkg.Log(pkg.Info, "Expected log record query results to contain at least one hit")
		return -1
	}
	for _, h := range hits.([]interface{}) {
		hit := h.(map[string]interface{})
		src := hit["_source"].(map[string]interface{})
		log := src["log"].(string)
		if badSocket.MatchString(log) {
			numMessages++
		}
	}

	pkg.Log(pkg.Info, fmt.Sprintf("Found %d bad socket messages over the last hour", numMessages))
	return numMessages
}
