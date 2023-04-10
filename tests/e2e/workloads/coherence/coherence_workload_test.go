// Copyright (c) 2022, 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package coherence

import (
	"fmt"
	"net/http"
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
	shortWaitTimeout         = 10 * time.Minute
	shortPollingInterval     = 10 * time.Second
	longWaitTimeout          = 20 * time.Minute
	longPollingInterval      = 20 * time.Second
	imagePullWaitTimeout     = 40 * time.Minute
	imagePullPollingInterval = 30 * time.Second

	appConfiguration  = "tests/testdata/test-applications/coherence/hello-coherence/hello-coherence-app.yaml"
	compConfiguration = "tests/testdata/test-applications/coherence/hello-coherence/hello-coherence-comp.yaml"

	appEndPoint       = "catalogue"
	expectedResponse  = "A perfect example of a swivel chair trained calf"
	skipVerifications = "Skip Verifications"

	ingress          = "hello-ingress-rule"
	coherenceService = "hello-coh-http"
	helloCreds       = "hello-coh"
)

var (
	t                  = framework.NewTestFramework("coherence")
	generatedNamespace = pkg.GenerateNamespace("hello-coherence")
	expectedPods       = []string{"hello-coh-"}
	host               = ""
	metricsTest        pkg.MetricsTest
)

var beforeSuite = t.BeforeSuiteFunc(func() {
	if !skipDeploy {
		start := time.Now()
		deployCoherenceApp(namespace)
		metrics.Emit(t.Metrics.With("deployment_elapsed_time", time.Since(start).Milliseconds()))

		t.Logs.Info("Container image pull check")
		Eventually(func() bool {
			return pkg.ContainerImagePullWait(namespace, expectedPods)
		}, imagePullWaitTimeout, imagePullPollingInterval).Should(BeTrue())
	}
	if !skipVerify {
		t.Logs.Info("Coherence Application: check expected pod is running")
		Eventually(func() bool {
			result, err := pkg.PodsRunning(namespace, expectedPods)
			if err != nil {
				AbortSuite(fmt.Sprintf("Coherence application pod is not running in the namespace: %v, error: %v", namespace, err))
			}
			return result
		}, longWaitTimeout, longPollingInterval).Should(BeTrue(), "Coherence Application Failed to Deploy: Pods are not ready\"")

		t.Logs.Info("Coherence Application: check expected Services are running")
		Eventually(func() bool {
			result, err := pkg.DoesServiceExist(namespace, coherenceService)
			if err != nil {
				AbortSuite(fmt.Sprintf("App Service %s is not running in the namespace: %v, error: %v", coherenceService, namespace, err))
			}
			return result
		}, longWaitTimeout, longPollingInterval).Should(BeTrue(), "Coherence Application Failed to Deploy: Services are not ready")

		t.Logs.Info("Coherence Application: check expected VirtualService is ready")
		Eventually(func() bool {
			result, err := pkg.DoesVirtualServiceExist(namespace, ingress)
			if err != nil {
				AbortSuite(fmt.Sprintf("App VirtualService %s is not running in the namespace: %v, error: %v", ingress, namespace, err))
			}
			return result
		}, shortWaitTimeout, longPollingInterval).Should(BeTrue(), "Coherence Application Failed to Deploy: VirtualService is not ready")

		t.Logs.Info("Coherence Application: check expected Secret exists")
		Eventually(func() bool {
			result, err := pkg.DoesSecretExist(namespace, helloCreds)
			if err != nil {
				AbortSuite(fmt.Sprintf("App Secret %s does not exist in the namespace: %v, error: %v", helloCreds, namespace, err))
			}
			return result
		}, shortWaitTimeout, longPollingInterval).Should(BeTrue(), "Coherence Application Failed to Deploy: Secret does not exist")

		var err error
		// Get the host from the Istio gateway resource.
		start := time.Now()
		t.Logs.Info("Coherence Application: check expected Gateway is ready")
		Eventually(func() (string, error) {
			host, err = k8sutil.GetHostnameFromGateway(namespace, "")
			return host, err
		}, shortWaitTimeout, shortPollingInterval).Should(Not(BeEmpty()), "Coherence Application Failed to Deploy: Gateway is not ready")
		metrics.Emit(t.Metrics.With("get_host_name_elapsed_time", time.Since(start).Milliseconds()))
	}

	kubeconfigPath, err := k8sutil.GetKubeConfigLocation()
	if err != nil {
		AbortSuite(fmt.Sprintf("Failed to find Kubeconfig location: %v", err))
	}
	metricsTest, err = pkg.NewMetricsTest([]string{kubeconfigPath}, kubeconfigPath, map[string]string{})
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
		undeployCoherenceApp()
	}
})

var _ = AfterSuite(afterSuite)

var _ = t.Describe("Validate deployment of VerrazzanoCoherenceWorkload", Label("f:app-lcm.oam", "f:app-lcm.coherence-workload"), func() {

	t.Context("Ingress", Label("f:mesh.ingress"), func() {
		// Verify the application endpoints
		t.It("Verify '/catalogue' UI endpoint is working", func() {
			if skipVerify {
				Skip(skipVerifications)
			}
			Eventually(func() (*pkg.HTTPResponse, error) {
				url := fmt.Sprintf("https://%s/%s", host, appEndPoint)
				return pkg.GetWebPage(url, host)
			}, shortWaitTimeout, shortPollingInterval).Should(And(pkg.HasStatus(http.StatusOK), pkg.BodyContains(expectedResponse)))
		})
	})

	t.Context("Logging.", Label("f:observability.logging.es"), func() {
		var indexName string
		var err error
		Eventually(func() error {
			indexName, err = pkg.GetOpenSearchAppIndex(namespace)
			return err
		}, shortWaitTimeout, shortPollingInterval).Should(BeNil(), "Expected to get OpenSearch App Index")

		t.It("Verify Opensearch index exists", func() {
			if skipVerify {
				Skip(skipVerifications)
			}
			Eventually(func() bool {
				return pkg.LogIndexFound(indexName)
			}, longWaitTimeout, longPollingInterval).Should(BeTrue(), "Expected to find Opensearch index for Coherence application.")
		})

		t.It("Verify recent Opensearch log record exists", func() {
			if skipVerify {
				Skip(skipVerifications)
			}
			Eventually(func() bool {
				return pkg.LogRecordFound(indexName, time.Now().Add(-24*time.Hour), map[string]string{
					"kubernetes.labels.app_oam_dev\\/component": "hello-coherence",
					"kubernetes.labels.app_oam_dev\\/name":      "hello-appconf",
					"kubernetes.container_name":                 "hello-coherence",
				})
			}, longWaitTimeout, longPollingInterval).Should(BeTrue(), "Expected to find a recent log record.")
		})

		t.It("Verify Coherence log records", func() {
			if skipVerify {
				Skip(skipVerifications)
			}
			Eventually(func() bool {
				return pkg.LogRecordFound(indexName, time.Now().Add(-24*time.Hour), map[string]string{
					"kubernetes.labels.coherenceCluster":                "HelloCoherence",
					"kubernetes.labels.app_oam_dev\\/component.keyword": "hello-coherence",
					"kubernetes.pod_name":                               "hello-coh-0",
					"kubernetes.container_name.keyword":                 "coherence",
				})
			}, longWaitTimeout, longPollingInterval).Should(BeTrue(), "Expected to find a recent log record.")
		})
	})

	t.Context("Metrics", Label("f:observability.monitoring.prom"), func() {
		// Verify Coherence metrics
		t.It("Retrieve Coherence metrics", func() {
			if skipVerify {
				Skip(skipVerifications)
			}
			kubeConfig, err := k8sutil.GetKubeConfigLocation()
			if err != nil {
				Expect(err).To(BeNil(), fmt.Sprintf("Failed to get default kubeconfig path: %s", err.Error()))
			}
			// Coherence metric fix available only from 1.3.0
			if ok, _ := pkg.IsVerrazzanoMinVersion("1.3.0", kubeConfig); ok {
				Eventually(func() bool {
					return metricsTest.MetricsExist("vendor:coherence_service_messages_local", map[string]string{"role": "HelloCoherenceRole"})
				}, longWaitTimeout, longPollingInterval).Should(BeTrue())
			}
		})

		t.It("Retrieve application metrics", func() {
			if skipVerify {
				Skip(skipVerifications)
			}
			pkg.Concurrently(
				func() {
					Eventually(func() bool {
						return metricsTest.MetricsExist("base_jvm_uptime_seconds", map[string]string{"app_oam_dev_name": "hello-appconf"})
					}, longWaitTimeout, longPollingInterval).Should(BeTrue())
				},
				func() {
					Eventually(func() bool {
						return metricsTest.MetricsExist("vendor_requests_count_total", map[string]string{"app_oam_dev_name": "hello-appconf"})
					}, longWaitTimeout, longPollingInterval).Should(BeTrue())
				},
			)
		})
	})
})

func deployCoherenceApp(namespace string) {
	t.Logs.Info("Deploy Coherence application")

	t.Logs.Info("Create namespace")
	Eventually(func() (*v1.Namespace, error) {
		nsLabels := map[string]string{
			"verrazzano-managed": "true"}
		return pkg.CreateNamespace(namespace, nsLabels)
	}, shortWaitTimeout, shortPollingInterval).ShouldNot(BeNil())

	t.Logs.Info("Create component resources")
	Eventually(func() error {
		file, err := pkg.FindTestDataFile(compConfiguration)
		if err != nil {
			return err
		}
		return resource.CreateOrUpdateResourceFromFileInGeneratedNamespace(file, namespace)
	}, shortWaitTimeout, shortPollingInterval, "Failed to create component resources for Coherence application").ShouldNot(HaveOccurred())

	t.Logs.Info("Create application resources")
	Eventually(func() error {
		file, err := pkg.FindTestDataFile(appConfiguration)
		if err != nil {
			return err
		}
		return resource.CreateOrUpdateResourceFromFileInGeneratedNamespace(file, namespace)
	}, shortWaitTimeout, shortPollingInterval).ShouldNot(HaveOccurred())
}

func undeployCoherenceApp() {
	t.Logs.Info("Undeploy Coherence application")
	t.Logs.Info("Delete application")
	start := time.Now()
	Eventually(func() error {
		file, err := pkg.FindTestDataFile(appConfiguration)
		if err != nil {
			return err
		}
		return resource.DeleteResourceFromFileInGeneratedNamespace(file, namespace)
	}, shortWaitTimeout, shortPollingInterval).ShouldNot(HaveOccurred())

	t.Logs.Info("Delete component")
	Eventually(func() error {
		file, err := pkg.FindTestDataFile(compConfiguration)
		if err != nil {
			return err
		}
		return resource.DeleteResourceFromFileInGeneratedNamespace(file, namespace)
	}, shortWaitTimeout, shortPollingInterval).ShouldNot(HaveOccurred())

	t.Logs.Info("Wait for pod to terminate")
	Eventually(func() bool {
		podsTerminated, _ := pkg.PodsNotRunning(namespace, expectedPods)
		return podsTerminated
	}, shortWaitTimeout, shortPollingInterval).Should(BeTrue())

	t.Logs.Info("Delete namespace")
	Eventually(func() error {
		return pkg.DeleteNamespace(namespace)
	}, shortWaitTimeout, shortPollingInterval).ShouldNot(HaveOccurred())

	t.Logs.Info("Wait for namespace finalizer to be removed")
	Eventually(func() bool {
		return pkg.CheckNamespaceFinalizerRemoved(namespace)
	}, shortWaitTimeout, shortPollingInterval).Should(BeTrue())

	t.Logs.Info("Wait for namespace deletion")
	Eventually(func() bool {
		_, err := pkg.GetNamespace(namespace)
		return err != nil && errors.IsNotFound(err)
	}, shortWaitTimeout, shortPollingInterval).Should(BeTrue())

	metrics.Emit(t.Metrics.With("undeployment_elapsed_time", time.Since(start).Milliseconds()))
}
