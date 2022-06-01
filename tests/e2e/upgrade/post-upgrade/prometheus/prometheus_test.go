// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package prometheus

import (
	"github.com/verrazzano/verrazzano/tests/e2e/upgrade"
	"k8s.io/apimachinery/pkg/api/errors"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/verrazzano/verrazzano/pkg/test/framework"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg"
)

const (
	threeMinutes    = 3 * time.Minute
	pollingInterval = 10 * time.Second
	longTimeout     = 10 * time.Minute

	promConfigJobName = "deploymetrics-appconf_default_deploymetrics_deploymetrics-deployment"
)

var t = framework.NewTestFramework("prometheus")

var _ = t.BeforeSuite(func() {
	upgrade.SkipIfPrometheusDisabled()
})

var _ = t.AfterSuite(func() {
	undeployMetricsApplication()
})

var _ = t.Describe("Post upgrade Prometheus", Label("f:observability.monitoring.prom"), func() {

	// GIVEN a running Prometheus instance,
	// WHEN a sample NGINX metric is queried,
	// THEN verify that the metric could be retrieved.
	t.It("Verify sample NGINX metrics can be queried from Prometheus",
		upgrade.VerifyNginxMetric())

	// GIVEN a running Prometheus instance,
	// WHEN a sample Container advisor metric is queried,
	// THEN verify that the metric could be retrieved.
	t.It("Verify sample Container Advisor metrics can be queried from Prometheus",
		upgrade.VerifyContainerAdvisorMetric())

	// GIVEN a running Prometheus instance,
	// WHEN a sample node exporter metric is queried,
	// THEN verify that the metric could be retrieved.
	t.It("Verify sample Node Exporter metrics can be queried from Prometheus",
		upgrade.VerifyNodeExporterMetric())

	// GIVEN a running Prometheus instance,
	// WHEN checking for the test metric created during pre-upgrade,
	// THEN verify that the metric is present.
	It("Check if the created test metrics is present",
		upgrade.VerifyDeploymentMetric())
})

func undeployMetricsApplication() {
	t.Logs.Info("Undeploy DeployMetrics Application")

	Eventually(func() error {
		return pkg.DeleteResourceFromFile("testdata/deploymetrics/deploymetrics-app.yaml")
	}, threeMinutes, pollingInterval).ShouldNot(HaveOccurred())

	t.Logs.Info("Delete components")
	Eventually(func() error {
		return pkg.DeleteResourceFromFile("testdata/deploymetrics/deploymetrics-comp.yaml")
	}, threeMinutes, pollingInterval).ShouldNot(HaveOccurred())

	t.Logs.Info("Wait for pods to terminate")
	Eventually(func() bool {
		podsNotRunning, _ := pkg.PodsNotRunning(upgrade.PromAppNamespace, upgrade.ExpectedPodsDeploymetricsApp)
		return podsNotRunning
	}, threeMinutes, pollingInterval).Should(BeTrue())

	Eventually(func() bool {
		return pkg.IsAppInPromConfig(promConfigJobName)
	}, threeMinutes, pollingInterval).Should(BeFalse(),
		"Expected App to be removed from Prometheus Config")

	t.Logs.Info("Delete namespace")
	Eventually(func() error {
		return pkg.DeleteNamespace(upgrade.PromAppNamespace)
	}, threeMinutes, pollingInterval).ShouldNot(HaveOccurred())

	t.Logs.Info("Wait for Finalizer to be removed")
	Eventually(func() bool {
		return pkg.CheckNamespaceFinalizerRemoved(upgrade.PromAppNamespace)
	}, threeMinutes, pollingInterval).Should(BeTrue())

	t.Logs.Info("Waiting for namespace deletion")
	Eventually(func() bool {
		_, err := pkg.GetNamespace(upgrade.PromAppNamespace)
		return err != nil && errors.IsNotFound(err)
	}, longTimeout, pollingInterval).Should(BeTrue())
}
