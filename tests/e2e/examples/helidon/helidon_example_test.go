// Copyright (c) 2020, 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package helidon

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
	"time"

	"github.com/hashicorp/go-retryablehttp"

	"github.com/verrazzano/verrazzano/pkg/k8sutil"
	"github.com/verrazzano/verrazzano/pkg/test/framework"
	"github.com/verrazzano/verrazzano/pkg/test/framework/metrics"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

const (
	longWaitTimeout          = 20 * time.Minute
	longPollingInterval      = 20 * time.Second
	shortPollingInterval     = 10 * time.Second
	shortWaitTimeout         = 5 * time.Minute
	imagePullWaitTimeout     = 40 * time.Minute
	imagePullPollingInterval = 30 * time.Second
	skipVerifications        = "Skip Verifications"
)

var (
	t                  = framework.NewTestFramework("helidon")
	generatedNamespace = pkg.GenerateNamespace("hello-helidon")
	//yamlApplier              = k8sutil.YAMLApplier{}
	expectedPodsHelloHelidon = []string{"hello-helidon-deployment"}
)

var _ = t.BeforeSuite(func() {
	if !skipDeploy {
		start := time.Now()
		pkg.DeployHelloHelidonApplication(namespace, "", istioInjection)
		metrics.Emit(t.Metrics.With("deployment_elapsed_time", time.Since(start).Milliseconds()))
	}

	Eventually(func() bool {
		return pkg.ContainerImagePullWait(namespace, expectedPodsHelloHelidon)
	}, imagePullWaitTimeout, imagePullPollingInterval).Should(BeTrue())
	// Verify hello-helidon-deployment pod is running
	// GIVEN OAM hello-helidon app is deployed
	// WHEN the component and appconfig are created
	// THEN the expected pod must be running in the test namespace
	if !skipVerify {
		Eventually(helloHelidonPodsRunning, longWaitTimeout, longPollingInterval).Should(BeTrue())
	}
	beforeSuitePassed = true
})

var failed = false
var beforeSuitePassed = false

var _ = t.AfterEach(func() {
	failed = failed || CurrentSpecReport().Failed()
})

var _ = t.AfterSuite(func() {
	if failed || !beforeSuitePassed {
		pkg.ExecuteClusterDumpWithEnvVarConfig()
	}
	if !skipUndeploy {
		start := time.Now()
		pkg.UndeployHelloHelidonApplication(namespace)
		metrics.Emit(t.Metrics.With("undeployment_elapsed_time", time.Since(start).Milliseconds()))
	}
})

var _ = t.Describe("Hello Helidon OAM App test", Label("f:app-lcm.oam",
	"f:app-lcm.helidon-workload"), func() {
	var host = ""
	var err error
	// Get the host from the Istio gateway resource.
	// GIVEN the Istio gateway for the hello-helidon namespace
	// WHEN GetHostnameFromGateway is called
	// THEN return the host name found in the gateway.
	t.BeforeEach(func() {
		Eventually(func() (string, error) {
			host, err = k8sutil.GetHostnameFromGateway(namespace, "")
			return host, err
		}, shortWaitTimeout, shortPollingInterval).Should(Not(BeEmpty()))
	})

	// Verify Hello Helidon app is working
	// GIVEN OAM hello-helidon app is deployed
	// WHEN the component and appconfig with ingress trait are created
	// THEN the application endpoint must be accessible
	t.Describe("for Ingress.", Label("f:mesh.ingress"), func() {
		t.It("Access /greet App Url.", func() {
			if skipVerify {
				Skip(skipVerifications)
			}
			url := fmt.Sprintf("https://%s/greet", host)
			Eventually(func() bool {
				return appEndpointAccessible(url, host)
			}, longWaitTimeout, longPollingInterval).Should(BeTrue())
		})
	})

	// Verify Prometheus scraped metrics
	// GIVEN OAM hello-helidon app is deployed
	// WHEN the component and appconfig without metrics-trait(using default) are created
	// THEN the application metrics must be accessible
	t.Describe("for Metrics.", Label("f:observability.monitoring.prom"), FlakeAttempts(5), func() {
		t.It("Retrieve Prometheus scraped metrics", func() {
			if skipVerify {
				Skip(skipVerifications)
			}
			pkg.Concurrently(
				func() {
					Eventually(appMetricsExists, longWaitTimeout, longPollingInterval).Should(BeTrue())
				},
				func() {
					Eventually(appComponentMetricsExists, longWaitTimeout, longPollingInterval).Should(BeTrue())
				},
				func() {
					Eventually(appConfigMetricsExists, longWaitTimeout, longPollingInterval).Should(BeTrue())
				},
				func() {
					Eventually(nodeExporterProcsRunning, longWaitTimeout, longPollingInterval).Should(BeTrue())
				},
				func() {
					Eventually(nodeExporterDiskIoNow, longWaitTimeout, longPollingInterval).Should(BeTrue())
				},
			)
		})
	})

	t.Context("Logging.", Label("f:observability.logging.es"), FlakeAttempts(5), func() {

		indexName, err := pkg.GetOpenSearchAppIndex(namespace)
		Expect(err).To(BeNil())
		// GIVEN an application with logging enabled
		// WHEN the Elasticsearch index is retrieved
		// THEN verify that it is found
		t.It("Verify Elasticsearch index exists", func() {
			if skipVerify {
				Skip(skipVerifications)
			}
			Eventually(func() bool {
				return pkg.LogIndexFound(indexName)
			}, longWaitTimeout, longPollingInterval).Should(BeTrue(), "Expected to find log index for hello helidon")
		})

		// GIVEN an application with logging enabled
		// WHEN the log records are retrieved from the Elasticsearch index
		// THEN verify that at least one recent log record is found
		t.It("Verify recent Elasticsearch log record exists", func() {
			if skipVerify {
				Skip(skipVerifications)
			}
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
	result, err := pkg.PodsRunning(namespace, expectedPodsHelloHelidon)
	if err != nil {
		AbortSuite(fmt.Sprintf("One or more pods are not running in the namespace: %v, error: %v", namespace, err))
	}
	return result
}

func appEndpointAccessible(url string, hostname string) bool {
	req, err := retryablehttp.NewRequest("GET", url, nil)
	if err != nil {
		t.Logs.Errorf("Unexpected error=%v", err)
		return false
	}

	kubeconfigPath, err := k8sutil.GetKubeConfigLocation()
	if err != nil {
		t.Logs.Errorf("Unexpected error=%v", err)
		return false
	}

	httpClient, err := pkg.GetVerrazzanoHTTPClient(kubeconfigPath)
	if err != nil {
		t.Logs.Errorf("Unexpected error=%v", err)
		return false
	}
	req.Host = hostname
	//req.Close = true
	resp, err := httpClient.Do(req)
	if err != nil {
		t.Logs.Errorf("Unexpected error=%v", err)
		bodyRaw, _ := ioutil.ReadAll(resp.Body)
		t.Logs.Errorf("Response=%v", string(bodyRaw))
		resp.Body.Close()
		return false
	}
	bodyRaw, err := ioutil.ReadAll(resp.Body)
	resp.Body.Close()
	if err != nil {
		t.Logs.Errorf("Unexpected error=%v", err)
		return false
	}
	if resp.StatusCode != http.StatusOK {
		t.Logs.Errorf("Unexpected status code=%v", resp.StatusCode)
		return false
	}
	// HTTP Server headers should never be returned.
	for headerName, headerValues := range resp.Header {
		if strings.EqualFold(headerName, "Server") {
			t.Logs.Errorf("Unexpected Server header=%v", headerValues)
			return false
		}
	}
	bodyStr := string(bodyRaw)
	if !strings.Contains(bodyStr, "Hello World") {
		t.Logs.Errorf("Unexpected response body=%v", bodyStr)
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
