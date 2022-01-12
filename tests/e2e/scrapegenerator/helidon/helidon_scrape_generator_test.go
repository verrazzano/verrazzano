// Copyright (c) 2021, 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

// +build metrics_template_test

package helidonscrapegenerator

import (
	"github.com/verrazzano/verrazzano/tests/e2e/pkg"
	"k8s.io/apimachinery/pkg/types"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/verrazzano/verrazzano/tests/e2e/scrapegenerator"
)

const (
	longWaitTimeout      = 15 * time.Minute
	longPollingInterval  = 20 * time.Second
	namespace            = "hello-helidon-scrape"
	applicationPodPrefix = "hello-helidon-deployment-"
	metricsTemplateName  = "hello-helidon-metrics-template"
	yamlPath             = "testdata/scrapegenerator/helidon/helidon-scrape-generator.yaml"
)

var t = framework.NewTestFramework("helidonscrapegenerator")

var _ = t.BeforeSuite(func() {
	start := time.Now()
	scrapegenerator.DeployApplication(namespace, yamlPath)
	metrics.Emit(t.Metrics.With("deployment_elapsed_time", time.Since(start).Milliseconds()))
})

var clusterDump = pkg.NewClusterDumpWrapper()
var _ = clusterDump.AfterEach(func() {}) // Dump cluster if spec fails
var _ = clusterDump.AfterSuite(func() {  // Dump cluster if aftersuite fails
	scrapegenerator.UndeployApplication(namespace, yamlPath)
})

var _ = t.AfterEach(func() {})

var _ = t.Describe("Verify application.", func() {
	t.Context("Deployment.", func() {
		// GIVEN the app is deployed
		// WHEN the running pods are checked
		// THEN the Helidon pod should exist
		t.It("Verify 'hello-helidon-scrape' pod is running", func() {
			Eventually(func() bool {
				return pkg.PodsRunning(namespace, []string{applicationPodPrefix})
			}, longWaitTimeout, longPollingInterval).Should(BeTrue())
		})
		t.It("Verify metrics template exists", func() {
			Eventually(func() bool {
				return pkg.DoesMetricsTemplateExist(types.NamespacedName{Name: metricsTemplateName, Namespace: namespace})
			}, longWaitTimeout, longPollingInterval).Should(BeTrue())
		})
	})

	// GIVEN the Helidon app is deployed and the pods are running
	// WHEN the Prometheus metrics in the app namespace are scraped
	// THEN the Helidon application metrics should exist
	t.Context("Verify Prometheus scraped metrics.", func() {
		t.It("Retrieve Prometheus scraped metrics for Helidon Pod", func() {
			Eventually(func() bool {
				return pkg.MetricsExist("base_jvm_uptime_seconds", "app", "hello-helidon-scrape-application")
			}, longWaitTimeout, longPollingInterval).Should(BeTrue(), "Expected to find Prometheus scraped metrics for Helidon application.")
		})
	})
})
