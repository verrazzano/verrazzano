// Copyright (c) 2020, 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package springboot

import (
	"fmt"
	"net/http"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
)

const testNamespace string = "springboot"

var expectedPodsSpringBootApp = []string{"springboot-workload"}
var waitTimeout = 10 * time.Minute
var pollingInterval = 30 * time.Second
var shortPollingInterval = 10 * time.Second
var shortWaitTimeout = 5 * time.Minute
var longWaitTimeout = 10 * time.Minute
var longPollingInterval = 20 * time.Second

var _ = BeforeSuite(func() {
	deploySpringBootApplication()
})

var failed = false
var _ = AfterEach(func() {
	failed = failed || CurrentGinkgoTestDescription().Failed
})

var _ = AfterSuite(func() {
	if failed {
		pkg.ExecuteClusterDumpWithEnvVarConfig()
	}
	undeploySpringBootApplication()
})

func deploySpringBootApplication() {
	pkg.Log(pkg.Info, "Deploy Spring Boot Application")

	pkg.Log(pkg.Info, "Create namespace")
	Eventually(func() (*v1.Namespace, error) {
		nsLabels := map[string]string{
			"verrazzano-managed": "true",
			"istio-injection":    "enabled"}
		return pkg.CreateNamespace(testNamespace, nsLabels)
	}, shortWaitTimeout, shortPollingInterval).ShouldNot(BeNil())

	pkg.Log(pkg.Info, "Create component resource")
	Eventually(func() error {
		return pkg.CreateOrUpdateResourceFromFile("examples/springboot-app/springboot-comp.yaml")
	}, shortWaitTimeout, shortPollingInterval).ShouldNot(HaveOccurred())

	pkg.Log(pkg.Info, "Create application resource")
	Eventually(func() error {
		return pkg.CreateOrUpdateResourceFromFile("examples/springboot-app/springboot-app.yaml")
	}, shortWaitTimeout, shortPollingInterval).ShouldNot(HaveOccurred(), "Failed to create Spring Boot application resource")
}

func undeploySpringBootApplication() {
	pkg.Log(pkg.Info, "Undeploy Spring Boot Application")
	pkg.Log(pkg.Info, "Delete application")
	Eventually(func() error {
		return pkg.DeleteResourceFromFile("examples/springboot-app/springboot-app.yaml")
	}, shortWaitTimeout, shortPollingInterval).ShouldNot(HaveOccurred())

	pkg.Log(pkg.Info, "Delete components")
	Eventually(func() error {
		return pkg.DeleteResourceFromFile("examples/springboot-app/springboot-comp.yaml")
	}, shortWaitTimeout, shortPollingInterval).ShouldNot(HaveOccurred())

	pkg.Log(pkg.Info, "Delete namespace")
	Eventually(func() error {
		return pkg.DeleteNamespace(testNamespace)
	}, shortWaitTimeout, shortPollingInterval).ShouldNot(HaveOccurred())

	Eventually(func() bool {
		_, err := pkg.GetNamespace(testNamespace)
		return err != nil && errors.IsNotFound(err)
	}, shortWaitTimeout, shortPollingInterval).Should(BeTrue())
}

var _ = Describe("Verify Spring Boot Application", func() {
	// Verify springboot-workload pod is running
	// GIVEN springboot app is deployed
	// WHEN the component and appconfig are created
	// THEN the expected pod must be running in the test namespace
	Context("Deployment.", func() {
		It("and waiting for expected pods must be running", func() {
			Eventually(func() bool {
				return pkg.PodsRunning(testNamespace, expectedPodsSpringBootApp)
			}, waitTimeout, pollingInterval).Should(BeTrue())
		})
	})

	var host = ""
	// Get the host from the Istio gateway resource.
	// GIVEN the Istio gateway for the springboot namespace
	// WHEN GetHostnameFromGateway is called
	// THEN return the host name found in the gateway.
	It("Get host from gateway.", func() {
		Eventually(func() string {
			host = pkg.GetHostnameFromGateway(testNamespace, "")
			return host
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
