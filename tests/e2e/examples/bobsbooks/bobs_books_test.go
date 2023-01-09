// Copyright (c) 2021, 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package bobsbooks

import (
	"fmt"
	dump "github.com/verrazzano/verrazzano/tests/e2e/pkg/test/clusterdump"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/verrazzano/verrazzano/pkg/k8s/resource"
	"github.com/verrazzano/verrazzano/pkg/k8sutil"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg"
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

	// application specific constants
	robertCoh            = "robert-coh"
	bobsBookStore        = "bobs-bookstore"
	robertsCoherence     = "roberts-coherence"
	bobbysFrontEnd       = "bobbys-front-end"
	managedServer1       = "managed-server1"
	fluentdStdoutSidecar = "fluentd-stdout-sidecar"

	// various labels
	k8sLabelDomainUID        = "kubernetes.labels.weblogic_domainUID"
	k8sLabelWLServerName     = "kubernetes.labels.weblogic_serverName"
	k8sPodName               = "kubernetes.pod_name"
	k8sLabelContainerName    = "kubernetes.container_name"
	K8sLabelCoherenceCluster = "kubernetes.labels.coherenceCluster"
)

var (
	t                  = framework.NewTestFramework("bobsbooks")
	generatedNamespace = pkg.GenerateNamespace("bobs-books")
	expectedPods       = []string{
		"bobbys-front-end-adminserver",
		"bobs-bookstore-adminserver",
		"bobbys-coherence-0",
		"roberts-coherence-0",
		"roberts-coherence-1",
		"bobbys-helidon-stock-application",
		"robert-helidon",
		"mysql"}
	appName = "bobs-books"
)

var beforeSuite = t.BeforeSuiteFunc(func() {
	if !skipDeploy {
		start := time.Now()
		deployBobsBooksExample(namespace)
		metrics.Emit(t.Metrics.With("deployment_elapsed_time", time.Since(start).Milliseconds()))
	}
	t.Logs.Info("Container image pull check")
	Eventually(func() bool {
		return pkg.ContainerImagePullWait(namespace, expectedPods)
	}, imagePullWaitTimeout, imagePullPollingInterval).Should(BeTrue())
	t.Logs.Info("Bobs Books Application expected pods running check.")
	Eventually(func() bool {
		result, err := pkg.PodsRunning(namespace, expectedPods)
		if err != nil {
			AbortSuite(fmt.Sprintf("One or more pods are not running in the namespace: %v, error: %v", namespace, err))
		}
		return result
	}, longWaitTimeout, longPollingInterval).Should(BeTrue(), "Bobs Books Application Failed to Deploy")
	beforeSuitePassed = true
})

var failed = false
var beforeSuitePassed = false
var _ = t.AfterEach(func() {
	failed = failed || CurrentSpecReport().Failed()
})

var afterSuite = t.AfterSuiteFunc(func() {
	if failed || !beforeSuitePassed {
		// bobbys frontend
		dump.CaptureContainerLogs(namespace, "bobbys-front-end-adminserver", "weblogic-server", "/scratch/logs/bobbys-front-end")
		dump.CaptureContainerLogs(namespace, "bobbys-front-end-managed-server1", "weblogic-server", "/scratch/logs/bobbys-front-end")
		// Bobs Bookstore
		dump.CaptureContainerLogs(namespace, "bobs-bookstore-adminserver", "weblogic-server", "/scratch/logs/bobs-bookstore")
		dump.CaptureContainerLogs(namespace, "bobs-bookstore-managed-server1", "weblogic-server", "/scratch/logs/bobs-bookstore")
		dump.ExecuteBugReport(namespace)
	}
	if !skipUndeploy {
		undeployBobsBooksExample()
	}
})

var _ = AfterSuite(afterSuite)
var _ = BeforeSuite(beforeSuite)

func deployBobsBooksExample(namespace string) {
	t.Logs.Info("Deploy BobsBooks example")
	wlsUser := "weblogic"
	wlsPass := pkg.GetRequiredEnvVarOrFail("WEBLOGIC_PSW")
	dbPass := pkg.GetRequiredEnvVarOrFail("DATABASE_PSW")
	regServ := pkg.GetRequiredEnvVarOrFail("OCR_REPO")
	regUser := pkg.GetRequiredEnvVarOrFail("OCR_CREDS_USR")
	regPass := pkg.GetRequiredEnvVarOrFail("OCR_CREDS_PSW")

	start := time.Now()
	t.Logs.Info("Create namespace")
	Eventually(func() (*v1.Namespace, error) {
		nsLabels := map[string]string{
			"verrazzano-managed": "true",
			"istio-injection":    istioInjection}
		return pkg.CreateNamespace(namespace, nsLabels)
	}, shortWaitTimeout, shortPollingInterval).ShouldNot(BeNil())

	t.Logs.Info("Create Docker repository secret")
	Eventually(func() (*v1.Secret, error) {
		return pkg.CreateDockerSecret(namespace, "bobs-books-repo-credentials", regServ, regUser, regPass)
	}, shortWaitTimeout, shortPollingInterval).ShouldNot(BeNil())

	t.Logs.Info("Create Bobbys front end WebLogic credentials secret")
	Eventually(func() (*v1.Secret, error) {
		return pkg.CreateCredentialsSecret(namespace, "bobbys-front-end-weblogic-credentials", wlsUser, wlsPass, nil)
	}, shortWaitTimeout, shortPollingInterval).ShouldNot(BeNil())

	t.Logs.Info("Create Bobs Bookstore WebLogic credentials secret")
	Eventually(func() (*v1.Secret, error) {
		return pkg.CreateCredentialsSecret(namespace, "bobs-bookstore-weblogic-credentials", wlsUser, wlsPass, nil)
	}, shortWaitTimeout, shortPollingInterval).ShouldNot(BeNil())

	t.Logs.Info("Create database credentials secret")
	Eventually(func() (*v1.Secret, error) {
		m := map[string]string{"password": dbPass, "username": wlsUser, "url": "jdbc:mysql://mysql:3306/books"}
		return pkg.CreateCredentialsSecretFromMap(namespace, "mysql-credentials", m, nil)
	}, shortWaitTimeout, shortPollingInterval).ShouldNot(BeNil())

	// Note: creating the app config first to verify that default metrics traits are created properly if the app config exists before the components
	t.Logs.Info("Create application resources")
	Eventually(func() error {
		file, err := pkg.FindTestDataFile("examples/bobs-books/bobs-books-app.yaml")
		if err != nil {
			return err
		}
		return resource.CreateOrUpdateResourceFromFileInGeneratedNamespace(file, namespace)
	}, shortWaitTimeout, shortPollingInterval).ShouldNot(HaveOccurred())

	t.Logs.Info("Create component resources")
	Eventually(func() error {
		file, err := pkg.FindTestDataFile("examples/bobs-books/bobs-books-comp.yaml")
		if err != nil {
			return err
		}
		return resource.CreateOrUpdateResourceFromFileInGeneratedNamespace(file, namespace)
	}, shortWaitTimeout, shortPollingInterval, "Failed to create Bobs Books component resources").ShouldNot(HaveOccurred())
	metrics.Emit(t.Metrics.With("deployment_elapsed_time", time.Since(start).Milliseconds()))
}

func undeployBobsBooksExample() {
	t.Logs.Info("Undeploy BobsBooks example")
	t.Logs.Info("Delete application")
	start := time.Now()
	Eventually(func() error {
		file, err := pkg.FindTestDataFile("examples/bobs-books/bobs-books-app.yaml")
		if err != nil {
			return err
		}
		return resource.DeleteResourceFromFileInGeneratedNamespace(file, namespace)
	}, shortWaitTimeout, shortPollingInterval).ShouldNot(HaveOccurred())

	t.Logs.Info("Delete components")
	Eventually(func() error {
		file, err := pkg.FindTestDataFile("examples/bobs-books/bobs-books-comp.yaml")
		if err != nil {
			return err
		}
		return resource.DeleteResourceFromFileInGeneratedNamespace(file, namespace)
	}, shortWaitTimeout, shortPollingInterval).ShouldNot(HaveOccurred())

	t.Logs.Info("Wait for pods to terminate")
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

var _ = t.Describe("Bobs Books test", Label("f:app-lcm.oam",
	"f:app-lcm.helidon-workload",
	"f:app-lcm.weblogic-workload",
	"f:app-lcm.coherence-workload"), func() {

	var host = ""
	var err error
	// Get the host from the Istio gateway resource.
	// GIVEN the Istio gateway for the bobs-books namespace
	// WHEN GetHostnameFromGateway is called
	// THEN return the host name found in the gateway.
	t.BeforeEach(func() {
		start := time.Now()
		Eventually(func() (string, error) {
			host, err = k8sutil.GetHostnameFromGateway(namespace, "")
			return host, err
		}, shortWaitTimeout, shortPollingInterval).Should(Not(BeEmpty()))
		metrics.Emit(t.Metrics.With("get_host_name_elapsed_time", time.Since(start).Milliseconds()))
	})
	t.Context("Ingress.", Label("f:mesh.ingress"), FlakeAttempts(8), func() {
		// Verify the application endpoint is working.
		// GIVEN the Bobs Books app is deployed
		// WHEN the roberts-books UI is accessed
		// THEN the expected returned page should contain an expected value.
		t.It("Verify roberts-books UI endpoint is working.", func() {
			Eventually(func() (*pkg.HTTPResponse, error) {
				url := fmt.Sprintf("https://%s", host)
				return pkg.GetWebPage(url, host)
			}, shortWaitTimeout, shortPollingInterval).Should(And(pkg.HasStatus(200), pkg.BodyContains("Robert's Books")))
		})
		// Verify the application endpoint is working.
		// GIVEN the Bobs Books app is deployed
		// WHEN the bobbys-books UI is accessed
		// THEN the expected returned page should contain an expected value.
		t.It("Verify bobbys-books UI endpoint is working.", func() {
			Eventually(func() (*pkg.HTTPResponse, error) {
				url := fmt.Sprintf("https://%s/bobbys-front-end/", host)
				return pkg.GetWebPage(url, host)
			}, longWaitTimeout, longPollingInterval).Should(And(pkg.HasStatus(200), pkg.BodyContains("Bobby's Books")))
		})
		// Verify the application endpoint is working even without trailing slash.
		// GIVEN the Bobs Books app is deployed
		// WHEN the bobbys-books UI is accessed
		// THEN the expected returned page should contain an expected value.
		t.It("Verify bobbys-books UI endpoint without trailing slash is working.", func() {
			Eventually(func() (*pkg.HTTPResponse, error) {
				url := fmt.Sprintf("https://%s/bobbys-front-end", host)
				return pkg.GetWebPage(url, host)
			}, longWaitTimeout, longPollingInterval).Should(And(pkg.HasStatus(200), pkg.BodyContains("Bobby's Books")))
		})
		// Verify the application endpoint is working.
		// GIVEN the Bobs Books app is deployed
		// WHEN the bobs-orders UI is accessed
		// THEN the expected returned page should contain an expected value.
		t.It("Verify bobs-orders UI endpoint for orders is working.", func() {
			Eventually(func() (*pkg.HTTPResponse, error) {
				url := fmt.Sprintf("https://%s/bobs-bookstore-order-manager/orders", host)
				return pkg.GetWebPage(url, host)
			}, longWaitTimeout, longPollingInterval).Should(And(pkg.HasStatus(200), pkg.BodyContains("Bob's Order Manager")))
		})
		t.It("Verify bobs-orders UI endpoint for books is working.", func() {
			Eventually(func() (*pkg.HTTPResponse, error) {
				url := fmt.Sprintf("https://%s/bobs-bookstore-order-manager/books", host)
				return pkg.GetWebPage(url, host)
			}, longWaitTimeout, longPollingInterval).Should(And(pkg.HasStatus(200), pkg.BodyContains("Bob's Order Manager")))
		})
	})
	t.Context("Metrics.", Label("f:observability.monitoring.prom"), FlakeAttempts(5), func() {
		// Verify application Prometheus scraped targets
		// GIVEN a deployed Bob's Books application
		// WHEN the application configuration uses a default metrics trait
		// THEN confirm that all the scrape targets are healthy
		t.It("Verify all scrape targets are healthy for the application", func() {
			Eventually(func() (bool, error) {
				var componentNames = []string{"bobby-coh", "bobby-helidon", "bobby-wls", "bobs-mysql-deployment", "bobs-mysql-service", "bobs-orders-wls", robertCoh, "robert-helidon"}
				return pkg.ScrapeTargetsHealthy(pkg.GetScrapePools(namespace, "bob-books", componentNames))
			}, shortWaitTimeout, shortPollingInterval).Should(BeTrue())
		})
		// Verify Istio Prometheus scraped metrics
		// GIVEN a deployed Bob's Books application
		// WHEN the application configuration is deployed
		// THEN confirm that Istio metrics are being collected
		t.It("Retrieve Istio Prometheus scraped metrics", func() {
			pkg.Concurrently(
				func() {
					Eventually(func() bool {
						return pkg.MetricsExist("istio_tcp_received_bytes_total", "destination_canonical_service", "bobbys-helidon-stock-application")
					}, shortWaitTimeout, shortPollingInterval).Should(BeTrue())
				},
			)
		})
	})
	t.Context("WebLogic logging.", Label("f:observability.logging.es"), func() {
		var bobsIndexName string
		Eventually(func() error {
			bobsIndexName, err = pkg.GetOpenSearchAppIndex(namespace)
			return err
		}, shortWaitTimeout, shortPollingInterval).Should(BeNil(), "Expected to get OpenSearch App Index")

		// GIVEN a WebLogic application with logging enabled
		// WHEN the Opensearch index is retrieved
		// THEN verify that it is found
		t.It("Verify Opensearch index exists", func() {
			Eventually(func() bool {
				return pkg.LogIndexFound(bobsIndexName)
			}, shortWaitTimeout, shortPollingInterval).Should(BeTrue(), "Expected to find log index "+bobsIndexName)
		})
		pkg.Concurrently(
			// GIVEN a WebLogic application with logging enabled
			// WHEN the log records are retrieved from the Opensearch index
			// THEN verify that a recent log record of bobbys-front-end-adminserver stdout is found
			func() {
				t.It("Verify recent bobbys-front-end-adminserver log record exists", func() {
					Eventually(func() bool {
						return pkg.LogRecordFound(bobsIndexName, time.Now().Add(-24*time.Hour), map[string]string{
							k8sLabelDomainUID:     bobbysFrontEnd,
							k8sLabelWLServerName:  "AdminServer",
							k8sPodName:            "bobbys-front-end-adminserver",
							k8sLabelContainerName: "weblogic-server",
						})
					}, shortWaitTimeout, shortPollingInterval).Should(BeTrue(), "Expected to find a recent log record")
				})
			},
			// GIVEN a WebLogic application with logging enabled
			// WHEN the log records are retrieved from the Opensearch index
			// THEN verify that a recent log record of bobbys-front-end-adminserver log file is found
			func() {
				t.It("Verify recent bobbys-front-end-adminserver log record exists", func() {
					Eventually(func() bool {
						return pkg.LogRecordFound(bobsIndexName, time.Now().Add(-24*time.Hour), map[string]string{
							k8sLabelDomainUID:     bobbysFrontEnd,
							k8sLabelWLServerName:  "AdminServer",
							k8sPodName:            "bobbys-front-end-adminserver",
							k8sLabelContainerName: fluentdStdoutSidecar,
						})
					}, shortWaitTimeout, shortPollingInterval).Should(BeTrue(), "Expected to find a recent log record")
				})
			},
			// GIVEN a WebLogic application with logging enabled
			// WHEN the log records are retrieved from the Opensearch index
			// THEN verify that a recent log record of bobbys-front-end-managed-server stdout is found
			func() {
				t.It("Verify recent bobbys-front-end-managed-server1 log record exists", func() {
					Eventually(func() bool {
						return pkg.LogRecordFound(bobsIndexName, time.Now().Add(-24*time.Hour), map[string]string{
							k8sLabelDomainUID:     bobbysFrontEnd,
							k8sLabelWLServerName:  managedServer1,
							k8sPodName:            "bobbys-front-end-managed-server1",
							k8sLabelContainerName: "weblogic-server",
						})
					}, shortWaitTimeout, shortPollingInterval).Should(BeTrue(), "Expected to find a recent log record")
				})
			},

			// GIVEN a WebLogic application with logging enabled
			// WHEN the log records are retrieved from the Opensearch index
			// THEN verify that a recent pattern-matched log record of bobbys-front-end-adminserver stdout is found
			func() {
				t.It("Verify recent pattern-matched AdminServer log record exists", func() {
					Eventually(func() bool {
						return pkg.FindLog(bobsIndexName,
							[]pkg.Match{
								{Key: "kubernetes.container_name.keyword", Value: fluentdStdoutSidecar},
								{Key: "subSystem.keyword", Value: "WorkManager"},
								{Key: "serverName.keyword", Value: "bobbys-front-end-adminserver"},
								{Key: "serverName2.keyword", Value: "AdminServer"},
								{Key: "message", Value: "standby threads"}},
							[]pkg.Match{})
					}, longWaitTimeout, longPollingInterval).Should(BeTrue(), "Expected to find a recent log record")
				})
			},
			// GIVEN a WebLogic application with logging enabled
			// WHEN the log records are retrieved from the Opensearch index
			// THEN verify that a recent pattern-matched log record of bobbys-front-end-adminserver stdout is found
			func() {
				t.It("Verify recent pattern-matched AdminServer log record exists", func() {
					Eventually(func() bool {
						return pkg.FindLog(bobsIndexName,
							[]pkg.Match{
								{Key: "kubernetes.container_name.keyword", Value: fluentdStdoutSidecar},
								{Key: "subSystem", Value: "WorkManager"},
								{Key: "serverName", Value: "bobbys-front-end-adminserver"},
								{Key: "serverName2", Value: "AdminServer"},
								{Key: "message", Value: "Self-tuning"}},
							[]pkg.Match{})
					}, longWaitTimeout, longPollingInterval).Should(BeTrue(), "Expected to find a recent log record")
				})
			},
			// GIVEN a WebLogic application with logging enabled
			// WHEN the log records are retrieved from the Opensearch index
			// THEN verify that a recent log record of bobbys-front-end-managed-server log file is found
			func() {
				t.It("Verify recent bobbys-front-end-managed-server1 log record exists", func() {
					Eventually(func() bool {
						return pkg.FindLog(bobsIndexName,
							[]pkg.Match{
								{Key: "kubernetes.container_name.keyword", Value: fluentdStdoutSidecar},
								{Key: k8sLabelDomainUID, Value: bobbysFrontEnd},
								{Key: k8sLabelWLServerName, Value: managedServer1},
								{Key: "messageID", Value: "BEA-"},         //matches BEA-*
								{Key: "message", Value: "Tunneling Ping"}, //"Tunneling Ping" in last line
								{Key: "serverName", Value: "bobbys-front-end-managed-server1"},
								{Key: "subSystem.keyword", Value: "RJVM"}},
							[]pkg.Match{})
					}, longWaitTimeout, longPollingInterval).Should(BeTrue(), "Expected to find a recent log record")
				})
			},
			// GIVEN a WebLogic application with logging enabled
			// WHEN the log records are retrieved from the Opensearch index
			// THEN verify that a recent pattern-matched log record of bobbys-front-end-managed-server stdout is found
			func() {
				t.It("Verify recent pattern-matched managed-server log record exists", func() {
					Eventually(func() bool {
						return pkg.FindLog(bobsIndexName,
							[]pkg.Match{
								{Key: "kubernetes.container_name.keyword", Value: fluentdStdoutSidecar},
								{Key: "subSystem.keyword", Value: "WorkManager"},
								{Key: "serverName.keyword", Value: "bobbys-front-end-managed-server1"},
								{Key: "serverName2.keyword", Value: managedServer1},
								{Key: "message", Value: "standby threads"}},
							[]pkg.Match{})
					}, longWaitTimeout, longPollingInterval).Should(BeTrue(), "Expected to find a recent log record")
				})
			},
			// GIVEN a WebLogic application with logging enabled
			// WHEN the log records are retrieved from the Opensearch index
			// THEN verify that a recent pattern-matched log record of bobbys-front-end-managed-server stdout is found
			func() {
				t.It("Verify recent pattern-matched managed-server log record exists", func() {
					Eventually(func() bool {
						return pkg.FindLog(bobsIndexName,
							[]pkg.Match{
								{Key: "kubernetes.container_name.keyword", Value: fluentdStdoutSidecar},
								{Key: "subSystem", Value: "WorkManager"},
								{Key: "serverName", Value: "bobbys-front-end-managed-server1"},
								{Key: "serverName2", Value: managedServer1},
								{Key: "message", Value: "Self-tuning"}},
							[]pkg.Match{})
					}, longWaitTimeout, longPollingInterval).Should(BeTrue(), "Expected to find a recent log record")
				})
			},
		)
		pkg.Concurrently(
			// GIVEN a WebLogic application with logging enabled
			// WHEN the log records are retrieved from the Opensearch index
			// THEN verify that a recent log record of bobs-bookstore-adminserver stdout is found
			func() {
				t.It("Verify recent bobs-bookstore-adminserver log record exists", func() {
					Eventually(func() bool {
						return pkg.LogRecordFound(bobsIndexName, time.Now().Add(-24*time.Hour), map[string]string{
							k8sLabelDomainUID:     bobsBookStore,
							k8sLabelWLServerName:  "AdminServer",
							k8sPodName:            "bobs-bookstore-adminserver",
							k8sLabelContainerName: "weblogic-server",
						})
					}, shortWaitTimeout, shortPollingInterval).Should(BeTrue(), "Expected to find a recent log record")
				})
			},
			// GIVEN a WebLogic application with logging enabled
			// WHEN the log records are retrieved from the Opensearch index
			// THEN verify that a recent log record of bobs-bookstore-adminserver log file is found
			func() {
				t.It("Verify recent bobs-bookstore-adminserver log record exists", func() {
					Eventually(func() bool {
						return pkg.LogRecordFound(bobsIndexName, time.Now().Add(-24*time.Hour), map[string]string{
							k8sLabelDomainUID:     bobsBookStore,
							k8sLabelWLServerName:  "AdminServer",
							k8sPodName:            "bobs-bookstore-adminserver",
							k8sLabelContainerName: fluentdStdoutSidecar,
						})
					}, shortWaitTimeout, shortPollingInterval).Should(BeTrue(), "Expected to find a recent log record")
				})
			},
			// GIVEN a WebLogic application with logging enabled
			// WHEN the log records are retrieved from the Opensearch index
			// THEN verify that a recent log record of bobs-bookstore-managed-server stdout is found
			func() {
				t.It("Verify recent bobs-bookstore-managed-server1 log record exists", func() {
					Eventually(func() bool {
						return pkg.LogRecordFound(bobsIndexName, time.Now().Add(-24*time.Hour), map[string]string{
							k8sLabelDomainUID:     bobsBookStore,
							k8sLabelWLServerName:  managedServer1,
							k8sPodName:            "bobs-bookstore-managed-server1",
							k8sLabelContainerName: "weblogic-server",
						})
					}, shortWaitTimeout, shortPollingInterval).Should(BeTrue(), "Expected to find a recent log record")
				})
			},
			// GIVEN a WebLogic application with logging enabled
			// WHEN the log records are retrieved from the Opensearch index
			// THEN verify that a recent log record of bobs-bookstore-managed-server log file is found
			func() {
				t.It("Verify recent bobs-bookstore-managed-server1 log record exists", func() {
					Eventually(func() bool {
						return pkg.FindLog(bobsIndexName,
							[]pkg.Match{
								{Key: "kubernetes.container_name.keyword", Value: fluentdStdoutSidecar},
								{Key: k8sLabelDomainUID, Value: bobsBookStore},
								{Key: k8sLabelWLServerName, Value: managedServer1},
								{Key: "messageID", Value: "BEA-"},                //matches BEA-*
								{Key: "message", Value: "Admin Traffic Enabled"}, //"Admin Traffic Enabled" in last line
								{Key: "serverName", Value: "bobs-bookstore-managed-server1"},
								{Key: "subSystem.keyword", Value: "RJVM"}},
							[]pkg.Match{})
					}, longWaitTimeout, longPollingInterval).Should(BeTrue(), "Expected to find a recent log record")
				})
			},
		)
	})
	t.Context("Coherence logging.", Label("f:observability.logging.es"), func() {
		var indexName string
		Eventually(func() error {
			indexName, err = pkg.GetOpenSearchAppIndex(namespace)
			return err
		}, shortWaitTimeout, shortPollingInterval).Should(BeNil(), "Expected to get OpenSearch App Index")

		// GIVEN a Coherence application with logging enabled
		// WHEN the Opensearch index is retrieved
		// THEN verify that it is found
		t.It("Verify Opensearch index exists", func() {
			Eventually(func() bool {
				return pkg.LogIndexFound(indexName)
			}, shortWaitTimeout, shortPollingInterval).Should(BeTrue(), "Expected to find log index "+indexName)
		})
		pkg.Concurrently(
			// GIVEN a Coherence application with logging enabled
			// WHEN the log records are retrieved from the Opensearch index
			// THEN verify that a recent log record of roberts-coherence-0 stdout is found
			func() {
				t.It("Verify recent roberts-coherence-0 log record exists", func() {
					Eventually(func() bool {
						return pkg.LogRecordFound(indexName, time.Now().Add(-24*time.Hour), map[string]string{
							K8sLabelCoherenceCluster:                            robertsCoherence,
							"kubernetes.labels.app_oam_dev\\/component.keyword": robertCoh,
							k8sPodName:                          "roberts-coherence-0",
							"kubernetes.container_name.keyword": "coherence",
						})
					}, shortWaitTimeout, shortPollingInterval).Should(BeTrue(), "Expected to find a recent log record")
				})
			},
			// GIVEN a Coherence application with logging enabled
			// WHEN the log records are retrieved from the Opensearch index
			// THEN verify that a recent log record of roberts-coherence-0 log file is found
			func() {
				t.It("Verify recent roberts-coherence-0 log record exists", func() {
					Eventually(func() bool {
						return pkg.FindLog(indexName,
							[]pkg.Match{
								{Key: "kubernetes.labels.app_oam_dev/component", Value: robertCoh},
								{Key: K8sLabelCoherenceCluster, Value: robertsCoherence},
								{Key: k8sPodName, Value: "roberts-coherence-0"},
								{Key: "product", Value: "Oracle Coherence"},
								{Key: k8sLabelContainerName, Value: fluentdStdoutSidecar}},
							[]pkg.Match{ //MustNot
								{Key: k8sLabelContainerName, Value: "coherence"}})
					}, 5*time.Minute, 10*time.Second).Should(BeTrue(), "Expected to find a systemd log record")
				})
			},
			// GIVEN a Coherence application with logging enabled
			// WHEN the log records are retrieved from the Opensearch index
			// THEN verify that a recent log record of roberts-coherence-1 stdout is found
			func() {
				t.It("Verify recent roberts-coherence-1 log record exists", func() {
					Eventually(func() bool {
						return pkg.FindLog(indexName,
							[]pkg.Match{
								{Key: K8sLabelCoherenceCluster, Value: robertsCoherence},
								{Key: k8sPodName, Value: "roberts-coherence-1"},
								{Key: "kubernetes.container_name.keyword", Value: "coherence"}},
							[]pkg.Match{ //MustNot
								{Key: k8sLabelContainerName, Value: fluentdStdoutSidecar},
							})
					}, 5*time.Minute, 10*time.Second).Should(BeTrue(), "Expected to find a systemd log record")

				})
			},
			// GIVEN a Coherence application with logging enabled
			// WHEN the log records are retrieved from the Opensearch index
			// THEN verify that a recent log record of roberts-coherence-1 log file is found
			func() {
				t.It("Verify recent roberts-coherence-1 log record exists", func() {
					Eventually(func() bool {
						return pkg.FindLog(indexName,
							[]pkg.Match{
								{Key: K8sLabelCoherenceCluster, Value: robertsCoherence},
								{Key: k8sPodName, Value: "roberts-coherence-1"},
								{Key: "product", Value: "Oracle Coherence"},
								{Key: k8sLabelContainerName, Value: fluentdStdoutSidecar}},
							[]pkg.Match{})
					}, 5*time.Minute, 10*time.Second).Should(BeTrue(), "Expected to find a systemd log record")
				})
			},
			// GIVEN a Coherence application with logging enabled
			// WHEN the log records are retrieved from the Opensearch index
			// THEN verify that a recent log record of bobbys-coherence log file is found
			func() {
				t.It("Verify recent roberts-coherence-1 log record exists", func() {
					Eventually(func() bool {
						return pkg.FindLog(indexName,
							[]pkg.Match{
								{Key: "kubernetes.labels.app_oam_dev/component", Value: "bobby-coh"},
								{Key: K8sLabelCoherenceCluster, Value: "bobbys-coherence"},
								{Key: "coherence.cluster.name", Value: "bobbys-coherence"},
								{Key: "product", Value: "Oracle Coherence"},
								{Key: k8sLabelContainerName, Value: fluentdStdoutSidecar}},
							[]pkg.Match{})
					}, 5*time.Minute, 10*time.Second).Should(BeTrue(), "Expected to find a systemd log record")
				})
			},
		)
	})
})
