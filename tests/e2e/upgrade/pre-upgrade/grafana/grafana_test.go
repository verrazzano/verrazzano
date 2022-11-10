// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package grafana

import (
	"encoding/json"
	"fmt"
	"github.com/verrazzano/verrazzano/pkg/k8sutil"
	"net/http"
	"os"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg/test/framework"
)

const (
	waitTimeout     = 3 * time.Minute
	pollingInterval = 10 * time.Second
	documentFile    = "testdata/upgrade/grafana/dashboard.json"
)

var testDashboard pkg.DashboardMetadata
var t = framework.NewTestFramework("grafana")

var _ = t.BeforeSuite(func() {
	kubeconfigPath, err := k8sutil.GetKubeConfigLocation()
	if err != nil {
		Fail(fmt.Sprintf(pkg.KubeConfigErrorFmt, err))
	}
	supported := pkg.IsGrafanaEnabled(kubeconfigPath)
	// Only run tests if Grafana component is enabled in Verrazzano CR
	if !supported {
		Skip("Grafana component is not enabled")
	}
	// Create the test Grafana dashboard
	file, err := pkg.FindTestDataFile(documentFile)
	if err != nil {
		pkg.Log(pkg.Error, fmt.Sprintf("failed to find test data file: %v", err))
		Fail(err.Error())
	}
	data, err := os.ReadFile(file)
	if err != nil {
		pkg.Log(pkg.Error, fmt.Sprintf("failed to read test data file: %v", err))
		Fail(err.Error())
	}
	Eventually(func() bool {
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
	}).WithPolling(pollingInterval).WithTimeout(waitTimeout).Should(BeTrue(),
		"It should be possible to create a Grafana dashboard and persist it.")
})

var _ = t.Describe("Pre Upgrade Grafana Dashboard", Label("f:observability.logging.es"), func() {

	// GIVEN a running grafana instance,
	// WHEN a GET call is made  to Grafana with the UID of the newly created testDashboard,
	// THEN the testDashboard metadata of the corresponding testDashboard is returned.
	t.It("Get details of the test Grafana Dashboard", func() {
		pkg.TestGrafanaTestDashboard(testDashboard, pollingInterval, waitTimeout)
	})

	// GIVEN a running Grafana instance,
	// WHEN a search is done based on the dashboard title,
	// THEN the details of the dashboards matching the search query is returned.
	t.It("Search the test Grafana Dashboard using its title", func() {
		pkg.TestSearchGrafanaDashboard(pollingInterval, waitTimeout)
	})

	// GIVEN a running grafana instance,
	// WHEN a GET call is made  to Grafana with the system dashboard UID,
	// THEN the dashboard metadata of the corresponding testDashboard is returned.
	t.It("Get details of the system Grafana Dashboard", func() {
		pkg.TestSystemGrafanaDashboard(pollingInterval, waitTimeout)
	})

	kubeconfigPath, err := k8sutil.GetKubeConfigLocation()
	if err != nil {
		Expect(err).To(BeNil(), fmt.Sprintf(pkg.KubeConfigErrorFmt, err))
	}

	// GIVEN a running grafana instance
	// WHEN a call is made to Grafana Dashboard with UID corresponding to OpenSearch Summary Dashboard
	// THEN the dashboard metadata of the corresponding dashboard is returned
	if ok, _ := pkg.IsVerrazzanoMinVersion("1.3.0", kubeconfigPath); ok {
		t.It("Get details of the OpenSearch Grafana Dashboard", func() {
			pkg.TestOpenSearchGrafanaDashBoard(pollingInterval, waitTimeout)
		})
	}
})
