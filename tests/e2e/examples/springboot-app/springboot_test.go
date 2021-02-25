// Copyright (c) 2020, 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package springboot_test

import (
	"fmt"
	"time"

	"github.com/verrazzano/verrazzano/tests/e2e/pkg"

	"github.com/onsi/ginkgo"
	"github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/api/errors"
)

const testNamespace string = "springboot"
const hostHeaderValue string = "springboot.example.com"

var expectedPodsSpringBootApp = []string{"springboot-workload"}
var waitTimeout = 10 * time.Minute
var pollingInterval = 30 * time.Second
var shortPollingInterval = 10 * time.Second
var shortWaitTimeout = 5 * time.Minute
var longWaitTimeout      = 10 * time.Minute
var longPollingInterval  = 20 * time.Second

var _ = ginkgo.BeforeSuite(func() {
	deploySpringBootApplication()
})

var _ = ginkgo.AfterSuite(func() {
	undeploySpringBootApplication()
})

func deploySpringBootApplication() {
	pkg.Log(pkg.Info, "Deploy Spring Boot Application")

	pkg.Log(pkg.Info, "Create namespace")
	if _, err := pkg.CreateNamespace(testNamespace, map[string]string{"verrazzano-managed": "true"}); err != nil {
		ginkgo.Fail(fmt.Sprintf("Failed to create namespace: %v", err))
	}

	pkg.Log(pkg.Info, "Create logging scope resource")
	if err := pkg.CreateOrUpdateResourceFromFile("examples/springboot-app/springboot-comp.yaml"); err != nil {
		ginkgo.Fail(fmt.Sprintf("Failed to create Spring Boot component resources: %v", err))
	}
	pkg.Log(pkg.Info, "Create component resources")
	if err := pkg.CreateOrUpdateResourceFromFile("examples/springboot-app/springboot-app.yaml"); err != nil {
		ginkgo.Fail(fmt.Sprintf("Failed to create Spring Boot application resources: %v", err))
	}
}

func undeploySpringBootApplication() {
	pkg.Log(pkg.Info, "Undeploy Spring Boot Application")
	pkg.Log(pkg.Info, "Delete application")
	if err := pkg.DeleteResourceFromFile("examples/springboot-app/springboot-app.yaml"); err != nil {
		pkg.Log(pkg.Error, fmt.Sprintf("Failed to delete the application: %v", err))
	}
	pkg.Log(pkg.Info, "Delete components")
	if err := pkg.DeleteResourceFromFile("examples/springboot-app/springboot-comp.yaml"); err != nil {
		pkg.Log(pkg.Error, fmt.Sprintf("Failed to delete the component: %v", err))
	}
	pkg.Log(pkg.Info, "Delete namespace")
	if err := pkg.DeleteNamespace(testNamespace); err != nil {
		pkg.Log(pkg.Error, fmt.Sprintf("Failed to delete the namespace: %v", err))
	}
	gomega.Eventually(func() bool {
		ns, err := pkg.GetNamespace(testNamespace)
		return ns == nil && err != nil && errors.IsNotFound(err)
	}, 3*time.Minute, 15*time.Second).Should(gomega.BeFalse())
}

var _ = ginkgo.Describe("Verify Spring Boot Application", func() {
	// Verify springboot-workload pod is running
	// GIVEN springboot app is deployed
	// WHEN the component and appconfig are created
	// THEN the expected pod must be running in the test namespace
	ginkgo.Context("Deployment.", func() {
		ginkgo.It("and waiting for expected pods must be running", func() {
			gomega.Eventually(func() bool {
				return pkg.PodsRunning(testNamespace, expectedPodsSpringBootApp)
			}, waitTimeout, pollingInterval).Should(gomega.BeTrue())
		})
	})


	// Verify Spring Boot application is working
	// GIVEN springboot app is deployed
	// WHEN the component and appconfig with ingress trait are created
	// THEN the application endpoint must be accessible
	ginkgo.It("Verify welcome page of Spring Boot application is working.", func() {
		gomega.Eventually(func() bool {
			ingress := pkg.Ingress()
			pkg.Log(pkg.Info, fmt.Sprintf("Ingress: %s", ingress))
			url := fmt.Sprintf("http://%s/", ingress)
			host := pkg.GetHostnameFromGateway(testNamespace, "")
			status, content := pkg.GetWebPageWithCABundle(url, host)
			return gomega.Expect(status).To(gomega.Equal(200)) &&
				gomega.Expect(content).To(gomega.ContainSubstring("Greetings from Verrazzano Enterprise Container Platform"))
		}, shortWaitTimeout, shortPollingInterval).Should(gomega.BeTrue())
	})

	ginkgo.It("Verify Verrazzano facts endpoint is working.", func() {
		gomega.Eventually(func() bool {
			ingress := pkg.Ingress()
			url := fmt.Sprintf("http://%s/facts", ingress)
			host := pkg.GetHostnameFromGateway(testNamespace, "")
			status, content := pkg.GetWebPageWithCABundle(url, host)
			gomega.Expect(len(content) > 0, fmt.Sprintf("An empty string returned from /facts endpoint %v", content))
			return gomega.Expect(status).To(gomega.Equal(200))
		}, shortWaitTimeout, shortPollingInterval).Should(gomega.BeTrue())
	})

	ginkgo.Context("Logging.", func() {
		indexName := "springboot-springboot-appconf-springboot-component"
		ginkgo.It("Verify Elasticsearch index exists", func() {
			gomega.Eventually(func() bool {
				return logIndexFound(indexName)
			}, longWaitTimeout, longPollingInterval).Should(gomega.BeTrue(), "Expected to find log index for Spring Boot application")
		})

		ginkgo.It("Verify recent Elasticsearch log record exists", func() {
			gomega.Eventually(func() bool {
				return logRecordFound(indexName)
			}, longWaitTimeout, longPollingInterval).Should(gomega.BeTrue(), "Expected to find a recent log record")
		})
	})

	ginkgo.Context("Verify Prometheus scraped metrics.", func() {
		ginkgo.It("Retrieve Prometheus scraped metrics for App Component Metrics", func() {
			gomega.Eventually(func() bool {
				return appComponentMetricsExists()
			}, longWaitTimeout, longPollingInterval).Should(gomega.BeTrue(), "Expected to find log index for Spring Boot application")
		})
		ginkgo.It("Retrieve Prometheus scraped metrics for App Config Metrics", func() {
			gomega.Eventually(func() bool {
				return appConfigMetricsExists()
			}, longWaitTimeout, longPollingInterval).Should(gomega.BeTrue(), "Expected to find log index for Spring Boot application")
		})
	})

})

// appComponentMetricsExists checks whether component related metrics are available
func appComponentMetricsExists() bool {
	return pkg.MetricsExist("http_server_requests_seconds_count", "app_oam_dev_name", "springboot-appconf")
}

// appConfigMetricsExists checks whether config metrics are available
func appConfigMetricsExists() bool {
	return pkg.MetricsExist("tomcat_sessions_created_sessions_total", "app_oam_dev_component", "springboot-component")
}

// logIndexFound confirms a named index can be found.
func logIndexFound(indexName string) bool {
	for _, name := range pkg.ListSystemElasticSearchIndices() {
		if name == indexName {
			return true
		}
	}
	pkg.Log(pkg.Error, fmt.Sprintf("Expected to find log index %s", indexName))
	return false
}

func logRecordFound(indexName string) bool {
	searchResult := pkg.QuerySystemElasticSearch(indexName, map[string]string{})
	hits := pkg.Jq(searchResult, "hits", "hits")
	if hits == nil {
		pkg.Log(pkg.Info, "Expected to find hits in log record query results")
		return false
	}
	return true
}
