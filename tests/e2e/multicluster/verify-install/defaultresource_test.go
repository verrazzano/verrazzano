// Copyright (c) 2021, 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package defaultresource_test

import (
	"encoding/json"
	"fmt"
	"github.com/verrazzano/verrazzano/pkg/test/framework"
	"os"
	"regexp"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/verrazzano/verrazzano/pkg/k8sutil"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	waitTimeout           = 5 * time.Minute
	pollingInterval       = 10 * time.Second
	cattleSystemNamespace = "cattle-system"
	searchTimeWindow      = "1h"
)

var expectedPodsKubeSystem = []string{
	"coredns",
	"kube-proxy"}

var t = framework.NewTestFramework("defaultresource_test")

var _ = t.BeforeSuite(func() {})
var _ = t.AfterEach(func() {})

var _ = t.AfterSuite(func() {
	Eventually(func() error {
		return listPodsInKubeSystem()
	}, waitTimeout, pollingInterval).Should(BeNil())
})

var _ = t.Describe("Multi Cluster Install Validation", Label("f:platform-lcm.install"),
	func() {
		t.It("has the expected namespaces", func() {
			kubeConfig := os.Getenv("KUBECONFIG")
			pkg.Log(pkg.Info, fmt.Sprintf("Kube config: %s", kubeConfig))
			namespaces, err := pkg.ListNamespaces(metav1.ListOptions{})
			Expect(err).ShouldNot(HaveOccurred())
			Expect(nsListContains(namespaces.Items, "default")).To(Equal(true))
			Expect(nsListContains(namespaces.Items, "kube-public")).To(Equal(true))
			Expect(nsListContains(namespaces.Items, "kube-system")).To(Equal(true))
			Expect(nsListContains(namespaces.Items, "kube-node-lease")).To(Equal(true))
		})

		t.Context("Expected pods are running.", func() {
			t.It("and waiting for expected pods must be running", func() {
				Eventually(func() bool {
					result, err := pkg.PodsRunning("kube-system", expectedPodsKubeSystem)
					if err != nil {
						AbortSuite(fmt.Sprintf("One or more pods are not running in the namespace: kube-system, error: %v", err))
					}
					return result
				}, waitTimeout, pollingInterval).Should(BeTrue())
			})
		})

		t.It("Rancher log records do not contain any websocket bad handshake messages", func() {
			// GIVEN existing system logs
			// WHEN the Elasticsearch index for the cattle-system namespace is retrieved
			// THEN is has a limited number of bad socket messages
			indexName, err := pkg.GetOpenSearchSystemIndex(cattleSystemNamespace)
			Expect(err).To(BeNil())
			Eventually(func() bool {
				return pkg.LogIndexFound(indexName)
			}, waitTimeout, pollingInterval).Should(BeTrue(), "Expected to find Elasticsearch index cattle-system")

			Expect(getNumBadSocketMessages()).To(BeNumerically("<", 20))

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

func nsListContains(list []v1.Namespace, target string) bool {
	for i := range list {
		if list[i].Name == target {
			return true
		}
	}
	return false
}

func listPodsInKubeSystem() error {
	// Get the Kubernetes clientset and list pods in cluster
	clientset, err := k8sutil.GetKubernetesClientset()
	if err != nil {
		pkg.Log(pkg.Error, fmt.Sprintf("Error getting Kubernetes clientset: %v", err))
		return err
	}
	pods, err := pkg.ListPodsInCluster("kube-system", clientset)
	if err != nil {
		pkg.Log(pkg.Error, fmt.Sprintf("Error listing pods: %v", err))
		return err
	}
	for _, podInfo := range (*pods).Items {
		pkg.Log(pkg.Info, fmt.Sprintf("pods-name=%v\n", podInfo.Name))
		pkg.Log(pkg.Info, fmt.Sprintf("pods-status=%v\n", podInfo.Status.Phase))
		pkg.Log(pkg.Info, fmt.Sprintf("pods-condition=%v\n", podInfo.Status.Conditions))
	}
	return nil
}
