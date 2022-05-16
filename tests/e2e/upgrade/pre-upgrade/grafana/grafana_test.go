// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package opensearch

import (
	"fmt"
	"io/ioutil"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/verrazzano/verrazzano/pkg/test/framework"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg"
)

const (
	threeMinutes    = 3 * time.Minute
	pollingInterval = 10 * time.Second
	documentFile    = "testdata/upgrade/grafana/dashboard.json"
)

var t = framework.NewTestFramework("grafana")

var _ = t.Describe("Pre Upgrade Grafana Dashboard", Label("f:observability.logging.es"), func() {
	// GIVEN a running Grafana instance
	// WHEN a dashboard is created
	// THEN verify that the dashboard is created successfully.
	It("Grafana Create Dashboard", func() {
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
				pkg.Log(pkg.Error, fmt.Sprintf("Failed to write to Grafana: %v", err))
				return false
			}
			if resp.StatusCode != 200 {
				pkg.Log(pkg.Error, fmt.Sprintf("Failed to write to Grafana: status=%d: body=%s", resp.StatusCode, string(resp.Body)))
				return false
			}

			return true
		}).WithPolling(pollingInterval).WithTimeout(threeMinutes).Should(BeTrue(), "Expected not to fail while writing data to OpenSearch")
	})

	// GIVEN a running grafana instance
	// WHEN a GET call is made  to Grafana with the specific UID
	// THEN the dashboard metadata of the corresponding dashboard is returned.
	It("Grafana Get Dasbhoard details" , func() {
		uid := "pIZicTl8z"
		Eventually( func() bool {
			resp, err := pkg.GetGrafanaDashboard(uid)
			if err != nil {
				pkg.Log(pkg.Error, err.Error())
				return false
			}
			if resp.StatusCode != 200 {
				pkg.Log(pkg.Error, fmt.Sprintf("Failed to GET Grafana dashboard: status=%d: body=%s", resp.StatusCode, string(resp.Body)))
				return false
			}
			return true

		}).WithPolling(pollingInterval).WithTimeout(threeMinutes)
	})
})
