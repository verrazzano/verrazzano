// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package helidonmetrics

import (
	"fmt"
	"strings"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/verrazzano/verrazzano/pkg/k8s/resource"
	"github.com/verrazzano/verrazzano/pkg/k8sutil"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg"
	dump "github.com/verrazzano/verrazzano/tests/e2e/pkg/test/clusterdump"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg/test/framework"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg/test/framework/metrics"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
)

const (
	longWaitTimeout      = 20 * time.Minute
	longPollingInterval  = 20 * time.Second
	shortPollingInterval = 10 * time.Second
	shortWaitTimeout     = 5 * time.Minute

	ingress        = "hello-helidon-ingress-rule"
	helidonService = "hello-helidon-deployment"
)

var (
	t                  = framework.NewTestFramework("helidonmetrics")
	generatedNamespace = pkg.GenerateNamespace("helidon-metrics")
	host               = ""
)

var beforeSuite = t.BeforeSuiteFunc(func() {
	if !skipDeploy {
		start := time.Now()
		Eventually(func() (*v1.Namespace, error) {
			nsLabels := map[string]string{
				"verrazzano-managed": "true",
				"istio-injection":    istioInjection}
			return pkg.CreateNamespace(namespace, nsLabels)
		}, shortWaitTimeout, shortPollingInterval).ShouldNot(BeNil())

		Eventually(func() error {
			file, err := pkg.FindTestDataFile("examples/hello-helidon/hello-helidon-comp.yaml")
			if err != nil {
				return err
			}
			return resource.CreateOrUpdateResourceFromFileInGeneratedNamespace(file, namespace)
		}, shortWaitTimeout, shortPollingInterval).ShouldNot(HaveOccurred())
		Eventually(func() error {
			file, err := pkg.FindTestDataFile("examples/hello-helidon/hello-helidon-app.yaml")
			if err != nil {
				return err
			}
			return resource.CreateOrUpdateResourceFromFileInGeneratedNamespace(file, namespace)
		}, shortWaitTimeout, shortPollingInterval).ShouldNot(HaveOccurred())

		beforeSuitePassed = true
		metrics.Emit(t.Metrics.With("deployment_elapsed_time", time.Since(start).Milliseconds()))
	}

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
		// undeploy the application here
		pkg.Log(pkg.Info, "Delete application")
		Eventually(func() error {
			file, err := pkg.FindTestDataFile("tests/e2e/examples/helidonmetrics/testdata/hello-helidon-app-metrics-disabled.yaml")
			if err != nil {
				return err
			}
			return resource.DeleteResourceFromFileInGeneratedNamespace(file, namespace)
		}, shortWaitTimeout, shortPollingInterval).ShouldNot(HaveOccurred())

		pkg.Log(pkg.Info, "Delete components")
		Eventually(func() error {
			file, err := pkg.FindTestDataFile("examples/hello-helidon/hello-helidon-comp.yaml")
			if err != nil {
				return err
			}
			return resource.DeleteResourceFromFileInGeneratedNamespace(file, namespace)
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

var _ = AfterSuite(afterSuite)

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
				file, err := pkg.FindTestDataFile("tests/e2e/examples/helidonmetrics/testdata/hello-helidon-app-metrics-disabled.yaml")
				if err != nil {
					return err
				}
				return resource.CreateOrUpdateResourceFromFileInGeneratedNamespace(file, namespace)
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

func serviceMonitorExists() bool {
	smName := pkg.GetAppServiceMonitorName(namespace, "hello-helidon", "hello-helidon-component")
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
