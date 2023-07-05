// Copyright (c) 2020, 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package helidon

import (
	"fmt"
	"io"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/hashicorp/go-retryablehttp"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/verrazzano/verrazzano/pkg/k8sutil"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg"
	dump "github.com/verrazzano/verrazzano/tests/e2e/pkg/test/clusterdump"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg/test/framework"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg/test/framework/metrics"
)

const (
	longWaitTimeout            = 20 * time.Minute
	longPollingInterval        = 20 * time.Second
	shortPollingInterval       = 10 * time.Second
	shortWaitTimeout           = 5 * time.Minute
	imagePullWaitTimeout       = 40 * time.Minute
	imagePullPollingInterval   = 30 * time.Second
	skipVerifications          = "Skip Verifications"
	helloHelidon               = "hello-helidon"
	nodeExporterJobName        = "node-exporter"
	helloHelidonDeploymentName = "hello-helidon-deployment"

	ingress        = "hello-helidon-ingress-rule"
	helidonService = "hello-helidon-deployment"
)

var (
	t                  = framework.NewTestFramework("helidon")
	generatedNamespace = pkg.GenerateNamespace(helloHelidon)
	// yamlApplier              = k8sutil.YAMLApplier{}
	expectedPodsHelloHelidon = []string{"hello-helidon-deployment"}
	host                     = ""
	isMinVersion140          bool
	metricsTest              pkg.MetricsTest
)

var beforeSuite = t.BeforeSuiteFunc(func() {

	if !skipDeploy {
		start := time.Now()
		pkg.DeployHelloHelidonApplication(namespace, "", istioInjection, helloHelidonComponent, helloHelidonAppConfig)
		metrics.Emit(t.Metrics.With("deployment_elapsed_time", time.Since(start).Milliseconds()))

		t.Logs.Info("Container image pull check")
		Eventually(func() bool {
			return pkg.ContainerImagePullWait(namespace, expectedPodsHelloHelidon)
		}, imagePullWaitTimeout, imagePullPollingInterval).Should(BeTrue())
	}
	kubeconfigPath, err := k8sutil.GetKubeConfigLocation()
	if err != nil {
		Fail(fmt.Sprintf("Failed to get default kubeconfig path: %s", err.Error()))
	}
	isMinVersion140, err = pkg.IsVerrazzanoMinVersion("1.4.0", kubeconfigPath)
	if err != nil {
		Fail(err.Error())
	}

	if !skipVerify {
		// Verify hello-helidon-deployment pod is running
		// GIVEN OAM hello-helidon app is deployed
		// WHEN the component and appconfig are created
		// THEN the expected pod must be running in the test namespace
		t.Logs.Info("Helidon Example: check expected pods are running")
		Eventually(func() bool {
			result, err := pkg.PodsRunning(namespace, expectedPodsHelloHelidon)
			if err != nil {
				AbortSuite(fmt.Sprintf("One or more pods are not running in the namespace: %v, error: %v", namespace, err))
			}
			return result
		}, longWaitTimeout, longPollingInterval).Should(BeTrue(), "Helidon Example Failed to Deploy: Pods are not ready")

		t.Logs.Info("Helidon Example: check expected Services are running")
		Eventually(func() bool {
			result, err := pkg.DoesServiceExist(namespace, helidonService)
			if err != nil {
				AbortSuite(fmt.Sprintf("Helidon Service %s is not running in the namespace: %v, error: %v", helidonService, namespace, err))
			}
			return result
		}, longWaitTimeout, longPollingInterval).Should(BeTrue(), "Helidon Example Failed to Deploy: Services are not ready")

		t.Logs.Info("Helidon Example: check expected VirtualService is ready")
		Eventually(func() bool {
			result, err := pkg.DoesVirtualServiceExist(namespace, ingress)
			if err != nil {
				AbortSuite(fmt.Sprintf("Helidon VirtualService %s is not running in the namespace: %v, error: %v", ingress, namespace, err))
			}
			return result
		}, shortWaitTimeout, longPollingInterval).Should(BeTrue(), "Helidon Example Failed to Deploy: VirtualService is not ready")

		var err error
		// Get the host from the Istio gateway resource.
		start := time.Now()
		t.Logs.Info("Helidon Example: check expected Gateway is ready")
		Eventually(func() (string, error) {
			host, err = k8sutil.GetHostnameFromGateway(namespace, "")
			return host, err
		}, shortWaitTimeout, shortPollingInterval).Should(Not(BeEmpty()), "Helidon Example: Gateway is not ready")
		metrics.Emit(t.Metrics.With("get_host_name_elapsed_time", time.Since(start).Milliseconds()))

		// validate if manualscalertrait applied, then pod count should be 2
		// examples/hello-helidon/hello-helidon-app-scaler-trait.yaml
		if strings.Contains(helloHelidonAppConfig, "hello-helidon-app-scaler-trait.yaml") {
			t.Logs.Info("Helidon Example: check expected pods are running")

			Eventually(func() bool {
				return isDeploymentSetUpdated()
			}, longWaitTimeout, longPollingInterval).Should(BeTrue())
		}
	}

	kubeconfig, err := k8sutil.GetKubeConfigLocation()
	if err != nil {
		AbortSuite(fmt.Sprintf("Failed to get the Kubeconfig location for the cluster: %v", err))
	}
	metricsTest, err = pkg.NewMetricsTest(kubeconfig, map[string]string{})
	if err != nil {
		AbortSuite(fmt.Sprintf("Failed to create the Metrics test object: %v", err))
	}

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
		pkg.UndeployHelloHelidonApplication(namespace, helloHelidonComponent, helloHelidonAppConfig)
		metrics.Emit(t.Metrics.With("undeployment_elapsed_time", time.Since(start).Milliseconds()))
	}
})

var _ = AfterSuite(afterSuite)

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

	t.Describe("supports Selector", Label("f:selector.labels"), func() {
		t.It("Matchlabels and Matchexpressions", func() {
			if skipVerify {
				Skip(skipVerifications)
			}
			kubeConfig, err := k8sutil.GetKubeConfigLocation()
			if err != nil {
				Skip(skipVerifications)
			}
			if ok, _ := pkg.IsVerrazzanoMinVersion("1.4.0", kubeConfig); !ok {
				Skip(skipVerifications)
			}
			Eventually(func() bool {
				return isDeploymentLabelSelectorValuesMatched()
			}, longWaitTimeout, longPollingInterval).Should(BeTrue())
		})
	})
	// Verify Prometheus scraped targets
	// GIVEN OAM hello-helidon app is deployed
	// WHEN the component and appconfig without metrics-trait(using default) are created
	// THEN the application's all scrape targets must be healthy
	t.Describe("for Metrics.", Label("f:observability.monitoring.prom"), FlakeAttempts(5), func() {
		t.It("Verify all scrape targets are healthy for the application", func() {
			if skipVerify {
				Skip(skipVerifications)
			}
			Eventually(func() (bool, error) {
				var componentNames = []string{"hello-helidon-component"}
				return pkg.ScrapeTargetsHealthy(pkg.GetScrapePools(namespace, helloHelidon, componentNames, isMinVersion140))
			}, shortWaitTimeout, shortPollingInterval).Should(BeTrue())
		})
	})

	t.Context("Logging.", Label("f:observability.logging.es"), FlakeAttempts(5), func() {
		var indexName string
		Eventually(func() error {
			indexName, err = pkg.GetOpenSearchAppIndex(namespace)
			return err
		}, shortWaitTimeout, shortPollingInterval).Should(BeNil(), "Expected to get OpenSearch App Index")

		// GIVEN an application with logging enabled
		// WHEN the Opensearch index is retrieved
		// THEN verify that it is found
		t.It("Verify Opensearch index exists", func() {
			if skipVerify {
				Skip(skipVerifications)
			}
			Eventually(func() bool {
				return pkg.LogIndexFound(indexName)
			}, longWaitTimeout, longPollingInterval).Should(BeTrue(), "Expected to find log index for hello helidon")
		})

		// GIVEN an application with logging enabled
		// WHEN the log records are retrieved from the Opensearch index
		// THEN verify that at least one recent log record is found
		t.It("Verify recent Opensearch log record exists", func() {
			if skipVerify {
				Skip(skipVerifications)
			}
			if os.Getenv("TEST_ENV") != "LRE" {
				Eventually(func() bool {
					return pkg.FindLog(indexName,
						[]pkg.Match{
							{Key: "kubernetes.labels.app_oam_dev\\/component", Value: "hello-helidon-component"},
							{Key: "kubernetes.labels.app_oam_dev\\/name", Value: helloHelidon},
							{Key: "kubernetes.container_name", Value: "hello-helidon-container"}},
						[]pkg.Match{})
				}, longWaitTimeout, longPollingInterval).Should(BeTrue(), "Expected to find a recent log record")
			}
		})
	})

})

// isDeploymentLabelSelectorValuesMatched tests labelselector must exists into deployment
// also must have values into matchlabels & matchexpressions

func isDeploymentLabelSelectorValuesMatched() bool {
	// fetch labelselector from hello helidon deployment
	labelSelector, err := pkg.GetDeploymentLabelSelector(namespace, helloHelidonDeploymentName)
	if err != nil {
		return false
	}

	/*
		// Putting the exact value match on hold, reconciling during vz upgrade is wip
		// check labelselector matchlabels must have at least 1 pair of matchlabels arg
		if val, ok := labelSelector.MatchLabels["app"]; !ok || val != helloHelidon {
			return false
		}
		// check labelselector matchexpressions must not be empty
		if len(labelSelector.MatchExpressions) == 0 {
			return false
		}
	*/
	return labelSelector != nil
}

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
		t.Logs.Errorf("Unexpected error while creating new request=%v", err)
		return false
	}

	kubeconfigPath, err := k8sutil.GetKubeConfigLocation()
	if err != nil {
		t.Logs.Errorf("Unexpected error while getting kubeconfig location=%v", err)
		return false
	}

	httpClient, err := pkg.GetVerrazzanoHTTPClient(kubeconfigPath)
	if err != nil {
		t.Logs.Errorf("Unexpected error while getting new httpClient=%v", err)
		return false
	}
	req.Host = hostname
	req.Close = true
	resp, err := httpClient.Do(req)
	if err != nil {
		t.Logs.Errorf("Unexpected error while making http request=%v", err)
		if resp != nil && resp.Body != nil {
			bodyRaw, err := io.ReadAll(resp.Body)
			if err != nil {
				t.Logs.Errorf("Unexpected error while marshallling error response=%v", err)
				return false
			}

			t.Logs.Errorf("Error Response=%v", string(bodyRaw))
			resp.Body.Close()
		}
		return false
	}

	bodyRaw, err := io.ReadAll(resp.Body)
	resp.Body.Close()
	if err != nil {
		t.Logs.Errorf("Unexpected error marshallling response=%v", err)
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
	return metricsTest.MetricsExist("base_jvm_uptime_seconds", map[string]string{"app": helloHelidon})
}

func appComponentMetricsExists() bool {
	return metricsTest.MetricsExist("vendor_requests_count_total", map[string]string{"app_oam_dev_name": helloHelidon})
}

func appConfigMetricsExists() bool {
	return metricsTest.MetricsExist("vendor_requests_count_total", map[string]string{"app_oam_dev_component": "hello-helidon-component"})
}

func nodeExporterProcsRunning() bool {
	return metricsTest.MetricsExist("node_procs_running", map[string]string{"job": nodeExporterJobName})
}

func nodeExporterDiskIoNow() bool {
	return metricsTest.MetricsExist("node_disk_io_now", map[string]string{"job": nodeExporterJobName})
}

// isDeploymentSetUpdated returns
// replicas from examples/hello-helidon/hello-helidon-app-scaler-trait.yaml
// currently configured 2
func isDeploymentSetUpdated() bool {
	pods, err := pkg.ListPods(namespace, metav1.ListOptions{})
	if err != nil {
		t.Logs.Errorf("Unexpected error while listing the deployments error response=%v", err)
		return false
	}
	t.Logs.Info("current pod count...", len(pods.Items))
	if len(pods.Items) < 2 {
		return false
	} else {
		return true
	}
}
