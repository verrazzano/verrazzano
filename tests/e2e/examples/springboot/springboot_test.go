// Copyright (c) 2020, 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package springboot

import (
	"fmt"
	"github.com/verrazzano/verrazzano/pkg/test/framework"
	"github.com/verrazzano/verrazzano/pkg/test/framework/metrics"
	"net/http"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/verrazzano/verrazzano/pkg/k8sutil"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg"
)

var expectedPodsSpringBootApp = []string{"springboot-workload"}
var pollingInterval = 30 * time.Second
var shortPollingInterval = 10 * time.Second
var shortWaitTimeout = 5 * time.Minute
var longWaitTimeout = 15 * time.Minute
var longPollingInterval = 20 * time.Second

var t = framework.NewTestFramework("springboot")

var _ = t.BeforeSuite(func() {
	if !skipDeploy {
		start := time.Now()
		pkg.DeploySpringBootApplication()
		metrics.Emit(t.Metrics.With("deployment_elapsed_time", time.Since(start).Milliseconds()))
	}
})

var failed = false
var _ = t.AfterEach(func() {
	failed = failed || CurrentSpecReport().Failed()
})

var _ = t.AfterSuite(func() {
	if failed {
		pkg.ExecuteClusterDumpWithEnvVarConfig()
	}
	if !skipUndeploy {
		start := time.Now()
		pkg.UndeploySpringBootApplication()
		metrics.Emit(t.Metrics.With("undeployment_elapsed_time", time.Since(start).Milliseconds()))
	}
})

var _ = t.Describe("Spring Boot test", func() {
	// Verify springboot-workload pod is running
	// GIVEN springboot app is deployed
	// WHEN the component and appconfig are created
	// THEN the expected pod must be running in the test namespace
	t.Context("expected pods", func() {
		t.It("are running", func() {
			t.Eventually(func() bool {
				return pkg.PodsRunning(pkg.SpringbootNamespace, expectedPodsSpringBootApp)
			}, longWaitTimeout, pollingInterval).Should(BeTrue())
		})
	})

	var host = ""
	var err error
	// Get the host from the Istio gateway resource.
	// GIVEN the Istio gateway for the springboot namespace
	// WHEN GetHostnameFromGateway is called
	// THEN return the host name found in the gateway.
	t.It("Get host from gateway.", func() {
		t.Eventually(func() (string, error) {
			host, err = k8sutil.GetHostnameFromGateway(pkg.SpringbootNamespace, "")
			return host, err
		}, shortWaitTimeout, shortPollingInterval).Should(Not(BeEmpty()))
	})

	// Verify Spring Boot application is working
	// GIVEN springboot app is deployed
	// WHEN the component and appconfig with ingress trait are created
	// THEN the application endpoint must be accessible
	t.It("Verify welcome page of Spring Boot application is working.", func() {
		t.Eventually(func() (*pkg.HTTPResponse, error) {
			url := fmt.Sprintf("https://%s/", host)
			return pkg.GetWebPage(url, host)
		}, longWaitTimeout, longPollingInterval).Should(And(pkg.HasStatus(http.StatusOK), pkg.BodyContains("Greetings from Verrazzano Enterprise Container Platform")))
	})

	t.It("Verify Verrazzano facts endpoint is working.", func() {
		t.Eventually(func() (*pkg.HTTPResponse, error) {
			url := fmt.Sprintf("https://%s/facts", host)
			return pkg.GetWebPage(url, host)
		}, longWaitTimeout, longPollingInterval).Should(And(pkg.HasStatus(http.StatusOK), pkg.BodyNotEmpty()))
	})

	t.Context("for Logging.", func() {
		indexName := "verrazzano-namespace-springboot"
		t.It("Verify Elasticsearch index exists", func() {
			t.Eventually(func() bool {
				return pkg.LogIndexFound(indexName)
			}, longWaitTimeout, longPollingInterval).Should(BeTrue(), "Expected to find Elasticsearch index for Spring Boot application.")
		})
		t.It("Verify recent Elasticsearch log record exists", func() {
			t.Eventually(func() bool {
				return pkg.LogRecordFound(indexName, time.Now().Add(-24*time.Hour), map[string]string{
					"kubernetes.labels.app_oam_dev\\/component": "springboot-component",
					"kubernetes.container_name":                 "springboot-container",
				})
			}, longWaitTimeout, longPollingInterval).Should(BeTrue(), "Expected to find a recent log record.")
			t.Eventually(func() bool {
				return pkg.LogRecordFound(indexName, time.Now().Add(-24*time.Hour), map[string]string{
					"kubernetes.labels.app_oam_dev\\/component": "springboot-component",
					"kubernetes.labels.app_oam_dev\\/name":      "springboot-appconf",
					"kubernetes.container_name":                 "springboot-container",
				})
			}, longWaitTimeout, longPollingInterval).Should(BeTrue(), "Expected to find a recent log record.")
		})
	})

	t.Context("for metrics.", func() {
		t.It("Retrieve Prometheus scraped metrics for App Component", func() {
			t.Eventually(func() bool {
				return pkg.MetricsExist("http_server_requests_seconds_count", "app_oam_dev_name", "springboot-appconf")
			}, longWaitTimeout, longPollingInterval).Should(BeTrue(), "Expected to find Prometheus scraped metrics for App Component.")
		})
		t.It("Retrieve Prometheus scraped metrics for App Config", func() {
			t.Eventually(func() bool {
				return pkg.MetricsExist("tomcat_sessions_created_sessions_total", "app_oam_dev_component", "springboot-component")
			}, longWaitTimeout, longPollingInterval).Should(BeTrue(), "Expected to find Prometheus scraped metrics for App Config.")
		})
	})
})
