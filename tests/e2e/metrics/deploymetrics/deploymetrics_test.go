// Copyright (c) 2021, 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package deploymetrics

import (
	"github.com/verrazzano/verrazzano/pkg/test/framework"
	"github.com/verrazzano/verrazzano/pkg/test/framework/metrics"
	"time"

	. "github.com/onsi/gomega"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
)

const (
	testNamespace     = "deploymetrics"
	promConfigJobName = "deploymetrics-appconf_default_deploymetrics_deploymetrics-deployment"
)

var expectedPodsDeploymetricsApp = []string{"deploymetrics-workload"}
var waitTimeout = 10 * time.Minute
var pollingInterval = 30 * time.Second
var shortPollingInterval = 10 * time.Second
var shortWaitTimeout = 5 * time.Minute
var longWaitTimeout = 10 * time.Minute
var longPollingInterval = 20 * time.Second

var t = framework.NewTestFramework("deploymetrics")

var _ = t.BeforeSuite(func() {
	deployMetricsApplication()
})

var clusterDump = pkg.NewClusterDumpWrapper()
var _ = clusterDump.AfterEach(func() {}) // Dump cluster if spec fails
var _ = clusterDump.AfterSuite(func() {  // Dump cluster if aftersuite fails
	undeployMetricsApplication()
})

func deployMetricsApplication() {
	pkg.Log(pkg.Info, "Deploy DeployMetrics Application")

	pkg.Log(pkg.Info, "Create namespace")
	start := time.Now()
	Eventually(func() (*v1.Namespace, error) {
		nsLabels := map[string]string{
			"verrazzano-managed": "true",
			"istio-injection":    "enabled"}
		return pkg.CreateNamespace(testNamespace, nsLabels)
	}, shortWaitTimeout, shortPollingInterval).ShouldNot(BeNil())

	pkg.Log(pkg.Info, "Create component resource")
	Eventually(func() error {
		return pkg.CreateOrUpdateResourceFromFile("testdata/deploymetrics/deploymetrics-comp.yaml")
	}, shortWaitTimeout, shortPollingInterval).ShouldNot(HaveOccurred())

	pkg.Log(pkg.Info, "Create application resource")
	Eventually(func() error {
		return pkg.CreateOrUpdateResourceFromFile("testdata/deploymetrics/deploymetrics-app.yaml")
	}, shortWaitTimeout, shortPollingInterval).ShouldNot(HaveOccurred(), "Failed to create DeployMetrics application resource")
	metrics.Emit(t.Metrics.With("deployment_elapsed_time", time.Since(start).Milliseconds()))
}

func undeployMetricsApplication() {
	pkg.Log(pkg.Info, "Undeploy DeployMetrics Application")

	pkg.Log(pkg.Info, "Delete application")
	start := time.Now()
	Eventually(func() error {
		return pkg.DeleteResourceFromFile("testdata/deploymetrics/deploymetrics-app.yaml")
	}, shortWaitTimeout, shortPollingInterval).ShouldNot(HaveOccurred())

	pkg.Log(pkg.Info, "Delete components")
	Eventually(func() error {
		return pkg.DeleteResourceFromFile("testdata/deploymetrics/deploymetrics-comp.yaml")
	}, shortWaitTimeout, shortPollingInterval).ShouldNot(HaveOccurred())

	Eventually(func() bool {
		return pkg.IsAppInPromConfig(promConfigJobName)
	}, waitTimeout, pollingInterval).Should(BeFalse(), "Expected App to be removed from Prometheus Config")

	pkg.Log(pkg.Info, "Delete namespace")
	Eventually(func() error {
		return pkg.DeleteNamespace(testNamespace)
	}, shortWaitTimeout, shortPollingInterval).ShouldNot(HaveOccurred())

	Eventually(func() bool {
		ns, err := pkg.GetNamespace(testNamespace)
		if err == nil {
			finalizeErr := pkg.RemoveNamespaceFinalizers(ns)
			if finalizeErr != nil {
				return false
			}
		}
		return err != nil && errors.IsNotFound(err)
	}, shortWaitTimeout, shortPollingInterval).Should(BeTrue())
	metrics.Emit(t.Metrics.With("undeployment_elapsed_time", time.Since(start).Milliseconds()))
}

var _ = t.Describe("Verify DeployMetrics Application", func() {
	// Verify deploymetrics-workload pod is running
	// GIVEN deploymetrics app is deployed
	// WHEN the component and appconfig are created
	// THEN the expected pod must be running in the test namespace
	t.Context("Deployment.", func() {
		t.It("and waiting for expected pods must be running", func() {
			Eventually(func() bool {
				return pkg.PodsRunning(testNamespace, expectedPodsDeploymetricsApp)
			}, waitTimeout, pollingInterval).Should(BeTrue())
		})
	})

	t.Context("Prometheus Config.", func() {
		t.It("Verify that Prometheus Config Data contains deploymetrics-appconf_default_deploymetrics_deploymetrics-deployment", func() {
			Eventually(func() bool {
				return pkg.IsAppInPromConfig(promConfigJobName)
			}, waitTimeout, pollingInterval).Should(BeTrue(), "Expected to find App in Prometheus Config")
		})
	})

	t.Context("Verify Prometheus scraped metrics.", func() {
		t.It("Retrieve Prometheus scraped metrics for App Component", func() {
			Eventually(func() bool {
				return pkg.MetricsExist("http_server_requests_seconds_count", "app_oam_dev_name", "deploymetrics-appconf")
			}, longWaitTimeout, longPollingInterval).Should(BeTrue(), "Expected to find Prometheus scraped metrics for App Component.")
		})
		t.It("Retrieve Prometheus scraped metrics for App Config", func() {
			Eventually(func() bool {
				return pkg.MetricsExist("tomcat_sessions_created_sessions_total", "app_oam_dev_component", "deploymetrics-deployment")
			}, longWaitTimeout, longPollingInterval).Should(BeTrue(), "Expected to find Prometheus scraped metrics for App Config.")
		})
	})

})
