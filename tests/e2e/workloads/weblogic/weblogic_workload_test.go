// Copyright (c) 2022, 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package weblogic

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
	longPollingInterval      = 20 * time.Second
	imagePullWaitTimeout     = 40 * time.Minute
	imagePullPollingInterval = 30 * time.Second

	appConfiguration  = "tests/testdata/test-applications/weblogic/hello-weblogic/hello-wls-app.yaml"
	compConfiguration = "tests/testdata/test-applications/weblogic/hello-weblogic/hello-wls-comp.yaml"

	appURL         = "hello/weblogic/greetings/message"
	welcomeMessage = "Hello WebLogic"

	wlsUser        = "weblogic"
	wlDomain       = "hellodomain"
	wlsAdminServer = "hellodomain-adminserver"
	trait          = "hello-domain-trait"

	helloDomainRepoCreds     = "hellodomain-repo-credentials"
	helloDomainWeblogicCreds = "hellodomain-weblogic-credentials"
)

var (
	t                  = framework.NewTestFramework("weblogicworkload")
	generatedNamespace = pkg.GenerateNamespace("hello-wls")
	expectedPods       = []string{wlsAdminServer}
	host               = ""
	metricsTest        pkg.MetricsTest
)

var beforeSuite = t.BeforeSuiteFunc(func() {
	if !skipDeploy {
		start := time.Now()
		deployWebLogicApp(namespace)
		metrics.Emit(t.Metrics.With("deployment_elapsed_time", time.Since(start).Milliseconds()))

		t.Logs.Info("Container image pull check")
		Eventually(func() bool {
			return pkg.ContainerImagePullWait(namespace, expectedPods)
		}, imagePullWaitTimeout, imagePullPollingInterval).Should(BeTrue())
	}

	t.Logs.Info("WebLogic Application: check expected admin server pod is running")
	Eventually(func() bool {
		result, err := pkg.PodsRunning(namespace, expectedPods)
		if err != nil {
			AbortSuite(fmt.Sprintf("WebLogic admin server pod is not running in the namespace: %v, error: %v", namespace, err))
		}
		return result
	}, shortWaitTimeout, longPollingInterval).Should(BeTrue(), "Failed to deploy the WebLogic Application: Admin server pod is not ready")

	t.Logs.Info("WebLogic Application: check expected VirtualService is ready")
	Eventually(func() bool {
		result, err := pkg.DoesVirtualServiceExist(namespace, trait)
		if err != nil {
			AbortSuite(fmt.Sprintf("WebLogic VirtualService %s is not running in the namespace: %v, error: %v", trait, namespace, err))
		}
		return result
	}, shortWaitTimeout, longPollingInterval).Should(BeTrue(), "Failed to deploy the WebLogic Application: VirtualService is not ready")

	t.Logs.Info("WebLogic Application: check expected Secrets exist")
	Eventually(func() bool {
		result, err := pkg.DoesSecretExist(namespace, helloDomainWeblogicCreds)
		if err != nil {
			AbortSuite(fmt.Sprintf("WebLogic Secret %s does not exist in the namespace: %v, error: %v", helloDomainWeblogicCreds, namespace, err))
		}
		return result
	}, shortWaitTimeout, longPollingInterval).Should(BeTrue(), "Failed to deploy the WebLogic Application: Secret does not exist")

	Eventually(func() bool {
		result, err := pkg.DoesSecretExist(namespace, helloDomainRepoCreds)
		if err != nil {
			AbortSuite(fmt.Sprintf("WebLogic Secret %s does not exist in the namespace: %v, error: %v", helloDomainRepoCreds, namespace, err))
		}
		return result
	}, shortWaitTimeout, longPollingInterval).Should(BeTrue(), "Failed to deploy the WebLogic Application: Secret does not exist")

	var err error
	// Get the host from the Istio gateway resource.
	start := time.Now()
	t.Logs.Info("WebLogic Application: check expected Gateway is ready")
	Eventually(func() (string, error) {
		host, err = k8sutil.GetHostnameFromGateway(namespace, "")
		return host, err
	}, shortWaitTimeout, shortPollingInterval).Should(Not(BeEmpty()), "Failed to deploy the WebLogic Application: Gateway is not ready")
	metrics.Emit(t.Metrics.With("get_host_name_elapsed_time", time.Since(start).Milliseconds()))

	kubeconfigPath, err := k8sutil.GetKubeConfigLocation()
	metricsTest, err = pkg.NewMetricsTest(kubeconfigPath, map[string]string{})
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
		dump.CaptureContainerLogs(namespace, wlsAdminServer, "weblogic-server", "/scratch/logs/hello-domain")
		dump.ExecuteBugReport(namespace)
	}
	if !skipUndeploy {
		undeployWebLogicApp()
	}
})

var _ = AfterSuite(afterSuite)

func deployWebLogicApp(namespace string) {
	t.Logs.Info("Deploy WebLogic application")
	wlsPass := pkg.GetRequiredEnvVarOrFail("WEBLOGIC_PSW")
	regServ := pkg.GetRequiredEnvVarOrFail("OCR_REPO")
	regUser := pkg.GetRequiredEnvVarOrFail("OCR_CREDS_USR")
	regPass := pkg.GetRequiredEnvVarOrFail("OCR_CREDS_PSW")

	t.Logs.Info("Create namespace")
	Eventually(func() (*v1.Namespace, error) {
		nsLabels := map[string]string{
			"verrazzano-managed": "true",
			"istio-injection":    istioInjection}
		return pkg.CreateNamespace(namespace, nsLabels)
	}, shortWaitTimeout, shortPollingInterval).ShouldNot(BeNil())

	t.Logs.Info("Create docker-registry secret to enable pulling image from the registry")
	Eventually(func() (*v1.Secret, error) {
		return pkg.CreateDockerSecret(namespace, helloDomainRepoCreds, regServ, regUser, regPass)
	}, shortWaitTimeout, shortPollingInterval).ShouldNot(BeNil())

	t.Logs.Info("Create secret for the WebLogic domain")
	Eventually(func() (*v1.Secret, error) {
		return pkg.CreateCredentialsSecret(namespace, helloDomainWeblogicCreds, wlsUser, wlsPass, nil)
	}, shortWaitTimeout, shortPollingInterval).ShouldNot(BeNil())

	// Note: creating the app config first to verify that default metrics traits are created properly if the app config exists before the components
	t.Logs.Info("Create application resources")
	Eventually(func() error {
		file, err := pkg.FindTestDataFile(appConfiguration)
		if err != nil {
			return err
		}
		return resource.CreateOrUpdateResourceFromFileInGeneratedNamespace(file, namespace)
	}, shortWaitTimeout, shortPollingInterval).ShouldNot(HaveOccurred())

	t.Logs.Info("Create component resources")
	Eventually(func() error {
		file, err := pkg.FindTestDataFile(compConfiguration)
		if err != nil {
			return err
		}
		return resource.CreateOrUpdateResourceFromFileInGeneratedNamespace(file, namespace)
	}, shortWaitTimeout, shortPollingInterval, "Failed to create component resources for WebLogic application").ShouldNot(HaveOccurred())
}

func undeployWebLogicApp() {
	t.Logs.Info("Undeploy WebLogic application")
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

var _ = t.Describe("Validate deployment of VerrazzanoWebLogicWorkload", Label("f:app-lcm.oam", "f:app-lcm.weblogic-workload"), func() {

	t.Context("Ingress", Label("f:mesh.ingress"), FlakeAttempts(8), func() {
		// Verify the application endpoint is working.
		// GIVEN the sample WebLogic app is deployed
		// WHEN the application endpoint is accessed
		// THEN the expected returned page should contain an expected value.
		t.It("Verify application endpoint is working", func() {
			Eventually(func() (*pkg.HTTPResponse, error) {
				url := fmt.Sprintf("https://%s/%s", host, appURL)
				return pkg.GetWebPage(url, host)
			}, shortWaitTimeout, shortPollingInterval).Should(And(pkg.HasStatus(http.StatusOK), pkg.BodyEquals(welcomeMessage)))
		})
	})

	t.Context("Metrics", Label("f:observability.monitoring.prom"), FlakeAttempts(5), func() {
		kubeconfigPath, err := k8sutil.GetKubeConfigLocation()
		if err != nil {
			Expect(err).To(BeNil(), fmt.Sprintf("Failed to get default kubeconfig path: %s", err.Error()))
		}
		ok, _ := pkg.IsVerrazzanoMinVersion("1.4.0", kubeconfigPath)
		// Verify application Prometheus scraped targets
		// GIVEN the sample WebLogic app is deployed
		// WHEN the application configuration uses a default metrics trait
		// THEN confirm that all the scrape targets are healthy
		t.It("Verify all scrape targets are healthy for the application", func() {
			Eventually(func() (bool, error) {
				var componentNames = []string{"hello-domain"}
				return pkg.ScrapeTargetsHealthy(pkg.GetScrapePools(namespace, "hello-appconf", componentNames, ok))
			}, shortWaitTimeout, shortPollingInterval).Should(BeTrue())
		})

		// Verify Istio Prometheus scraped metrics
		// GIVEN the sample WebLogic app is deployed
		// WHEN the application configuration is deployed
		// THEN confirm that Istio metrics are being collected
		if istioInjection == "enabled" {
			t.It("Retrieve Istio Prometheus scraped metrics", func() {
				pkg.Concurrently(
					func() {
						Eventually(func() bool {
							return metricsTest.MetricsExist("istio_tcp_received_bytes_total", map[string]string{"destination_canonical_service": "hello-domain"})
						}, shortWaitTimeout, shortPollingInterval).Should(BeTrue())
					},
					func() {
						Eventually(func() bool {
							return metricsTest.MetricsExist("envoy_cluster_http2_pending_send_bytes", map[string]string{"pod_name": wlsAdminServer})
						}, shortWaitTimeout, longPollingInterval).Should(BeTrue())
					},
				)
			})
		}
	})

	t.Context("WebLogic logging", Label("f:observability.logging.es"), func() {
		var indexName string
		var err error
		Eventually(func() error {
			indexName, err = pkg.GetOpenSearchAppIndex(namespace)
			return err
		}, shortWaitTimeout, shortPollingInterval).Should(BeNil(), "Expected to get OpenSearch App Index")

		// GIVEN a WebLogic application with logging enabled
		// WHEN the Opensearch index is retrieved
		// THEN verify that it is found
		t.It("Verify Opensearch index exists", func() {
			Eventually(func() bool {
				return pkg.LogIndexFound(indexName)
			}, shortWaitTimeout, shortPollingInterval).Should(BeTrue(), "Expected to find log index "+indexName)
		})
		pkg.Concurrently(
			// GIVEN a WebLogic application with logging enabled
			// WHEN the log records are retrieved from the Opensearch index
			// THEN verify that a recent log record of hellodomain-adminserver stdout is found
			func() {
				t.It("Verify recent hellodomain-adminserver log record exists", func() {
					Eventually(func() bool {
						return pkg.LogRecordFound(indexName, time.Now().Add(-24*time.Hour), map[string]string{
							"kubernetes.labels.weblogic_domainUID":  wlDomain,
							"kubernetes.labels.weblogic_serverName": "AdminServer",
							"kubernetes.pod_name":                   wlsAdminServer,
							"kubernetes.container_name":             "weblogic-server",
						})
					}, shortWaitTimeout, shortPollingInterval).Should(BeTrue(), "Expected to find a recent log record")
				})
			},

			// GIVEN a WebLogic application with logging enabled
			// WHEN the log records are retrieved from the Opensearch index
			// THEN verify that a recent log record of hellodomain-adminserver log file is found
			func() {
				t.It("Verify recent hellodomain-adminserver log record exists", func() {
					Eventually(func() bool {
						return pkg.LogRecordFound(indexName, time.Now().Add(-24*time.Hour), map[string]string{
							"kubernetes.labels.weblogic_domainUID":  wlDomain,
							"kubernetes.labels.weblogic_serverName": "AdminServer",
							"kubernetes.pod_name":                   wlsAdminServer,
							"kubernetes.container_name":             "fluentd-stdout-sidecar",
						})
					}, shortWaitTimeout, shortPollingInterval).Should(BeTrue(), "Expected to find a recent log record")
				})
			},
			// GIVEN a WebLogic application with logging enabled
			// WHEN the log records are retrieved from the Opensearch index
			// THEN verify that a recent pattern-matched log record of hellodomain-adminserver stdout is found
			func() {
				t.It("Verify recent pattern-matched AdminServer log record exists", func() {
					Eventually(func() bool {
						return pkg.FindLog(indexName,
							[]pkg.Match{
								{Key: "kubernetes.container_name.keyword", Value: "fluentd-stdout-sidecar"},
								{Key: "subSystem.keyword", Value: "WorkManager"},
								{Key: "serverName.keyword", Value: wlsAdminServer},
								{Key: "serverName2.keyword", Value: "AdminServer"},
								{Key: "message", Value: "standby threads"}},
							[]pkg.Match{})
					}, shortWaitTimeout, longPollingInterval).Should(BeTrue(), "Expected to find a recent log record")
				})
			},
			// GIVEN a WebLogic application with logging enabled
			// WHEN the log records are retrieved from the Opensearch index
			// THEN verify that a recent pattern-matched log record of hellodomain-adminserver stdout is found
			func() {
				t.It("Verify recent pattern-matched AdminServer log record exists", func() {
					Eventually(func() bool {
						return pkg.FindLog(indexName,
							[]pkg.Match{
								{Key: "kubernetes.container_name.keyword", Value: "fluentd-stdout-sidecar"},
								{Key: "subSystem", Value: "WorkManager"},
								{Key: "serverName", Value: wlsAdminServer},
								{Key: "serverName2", Value: "AdminServer"},
								{Key: "message", Value: "Self-tuning"}},
							[]pkg.Match{})
					}, shortWaitTimeout, longPollingInterval).Should(BeTrue(), "Expected to find a recent log record")
				})
			},
		)
	})
})
