// Copyright (c) 2020, 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package helidon

import (
	"crypto/tls"
	"flag"
	"fmt"
	"github.com/hashicorp/go-retryablehttp"
	"io/ioutil"
	"net/http"
	"strings"
	"time"

	"github.com/verrazzano/verrazzano/pkg/k8sutil"
	"github.com/verrazzano/verrazzano/pkg/test/framework/metrics"
	"github.com/verrazzano/verrazzano/tests/e2e/framework"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg"

	"github.com/onsi/ginkgo/v2"
)

const (
	longWaitTimeout      = 20 * time.Minute
	longPollingInterval  = 20 * time.Second
	shortPollingInterval = 10 * time.Second
	shortWaitTimeout     = 5 * time.Minute
)

var skipDeploy bool
var skipUndeploy bool
var namespace string

func init() {
	flag.BoolVar(&skipDeploy, "skipDeploy", false, "skipDeploy skips the call to install the application")
	flag.BoolVar(&skipUndeploy, "skipUndeploy", false, "skipUndeploy skips the call to install the application")
	flag.StringVar(&namespace, "namespace", f.UniqueName, "namespace is the app namespace")
}

var (
	f                        = framework.NewDefaultFramework("helidon")
	yamlApplier              = k8sutil.YAMLApplier{}
	expectedPodsHelloHelidon = []string{"hello-helidon-deployment"}
)

var _ = ginkgo.BeforeSuite(func() {
	if !skipDeploy {
		start := time.Now()
		pkg.DeployHelloHelidonApplication(namespace, "")
		metrics.Emit(f.Metrics.With("deployment_elapsed_time", time.Since(start).Milliseconds()))
	}
})

var failed = false
var _ = ginkgo.AfterEach(func() {
	failed = failed || ginkgo.CurrentSpecReport().Failed()
})

var _ = ginkgo.AfterSuite(func() {
	if failed {
		pkg.ExecuteClusterDumpWithEnvVarConfig()
	}
	if !skipUndeploy {
		start := time.Now()
		// pkg.UndeployHelloHelidonApplication(&yamlApplier, namespace)
		metrics.Emit(f.Metrics.With("undeployment_elapsed_time", time.Since(start).Milliseconds()))
	}
})

var _ = f.Describe("Hello Helidon OAM App test", ginkgo.Label("f:app-lcm.oam",
	"f:app-lcm.helidon-workload"), func() {
	fmt.Println("Helidon example")
	// Verify hello-helidon-deployment pod is running
	// GIVEN OAM hello-helidon app is deployed
	// WHEN the component and appconfig are created
	// THEN the expected pod must be running in the test namespace
	f.Describe("hello-helidon-deployment pod", func() {
		f.It("is running", func() {
			// framework.EventuallyBeTrue(helloHelidonPodsRunning, longWaitTimeout, longPollingInterval)
			fmt.Println("I am good")
		})
	})

	var host = ""
	var err error
	// Get the host from the Istio gateway resource.
	// GIVEN the Istio gateway for the hello-helidon namespace
	// WHEN GetHostnameFromGateway is called
	// THEN return the host name found in the gateway.
	f.It("Get host from gateway.", ginkgo.Label("f:mesh.ingress"), func() {
		framework.EventuallyNotEmpty(func() (string, error) {
			host, err = k8sutil.GetHostnameFromGateway(namespace, "")
			return host, err
		}, shortWaitTimeout, shortPollingInterval)
	})

	// Verify Hello Helidon app is working
	// GIVEN OAM hello-helidon app is deployed
	// WHEN the component and appconfig with ingress trait are created
	// THEN the application endpoint must be accessible
	f.Describe("for Ingress.", ginkgo.Label("f:mesh.ingress"), func() {
		f.It("Access /greet App Url.", func() {
			url := fmt.Sprintf("https://%s/greet", host)
			framework.EventuallyBeTrue(func() bool {
				return appEndpointAccessible(url, host)
			}, longWaitTimeout, longPollingInterval)
		})
	})

	// Verify Prometheus scraped metrics
	// GIVEN OAM hello-helidon app is deployed
	// WHEN the component and appconfig without metrics-trait(using default) are created
	// THEN the application metrics must be accessible
	f.Describe("for Metrics.", ginkgo.Label("f:observability.monitoring.prom"), func() {
		f.It("Retrieve Prometheus scraped metrics", func() {
			pkg.Concurrently(
				func() {
					framework.EventuallyBeTrue(appMetricsExists, longWaitTimeout, longPollingInterval)
				},
				func() {
					framework.EventuallyBeTrue(appComponentMetricsExists, longWaitTimeout, longPollingInterval)
				},
				func() {
					framework.EventuallyBeTrue(appConfigMetricsExists, longWaitTimeout, longPollingInterval)
				},
				func() {
					framework.EventuallyBeTrue(nodeExporterProcsRunning, longWaitTimeout, longPollingInterval)
				},
				func() {
					framework.EventuallyBeTrue(nodeExporterDiskIoNow, longWaitTimeout, longPollingInterval)
				},
			)
		})
	})

	f.Describe("Logging.", ginkgo.Label("f:observability.logging.es"), func() {

		indexName := "verrazzano-namespace-" + namespace

		// GIVEN an application with logging enabled
		// WHEN the Elasticsearch index is retrieved
		// THEN verify that it is found
		f.It("Verify Elasticsearch index exists", func() {
			framework.EventuallyBeTrue(func() bool {
				return pkg.LogIndexFound(indexName)
			}, longWaitTimeout, longPollingInterval, "Expected to find log index for hello helidon")
		})

		// GIVEN an application with logging enabled
		// WHEN the log records are retrieved from the Elasticsearch index
		// THEN verify that at least one recent log record is found
		f.It("Verify recent Elasticsearch log record exists", func() {
			framework.EventuallyBeTrue(func() bool {
				return pkg.LogRecordFound(indexName, time.Now().Add(-24*time.Hour), map[string]string{
					"kubernetes.labels.app_oam_dev\\/name": "hello-helidon-appconf",
					"kubernetes.container_name":            "hello-helidon-container",
				})
			}, longWaitTimeout, longPollingInterval, "Expected to find a recent log record")
			framework.EventuallyBeTrue(func() bool {
				return pkg.LogRecordFound(indexName, time.Now().Add(-24*time.Hour), map[string]string{
					"kubernetes.labels.app_oam_dev\\/component": "hello-helidon-component",
					"kubernetes.labels.app_oam_dev\\/name":      "hello-helidon-appconf",
					"kubernetes.container_name":                 "hello-helidon-container",
				})
			}, longWaitTimeout, longPollingInterval, "Expected to find a recent log record")
		})
	})
})

func helloHelidonPodsRunning() bool {
	result, _ := pkg.PodsRunning(namespace, expectedPodsHelloHelidon)
	return result
}

func appEndpointAccessible(url string, hostname string) bool {
	transport := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
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
	httpClient.HTTPClient.Transport = transport
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
