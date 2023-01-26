// Copyright (c) 2020, 2023, Oracle and/or its affiliates.
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
	dump "github.com/verrazzano/verrazzano/tests/e2e/pkg/test/clusterdump"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg/test/framework"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg/test/framework/metrics"
)

const (
	shortWaitTimeout         = 5 * time.Minute
	shortPollingInterval     = 10 * time.Second
	longWaitTimeout          = 20 * time.Minute
	longPollingInterval      = 20 * time.Second
	imagePullWaitTimeout     = 40 * time.Minute
	imagePullPollingInterval = 30 * time.Second

	springService = "springboot-workload"
	ingress       = "springboot-ingress-rule"
	appCreds      = "springboot-appconf"
)

var (
	t                         = framework.NewTestFramework("springboot")
	generatedNamespace        = pkg.GenerateNamespace("springboot")
	expectedPodsSpringBootApp = []string{"springboot-workload"}
	host                      = ""
)

var beforeSuite = t.BeforeSuiteFunc(func() {
	if !skipDeploy {
		start := time.Now()
		pkg.DeploySpringBootApplication(namespace, istioInjection)
		metrics.Emit(t.Metrics.With("deployment_elapsed_time", time.Since(start).Milliseconds()))

		// Verify springboot-workload pod is running
		// GIVEN springboot app is deployed
		// WHEN the component and appconfig are created
		// THEN the expected pod must be running in the test namespace
		t.Logs.Info("Container image pull check")
		Eventually(func() bool {
			return pkg.ContainerImagePullWait(namespace, expectedPodsSpringBootApp)
		}, imagePullWaitTimeout, imagePullPollingInterval).Should(BeTrue())
	}

	t.Logs.Info("Spring Boot Application: check expected pods are running")
	Eventually(func() bool {
		result, err := pkg.PodsRunning(namespace, expectedPodsSpringBootApp)
		if err != nil {
			AbortSuite(fmt.Sprintf("One or more pods are not running in the namespace: %v, error: %v", namespace, err))
		}
		return result
	}, longWaitTimeout, longPollingInterval).Should(BeTrue(), "Spring Boot Application Failed to Deploy: Pods are not ready")

	t.Logs.Info("Spring Boot Application: check expected Service is running")
	Eventually(func() bool {
		result, err := pkg.DoesServiceExist(namespace, springService)
		if err != nil {
			AbortSuite(fmt.Sprintf("App Service %s is not running in the namespace: %v, error: %v", springService, namespace, err))
		}
		return result
	}, longWaitTimeout, longPollingInterval).Should(BeTrue(), "Spring Boot Application Failed to Deploy: Services are not ready")

	t.Logs.Info("Spring Boot Application: check expected VirtualService is ready")
	Eventually(func() bool {
		result, err := pkg.DoesVirtualServiceExist(namespace, ingress)
		if err != nil {
			AbortSuite(fmt.Sprintf("App VirtualService %s is not running in the namespace: %v, error: %v", ingress, namespace, err))
		}
		return result
	}, shortWaitTimeout, longPollingInterval).Should(BeTrue(), "Spring Boot Application Failed to Deploy: VirtualService is not ready")

	t.Logs.Info("Spring Boot Application: check expected Secrets exist")
	Eventually(func() bool {
		result, err := pkg.DoesSecretExist(namespace, appCreds)
		if err != nil {
			AbortSuite(fmt.Sprintf("App Secret %s does not exist in the namespace: %v, error: %v", appCreds, namespace, err))
		}
		return result
	}, shortWaitTimeout, longPollingInterval).Should(BeTrue(), "Spring Boot Application Failed to Deploy: Secret does not exist")

	var err error
	// Get the host from the Istio gateway resource.
	start := time.Now()
	t.Logs.Info("Spring Boot Application: check expected Gateway is ready")
	Eventually(func() (string, error) {
		host, err = k8sutil.GetHostnameFromGateway(namespace, "")
		return host, err
	}, shortWaitTimeout, shortPollingInterval).Should(Not(BeEmpty()), "Spring Boot Application Failed to Deploy: Gateway is not ready")
	metrics.Emit(t.Metrics.With("get_host_name_elapsed_time", time.Since(start).Milliseconds()))

	beforeSuitePassed = true
})

var _ = BeforeSuite(beforeSuite)

var failed = false
var beforeSuitePassed = false

var _ = t.AfterEach(func() {
	failed = failed || CurrentSpecReport().Failed()
})

var afterSuite = t.AfterSuiteFunc(func() {
	if failed || !beforeSuitePassed {
		dump.ExecuteBugReport(namespace)
	}
	if !skipUndeploy {
		start := time.Now()
		pkg.UndeploySpringBootApplication(namespace)
		metrics.Emit(t.Metrics.With("undeployment_elapsed_time", time.Since(start).Milliseconds()))
	}
})

var _ = AfterSuite(afterSuite)

var _ = t.Describe("Spring Boot test", Label("f:app-lcm.oam",
	"f:app-lcm.spring-workload"), func() {

	var host = ""
	var err error
	// Get the host from the Istio gateway resource.
	// GIVEN the Istio gateway for the springboot namespace
	// WHEN GetHostnameFromGateway is called
	// THEN return the host name found in the gateway.
	t.BeforeEach(func() {
		Eventually(func() (string, error) {
			host, err = k8sutil.GetHostnameFromGateway(namespace, "")
			return host, err
		}, shortWaitTimeout, shortPollingInterval).Should(Not(BeEmpty()))
	})

	// Verify Spring Boot application is working
	// GIVEN springboot app is deployed
	// WHEN the component and appconfig with ingress trait are created
	// THEN the application endpoint must be accessible
	t.It("Verify welcome page of Spring Boot application is working.", Label("f:mesh.ingress"), func() {
		Eventually(func() (*pkg.HTTPResponse, error) {
			url := fmt.Sprintf("https://%s/", host)
			return pkg.GetWebPage(url, host)
		}, longWaitTimeout, longPollingInterval).Should(And(pkg.HasStatus(http.StatusOK), pkg.BodyContains("Greetings from Verrazzano Enterprise Container Platform")))
	})

	t.It("Verify Verrazzano facts endpoint is working.", Label("f:mesh.ingress"), func() {
		Eventually(func() (*pkg.HTTPResponse, error) {
			url := fmt.Sprintf("https://%s/facts", host)
			return pkg.GetWebPage(url, host)
		}, longWaitTimeout, longPollingInterval).Should(And(pkg.HasStatus(http.StatusOK), pkg.BodyNotEmpty()))
	})

	t.Context("for Logging.", Label("f:observability.logging.es"), FlakeAttempts(5), func() {
		var indexName string
		Eventually(func() error {
			indexName, err = pkg.GetOpenSearchAppIndex(namespace)
			return err
		}, shortWaitTimeout, shortPollingInterval).Should(BeNil(), "Expected to get OpenSearch App Index")

		t.It("Verify Opensearch index exists", func() {
			Eventually(func() bool {
				return pkg.LogIndexFound(indexName)
			}, longWaitTimeout, longPollingInterval).Should(BeTrue(), "Expected to find Opensearch index for Spring Boot application.")
		})
		t.It("Verify recent Opensearch log record exists", func() {
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

	t.Context("for metrics.", Label("f:observability.monitoring.prom"), func() {
		t.It("Retrieve Prometheus scraped metrics for App Component", func() {
			Eventually(func() bool {
				return pkg.MetricsExist("http_server_requests_seconds_count", "app_oam_dev_name", "springboot-appconf")
			}, longWaitTimeout, longPollingInterval).Should(BeTrue(), "Expected to find Prometheus scraped metrics for App Component.")
		})
		t.It("Retrieve Prometheus scraped metrics for App Config", func() {
			Eventually(func() bool {
				return pkg.MetricsExist("tomcat_sessions_created_sessions_total", "app_oam_dev_component", "springboot-component")
			}, longWaitTimeout, longPollingInterval).Should(BeTrue(), "Expected to find Prometheus scraped metrics for App Config.")
		})
	})
})
