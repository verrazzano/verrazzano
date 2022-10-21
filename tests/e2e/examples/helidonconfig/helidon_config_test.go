// Copyright (c) 2021, 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package helidonconfig

import (
	"fmt"
	"time"

	"github.com/verrazzano/verrazzano/pkg/k8s/resource"
	"github.com/verrazzano/verrazzano/pkg/test/framework"
	"github.com/verrazzano/verrazzano/pkg/test/framework/metrics"
	"k8s.io/apimachinery/pkg/api/errors"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/verrazzano/verrazzano/pkg/k8sutil"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg"
	v1 "k8s.io/api/core/v1"
)

const (
	longWaitTimeout          = 20 * time.Minute
	longPollingInterval      = 20 * time.Second
	shortPollingInterval     = 10 * time.Second
	shortWaitTimeout         = 5 * time.Minute
	imagePullWaitTimeout     = 40 * time.Minute
	imagePullPollingInterval = 30 * time.Second
)

var (
	t                  = framework.NewTestFramework("helidonconfig")
	generatedNamespace = pkg.GenerateNamespace("helidon-config")
)

var _ = t.BeforeSuite(func() {
	if !skipDeploy {
		start := time.Now()
		Eventually(func() (*v1.Namespace, error) {
			nsLabels := map[string]string{
				"verrazzano-managed": "true",
				"istio-injection":    istioInjection}
			return pkg.CreateNamespace(namespace, nsLabels)
		}, shortWaitTimeout, shortPollingInterval).ShouldNot(BeNil())

		Eventually(func() error {
			file, err := pkg.FindTestDataFile("examples/helidon-config/helidon-config-comp.yaml")
			if err != nil {
				return err
			}
			return resource.CreateOrUpdateResourceFromFileInGeneratedNamespace(file, namespace)
		}, shortWaitTimeout, shortPollingInterval).ShouldNot(HaveOccurred())

		Eventually(func() error {
			file, err := pkg.FindTestDataFile("examples/helidon-config/helidon-config-app.yaml")
			if err != nil {
				return err
			}
			return resource.CreateOrUpdateResourceFromFileInGeneratedNamespace(file, namespace)
		}, shortWaitTimeout, shortPollingInterval, "Failed to create helidon-config application resource").ShouldNot(HaveOccurred())
		beforeSuitePassed = true
		metrics.Emit(t.Metrics.With("deployment_elapsed_time", time.Since(start).Milliseconds()))

		Eventually(func() bool {
			return pkg.ContainerImagePullWait(namespace, expectedPodsHelidonConfig)
		}, imagePullWaitTimeout, imagePullPollingInterval).Should(BeTrue())
	}

	// Verify helidon-config-deployment pod is running
	// GIVEN OAM helidon-config app is deployed
	// WHEN the component and appconfig are created
	// THEN the expected pod must be running in the test namespace
	Eventually(helidonConfigPodsRunning, longWaitTimeout, longPollingInterval).Should(BeTrue())
})

var failed = false
var beforeSuitePassed = false

var _ = t.AfterEach(func() {
	failed = failed || CurrentSpecReport().Failed()
})

var _ = t.AfterSuite(func() {

	if failed || !beforeSuitePassed {
		pkg.ExecuteBugReport(namespace)
	}
	if !skipUndeploy {
		start := time.Now()
		// undeploy the application here
		pkg.Log(pkg.Info, "Delete application")
		Eventually(func() error {
			file, err := pkg.FindTestDataFile("examples/helidon-config/helidon-config-app.yaml")
			if err != nil {
				return err
			}
			return resource.DeleteResourceFromFileInGeneratedNamespace(file, namespace)
		}, shortWaitTimeout, shortPollingInterval).ShouldNot(HaveOccurred())

		pkg.Log(pkg.Info, "Delete components")
		Eventually(func() error {
			file, err := pkg.FindTestDataFile("examples/helidon-config/helidon-config-comp.yaml")
			if err != nil {
				return err
			}
			return resource.DeleteResourceFromFileInGeneratedNamespace(file, namespace)
		}, shortWaitTimeout, shortPollingInterval).ShouldNot(HaveOccurred())

		pkg.Log(pkg.Info, "Wait for application pods to terminate")
		Eventually(func() bool {
			podsTerminated, _ := pkg.PodsNotRunning(namespace, expectedPodsHelidonConfig)
			return podsTerminated
		}, shortWaitTimeout, shortPollingInterval).Should(BeTrue())

		pkg.Log(pkg.Info, "Delete namespace")
		Eventually(func() error {
			return pkg.DeleteNamespace(namespace)
		}, shortWaitTimeout, shortPollingInterval).ShouldNot(HaveOccurred())

		pkg.Log(pkg.Info, "Wait for Finalizer to be removed")
		Eventually(func() bool {
			return pkg.CheckNamespaceFinalizerRemoved(namespace)
		}, shortWaitTimeout, shortPollingInterval).Should(BeTrue())

		pkg.Log(pkg.Info, "Wait for namespace to be removed")
		Eventually(func() bool {
			_, err := pkg.GetNamespace(namespace)
			return err != nil && errors.IsNotFound(err)
		}, shortWaitTimeout, shortPollingInterval).Should(BeTrue())

		metrics.Emit(t.Metrics.With("undeployment_elapsed_time", time.Since(start).Milliseconds()))
	}
})

var (
	expectedPodsHelidonConfig = []string{"helidon-config-deployment"}
	waitTimeout               = 10 * time.Minute
	pollingInterval           = 30 * time.Second
)

var _ = t.Describe("Helidon Config OAM App test", Label("f:app-lcm.oam",
	"f:app-lcm.helidon-workload"), func() {

	var host = ""
	var err error
	// Get the host from the Istio gateway resource.
	// GIVEN the Istio gateway for the helidon-config namespace
	// WHEN GetHostnameFromGateway is called
	// THEN return the host name found in the gateway.
	t.BeforeEach(func() {
		Eventually(func() (string, error) {
			host, err = k8sutil.GetHostnameFromGateway(namespace, "")
			return host, err
		}, shortWaitTimeout, shortPollingInterval).Should(Not(BeEmpty()))
	})

	// Verify Helidon Config app is working
	// GIVEN OAM helidon-config app is deployed
	// WHEN the component and appconfig with ingress trait are created
	// THEN the application endpoint must be accessible
	t.Describe("Ingress.", Label("f:mesh.ingress"), func() {
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
	t.Describe("Metrics.", Label("f:observability.monitoring.prom"), func() {
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

	t.Context("Logging.", Label("f:observability.logging.es"), func() {
		indexName, err := pkg.GetOpenSearchAppIndex(namespace)
		Expect(err).To(BeNil())
		// GIVEN an application with logging enabled
		// WHEN the Elasticsearch index is retrieved
		// THEN verify that it is found
		t.It("Verify Elasticsearch index exists", func() {
			Eventually(func() bool {
				return pkg.LogIndexFound(indexName)
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
	result, err := pkg.PodsRunning(namespace, expectedPodsHelidonConfig)
	if err != nil {
		AbortSuite(fmt.Sprintf("One or more pods are not running in the namespace: %v, error: %v", namespace, err))
	}
	return result
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
