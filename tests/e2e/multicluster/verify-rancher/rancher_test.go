// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package rancher_test

import (
	"encoding/json"
	"fmt"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg/test/framework"
	"regexp"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg"
)

const (
	waitTimeout           = 5 * time.Minute
	pollingInterval       = 10 * time.Second
	cattleSystemNamespace = "cattle-system"
	searchTimeWindow      = "1h"
)

var t = framework.NewTestFramework("rancher_test")

var _ = t.BeforeSuite(func() {})
var _ = t.AfterEach(func() {})

var _ = t.AfterSuite(func() {})

var _ = t.Describe("Multi Cluster Rancher Validation", Label("f:platform-lcm.install"),
	func() {

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
