// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package podworkload

import (
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/verrazzano/verrazzano/pkg/test/framework"
	"github.com/verrazzano/verrazzano/pkg/test/framework/metrics"
	"github.com/verrazzano/verrazzano/tests/e2e/metricsbinding"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg"
)

const (
	shortWaitTimeout     = 10 * time.Minute
	shortPollingInterval = 10 * time.Second
	applicationPodPrefix = "hello-helidon-pod"
	yamlPath             = "tests/e2e/metricsbinding/testdata/hello-helidon-pod.yaml"
	promConfigJobName    = "_hello-helidon-pod_v1_Pod"
)

var (
	t                  = framework.NewTestFramework("podworkload")
	generatedNamespace = pkg.GenerateNamespace("hello-helidon-namespace")
	clusterDump        = pkg.NewClusterDumpWrapper()
)

var _ = clusterDump.BeforeSuite(func() {
	start := time.Now()
	metricsbinding.DeployApplication(namespace, yamlPath, applicationPodPrefix, istioInjection, *t)
	metrics.Emit(t.Metrics.With("deployment_elapsed_time", time.Since(start).Milliseconds()))
})

var _ = clusterDump.AfterEach(func() {}) // Dump cluster if spec fails
var _ = clusterDump.AfterSuite(func() {  // Dump cluster if aftersuite fails
	metricsbinding.UndeployApplication(namespace, yamlPath, namespace+promConfigJobName, *t)
})

var _ = t.AfterEach(func() {})

var _ = t.Describe("Verify application.", Label("f:app-lcm.poko"), func() {

	// GIVEN the Helidon app is deployed and the pods are running
	// WHEN the Prometheus metrics in the app namespace are scraped
	// THEN the Helidon application metrics should exist using the default metrics template for pods
	t.Context("Verify Prometheus scraped metrics.", Label("f:observability.monitoring.prom"),
		func() {
			t.It("Check Prometheus config map for scrape target", func() {
				Eventually(func() bool {
					return pkg.IsAppInPromConfig(namespace + promConfigJobName)
				}, shortWaitTimeout, shortPollingInterval).Should(BeTrue(), "Expected application to be found in Prometheus config")
			})
			t.It("Retrieve Prometheus scraped metrics for 'hello-helidon-pod' Pod", func() {
				Eventually(func() bool {
					return pkg.MetricsExist("base_jvm_uptime_seconds", "app_verrazzano_io_workload", "hello-helidon-pod-v1-pod")
				}, shortWaitTimeout, shortPollingInterval).Should(BeTrue(), "Expected to find Prometheus scraped metrics for Helidon application.")
			})
		})
})
