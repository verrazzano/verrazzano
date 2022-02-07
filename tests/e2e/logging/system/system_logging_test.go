// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package system

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
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

var _ = t.Describe("Elasticsearch data", Label("f:observability.logging.es"), func() {
	t.It("contains verrazzano-system index", func() {
		// GIVEN existing system logs
		// WHEN the Elasticsearch index for the verrazzano-system namespace is retrieved
		// THEN verify that it is found
		Eventually(func() bool {
			return pkg.LogIndexFound("verrazzano-namespace-verrazzano-system")
		}, shortWaitTimeout, shortPollingInterval).Should(BeTrue(), "Expected to find Elasticsearch index verrazzano-system")
	})

	t.It("contains valid verrazzano-install index and records", func() {
		// GIVEN existing system logs
		// WHEN the Elasticsearch index for the verrazzano-install namespace is retrieved
		// THEN verify that it is found
		Eventually(func() bool {
			return pkg.LogIndexFound("verrazzano-namespace-verrazzano-install")
		}, shortWaitTimeout, shortPollingInterval).Should(BeTrue(), "Expected to find Elasticsearch index verrazzano-install")

		// GIVEN...
		// Log message in Elasticsearch in the verrazzano-namespace-verrazzano-install index
		// With field kubernetes.labels.app==verrazzano-platform-operator
		// WHEN...
		// Log messages are retrieved from Elasticsearch
		// THEN...
		// Verify log records exist matching the criteria
		// Verify log records have a non-empty @timestamp field
		// Verify log records do not have a timestamp field
		// Verify log records have a non-empty level field
		// Verify log records have a non-empty message field
		query :=
			`{
				"size": 1000,
				"query": {
					"bool": {
						"filter" : [
							{"match_phrase": {"kubernetes.labels.app.keyword": "verrazzano-platform-operator"}},
							{"range": {"@timestamp": {"gte": "now-1h"}}}
						]
					}
				}
			}`
		resp, err := pkg.PostElasticsearch("verrazzano-namespace-verrazzano-install/_search", query)
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

		valid := pkg.ValidateElasticsearchHits(result, basicElasticsearchHitValidator)
		if !valid {
			t.Fail("Found invalid log record")
		}
	})
})

// Validate log records have a non-empty @timestamp field
// Validate log records do not have a timestamp field
// Validate log records have a non-empty level field
// Validate log records have a non-empty message field
func basicElasticsearchHitValidator(hit pkg.ElasticsearchHit) bool {
	//fmt.Printf("hit=%v\n", hit)
	ts := ""
	valid := true
	if val, ok := hit["@timestamp"]; !ok || len(val.(string)) == 0 {
		pkg.Log(pkg.Info, "Log record has missing or empty @timestamp field")
		valid = false
	} else {
		ts = hit["@timestamp"].(string)
	}
	if val, ok := hit["log"]; !ok || len(val.(string)) == 0 {
		pkg.Log(pkg.Info, "Log record has missing or empty log field")
		valid = false
	} else {
		re := regexp.MustCompile(`(\d{2}:\d{2}:\d{2}.\d{3})`)
		m := re.FindStringSubmatch(val.(string))
		if len(m) < 2 {
			pkg.Log(pkg.Info, "Log record log field does not contain a time")
			valid = false
		} else {
			if !strings.Contains(ts, m[1]) {
				pkg.Log(pkg.Info, fmt.Sprintf("Log record @timestamp field %s does not match log field %s content", ts, m[1]))
				valid = false
			}
		}
	}
	if val, ok := hit["message"]; !ok || len(val.(string)) == 0 {
		pkg.Log(pkg.Info, "Log record has missing or empty message field")
		valid = false
	}
	if hit["log"] == hit["message"] {
		pkg.Log(pkg.Info, "Log record has duplicate log and message field values")
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
}
