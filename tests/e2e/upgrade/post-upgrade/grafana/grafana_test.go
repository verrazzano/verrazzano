// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package opensearch

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/verrazzano/verrazzano/pkg/test/framework"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg"
)

const (
	threeMinutes    = 3 * time.Minute
	pollingInterval = 10 * time.Second
)

var t = framework.NewTestFramework("grafana")

var _ = t.Describe("Post Upgrade Grafana Dashboard", Label("f:observability.logging.es"), func() {
	// GIVEN a running Grafana instance
	// WHEN a search is made for the dashboard
	// THEN the dashboard metadata is returned
	It("Grafana search Dasbhoard details", func() {
		Eventually(func() bool {
			resp, err := pkg.SearchGrafanaDashboard(map[string]string{"query": "E2ETestDashboard"})
			if err != nil {
				pkg.Log(pkg.Error, err.Error())
				return false
			}
			if resp.StatusCode != http.StatusOK {
				pkg.Log(pkg.Error, fmt.Sprintf("Failed to GET Grafana testDashboard: status=%d: body=%s", resp.StatusCode, string(resp.Body)))
				return false
			}
			var body []map[string]string
			json.Unmarshal(resp.Body, &body)
			for _, dashboard := range body {
				if dashboard["title"] == "E2ETestDashboard" {
					return true
				}
			}
			return false

		}).WithPolling(pollingInterval).WithTimeout(threeMinutes).Should(BeTrue())
	})

	// GIVEN a running grafana instance
	// WHEN a GET call is made  to Grafana with the specific UID
	// THEN the dashboard metadata of the corresponding testDashboard is returned.
	It("Grafana System Dasbhoard details", func() {
		// UID of system testDashboard, which is created by the VMO on startup.
		uid := "H0xWYyyik"
		Eventually(func() bool {
			resp, err := pkg.GetGrafanaDashboard(uid)
			if err != nil {
				pkg.Log(pkg.Error, err.Error())
				return false
			}
			if resp.StatusCode != http.StatusOK {
				pkg.Log(pkg.Error, fmt.Sprintf("Failed to GET Grafana testDashboard: status=%d: body=%s", resp.StatusCode, string(resp.Body)))
				return false
			}
			body := make(map[string]map[string]string)
			json.Unmarshal(resp.Body, &body)
			return strings.Contains(body["dashboard"]["title"], "Host Metrics")
		}).WithPolling(pollingInterval).WithTimeout(threeMinutes).Should(BeTrue())
	})
})
