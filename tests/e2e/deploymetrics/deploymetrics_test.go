// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package deploymetrics

import (
	"fmt"
	"time"

	"github.com/onsi/ginkgo"
	"github.com/onsi/gomega"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg"
	"k8s.io/apimachinery/pkg/api/errors"
)

const testNamespace string = "deploymetrics"

var expectedPodsDeploymetricsApp = []string{"deploymetrics-workload"}
var waitTimeout = 10 * time.Minute
var pollingInterval = 30 * time.Second
var shortPollingInterval = 10 * time.Second
var shortWaitTimeout = 5 * time.Minute
var longWaitTimeout = 10 * time.Minute
var longPollingInterval = 20 * time.Second

var _ = ginkgo.BeforeSuite(func() {
	deployMetricsApplication()
})

var failed = false
var _ = ginkgo.AfterEach(func() {
	failed = failed || ginkgo.CurrentGinkgoTestDescription().Failed
})

var _ = ginkgo.AfterSuite(func() {
	if failed {
		pkg.ExecuteClusterDumpWithEnvVarConfig()
	}
	undeployMetricsApplication()
})

func deployMetricsApplication() {
<<<<<<< HEAD
	pkg.Log(pkg.Info, "Deploy DeployMetrics Application")
=======
	pkg.Log(pkg.Info, "Deploy Spring Boot Application")
>>>>>>> Add Acceptance Tests

	pkg.Log(pkg.Info, "Create namespace")
	nsLabels := map[string]string{
		"verrazzano-managed": "true",
		"istio-injection":    "enabled"}
	if _, err := pkg.CreateNamespace(testNamespace, nsLabels); err != nil {
		ginkgo.Fail(fmt.Sprintf("Failed to create namespace: %v", err))
	}

	pkg.Log(pkg.Info, "Create component resource")
	if err := pkg.CreateOrUpdateResourceFromFile("testdata/deploymetrics/deploymetrics-comp.yaml"); err != nil {
		ginkgo.Fail(fmt.Sprintf("Failed to create DeployMetrics component resources: %v", err))
	}
	pkg.Log(pkg.Info, "Create application resource")
	gomega.Eventually(func() error {
		return pkg.CreateOrUpdateResourceFromFile("testdata/deploymetrics/deploymetrics-app.yaml")
	}, shortWaitTimeout, shortPollingInterval).Should(gomega.BeNil(), "Failed to create DeployMetrics application resource")
}

func undeployMetricsApplication() {
<<<<<<< HEAD
	pkg.Log(pkg.Info, "Undeploy DeployMetrics Application")

=======
	pkg.Log(pkg.Info, "Undeploy Spring Boot Application")
>>>>>>> Add Acceptance Tests
	pkg.Log(pkg.Info, "Delete application")
	if err := pkg.DeleteResourceFromFile("testdata/deploymetrics/deploymetrics-app.yaml"); err != nil {
		pkg.Log(pkg.Error, fmt.Sprintf("Failed to delete the application: %v", err))
	}
	pkg.Log(pkg.Info, "Delete components")
	if err := pkg.DeleteResourceFromFile("testdata/deploymetrics/deploymetrics-comp.yaml"); err != nil {
		pkg.Log(pkg.Error, fmt.Sprintf("Failed to delete the component: %v", err))
	}
<<<<<<< HEAD

	gomega.Eventually(func() bool {
		return pkg.MetricsExist("http_server_requests_seconds_count", "app_oam_dev_name", "deploymetrics-appconf")
	}, longWaitTimeout, longPollingInterval).Should(gomega.BeFalse(), "Prometheus scraped metrics for App Component should have been deleted.")

	gomega.Eventually(func() bool {
		return pkg.MetricsExist("tomcat_sessions_created_sessions_total", "app_oam_dev_component", "deploymetrics-deployment")
	}, longWaitTimeout, longPollingInterval).Should(gomega.BeFalse(), "Prometheus scraped metrics for App Config should have been deleted.")

=======
>>>>>>> Add Acceptance Tests
	pkg.Log(pkg.Info, "Delete namespace")
	if err := pkg.DeleteNamespace(testNamespace); err != nil {
		pkg.Log(pkg.Error, fmt.Sprintf("Failed to delete the namespace: %v", err))
	}
	gomega.Eventually(func() bool {
		ns, err := pkg.GetNamespace(testNamespace)
		return ns == nil && err != nil && errors.IsNotFound(err)
	}, 3*time.Minute, 15*time.Second).Should(gomega.BeFalse())
}

var _ = ginkgo.Describe("Verify DeployMetrics Application", func() {
<<<<<<< HEAD
	// Verify deploymetrics-workload pod is running
	// GIVEN deploymetrics app is deployed
=======
	// Verify springboot-workload pod is running
	// GIVEN springboot app is deployed
>>>>>>> Add Acceptance Tests
	// WHEN the component and appconfig are created
	// THEN the expected pod must be running in the test namespace
	ginkgo.Context("Deployment.", func() {
		ginkgo.It("and waiting for expected pods must be running", func() {
			gomega.Eventually(func() bool {
				return pkg.PodsRunning(testNamespace, expectedPodsDeploymetricsApp)
			}, waitTimeout, pollingInterval).Should(gomega.BeTrue())
		})
	})

	ginkgo.Context("Verify Prometheus scraped metrics.", func() {
		ginkgo.It("Retrieve Prometheus scraped metrics for App Component", func() {
			gomega.Eventually(func() bool {
				return pkg.MetricsExist("http_server_requests_seconds_count", "app_oam_dev_name", "deploymetrics-appconf")
			}, longWaitTimeout, longPollingInterval).Should(gomega.BeTrue(), "Expected to find Prometheus scraped metrics for App Component.")
		})
		ginkgo.It("Retrieve Prometheus scraped metrics for App Config", func() {
			gomega.Eventually(func() bool {
				return pkg.MetricsExist("tomcat_sessions_created_sessions_total", "app_oam_dev_component", "deploymetrics-deployment")
			}, longWaitTimeout, longPollingInterval).Should(gomega.BeTrue(), "Expected to find Prometheus scraped metrics for App Config.")
		})
	})

})
