// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package grafana

import (
	"encoding/json"
	"fmt"
	"github.com/verrazzano/verrazzano/pkg/k8sutil"
	"io/ioutil"
	"net/http"
	"strings"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/verrazzano/verrazzano/pkg/test/framework"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg"
)

const (
	threeMinutes     = 3 * time.Minute
	pollingInterval  = 10 * time.Second
	documentFile     = "testdata/upgrade/grafana/dashboard.json"
	grafanaErrMsgFmt = "Failed to GET Grafana testDashboard: status=%d: body=%s"
)

var t = framework.NewTestFramework("grafana")

type DashboardMetadata struct {
	ID      int    `json:"id"`
	Slug    string `json:"slug"`
	Status  string `json:"status"`
	UID     string `json:"uid"`
	URL     string `json:"url"`
	Version int    `json:"version"`
}

var _ = t.Describe("Pre Upgrade Grafana Dashboard", Label("f:observability.logging.es"), func() {
	// It Wrapper to only run spec if component is supported on the current Verrazzano installation
	GrafanaSupportedIt := func(description string, f func()) {
		kubeconfigPath, err := k8sutil.GetKubeConfigLocation()
		if err != nil {
			t.It(description, func() {
				Fail(fmt.Sprintf("Failed to get default kubeconfig path: %s", err.Error()))
			})
		}
		supported := pkg.IsGrafanaEnabled(kubeconfigPath)
		// Only run tests if Verrazzano is at least version 1.3.0
		if supported {
			t.It(description, f)
		} else {
			pkg.Log(pkg.Info, fmt.Sprintf("Skipping check '%v', Verrazzano is not at version 1.3.0", description))
		}
	}

	var testDashboard DashboardMetadata
	// GIVEN a running Grafana instance,
	// WHEN a dashboard is created,
	// THEN verify that the dashboard is created/saved successfully.
	GrafanaSupportedIt("Grafana Create Dashboard", func() {
		Eventually(func() bool {
			file, err := pkg.FindTestDataFile(documentFile)
			if err != nil {
				pkg.Log(pkg.Error, fmt.Sprintf("failed to find test data file: %v", err))
				return false
			}
			data, err := ioutil.ReadFile(file)
			if err != nil {
				pkg.Log(pkg.Error, fmt.Sprintf("failed to read test data file: %v", err))
				return false
			}
			resp, err := pkg.CreateGrafanaDashboard(string(data))
			if err != nil {
				pkg.Log(pkg.Error, fmt.Sprintf("Failed to create Grafana testDashboard: %v", err))
				return false
			}
			if resp.StatusCode != http.StatusOK {
				pkg.Log(pkg.Error, fmt.Sprintf("Failed to create Grafana testDashboard: status=%d: body=%s", resp.StatusCode, string(resp.Body)))
				return false
			}
			json.Unmarshal(resp.Body, &testDashboard)
			return true
		}).WithPolling(pollingInterval).WithTimeout(threeMinutes).Should(BeTrue(), "Expected not to fail while writing data to OpenSearch")
	})

	// GIVEN a running grafana instance,
	// WHEN a GET call is made  to Grafana with the UID of the newly created testDashboard,
	// THEN the testDashboard metadata of the corresponding testDashboard is returned.
	GrafanaSupportedIt("Grafana Get Dasbhoard details", func() {
		Eventually(func() bool {
			// UID of testDashboard, which is created by the previous test.
			uid := testDashboard.UID
			if uid == "" {
				return false
			}
			resp, err := pkg.GetGrafanaDashboard(uid)
			if err != nil {
				pkg.Log(pkg.Error, err.Error())
				return false
			}
			if resp.StatusCode != http.StatusOK {
				pkg.Log(pkg.Error, fmt.Sprintf(grafanaErrMsgFmt, resp.StatusCode, string(resp.Body)))
				return false
			}
			body := make(map[string]map[string]string)
			json.Unmarshal(resp.Body, &body)
			return strings.Contains(body["dashboard"]["title"], "E2ETestDashboard")
		}).WithPolling(pollingInterval).WithTimeout(threeMinutes).Should(BeTrue())
	})

	GrafanaSupportedIt("Grafana search Dasbhoard details", func() {
		Eventually(func() bool {
			resp, err := pkg.SearchGrafanaDashboard(map[string]string{"query": "E2ETestDashboard"})
			if err != nil {
				pkg.Log(pkg.Error, err.Error())
				return false
			}
			if resp.StatusCode != http.StatusOK {
				pkg.Log(pkg.Error, fmt.Sprintf(grafanaErrMsgFmt, resp.StatusCode, string(resp.Body)))
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
	GrafanaSupportedIt("Grafana System Dasbhoard details", func() {
		// UID of system testDashboard, which is created by the VMO on startup.
		uid := "H0xWYyyik"
		Eventually(func() bool {
			resp, err := pkg.GetGrafanaDashboard(uid)
			if err != nil {
				pkg.Log(pkg.Error, err.Error())
				return false
			}
			if resp.StatusCode != http.StatusOK {
				pkg.Log(pkg.Error, fmt.Sprintf(grafanaErrMsgFmt, resp.StatusCode, string(resp.Body)))
				return false
			}
			body := make(map[string]map[string]string)
			json.Unmarshal(resp.Body, &body)
			return strings.Contains(body["dashboard"]["title"], "Host Metrics")
		}).WithPolling(pollingInterval).WithTimeout(threeMinutes).Should(BeTrue())
	})
})
