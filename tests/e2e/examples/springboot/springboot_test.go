// Copyright (c) 2020, 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package springboot

import (
	"fmt"
	"net/http"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/verrazzano/verrazzano/pkg/k8sutil"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg"
)

var expectedPodsSpringBootApp = []string{"springboot-workload"}
var waitTimeout = 10 * time.Minute
var pollingInterval = 30 * time.Second
var shortPollingInterval = 10 * time.Second
var shortWaitTimeout = 5 * time.Minute
var longWaitTimeout = 15 * time.Minute
var longPollingInterval = 20 * time.Second

var _ = BeforeSuite(func() {
	if !skipDeploy {
		pkg.DeploySpringBootApplication()
	}
})

var failed = false
var _ = AfterEach(func() {
	failed = failed || CurrentSpecReport().Failed()
})

var _ = AfterSuite(func() {
	if failed {
		pkg.ExecuteClusterDumpWithEnvVarConfig()
	}
	if !skipUndeploy {
		pkg.UndeploySpringBootApplication()
	}
})

var _ = Describe("Verify Spring Boot Application", func() {
	// Verify springboot-workload pod is running
	// GIVEN springboot app is deployed
	// WHEN the component and appconfig are created
	// THEN the expected pod must be running in the test namespace
	Context("Deployment.", func() {
		It("and waiting for expected pods must be running", func() {
			Eventually(func() bool {
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
	It("Get host from gateway.", func() {
		Eventually(func() (string, error) {
			host, err = k8sutil.GetHostnameFromGateway(pkg.SpringbootNamespace, "")
			return host, err
		}, shortWaitTimeout, shortPollingInterval).Should(Not(BeEmpty()))
	})

	// Verify Spring Boot application is working
	// GIVEN springboot app is deployed
	// WHEN the component and appconfig with ingress trait are created
	// THEN the application endpoint must be accessible
	It("Verify welcome page of Spring Boot application is working.", func() {
		Eventually(func() (*pkg.HTTPResponse, error) {
			url := fmt.Sprintf("https://%s/", host)
			return pkg.GetWebPage(url, host)
		}, longWaitTimeout, longPollingInterval).Should(And(pkg.HasStatus(http.StatusOK), pkg.BodyContains("Greetings from Verrazzano Enterprise Container Platform")))
	})

	It("Verify Verrazzano facts endpoint is working.", func() {
		Eventually(func() (*pkg.HTTPResponse, error) {
			url := fmt.Sprintf("https://%s/facts", host)
			return pkg.GetWebPage(url, host)
		}, longWaitTimeout, longPollingInterval).Should(And(pkg.HasStatus(http.StatusOK), pkg.BodyNotEmpty()))
	})

	Context("Logging.", func() {
		indexName := "verrazzano-namespace-springboot"
		It("Verify Elasticsearch index exists", func() {
			Eventually(func() bool {
				return pkg.LogIndexFound(indexName)
			}, longWaitTimeout, longPollingInterval).Should(BeTrue(), "Expected to find Elasticsearch index for Spring Boot application.")
		})
		It("Verify recent Elasticsearch log record exists", func() {
			Eventually(func() bool {
				return pkg.LogRecordFound(indexName, time.Now().Add(-24*time.Hour), map[string]string{
					"kubernetes.labels.app_oam_dev\\/component": "springboot-component",
					"kubernetes.container_name":                 "springboot-container",
				})
			}, longWaitTimeout, longPollingInterval).Should(BeTrue(), "Expected to find a recent log record.")
			Eventually(func() bool {
				return pkg.LogRecordFound(indexName, time.Now().Add(-24*time.Hour), map[string]string{
					"kubernetes.labels.app_oam_dev\\/component": "springboot-component",
					"kubernetes.labels.app_oam_dev\\/name":      "springboot-appconf",
					"kubernetes.container_name":                 "springboot-container",
				})
			}, longWaitTimeout, longPollingInterval).Should(BeTrue(), "Expected to find a recent log record.")
		})
	})

	Context("Verify Prometheus scraped metrics.", func() {
		It("Retrieve Prometheus scraped metrics for App Component", func() {
			Eventually(func() bool {
				return pkg.MetricsExist("http_server_requests_seconds_count", "app_oam_dev_name", "springboot-appconf")
			}, longWaitTimeout, longPollingInterval).Should(BeTrue(), "Expected to find Prometheus scraped metrics for App Component.")
		})
		It("Retrieve Prometheus scraped metrics for App Config", func() {
			Eventually(func() bool {
				return pkg.MetricsExist("tomcat_sessions_created_sessions_total", "app_oam_dev_component", "springboot-component")
			}, longWaitTimeout, longPollingInterval).Should(BeTrue(), "Expected to find Prometheus scraped metrics for App Config.")
		})
	})
})
