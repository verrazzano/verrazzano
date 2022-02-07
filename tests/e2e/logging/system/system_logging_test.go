// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package system

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/verrazzano/verrazzano/pkg/test/framework"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg"
)

const (
	shortPollingInterval = 10 * time.Second
	shortWaitTimeout     = 5 * time.Minute
)

var t = framework.NewTestFramework("system-logging")

var _ = t.BeforeSuite(func() {})

var failed = false
var _ = t.AfterEach(func() {
	failed = failed || CurrentSpecReport().Failed()
})

var _ = t.AfterSuite(func() {
	if failed {
		pkg.ExecuteClusterDumpWithEnvVarConfig()
	}
})

var _ = t.Describe("System logging test", Label(), func() {
	t.Context("for logging", Label("f:observability.logging.es"), func() {
		pkg.Concurrently(
			func() {
				// GIVEN existing system logs
				// WHEN the Elasticsearch index for the verrazzano-system namespace is retrieved
				// THEN verify that it is found
				t.It("Verify Elasticsearch index for logging exists", func() {
					Eventually(func() bool {
						return pkg.LogIndexFound("verrazzano-namespace-verrazzano-system")
					}, shortWaitTimeout, shortPollingInterval).Should(BeTrue(), "Expected to find log index for verrazzano-system")
				})
			},
			func() {
				// GIVEN existing system logs
				// WHEN the Elasticsearch index for the verrazzano-install namespace is retrieved
				// THEN verify that it is found
				t.It("Verify Elasticsearch index for logging exists", func() {
					Eventually(func() bool {
						return pkg.LogIndexFound("verrazzano-namespace-verrazzano-install")
					}, shortWaitTimeout, shortPollingInterval).Should(BeTrue(), "Expected to find log index for verrazzano-install")
				})
			},
		)
		// GIVEN...
		// Log message in Elasticsearch under the verrazzano-system index
		// With field kubernetes.labels.app==verrazzano-platform-operator
		// With field kubernetes.container_name in (verrazzano-platform-operator,webhook-init)
		// WHEN...
		// Log messages are returned from Elasticsearch
		// THEN...
		// Verify log records exist matching the criteria
		// Verify log records have a non-empty @timestamp field
		// Verify log records do not have a timestamp field
		// Verify log records have a non-empty level field
		// Verify log records have a non-empty message field
		t.It("from verrazzano-platform-operator", func() {
			// See https://coralogix.com/blog/42-elasticsearch-query-examples-hands-on-tutorial/
			query :=
				`{
					"from": 0,
					"size": 1000,
					"sort": [{
						"@timestamp": {
							"order": "desc"}}],
					"query": {
						"bool": {
							"must": [
								{"match": {"kubernetes.labels.app": "verrazzano-platform-operator"}},
								{"bool": {
									"should": [
										{"match": {"kubernetes.container_name": "verrazzano-platform-operator"}},
										{"match": {"kubernetes.container_name": "webhook-init"}}]}}]}}
				}`
			resp, err := pkg.PostElasticsearch("_search", query)
			if err != nil {
				pkg.Log(pkg.Error, fmt.Sprintf("Failed to query Elasticsearch: %v", err))
				t.Fail("Failed to query Elasticsearch")
			}
			if resp.StatusCode != 200 {
				pkg.Log(pkg.Error, fmt.Sprintf("Failed to query Elasticsearch: status=%d: body=%s", resp.StatusCode, string(resp.Body)))
				t.Fail("Failed to query Elasticsearch")
			}
			var result map[string]interface{}
			json.Unmarshal(resp.Body, &result)

			valid := pkg.ValidateElasticsearchHits(result, func(hit pkg.ElasticsearchHit) bool {
				fmt.Printf("hit=%v\n", hit)
				valid := true
				if val, ok := hit["@timestamp"]; !ok || len(val.(string)) == 0 {
					pkg.Log(pkg.Info, "Log record has missing or empty @timestamp field")
					valid = false
				}
				if val, ok := hit["message"]; !ok || len(val.(string)) == 0 {
					pkg.Log(pkg.Info, "Log record has missing or empty message field")
					valid = false
				}
				if val, ok := hit["level"]; !ok || len(val.(string)) == 0 {
					pkg.Log(pkg.Info, "Log record has missing or empty level field")
					valid = false
				}
				if _, ok := hit["timestamp"]; ok {
					pkg.Log(pkg.Info, "Log record has unwanted timestamp field")
					valid = false
				}
				if !valid {
					pkg.Log(pkg.Info, fmt.Sprintf("Log record is invalid: %v", hit))
				}
				return valid
			})
			if !valid {
				t.Fail("Found invalid log record")
			}
		})
	})
})
