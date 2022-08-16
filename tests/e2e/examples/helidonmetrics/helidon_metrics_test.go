// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package helidonmetrics

import (
	"github.com/verrazzano/verrazzano/pkg/test/framework"
	"github.com/verrazzano/verrazzano/pkg/test/framework/metrics"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"

	"fmt"
	"strings"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/verrazzano/verrazzano/pkg/k8sutil"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg"
)

const (
	longWaitTimeout      = 20 * time.Minute
	longPollingInterval  = 20 * time.Second
	shortPollingInterval = 10 * time.Second
	shortWaitTimeout     = 5 * time.Minute
)

var (
	t                  = framework.NewTestFramework("helidonmetrics")
	generatedNamespace = pkg.GenerateNamespace("helidon-metrics")
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
			return pkg.CreateOrUpdateResourceFromFileInGeneratedNamespace(
				"examples/hello-helidon/hello-helidon-comp.yaml", namespace)
		}, shortWaitTimeout, shortPollingInterval).ShouldNot(HaveOccurred())
		Eventually(func() error {
			return pkg.CreateOrUpdateResourceFromFileInGeneratedNamespace(
				"examples/hello-helidon/hello-helidon-app.yaml", namespace)
		}, shortWaitTimeout, shortPollingInterval).ShouldNot(HaveOccurred())

		beforeSuitePassed = true
		metrics.Emit(t.Metrics.With("deployment_elapsed_time", time.Since(start).Milliseconds()))
	}
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
			return pkg.DeleteResourceFromFileInGeneratedNamespace(
				"tests/e2e/examples/helidonmetrics/testdata/hello-helidon-app-metrics-disabled.yaml", namespace)
		}, shortWaitTimeout, shortPollingInterval).ShouldNot(HaveOccurred())

		pkg.Log(pkg.Info, "Delete components")
		Eventually(func() error {
			return pkg.DeleteResourceFromFileInGeneratedNamespace(
				"examples/hello-helidon/hello-helidon-comp.yaml", namespace)
		}, shortWaitTimeout, shortPollingInterval).ShouldNot(HaveOccurred())

		pkg.Log(pkg.Info, "Wait for application pods to terminate")
		Eventually(func() bool {
			podsTerminated, _ := pkg.PodsNotRunning(namespace, expectedPodsHelloHelidon)
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
	expectedPodsHelloHelidon = []string{"hello-helidon-deployment"}
	waitTimeout              = 10 * time.Minute
	pollingInterval          = 30 * time.Second
)

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

	// Verify Prometheus scrape targets
	// GIVEN OAM hello-helidon app is deployed
	// WHEN the appconfig is updated with metrics-trait.enabled=false
	// THEN the application metrics target must be removed
	t.Describe("for Metrics.", Label("f:observability.monitoring.prom"), FlakeAttempts(5), func() {
		t.It("MetricsTrait can be disabled", func() {
			pkg.Concurrently(
				func() {
					pkg.Log(pkg.Info, "Checking for scrape target existence")
					Eventually(scrapeTargetExists, shortWaitTimeout, longPollingInterval).Should(BeTrue())
				},
				func() {
					pkg.Log(pkg.Info, "Checking for service monitor existence")
					Eventually(serviceMonitorExists, shortWaitTimeout, longPollingInterval).Should(BeTrue())
				},
			)
			pkg.Log(pkg.Info, "Disabling metrics trait")
			Eventually(func() error {
				return pkg.CreateOrUpdateResourceFromFileInGeneratedNamespace(
					"tests/e2e/examples/helidonmetrics/testdata/hello-helidon-app-metrics-disabled.yaml", namespace)
			}, shortWaitTimeout, shortPollingInterval, "Failed to disable metrics").ShouldNot(HaveOccurred())
			pkg.Concurrently(
				func() {
					pkg.Log(pkg.Info, "Checking for scrape target to no longer exist")
					Eventually(scrapeTargetExists, shortWaitTimeout, longPollingInterval).Should(BeFalse())
				},
				func() {
					pkg.Log(pkg.Info, "Checking for service monitor to no longer exist")
					Eventually(serviceMonitorExists, shortWaitTimeout, longPollingInterval).Should(BeFalse())
				},
			)
		})
	})
})

func helidonConfigPodsRunning() bool {
	result, err := pkg.PodsRunning(namespace, expectedPodsHelloHelidon)
	if err != nil {
		AbortSuite(fmt.Sprintf("One or more pods are not running in the namespace: %v, error: %v", namespace, err))
	}
	return result
}

func serviceMonitorExists() bool {
	smName := pkg.GetAppServiceMonitorName(namespace, "hello-helidon")
	sm, err := pkg.GetServiceMonitor(namespace, smName)
	if err != nil {
		pkg.Log(pkg.Error, fmt.Sprintf("Failed to get the Service Monitor from the cluster: %v", err))
		return false
	}
	return sm != nil
}

func scrapeTargetExists() bool {
	targets, _ := pkg.ScrapeTargets()
	for _, t := range targets {
		m := t.(map[string]interface{})
		scrapePool := m["scrapePool"].(string)
		if strings.Contains(scrapePool, namespace) {
			return true
		}
	}
	return false
}
