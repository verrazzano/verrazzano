// Copyright (c) 2020, 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package oam

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/onsi/ginkgo"
	"github.com/onsi/gomega"
	"github.com/verrazzano/verrazzano/tests/e2e/util"
)

var (
	env                      VerrazzanoEnvironment
	testConfig               VerrazzanoTestConfig
	expectedPodsHelloHelidon = []string{"hello-helidon-workload"}
	prom                     *Prometheus
	waitTimeout              = 10 * time.Minute
	pollingInterval          = 30 * time.Second
)

const (
	testNamespace      = "oam-springboot"
)

var _ = BeforeSuite(func() {
	testConfig = GetTestConfig()
	env = NewVerrazzanoEnvironmentFromConfig(testConfig)
	prom = env.GetProm("system")
})

var _ = Describe("Verify Springboot OAM App.", func() {
	// Verify springboot-workload pod is running
	// GIVEN OAM springboot app is deployed
	// WHEN the component and appconfig are created
	// THEN the expected pod must be running in the test namespace
	Describe("Verify springboot-workload pod is running.", func() {
		It("and waiting for expected pods must be running", func() {
			Eventually(podsRunningInVerrazzanoApplication, waitTimeout, pollingInterval).Should(BeTrue())
		})
	})

	// Verify springboot app is working
	// GIVEN OAM springboot app is deployed
	// WHEN the component and appconfig with ingress trait are created
	// THEN the application endpoint must be accessible
	Describe("Verify Springboot app is working.", func() {
		It("Access / App Url.", func() {
			ingress := env.GetCluster1().Ingress()
			url := fmt.Sprintf("http://%s/", ingress)
			isEndpointAccessible := func() bool {
				return appEndpointAccessible(url)
			}
			Eventually(isEndpointAccessible, 15*time.Second, 1*time.Second).Should(BeTrue())
		})
	})

	// Verify Prometheus scraped metrics
	// GIVEN OAM hello-helidon app is deployed
	// WHEN the component and appconfig without metrics-trait(using default) are created
	// THEN the application metrics must be accessible
	Describe("Verify Prometheus scraped metrics", func() {
		It("Retrieve Prometheus scraped metrics", func() {
			Concurrently(
				func() {
					Eventually(appMetricsExists, waitTimeout, pollingInterval).Should(BeTrue())
				},
				func() {
					Eventually(appComponentMetricsExists, waitTimeout, pollingInterval).Should(BeTrue())
				},
				func() {
					Eventually(appConfigMetricsExists, waitTimeout, pollingInterval).Should(BeTrue())
				},
			)
		})
	})
})

func podsRunningInVerrazzanoApplication() bool {
	return env.GetCluster1().Namespace(testNamespace).
		PodsRunning(expectedPodsHelloHelidon)
}

func appEndpointAccessible(url string) bool {
	status, webpage := GetWebPage(url, "oam-springboot.example.com")
	return Expect(status).To(Equal(http.StatusOK), fmt.Sprintf("GET %v returns status %v expected 200.", url, status)) &&
		Expect(strings.Contains(webpage, "Hello World")).To(Equal(true), fmt.Sprintf("Webpage is NOT Hello World %v", webpage))
}

func findMetric(metrics []interface{}, key, value string) bool {
	for _, metric := range metrics {
		if Jq(metric, "metric", key) == value {
			return true
		}
	}
	return false
}

func metricsExist(metricsName, key, value string) bool {
	metrics := prom.Metrics(metricsName)
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
