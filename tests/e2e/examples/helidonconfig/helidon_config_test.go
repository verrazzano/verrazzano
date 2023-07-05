// Copyright (c) 2021, 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package helidonconfig

import (
	"fmt"
	"os"
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
	longWaitTimeout          = 20 * time.Minute
	longPollingInterval      = 20 * time.Second
	shortPollingInterval     = 10 * time.Second
	shortWaitTimeout         = 5 * time.Minute
	imagePullWaitTimeout     = 40 * time.Minute
	imagePullPollingInterval = 30 * time.Second

	ingress        = "helidon-config-ingress-rule"
	helidonService = "helidon-config-deployment"
	targetsVersion = "1.4.0"
)

var (
	t                  = framework.NewTestFramework("helidonconfig")
	generatedNamespace = pkg.GenerateNamespace("helidon-config")
	kubeConfig         = os.Getenv("KUBECONFIG")
	host               = ""
	metricsTest        pkg.MetricsTest
)
var isMinVersion140 bool

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

		t.Logs.Info("Container image pull check")
		Eventually(func() bool {
			return pkg.ContainerImagePullWait(namespace, expectedPodsHelidonConfig)
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

	// Verify helidon-config-deployment pod is running
	// GIVEN OAM helidon-config app is deployed
	// WHEN the component and appconfig are created
	// THEN the expected pod must be running in the test namespace
	t.Logs.Info("Helidon Config: check expected pods are running")
	Eventually(func() bool {
		result, err := pkg.PodsRunning(namespace, expectedPodsHelidonConfig)
		if err != nil {
			AbortSuite(fmt.Sprintf("One or more pods are not running in the namespace: %v, error: %v", namespace, err))
		}
		return result
	}, longWaitTimeout, longPollingInterval).Should(BeTrue(), "Helidon Config Failed to Deploy: Pods are not ready")

	t.Logs.Info("Helidon Config: check expected Services are running")
	Eventually(func() bool {
		result, err := pkg.DoesServiceExist(namespace, helidonService)
		if err != nil {
			AbortSuite(fmt.Sprintf("Helidon Service %s is not running in the namespace: %v, error: %v", helidonService, namespace, err))
		}
		return result
	}, longWaitTimeout, longPollingInterval).Should(BeTrue(), "Helidon Config Failed to Deploy: Services are not ready")

	t.Logs.Info("Helidon Config: check expected VirtualService is ready")
	Eventually(func() bool {
		result, err := pkg.DoesVirtualServiceExist(namespace, ingress)
		if err != nil {
			AbortSuite(fmt.Sprintf("Helidon VirtualService %s is not running in the namespace: %v, error: %v", ingress, namespace, err))
		}
		return result
	}, shortWaitTimeout, longPollingInterval).Should(BeTrue(), "Helidon Config Failed to Deploy: VirtualService is not ready")

	// Get the host from the Istio gateway resource.
	start := time.Now()
	t.Logs.Info("Helidon Config: check expected Gateway is ready")
	Eventually(func() (string, error) {
		host, err = k8sutil.GetHostnameFromGateway(namespace, "")
		return host, err
	}, shortWaitTimeout, shortPollingInterval).Should(Not(BeEmpty()), "Helidon Config: Gateway is not ready")
	metrics.Emit(t.Metrics.With("get_host_name_elapsed_time", time.Since(start).Milliseconds()))

	kubeconfig, err := k8sutil.GetKubeConfigLocation()
	if err != nil {
		AbortSuite(fmt.Sprintf("Failed to get the Kubeconfig location for the cluster: %v", err))
	}
	metricsTest, err = pkg.NewMetricsTest(kubeconfig, map[string]string{})
	if err != nil {
		AbortSuite(fmt.Sprintf("Failed to create the Metrics test object: %v", err))
	}
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

var _ = AfterSuite(afterSuite)

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

	// Verify Prometheus scraped targets
	// GIVEN OAM helidon-config app is deployed
	// WHEN the component and appconfig without metrics-trait(using default) are created
	// THEN the application scrape targets must be healthy
	t.Describe("Metrics.", Label("f:observability.monitoring.prom"), func() {
		t.It("Verify all scrape targets are healthy for the application", func() {
			Eventually(func() (bool, error) {
				var componentNames = []string{"helidon-config-component"}
				return pkg.ScrapeTargetsHealthy(pkg.GetScrapePools(namespace, "helidon-config-appconf", componentNames, isMinVersion140))
			}, shortWaitTimeout, shortPollingInterval).Should(BeTrue())
		})
	})

	t.Context("Logging.", Label("f:observability.logging.es"), func() {
		var indexName string
		Eventually(func() error {
			indexName, err = pkg.GetOpenSearchAppIndex(namespace)
			return err
		}, shortWaitTimeout, shortPollingInterval).Should(BeNil(), "Expected to get OpenSearch App Index")

		// GIVEN an application with logging enabled
		// WHEN the Opensearch index is retrieved
		// THEN verify that it is found
		t.It("Verify Opensearch index exists", func() {
			Eventually(func() bool {
				return pkg.LogIndexFound(indexName)
			}, longWaitTimeout, longPollingInterval).Should(BeTrue(), "Expected to find log index for helidon config")
		})

		// GIVEN an application with logging enabled
		// WHEN the log records are retrieved from the Opensearch index
		// THEN verify that at least one recent log record is found
		t.It("Verify recent Opensearch log record exists", func() {
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
