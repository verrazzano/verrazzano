// Copyright (c) 2020, 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package springboot

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
var longWaitTimeout = 10 * time.Minute
var longPollingInterval = 20 * time.Second

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
			host := pkg.GetHostnameFromGateway(testNamespace, "")
			url := fmt.Sprintf("https://%s/", host)
			status, content := pkg.GetWebPageWithCABundle(url, host)
			return gomega.Expect(status).To(gomega.Equal(200)) &&
				gomega.Expect(content).To(gomega.ContainSubstring("Greetings from Verrazzano Enterprise Container Platform"))
		}, shortWaitTimeout, shortPollingInterval).Should(gomega.BeTrue())
	})

	ginkgo.It("Verify Verrazzano facts endpoint is working.", func() {
		gomega.Eventually(func() bool {
			host := pkg.GetHostnameFromGateway(testNamespace, "")
			url := fmt.Sprintf("https://%s/facts", host)
			status, content := pkg.GetWebPageWithCABundle(url, host)
			gomega.Expect(len(content) > 0, fmt.Sprintf("An empty string returned from /facts endpoint %v", content))
			return gomega.Expect(status).To(gomega.Equal(200))
		}, shortWaitTimeout, shortPollingInterval).Should(gomega.BeTrue())
	})

	ginkgo.Context("Logging.", func() {
		indexName := "springboot-springboot-appconf-springboot-component-springboot-container"
		ginkgo.It("Verify Elasticsearch index exists", func() {
			gomega.Eventually(func() bool {
				return pkg.LogIndexFound(indexName)
			}, longWaitTimeout, longPollingInterval).Should(gomega.BeTrue(), "Expected to find Elasticsearch index for Spring Boot application.")
		})

		ginkgo.It("Verify recent Elasticsearch log record exists", func() {
			gomega.Eventually(func() bool {
				return pkg.LogRecordFound(indexName, time.Now().Add(-24*time.Hour), map[string]string{
					"oam.component.name": "springboot-component"})
			}, longWaitTimeout, longPollingInterval).Should(gomega.BeTrue(), "Expected to find a recent log record.")
		})
	})

	ginkgo.Context("Verify Prometheus scraped metrics.", func() {
		ginkgo.It("Retrieve Prometheus scraped metrics for App Component", func() {
			gomega.Eventually(func() bool {
				return pkg.MetricsExist("http_server_requests_seconds_count", "app_oam_dev_name", "springboot-appconf")
			}, longWaitTimeout, longPollingInterval).Should(gomega.BeTrue(), "Expected to find Prometheus scraped metrics for App Component.")
		})
		ginkgo.It("Retrieve Prometheus scraped metrics for App Config", func() {
			gomega.Eventually(func() bool {
				return pkg.MetricsExist("tomcat_sessions_created_sessions_total", "app_oam_dev_component", "springboot-component")
			}, longWaitTimeout, longPollingInterval).Should(gomega.BeTrue(), "Expected to find Prometheus scraped metrics for App Config.")
		})
	})
})
