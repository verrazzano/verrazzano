// Copyright (c) 2020, 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package helidon

import (
	"fmt"
	"github.com/verrazzano/verrazzano/pkg/test/framework"
	"github.com/verrazzano/verrazzano/pkg/test/framework/metrics"
	"io/ioutil"
	"net/http"
	"strings"
	"time"

	"github.com/hashicorp/go-retryablehttp"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/verrazzano/verrazzano/pkg/k8sutil"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg"
	v1 "k8s.io/api/core/v1"
)

const (
	longWaitTimeout      = 10 * time.Minute
	longPollingInterval  = 20 * time.Second
	shortPollingInterval = 10 * time.Second
	shortWaitTimeout     = 5 * time.Minute
)

var metricsLogger, _ = metrics.NewMetricsLogger("metrics")

var _ = framework.VzBeforeSuite(func() {
	if !skipDeploy {
		start := time.Now()
		Eventually(func() (*v1.Namespace, error) {
			nsLabels := map[string]string{
				"verrazzano-managed": "true",
				"istio-injection":    "enabled"}
			return pkg.CreateNamespace("hello-helidon", nsLabels)
		}, shortWaitTimeout, shortPollingInterval).ShouldNot(BeNil())

		Eventually(func() error {
			return pkg.CreateOrUpdateResourceFromFile("examples/hello-helidon/hello-helidon-comp.yaml")
		}, shortWaitTimeout, shortPollingInterval).ShouldNot(HaveOccurred())

		Eventually(func() error {
			return pkg.CreateOrUpdateResourceFromFile("examples/hello-helidon/hello-helidon-app.yaml")
		}, shortWaitTimeout, shortPollingInterval).ShouldNot(HaveOccurred(), "Failed to create hello-helidon application resource")
		metrics.Emit(metricsLogger.With("hello_helidon_deployment_duration_time", time.Since(start)))
	}
})

var _ = framework.VzAfterSuite(func() {
	if !skipUndeploy {
		start := time.Now()
		// undeploy the application here
		Eventually(func() error {
			return pkg.DeleteResourceFromFile("examples/hello-helidon/hello-helidon-app.yaml")
		}, shortWaitTimeout, shortPollingInterval).ShouldNot(HaveOccurred())

		Eventually(func() error {
			return pkg.DeleteResourceFromFile("examples/hello-helidon/hello-helidon-comp.yaml")
		}, shortWaitTimeout, shortPollingInterval).ShouldNot(HaveOccurred())

		Eventually(func() error {
			return pkg.DeleteNamespace("hello-helidon")
		}, shortWaitTimeout, shortPollingInterval).ShouldNot(HaveOccurred())
		metrics.Emit(metricsLogger.With("hello_helidon_undeployment_duration_time", time.Since(start)))
	}
})

var (
	expectedPodsHelloHelidon = []string{"hello-helidon-deployment"}
	waitTimeout              = 10 * time.Minute
	pollingInterval          = 30 * time.Second
)

const (
	testNamespace      = "hello-helidon"
	istioNamespace     = "istio-system"
	ingressServiceName = "istio-ingressgateway"
)

var _ = framework.VzDescribe("Verify Hello Helidon OAM App.", func() {
	// Verify hello-helidon-deployment pod is running
	// GIVEN OAM hello-helidon app is deployed
	// WHEN the component and appconfig are created
	// THEN the expected pod must be running in the test namespace
	framework.VzDescribe("Verify hello-helidon-deployment pod is running.", func() {
		framework.VzIt("and waiting for expected pods must be running", func() {
			start := time.Now()
			Eventually(helloHelidonPodsRunning, waitTimeout, pollingInterval).Should(BeTrue())
			metrics.Emit(metricsLogger.With("hello_helidon_pods_running_wait_time", time.Since(start)))
		})
	})

	var host = ""
	var err error
	// Get the host from the Istio gateway resource.
	// GIVEN the Istio gateway for the hello-helidon namespace
	// WHEN GetHostnameFromGateway is called
	// THEN return the host name found in the gateway.
	framework.VzIt("Get host from gateway.", func() {
		Eventually(func() (string, error) {
			host, err = k8sutil.GetHostnameFromGateway(testNamespace, "")
			return host, err
		}, shortWaitTimeout, shortPollingInterval).Should(Not(BeEmpty()))
	})

	// Verify Hello Helidon app is working
	// GIVEN OAM hello-helidon app is deployed
	// WHEN the component and appconfig with ingress trait are created
	// THEN the application endpoint must be accessible
	framework.VzDescribe("Verify Hello Helidon app is working.", func() {
		framework.VzIt("Access /greet App Url.", func() {
			start := time.Now()
			url := fmt.Sprintf("https://%s/greet", host)
			Eventually(func() bool {
				return appEndpointAccessible(url, host)
			}, longWaitTimeout, longPollingInterval).Should(BeTrue())
			metrics.Emit(metricsLogger.With("hello_helidon_web_app_ready_time", time.Since(start)))
		})
	})

	// Verify Prometheus scraped metrics
	// GIVEN OAM hello-helidon app is deployed
	// WHEN the component and appconfig without metrics-trait(using default) are created
	// THEN the application metrics must be accessible
	framework.VzDescribe("Verify Prometheus scraped metrics", func() {
		framework.VzIt("Retrieve Prometheus scraped metrics", func() {
			pkg.Concurrently(
				func() {
					Eventually(appMetricsExists, waitTimeout, pollingInterval).Should(BeTrue())
				},
				func() {
					Eventually(appComponentMetricsExists, waitTimeout, pollingInterval).Should(BeTrue())
				},
				func() {
					Eventually(appConfigMetricsExists, waitTimeout, pollingInterval).Should(BeTrue())
				},
				func() {
					Eventually(nodeExporterProcsRunning, waitTimeout, pollingInterval).Should(BeTrue())
				},
				func() {
					Eventually(nodeExporterDiskIoNow, waitTimeout, pollingInterval).Should(BeTrue())
				},
			)
		})
	})

	framework.VzContext("Logging.", func() {
		indexName := "verrazzano-namespace-hello-helidon"

		// GIVEN an application with logging enabled
		// WHEN the Elasticsearch index is retrieved
		// THEN verify that it is found
		framework.VzIt("Verify Elasticsearch index exists", func() {
			Eventually(func() bool {
				return pkg.LogIndexFound(indexName)
			}, longWaitTimeout, longPollingInterval).Should(BeTrue(), "Expected to find log index for hello helidon")
		})

		// GIVEN an application with logging enabled
		// WHEN the log records are retrieved from the Elasticsearch index
		// THEN verify that at least one recent log record is found
		framework.VzIt("Verify recent Elasticsearch log record exists", func() {
			Eventually(func() bool {
				return pkg.LogRecordFound(indexName, time.Now().Add(-24*time.Hour), map[string]string{
					"kubernetes.labels.app_oam_dev\\/name": "hello-helidon-appconf",
					"kubernetes.container_name":            "hello-helidon-container",
				})
			}, longWaitTimeout, longPollingInterval).Should(BeTrue(), "Expected to find a recent log record")
			Eventually(func() bool {
				return pkg.LogRecordFound(indexName, time.Now().Add(-24*time.Hour), map[string]string{
					"kubernetes.labels.app_oam_dev\\/component": "hello-helidon-component",
					"kubernetes.labels.app_oam_dev\\/name":      "hello-helidon-appconf",
					"kubernetes.container_name":                 "hello-helidon-container",
				})
			}, longWaitTimeout, longPollingInterval).Should(BeTrue(), "Expected to find a recent log record")
		})
	})
})

func helloHelidonPodsRunning() bool {
	return pkg.PodsRunning(testNamespace, expectedPodsHelloHelidon)
}

func appEndpointAccessible(url string, hostname string) bool {
	req, err := retryablehttp.NewRequest("GET", url, nil)
	if err != nil {
		pkg.Log(pkg.Error, fmt.Sprintf("Unexpected error=%v", err))
		return false
	}

	kubeconfigPath, err := k8sutil.GetKubeConfigLocation()
	if err != nil {
		pkg.Log(pkg.Error, fmt.Sprintf("Unexpected error=%v", err))
		return false
	}

	httpClient, err := pkg.GetVerrazzanoHTTPClient(kubeconfigPath)
	if err != nil {
		pkg.Log(pkg.Error, fmt.Sprintf("Unexpected error=%v", err))
		return false
	}
	req.Host = hostname
	resp, err := httpClient.Do(req)
	if err != nil {
		pkg.Log(pkg.Error, fmt.Sprintf("Unexpected error=%v", err))
		return false
	}
	bodyRaw, err := ioutil.ReadAll(resp.Body)
	resp.Body.Close()
	if err != nil {
		pkg.Log(pkg.Error, fmt.Sprintf("Unexpected error=%v", err))
		return false
	}
	if resp.StatusCode != http.StatusOK {
		pkg.Log(pkg.Error, fmt.Sprintf("Unexpected status code=%v", resp.StatusCode))
		return false
	}
	// HTTP Server headers should never be returned.
	for headerName, headerValues := range resp.Header {
		if strings.EqualFold(headerName, "Server") {
			pkg.Log(pkg.Error, fmt.Sprintf("Unexpected Server header=%v", headerValues))
			return false
		}
	}
	bodyStr := string(bodyRaw)
	if !strings.Contains(bodyStr, "Hello World") {
		pkg.Log(pkg.Error, fmt.Sprintf("Unexpected response body=%v", bodyStr))
		return false
	}
	return true
}

func appMetricsExists() bool {
	return pkg.MetricsExist("base_jvm_uptime_seconds", "app", "hello-helidon")
}

func appComponentMetricsExists() bool {
	return pkg.MetricsExist("vendor_requests_count_total", "app_oam_dev_name", "hello-helidon-appconf")
}

func appConfigMetricsExists() bool {
	return pkg.MetricsExist("vendor_requests_count_total", "app_oam_dev_component", "hello-helidon-component")
}

func nodeExporterProcsRunning() bool {
	return pkg.MetricsExist("node_procs_running", "job", "node-exporter")
}

func nodeExporterDiskIoNow() bool {
	return pkg.MetricsExist("node_disk_io_now", "job", "node-exporter")
}
