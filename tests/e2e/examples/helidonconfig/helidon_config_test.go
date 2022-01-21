// Copyright (c) 2021, 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package helidonconfig

import (
	"fmt"
	"github.com/verrazzano/verrazzano/pkg/test/framework"
	"github.com/verrazzano/verrazzano/pkg/test/framework/metrics"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/verrazzano/verrazzano/pkg/k8sutil"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg"
	v1 "k8s.io/api/core/v1"
)

const (
	longWaitTimeout      = 20 * time.Minute
	longPollingInterval  = 20 * time.Second
	shortPollingInterval = 10 * time.Second
	shortWaitTimeout     = 5 * time.Minute
)

var t = framework.NewTestFramework("helidonconfig")

var _ = t.BeforeSuite(func() {
	if !skipDeploy {
		start := time.Now()
		Eventually(func() (*v1.Namespace, error) {
			nsLabels := map[string]string{
				"verrazzano-managed": "true",
				"istio-injection":    "enabled"}
			return pkg.CreateNamespace("helidon-config", nsLabels)
		}, shortWaitTimeout, shortPollingInterval).ShouldNot(BeNil())

		Eventually(func() error {
			return pkg.CreateOrUpdateResourceFromFile("examples/helidon-config/helidon-config-comp.yaml")
		}, shortWaitTimeout, shortPollingInterval).ShouldNot(HaveOccurred())

		Eventually(func() error {
			return pkg.CreateOrUpdateResourceFromFile("examples/helidon-config/helidon-config-app.yaml")
		}, shortWaitTimeout, shortPollingInterval, "Failed to create helidon-config application resource").ShouldNot(HaveOccurred())
		metrics.Emit(t.Metrics.With("deployment_elapsed_time", time.Since(start).Milliseconds()))
	}
})

var failed = false
var _ = t.AfterEach(func() {
	failed = failed || CurrentSpecReport().Failed()
})

var _ = t.AfterSuite(func() {
	if failed {
		pkg.ExecuteClusterDumpWithEnvVarConfig()
	}
	if !skipUndeploy {
		start := time.Now()
		// undeploy the application here
		Eventually(func() error {
			return pkg.DeleteResourceFromFile("examples/helidon-config/helidon-config-app.yaml")
		}, shortWaitTimeout, shortPollingInterval).ShouldNot(HaveOccurred())

		Eventually(func() error {
			return pkg.DeleteResourceFromFile("examples/helidon-config/helidon-config-comp.yaml")
		}, shortWaitTimeout, shortPollingInterval).ShouldNot(HaveOccurred())

		Eventually(func() error {
			return pkg.DeleteNamespace("helidon-config")
		}, shortWaitTimeout, shortPollingInterval).ShouldNot(HaveOccurred())
		metrics.Emit(t.Metrics.With("undeployment_elapsed_time", time.Since(start).Milliseconds()))
	}
})

var (
	expectedPodsHelidonConfig = []string{"helidon-config-deployment"}
	waitTimeout               = 10 * time.Minute
	pollingInterval           = 30 * time.Second
)

const (
	testNamespace      = "helidon-config"
	istioNamespace     = "istio-system"
	ingressServiceName = "istio-ingressgateway"
)

var _ = t.Describe("Helidon Config OAM App test", func() {
	// Verify helidon-config-deployment pod is running
	// GIVEN OAM helidon-config app is deployed
	// WHEN the component and appconfig are created
	// THEN the expected pod must be running in the test namespace
	t.Describe("helidon-config-deployment pod", func() {
		t.It("is running", func() {
			Eventually(helidonConfigPodsRunning, waitTimeout, pollingInterval).Should(BeTrue())
		})
	})

	var host = ""
	var err error
	// Get the host from the Istio gateway resource.
	// GIVEN the Istio gateway for the helidon-config namespace
	// WHEN GetHostnameFromGateway is called
	// THEN return the host name found in the gateway.
	t.It("Get host from gateway.", func() {
		Eventually(func() (string, error) {
			host, err = k8sutil.GetHostnameFromGateway(testNamespace, "")
			return host, err
		}, shortWaitTimeout, shortPollingInterval).Should(Not(BeEmpty()))
	})

	// Verify Helidon Config app is working
	// GIVEN OAM helidon-config app is deployed
	// WHEN the component and appconfig with ingress trait are created
	// THEN the application endpoint must be accessible
	t.Describe("Ingress.", func() {
		t.It("Access /config App Url.", func() {
			url := fmt.Sprintf("https://%s/config", host)
			kubeconfigPath, err := k8sutil.GetKubeConfigLocation()
			Expect(err).ShouldNot(HaveOccurred())
			Eventually(func() (*pkg.HTTPResponse, error) {
				return pkg.GetWebPageWithBasicAuth(url, host, "", "", kubeconfigPath)
			}, shortWaitTimeout, shortPollingInterval).Should(And(pkg.HasStatus(200), pkg.BodyContains("HelloConfig World")))
		})
	})

	// Verify Prometheus scraped metrics
	// GIVEN OAM helidon-config app is deployed
	// WHEN the component and appconfig without metrics-trait(using default) are created
	// THEN the application metrics must be accessible
	t.Describe("Metrics.", func() {
		t.It("Retrieve Prometheus scraped metrics", func() {
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
			)
		})
	})

	t.Context("Logging.", func() {
		indexName := "verrazzano-namespace-helidon-config"
		// GIVEN an application with logging enabled
		// WHEN the Elasticsearch index is retrieved
		// THEN verify that it is found
		t.It("Verify Elasticsearch index exists", func() {
			Eventually(func() bool {
				return pkg.LogIndexFound(".ds-verrazzano-application-000001")
			}, longWaitTimeout, longPollingInterval).Should(BeTrue(), "Expected to find log index for helidon config")
		})

		// GIVEN an application with logging enabled
		// WHEN the log records are retrieved from the Elasticsearch index
		// THEN verify that at least one recent log record is found
		t.It("Verify recent Elasticsearch log record exists", func() {
			Eventually(func() bool {
				return pkg.LogRecordFound(indexName, time.Now().Add(-24*time.Hour), map[string]string{
					"kubernetes.labels.app_oam_dev\\/component": "helidon-config-component",
					"kubernetes.labels.app_oam_dev\\/name":      "helidon-config-appconf",
					"kubernetes.container_name":                 "helidon-config-container",
				})
			}, longWaitTimeout, longPollingInterval).Should(BeTrue(), "Expected to find a recent log record")
		})
	})
})

func helidonConfigPodsRunning() bool {
	return pkg.PodsRunning(testNamespace, expectedPodsHelidonConfig)
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
