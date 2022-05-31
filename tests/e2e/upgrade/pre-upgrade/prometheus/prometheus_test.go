// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package prometheus

import (
	"fmt"
	v1 "k8s.io/api/core/v1"
	"strings"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/verrazzano/verrazzano/pkg/test/framework"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg"
	"github.com/verrazzano/verrazzano/tests/e2e/upgrade/common"
)

const (
	threeMinutes    = 3 * time.Minute
	pollingInterval = 10 * time.Second
)

var adminKubeConfig string

var t = framework.NewTestFramework("prometheus")

var _ = t.BeforeSuite(func() {
	common.SkipIfPrometheusDisabled()
	adminKubeConfig = common.InitKubeConfigPath()
	deployMetricsApplication()
})

var _ = t.Describe("Pre upgrade Prometheus", Label("f:observability.monitoring.prom"), func() {

	// GIVEN a running Prometheus instance,
	// WHEN a scrape config is created,
	// THEN verify that the scrape config is created correctly
	It("Scrape targets can be listed and there is at least 1 scrape target",
		common.VerifyScrapeTargets(adminKubeConfig))

	// GIVEN a running Prometheus instance,
	// WHEN a sample NGINX metric is queried,
	// THEN verify that the metric could be retrieved.
	t.It("Verify sample NGINX metrics can be queried from Prometheus",
		common.VerifyNginxMetric(adminKubeConfig))

	// GIVEN a running Prometheus instance,
	// WHEN a sample Container advisor metric is queried,
	// THEN verify that the metric could be retrieved.
	t.It("Verify sample Container Advisor metrics can be queried from Prometheus",
		common.VerifyContainerAdvisorMetric(adminKubeConfig))

	// GIVEN a running Prometheus instance,
	// WHEN a sample node exporter metric is queried,
	// THEN verify that the metric could be retrieved.
	t.It("Verify sample Node Exporter metrics can be queried from Prometheus",
		common.VerifyNodeExporterMetric(adminKubeConfig))

	// GIVEN a running Prometheus instance,
	// WHEN a metric is created,
	// THEN verify that the metric is persisted in the prometheus time series DB.
	It("Validate if the test metric created by the test OAM deployment exists",
		common.VerifyDeploymentMetric(adminKubeConfig))

})

func deployMetricsApplication() {
	t.Logs.Info("Deploy DeployMetrics Application")
	Eventually(func() *v1.Namespace {
		nsLabels := map[string]string{
			"verrazzano-managed": "true",
			"istio-injection":    "true"}
		ns, err := pkg.CreateNamespace(common.TestNamespace, nsLabels)
		if err != nil && strings.Contains(err.Error(), "already exists") {
			ns, _ = pkg.GetNamespace(common.TestNamespace)
			return ns
		}
		return ns
	}, threeMinutes, pollingInterval).ShouldNot(BeNil())

	t.Logs.Info("Create component resource")
	Eventually(func() error {
		return pkg.CreateOrUpdateResourceFromFile("testdata/deploymetrics/deploymetrics-comp.yaml")
	}, threeMinutes, pollingInterval).ShouldNot(HaveOccurred())

	t.Logs.Info("Create application resource")
	Eventually(func() error {
		return pkg.CreateOrUpdateResourceFromFile("testdata/deploymetrics/deploymetrics-app.yaml")
	}, threeMinutes, pollingInterval).ShouldNot(HaveOccurred(), "Failed to create DeployMetrics application resource")

	Eventually(func() bool {
		return pkg.ContainerImagePullWait(common.TestNamespace, common.ExpectedPodsDeploymetricsApp)
	}, threeMinutes, pollingInterval).Should(BeTrue())

	t.Logs.Info("Verify deploymetrics-workload pod is running")
	Eventually(func() bool {
		result, err := pkg.PodsRunning(common.TestNamespace, common.ExpectedPodsDeploymetricsApp)
		if err != nil {
			pkg.Log(pkg.Error, fmt.Sprintf("One or more pods are not running in the namespace: %v, error: %v",
				common.TestNamespace, err))
			return false
		}
		return result
	}, threeMinutes, pollingInterval).Should(BeTrue())
}
