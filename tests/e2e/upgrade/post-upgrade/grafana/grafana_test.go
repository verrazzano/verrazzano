// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package grafana

import (
	"encoding/json"
	"fmt"
	"github.com/verrazzano/verrazzano/pkg/k8sutil"
	"net/http"
	"strings"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/verrazzano/verrazzano/pkg/test/framework"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg"
)

const (
	waitTimeout          = 3 * time.Minute
	pollingInterval      = 10 * time.Second
	testDashboardTitle   = "E2ETestDashboard"
	systemDashboardTitle = "Host Metrics"
)

var t = framework.NewTestFramework("grafana")

var _ = t.BeforeSuite(func() {
	kubeconfigPath, err := k8sutil.GetKubeConfigLocation()
	if err != nil {
		Fail(fmt.Sprintf("Failed to get default kubeconfig path: %s", err.Error()))
	}
	supported := pkg.IsGrafanaEnabled(kubeconfigPath)
	// Only run tests if Grafana component is enabled in Verrazzano CR
	if !supported {
		Skip("Grafana component is not enabled")
	}
})

var _ = t.Describe("Post Upgrade Grafana Dashboard", Label("f:observability.logging.es"), func() {

	// GIVEN a running Grafana instance,
	// WHEN a search is made for the dashboard using its title,
	// THEN the dashboard metadata is returned.
	It("Search the test Grafana Dashboard using its title", func() {
		Eventually(func() bool {
			resp, err := pkg.SearchGrafanaDashboard(map[string]string{"query": testDashboardTitle})
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
				if dashboard["title"] == testDashboardTitle {
					return true
				}
			}
			return false

		}).WithPolling(pollingInterval).WithTimeout(waitTimeout).Should(BeTrue())
	})

	// GIVEN a running grafana instance,
	// WHEN a GET call is made  to Grafana with the UID of the system dashboard,
	// THEN the dashboard metadata of the corresponding System dashboard is returned.
	It("Get details of the system Grafana dashboard", func() {
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
			return strings.Contains(body["dashboard"]["title"], systemDashboardTitle)
		}).WithPolling(pollingInterval).WithTimeout(waitTimeout).Should(BeTrue())
	})
})
