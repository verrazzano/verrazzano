// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package helidonconfig

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/onsi/ginkgo"
	"github.com/onsi/gomega"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg"
)

const (
	longWaitTimeout      = 10 * time.Minute
	longPollingInterval  = 20 * time.Second
	shortPollingInterval = 10 * time.Second
	shortWaitTimeout     = 5 * time.Minute
)

var _ = ginkgo.BeforeSuite(func() {
	nsLabels := map[string]string{
		"verrazzano-managed": "true",
		"istio-injection":    "enabled"}
	if _, err := pkg.CreateNamespace("helidon-config", nsLabels); err != nil {
		ginkgo.Fail(fmt.Sprintf("Failed to create namespace: %v", err))
	}

	if err := pkg.CreateOrUpdateResourceFromFile("examples/helidon-config/helidon-config-comp.yaml"); err != nil {
		ginkgo.Fail(fmt.Sprintf("Failed to create helidon-config component resources: %v", err))
	}
	if err := pkg.CreateOrUpdateResourceFromFile("examples/helidon-config/helidon-config-app.yaml"); err != nil {
		ginkgo.Fail(fmt.Sprintf("Failed to create helidon-config application resource: %v", err))
	}

})

var _ = ginkgo.AfterSuite(func() {
	// undeploy the application here
	err := pkg.DeleteResourceFromFile("examples/helidon-config/helidon-config-app.yaml")
	if err != nil {
		ginkgo.Fail(fmt.Sprintf("Could not delete helidon-config application resource: %v\n", err.Error()))
	}
	err = pkg.DeleteResourceFromFile("examples/helidon-config/helidon-config-comp.yaml")
	if err != nil {
		ginkgo.Fail(fmt.Sprintf("Could not delete helidon-config component resource: %v\n", err.Error()))
	}
	err = pkg.DeleteNamespace("helidon-config")
	if err != nil {
		ginkgo.Fail(fmt.Sprintf("Could not delete helidon-config namespace: %v\n", err.Error()))
	}
})

var (
	expectedPodsHelidonConfig = []string{"helidon-config-deployment"}
	waitTimeout              = 10 * time.Minute
	pollingInterval          = 30 * time.Second
)

const (
	testNamespace      = "helidon-config"
	istioNamespace     = "istio-system"
	ingressServiceName = "istio-ingressgateway"
)

var _ = ginkgo.Describe("Verify Helidon Config OAM App.", func() {
	// Verify helidon-config-deployment pod is running
	// GIVEN OAM helidon-config app is deployed
	// WHEN the component and appconfig are created
	// THEN the expected pod must be running in the test namespace
	ginkgo.Describe("Verify helidon-config-deployment pod is running.", func() {
		ginkgo.It("and waiting for expected pods must be running", func() {
			gomega.Eventually(helidonConfigPodsRunning, waitTimeout, pollingInterval).Should(gomega.BeTrue())
		})
	})

	var host = ""
	// Get the host from the Istio gateway resource.
	// GIVEN the Istio gateway for the helidon-config namespace
	// WHEN GetHostnameFromGateway is called
	// THEN return the host name found in the gateway.
	ginkgo.It("Get host from gateway.", func() {
		gomega.Eventually(func() string {
			host = pkg.GetHostnameFromGateway(testNamespace, "")
			return host
		}, shortWaitTimeout, shortPollingInterval).Should(gomega.Not(gomega.BeEmpty()))
	})

	// Verify Helidon Config app is working
	// GIVEN OAM helidon-config app is deployed
	// WHEN the component and appconfig with ingress trait are created
	// THEN the application endpoint must be accessible
	ginkgo.Describe("Verify Helidon Config app is working.", func() {
		ginkgo.It("Access /greet App Url.", func() {
			url := fmt.Sprintf("https://%s/greet", host)
			isEndpointAccessible := func() bool {
				return appEndpointAccessible(url, host)
			}
			gomega.Eventually(isEndpointAccessible, 15*time.Second, 1*time.Second).Should(gomega.BeTrue())
		})
	})

	// Verify Prometheus scraped metrics
	// GIVEN OAM helidon-config app is deployed
	// WHEN the component and appconfig without metrics-trait(using default) are created
	// THEN the application metrics must be accessible
	ginkgo.Describe("Verify Prometheus scraped metrics", func() {
		ginkgo.It("Retrieve Prometheus scraped metrics", func() {
			pkg.Concurrently(
				func() {
					gomega.Eventually(appMetricsExists, waitTimeout, pollingInterval).Should(gomega.BeTrue())
				},
				func() {
					gomega.Eventually(appComponentMetricsExists, waitTimeout, pollingInterval).Should(gomega.BeTrue())
				},
				func() {
					gomega.Eventually(appConfigMetricsExists, waitTimeout, pollingInterval).Should(gomega.BeTrue())
				},
			)
		})
	})

	ginkgo.Context("Logging.", func() {
		indexName := "helidon-config-helidon-config-appconf-helidon-config-component-helidon-config-container"

		// GIVEN an application with logging enabled via a logging scope
		// WHEN the Elasticsearch index is retrieved
		// THEN verify that it is found
		ginkgo.It("Verify Elasticsearch index exists", func() {
			gomega.Eventually(func() bool {
				return pkg.LogIndexFound(indexName)
			}, longWaitTimeout, longPollingInterval).Should(gomega.BeTrue(), "Expected to find log index for helidon config")
		})

		// GIVEN an application with logging enabled via a logging scope
		// WHEN the log records are retrieved from the Elasticsearch index
		// THEN verify that at least one recent log record is found
		ginkgo.It("Verify recent Elasticsearch log record exists", func() {
			gomega.Eventually(func() bool {
				return pkg.LogRecordFound(indexName, time.Now().Add(-24*time.Hour), map[string]string{
					"oam.applicationconfiguration.namespace": "helidon-config",
					"oam.applicationconfiguration.name":      "helidon-config-appconf"})
			}, longWaitTimeout, longPollingInterval).Should(gomega.BeTrue(), "Expected to find a recent log record")
		})
	})
})

func helidonConfigPodsRunning() bool {
	return pkg.PodsRunning(testNamespace, expectedPodsHelidonConfig)
}

func appEndpointAccessible(url string, hostname string) bool {
	status, webpage := pkg.GetWebPageWithBasicAuth(url, hostname, "", "")
	gomega.Expect(status).To(gomega.Equal(http.StatusOK), fmt.Sprintf("GET %v returns status %v expected 200.", url, status))
	gomega.Expect(strings.Contains(webpage, "Hello World")).To(gomega.Equal(true), fmt.Sprintf("Webpage is NOT Hello World %v", webpage))
	return true
}

func appMetricsExists() bool {
	return pkg.MetricsExist("base_jvm_uptime_seconds", "app", "helidon-config")
}

func appComponentMetricsExists() bool {
	return pkg.MetricsExist("vendor_requests_count_total", "app_oam_dev_name", "helidon-config-appconf")
}

func appConfigMetricsExists() bool {
	return pkg.MetricsExist("vendor_requests_count_total", "app_oam_dev_component", "helidon-config-component")
}
