// Copyright (c) 2020, 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package hello_helidon

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/onsi/ginkgo"
	"github.com/onsi/gomega"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg"
)

var _ = ginkgo.BeforeSuite(func() {
	if _, err := pkg.CreateNamespace("oam-hello-helidon", map[string]string{"verrazzano-managed": "true"}); err != nil {
		ginkgo.Fail(fmt.Sprintf("Failed to create namespace: %v", err))
	}

	if err := pkg.CreateOrUpdateResourceFromFile("examples/hello-helidon/hello-helidon-comp.yaml"); err != nil {
		ginkgo.Fail(fmt.Sprintf("Failed to create hello-helidon component resources: %v", err))
	}
	if err := pkg.CreateOrUpdateResourceFromFile("examples/hello-helidon/hello-helidon-app.yaml"); err != nil {
		ginkgo.Fail(fmt.Sprintf("Failed to create hello-helidon application resource: %v", err))
	}

})

var _ = ginkgo.AfterSuite(func() {
	// undeploy the application here
	err := pkg.DeleteResourceFromFile("examples/hello-helidon/hello-helidon-app.yaml")
	if err != nil {
		ginkgo.Fail(fmt.Sprintf("Could not delete hello-helidon application resource: %v\n", err.Error()))
	}
	err = pkg.DeleteResourceFromFile("examples/hello-helidon/hello-helidon-comp.yaml")
	if err != nil {
		ginkgo.Fail(fmt.Sprintf("Could not delete hello-helidon component resource: %v\n", err.Error()))
	}
	err = pkg.DeleteNamespace("oam-hello-helidon")
	if err != nil {
		ginkgo.Fail(fmt.Sprintf("Could not delete oam-hello-helidon namespace: %v\n", err.Error()))
	}
})

var (
	expectedPodsHelloHelidon = []string{"hello-helidon-workload"}
	waitTimeout              = 10 * time.Minute
	pollingInterval          = 30 * time.Second
)

const (
	testNamespace      = "oam-hello-helidon"
	helloHostHeader    = "hello-helidon.example.com"
	istioNamespace     = "istio-system"
	ingressServiceName = "istio-ingressgateway"
)

var _ = ginkgo.Describe("Verify Hello Helidon OAM App.", func() {
	// Verify hello-helidon-workload pod is running
	// GIVEN OAM hello-helidon app is deployed
	// WHEN the component and appconfig are created
	// THEN the expected pod must be running in the test namespace
	ginkgo.Describe("Verify hello-helidon-workload pod is running.", func() {
		ginkgo.It("and waiting for expected pods must be running", func() {
			gomega.Eventually(helloHelidonPodsRunning, waitTimeout, pollingInterval).Should(gomega.BeTrue())
		})
	})

	// Verify Hello Helidon app is working
	// GIVEN OAM hello-helidon app is deployed
	// WHEN the component and appconfig with ingress trait are created
	// THEN the application endpoint must be accessible
	ginkgo.Describe("Verify Hello Helidon app is working.", func() {
		ginkgo.It("Access /greet App Url.", func() {
			ingress := pkg.Ingress()
			url := fmt.Sprintf("http://%s/greet", ingress)
			isEndpointAccessible := func() bool {
				return appEndpointAccessible(url)
			}
			gomega.Eventually(isEndpointAccessible, 15*time.Second, 1*time.Second).Should(gomega.BeTrue())
		})
	})

	// Verify Prometheus scraped metrics
	// GIVEN OAM hello-helidon app is deployed
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
})

func helloHelidonPodsRunning() bool {
	return pkg.PodsRunning(testNamespace, expectedPodsHelloHelidon)
}

func appEndpointAccessible(url string) bool {
	status, webpage := pkg.GetWebPageWithBasicAuth(url, helloHostHeader, "", "")
	gomega.Expect(status).To(gomega.Equal(http.StatusOK), fmt.Sprintf("GET %v returns status %v expected 200.", url, status))
	gomega.Expect(strings.Contains(webpage, "Hello World")).To(gomega.Equal(true), fmt.Sprintf("Webpage is NOT Hello World %v", webpage))
	return true
}

// findMetric parses a Prometheus response to find a specified metric value
func findMetric(metrics []interface{}, key, value string) bool {
	for _, metric := range metrics {
		if pkg.Jq(metric, "metric", key) == value {
			return true
		}
	}
	return false
}

// metricsExist validates the availability of a specified metric
func metricsExist(metricsName, key, value string) bool {
	metrics := pkg.JTq(pkg.QueryMetric(metricsName), "data", "result").([]interface{})
	if metrics != nil {
		return findMetric(metrics, key, value)
	} else {
		return false
	}
}

func appMetricsExists() bool {
	return metricsExist("base_jvm_uptime_seconds", "app", "hello-helidon")
}

func appComponentMetricsExists() bool {
	return metricsExist("vendor_requests_count_total", "app_oam_dev_name", "hello-helidon-appconf")
}

func appConfigMetricsExists() bool {
	return metricsExist("vendor_requests_count_total", "app_oam_dev_component", "hello-helidon-component")
}
