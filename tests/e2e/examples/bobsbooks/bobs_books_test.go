// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package bobsbooks

import (
	"fmt"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
)

const (
	shortWaitTimeout     = 10 * time.Minute
	shortPollingInterval = 10 * time.Second
	longWaitTimeout      = 20 * time.Minute
	longPollingInterval  = 20 * time.Second
)

var _ = BeforeSuite(func() {
	deployBobsBooksExample()
})

var failed = false
var _ = AfterEach(func() {
	failed = failed || CurrentGinkgoTestDescription().Failed
})

var _ = AfterSuite(func() {
	if failed {
		pkg.ExecuteClusterDumpWithEnvVarConfig()
	}
	undeployBobsBooksExample()
})

func deployBobsBooksExample() {
	pkg.Log(pkg.Info, "Deploy BobsBooks example")
	wlsUser := "weblogic"
	wlsPass := pkg.GetRequiredEnvVarOrFail("WEBLOGIC_PSW")
	dbPass := pkg.GetRequiredEnvVarOrFail("DATABASE_PSW")
	regServ := pkg.GetRequiredEnvVarOrFail("OCR_REPO")
	regUser := pkg.GetRequiredEnvVarOrFail("OCR_CREDS_USR")
	regPass := pkg.GetRequiredEnvVarOrFail("OCR_CREDS_PSW")

	pkg.Log(pkg.Info, "Create namespace")
	Eventually(func() (*v1.Namespace, error) {
		nsLabels := map[string]string{
			"verrazzano-managed": "true",
			"istio-injection":    "enabled"}
		return pkg.CreateNamespace("bobs-books", nsLabels)
	}, shortWaitTimeout, shortPollingInterval).ShouldNot(BeNil())

	pkg.Log(pkg.Info, "Create Docker repository secret")
	Eventually(func() (*v1.Secret, error) {
		return pkg.CreateDockerSecret("bobs-books", "bobs-books-repo-credentials", regServ, regUser, regPass)
	}, shortWaitTimeout, shortPollingInterval).ShouldNot(BeNil())

	pkg.Log(pkg.Info, "Create Bobbys front end Weblogic credentials secret")
	Eventually(func() (*v1.Secret, error) {
		return pkg.CreateCredentialsSecret("bobs-books", "bobbys-front-end-weblogic-credentials", wlsUser, wlsPass, nil)
	}, shortWaitTimeout, shortPollingInterval).ShouldNot(BeNil())

	pkg.Log(pkg.Info, "Create Bobs Bookstore Weblogic credentials secret")
	Eventually(func() (*v1.Secret, error) {
		return pkg.CreateCredentialsSecret("bobs-books", "bobs-bookstore-weblogic-credentials", wlsUser, wlsPass, nil)
	}, shortWaitTimeout, shortPollingInterval).ShouldNot(BeNil())

	pkg.Log(pkg.Info, "Create database credentials secret")
	Eventually(func() (*v1.Secret, error) {
		m := map[string]string{"password": dbPass, "username": wlsUser, "url": "jdbc:mysql://mysql.bobs-books.svc.cluster.local:3306/books"}
		return pkg.CreateCredentialsSecretFromMap("bobs-books", "mysql-credentials", m, nil)
	}, shortWaitTimeout, shortPollingInterval).ShouldNot(BeNil())

	// Note: creating the app config first to verify that default metrics traits are created properly if the app config exists before the components
	pkg.Log(pkg.Info, "Create application resources")
	Eventually(func() error {
		return pkg.CreateOrUpdateResourceFromFile("examples/bobs-books/bobs-books-app.yaml")
	}, shortWaitTimeout, shortPollingInterval).ShouldNot(HaveOccurred())

	pkg.Log(pkg.Info, "Create component resources")
	Eventually(func() error {
		return pkg.CreateOrUpdateResourceFromFile("examples/bobs-books/bobs-books-comp.yaml")
	}, shortWaitTimeout, shortPollingInterval, "Failed to create Bobs Books component resources").ShouldNot(HaveOccurred())
}

func undeployBobsBooksExample() {
	pkg.Log(pkg.Info, "Undeploy BobsBooks example")
	pkg.Log(pkg.Info, "Delete application")
	Eventually(func() error {
		return pkg.DeleteResourceFromFile("examples/bobs-books/bobs-books-app.yaml")
	}, shortWaitTimeout, shortPollingInterval).ShouldNot(HaveOccurred())

	pkg.Log(pkg.Info, "Delete components")
	Eventually(func() error {
		return pkg.DeleteResourceFromFile("examples/bobs-books/bobs-books-comp.yaml")
	}, shortWaitTimeout, shortPollingInterval).ShouldNot(HaveOccurred())

	pkg.Log(pkg.Info, "Delete namespace")
	Eventually(func() error {
		return pkg.DeleteNamespace("bobs-books")
	}, shortWaitTimeout, shortPollingInterval).ShouldNot(HaveOccurred())

	Eventually(func() bool {
		_, err := pkg.GetNamespace("bobs-books")
		return err != nil && errors.IsNotFound(err)
	}, shortWaitTimeout, shortPollingInterval).Should(BeTrue())
}

var _ = Describe("Verify Bobs Books example application.", func() {
	It("Wait for deployment.", func() {
		Eventually(func() bool {
			expectedPods := []string{
				"bobbys-front-end-adminserver",
				"bobs-bookstore-adminserver",
				"bobbys-coherence-0",
				"roberts-coherence-0",
				"roberts-coherence-1",
				"bobbys-helidon-stock-application",
				"robert-helidon",
				"mysql",
			}
			return pkg.PodsRunning("bobs-books", expectedPods)
		}, longWaitTimeout, longPollingInterval).Should(BeTrue(), "Bobs Books Application Failed to Deploy")
	})

	var host = ""

	// Get the host from the Istio gateway resource.
	// GIVEN the Istio gateway for the bobs-books namespace
	// WHEN GetHostnameFromGateway is called
	// THEN return the host name found in the gateway.
	It("Get host from gateway.", func() {
		Eventually(func() string {
			host = pkg.GetHostnameFromGateway("bobs-books", "")
			return host
		}, shortWaitTimeout, shortPollingInterval).Should(Not(BeEmpty()))
	})
	Context("Ingress.", func() {
		// Verify the application endpoint is working.
		// GIVEN the Bobs Books app is deployed
		// WHEN the roberts-books UI is accessed
		// THEN the expected returned page should contain an expected value.
		It("Verify roberts-books UI endpoint is working.", func() {
			Eventually(func() (*pkg.HTTPResponse, error) {
				url := fmt.Sprintf("https://%s", host)
				return pkg.GetWebPage(url, host)
			}, shortWaitTimeout, shortPollingInterval).Should(And(pkg.HasStatus(200), pkg.BodyContains("Robert's Books")))
		})
		// Verify the application endpoint is working.
		// GIVEN the Bobs Books app is deployed
		// WHEN the bobbys-books UI is accessed
		// THEN the expected returned page should contain an expected value.
		It("Verify bobbys-books UI endpoint is working.", func() {
			Eventually(func() (*pkg.HTTPResponse, error) {
				url := fmt.Sprintf("https://%s/bobbys-front-end/", host)
				return pkg.GetWebPage(url, host)
			}, shortWaitTimeout, shortPollingInterval).Should(And(pkg.HasStatus(200), pkg.BodyContains("Bobby's Books")))
		})
		// Verify the application endpoint is working.
		// GIVEN the Bobs Books app is deployed
		// WHEN the bobs-orders UI is accessed
		// THEN the expected returned page should contain an expected value.
		It("Verify bobs-orders UI endpoint is working.", func() {
			Eventually(func() (*pkg.HTTPResponse, error) {
				url := fmt.Sprintf("https://%s/bobs-bookstore-order-manager/orders", host)
				return pkg.GetWebPage(url, host)
			}, shortWaitTimeout, shortPollingInterval).Should(And(pkg.HasStatus(200), pkg.BodyContains("Bob's Order Manager")))
		})
	})
	Context("Metrics.", func() {
		// Verify application Prometheus scraped metrics
		// GIVEN a deployed Bob's Books application
		// WHEN the application configuration uses a default metrics trait
		// THEN confirm that metrics are being collected
		It("Retrieve application Prometheus scraped metrics", func() {
			pkg.Concurrently(
				func() {
					Eventually(func() bool {
						return pkg.MetricsExist("base_jvm_uptime_seconds", "app", "bobbys-helidon-stock-application")
					}, shortWaitTimeout, shortPollingInterval).Should(BeTrue())
				},
				func() {
					Eventually(func() bool {
						return pkg.MetricsExist("base_jvm_uptime_seconds", "app", "robert-helidon")
					}, shortWaitTimeout, shortPollingInterval).Should(BeTrue())
				},
				func() {
					Eventually(func() bool {
						return pkg.MetricsExist("vendor_requests_count_total", "app_oam_dev_component", "bobby-helidon")
					}, shortWaitTimeout, shortPollingInterval).Should(BeTrue())
				},
				func() {
					Eventually(func() bool {
						return pkg.MetricsExist("vendor_requests_count_total", "app_oam_dev_component", "robert-helidon")
					}, longWaitTimeout, longPollingInterval).Should(BeTrue())
				},
				func() {
					Eventually(func() bool {
						return pkg.MetricsExist("wls_jvm_process_cpu_load", "weblogic_domainName", "bobbys-front-end")
					}, longWaitTimeout, longPollingInterval).Should(BeTrue())
				},
				func() {
					Eventually(func() bool {
						return pkg.MetricsExist("wls_jvm_process_cpu_load", "weblogic_domainName", "bobs-bookstore")
					}, shortWaitTimeout, shortPollingInterval).Should(BeTrue())
				},
				func() {
					Eventually(func() bool {
						return pkg.MetricsExist("wls_scrape_mbeans_count_total", "weblogic_domainName", "bobbys-front-end")
					}, longWaitTimeout, longPollingInterval).Should(BeTrue())
				},
				func() {
					Eventually(func() bool {
						return pkg.MetricsExist("wls_scrape_mbeans_count_total", "weblogic_domainName", "bobs-bookstore")
					}, shortWaitTimeout, shortPollingInterval).Should(BeTrue())
				},
				func() {
					Eventually(func() bool {
						return pkg.MetricsExist("vendor:coherence_cluster_size", "coherenceCluster", "bobbys-coherence")
					}, longWaitTimeout, longPollingInterval).Should(BeTrue())
				},
				func() {
					Eventually(func() bool {
						return pkg.MetricsExist("vendor:coherence_cluster_size", "coherenceCluster", "roberts-coherence")
					}, shortWaitTimeout, shortPollingInterval).Should(BeTrue())
				},
			)
		})
		// Verify Istio Prometheus scraped metrics
		// GIVEN a deployed Bob's Books application
		// WHEN the application configuration is deployed
		// THEN confirm that Istio metrics are being collected
		It("Retrieve Istio Prometheus scraped metrics", func() {
			pkg.Concurrently(
				func() {
					Eventually(func() bool {
						return pkg.MetricsExist("istio_tcp_received_bytes_total", "destination_canonical_service", "bobbys-helidon-stock-application")
					}, shortWaitTimeout, shortPollingInterval).Should(BeTrue())
				},
				func() {
					Eventually(func() bool {
						return pkg.MetricsExist("istio_tcp_received_bytes_total", "destination_canonical_service", "robert-helidon")
					}, shortWaitTimeout, shortPollingInterval).Should(BeTrue())
				},
				func() {
					Eventually(func() bool {
						return pkg.MetricsExist("istio_tcp_received_bytes_total", "destination_canonical_service", "bobbys-front-end-adminserver")
					}, shortWaitTimeout, shortPollingInterval).Should(BeTrue())
				},
				func() {
					Eventually(func() bool {
						return pkg.MetricsExist("istio_tcp_received_bytes_total", "destination_canonical_service", "bobs-bookstore-adminserver")
					}, longWaitTimeout, longPollingInterval).Should(BeTrue())
				},
				func() {
					Eventually(func() bool {
						return pkg.MetricsExist("envoy_cluster_ssl_handshake", "pod_name", "bobbys-front-end-adminserver")
					}, longWaitTimeout, longPollingInterval).Should(BeTrue())
				},
				func() {
					Eventually(func() bool {
						return pkg.MetricsExist("envoy_cluster_ssl_handshake", "pod_name", "bobs-bookstore-adminserver")
					}, shortWaitTimeout, shortPollingInterval).Should(BeTrue())
				},
			)
		})
	})
	Context("WebLogic logging.", func() {
		bobsIndexName := "verrazzano-namespace-bobs-books"
		// GIVEN a WebLogic application with logging enabled
		// WHEN the Elasticsearch index is retrieved
		// THEN verify that it is found
		It("Verify Elasticsearch index exists", func() {
			Eventually(func() bool {
				return pkg.LogIndexFound(bobsIndexName)
			}, shortWaitTimeout, shortPollingInterval).Should(BeTrue(), "Expected to find log index "+bobsIndexName)
		})
		pkg.Concurrently(
			// GIVEN a WebLogic application with logging enabled
			// WHEN the log records are retrieved from the Elasticsearch index
			// THEN verify that a recent log record of bobbys-front-end-adminserver stdout is found
			func() {
				It("Verify recent bobbys-front-end-adminserver log record exists", func() {
					Eventually(func() bool {
						return pkg.LogRecordFound(bobsIndexName, time.Now().Add(-24*time.Hour), map[string]string{
							"kubernetes.labels.weblogic_domainUID":  "bobbys-front-end",
							"kubernetes.labels.weblogic_serverName": "AdminServer",
							"kubernetes.pod_name":                   "bobbys-front-end-adminserver",
							"kubernetes.container_name":             "weblogic-server",
						})
					}, shortWaitTimeout, shortPollingInterval).Should(BeTrue(), "Expected to find a recent log record")
				})
			},
			// GIVEN a WebLogic application with logging enabled
			// WHEN the log records are retrieved from the Elasticsearch index
			// THEN verify that a recent log record of bobbys-front-end-adminserver log file is found
			func() {
				It("Verify recent bobbys-front-end-adminserver log record exists", func() {
					Eventually(func() bool {
						return pkg.LogRecordFound(bobsIndexName, time.Now().Add(-24*time.Hour), map[string]string{
							"kubernetes.labels.weblogic_domainUID":  "bobbys-front-end",
							"kubernetes.labels.weblogic_serverName": "AdminServer",
							"kubernetes.pod_name":                   "bobbys-front-end-adminserver",
							"kubernetes.container_name":             "fluentd-stdout-sidecar",
						})
					}, shortWaitTimeout, shortPollingInterval).Should(BeTrue(), "Expected to find a recent log record")
				})
			},
			// GIVEN a WebLogic application with logging enabled
			// WHEN the log records are retrieved from the Elasticsearch index
			// THEN verify that a recent log record of bobbys-front-end-managed-server stdout is found
			func() {
				It("Verify recent bobbys-front-end-managed-server1 log record exists", func() {
					Eventually(func() bool {
						return pkg.LogRecordFound(bobsIndexName, time.Now().Add(-24*time.Hour), map[string]string{
							"kubernetes.labels.weblogic_domainUID":  "bobbys-front-end",
							"kubernetes.labels.weblogic_serverName": "managed-server1",
							"kubernetes.pod_name":                   "bobbys-front-end-managed-server1",
							"kubernetes.container_name":             "weblogic-server",
						})
					}, shortWaitTimeout, shortPollingInterval).Should(BeTrue(), "Expected to find a recent log record")
				})
			},

			// GIVEN a WebLogic application with logging enabled
			// WHEN the log records are retrieved from the Elasticsearch index
			// THEN verify that a recent pattern-matched log record of bobbys-front-end-adminserver stdout is found
			func() {
				It("Verify recent pattern-matched AdminServer log record exists", func() {
					Eventually(func() bool {
						return pkg.FindLog(bobsIndexName,
							[]pkg.Match{
								{Key: "kubernetes.container_name.keyword", Value: "fluentd-stdout-sidecar"},
								{Key: "subSystem.keyword", Value: "WorkManager"},
								{Key: "serverName.keyword", Value: "bobbys-front-end-adminserver"},
								{Key: "serverName2.keyword", Value: "AdminServer"},
								{Key: "message", Value: "standby threads"}},
							[]pkg.Match{})
					}, longWaitTimeout, longPollingInterval).Should(BeTrue(), "Expected to find a recent log record")
				})
			},
			// GIVEN a WebLogic application with logging enabled
			// WHEN the log records are retrieved from the Elasticsearch index
			// THEN verify that a recent pattern-matched log record of bobbys-front-end-adminserver stdout is found
			func() {
				It("Verify recent pattern-matched AdminServer log record exists", func() {
					Eventually(func() bool {
						return pkg.FindLog(bobsIndexName,
							[]pkg.Match{
								{Key: "kubernetes.container_name.keyword", Value: "fluentd-stdout-sidecar"},
								{Key: "subSystem", Value: "WorkManager"},
								{Key: "serverName", Value: "bobbys-front-end-adminserver"},
								{Key: "serverName2", Value: "AdminServer"},
								{Key: "message", Value: "Self-tuning"}},
							[]pkg.Match{})
					}, longWaitTimeout, longPollingInterval).Should(BeTrue(), "Expected to find a recent log record")
				})
			},
			// GIVEN a WebLogic application with logging enabled
			// WHEN the log records are retrieved from the Elasticsearch index
			// THEN verify that no 'pattern not matched' log record of fluentd-stdout-sidecar is found
			func() {
				It("Verify recent 'pattern not matched' log records do not exist", func() {
					Eventually(func() bool {
						return pkg.FindLog(bobsIndexName,
							[]pkg.Match{
								{Key: "kubernetes.container_name.keyword", Value: "fluentd-stdout-sidecar"},
								{Key: "message", Value: "pattern not matched"}},
							[]pkg.Match{})
					}, longWaitTimeout, longPollingInterval).Should(BeFalse(), "Expected to find No pattern not matched log records")
				})
			},
			// GIVEN a WebLogic application with logging enabled
			// WHEN the log records are retrieved from the Elasticsearch index
			// THEN verify that a recent log record of bobbys-front-end-managed-server log file is found
			func() {
				It("Verify recent bobbys-front-end-managed-server1 log record exists", func() {
					Eventually(func() bool {
						return pkg.FindLog(bobsIndexName,
							[]pkg.Match{
								{Key: "kubernetes.container_name.keyword", Value: "fluentd-stdout-sidecar"},
								{Key: "kubernetes.labels.weblogic_domainUID", Value: "bobbys-front-end"},
								{Key: "kubernetes.labels.weblogic_serverName", Value: "managed-server1"},
								{Key: "messageID", Value: "BEA-"},         //matches BEA-*
								{Key: "message", Value: "Tunneling Ping"}, //"Tunneling Ping" in last line
								{Key: "serverName", Value: "bobbys-front-end-managed-server1"},
								{Key: "subSystem.keyword", Value: "RJVM"}},
							[]pkg.Match{})
					}, longWaitTimeout, longPollingInterval).Should(BeTrue(), "Expected to find a recent log record")
				})
			},
			// GIVEN a WebLogic application with logging enabled
			// WHEN the log records are retrieved from the Elasticsearch index
			// THEN verify that a recent pattern-matched log record of bobbys-front-end-managed-server stdout is found
			func() {
				It("Verify recent pattern-matched managed-server log record exists", func() {
					Eventually(func() bool {
						return pkg.FindLog(bobsIndexName,
							[]pkg.Match{
								{Key: "kubernetes.container_name.keyword", Value: "fluentd-stdout-sidecar"},
								{Key: "subSystem.keyword", Value: "WorkManager"},
								{Key: "serverName.keyword", Value: "bobbys-front-end-managed-server1"},
								{Key: "serverName2.keyword", Value: "managed-server1"},
								{Key: "message", Value: "standby threads"}},
							[]pkg.Match{})
					}, longWaitTimeout, longPollingInterval).Should(BeTrue(), "Expected to find a recent log record")
				})
			},
			// GIVEN a WebLogic application with logging enabled
			// WHEN the log records are retrieved from the Elasticsearch index
			// THEN verify that a recent pattern-matched log record of bobbys-front-end-managed-server stdout is found
			func() {
				It("Verify recent pattern-matched managed-server log record exists", func() {
					Eventually(func() bool {
						return pkg.FindLog(bobsIndexName,
							[]pkg.Match{
								{Key: "kubernetes.container_name.keyword", Value: "fluentd-stdout-sidecar"},
								{Key: "subSystem", Value: "WorkManager"},
								{Key: "serverName", Value: "bobbys-front-end-managed-server1"},
								{Key: "serverName2", Value: "managed-server1"},
								{Key: "message", Value: "Self-tuning"}},
							[]pkg.Match{})
					}, longWaitTimeout, longPollingInterval).Should(BeTrue(), "Expected to find a recent log record")
				})
			},
		)
		pkg.Concurrently(
			// GIVEN a WebLogic application with logging enabled
			// WHEN the log records are retrieved from the Elasticsearch index
			// THEN verify that a recent log record of bobs-bookstore-adminserver stdout is found
			func() {
				It("Verify recent bobs-bookstore-adminserver log record exists", func() {
					Eventually(func() bool {
						return pkg.LogRecordFound(bobsIndexName, time.Now().Add(-24*time.Hour), map[string]string{
							"kubernetes.labels.weblogic_domainUID":  "bobs-bookstore",
							"kubernetes.labels.weblogic_serverName": "AdminServer",
							"kubernetes.pod_name":                   "bobs-bookstore-adminserver",
							"kubernetes.container_name":             "weblogic-server",
						})
					}, shortWaitTimeout, shortPollingInterval).Should(BeTrue(), "Expected to find a recent log record")
				})
			},
			// GIVEN a WebLogic application with logging enabled
			// WHEN the log records are retrieved from the Elasticsearch index
			// THEN verify that a recent log record of bobs-bookstore-adminserver log file is found
			func() {
				It("Verify recent bobs-bookstore-adminserver log record exists", func() {
					Eventually(func() bool {
						return pkg.LogRecordFound(bobsIndexName, time.Now().Add(-24*time.Hour), map[string]string{
							"kubernetes.labels.weblogic_domainUID":  "bobs-bookstore",
							"kubernetes.labels.weblogic_serverName": "AdminServer",
							"kubernetes.pod_name":                   "bobs-bookstore-adminserver",
							"kubernetes.container_name":             "fluentd-stdout-sidecar",
						})
					}, shortWaitTimeout, shortPollingInterval).Should(BeTrue(), "Expected to find a recent log record")
				})
			},
			// GIVEN a WebLogic application with logging enabled
			// WHEN the log records are retrieved from the Elasticsearch index
			// THEN verify that a recent log record of bobs-bookstore-managed-server stdout is found
			func() {
				It("Verify recent bobs-bookstore-managed-server1 log record exists", func() {
					Eventually(func() bool {
						return pkg.LogRecordFound(bobsIndexName, time.Now().Add(-24*time.Hour), map[string]string{
							"kubernetes.labels.weblogic_domainUID":  "bobs-bookstore",
							"kubernetes.labels.weblogic_serverName": "managed-server1",
							"kubernetes.pod_name":                   "bobs-bookstore-managed-server1",
							"kubernetes.container_name":             "weblogic-server",
						})
					}, shortWaitTimeout, shortPollingInterval).Should(BeTrue(), "Expected to find a recent log record")
				})
			},
			// GIVEN a WebLogic application with logging enabled
			// WHEN the log records are retrieved from the Elasticsearch index
			// THEN verify that a recent log record of bobs-bookstore-managed-server log file is found
			func() {
				It("Verify recent bobs-bookstore-managed-server1 log record exists", func() {
					Eventually(func() bool {
						return pkg.FindLog(bobsIndexName,
							[]pkg.Match{
								{Key: "kubernetes.container_name.keyword", Value: "fluentd-stdout-sidecar"},
								{Key: "kubernetes.labels.weblogic_domainUID", Value: "bobs-bookstore"},
								{Key: "kubernetes.labels.weblogic_serverName", Value: "managed-server1"},
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
	Context("Coherence logging.", func() {
		indexName := "verrazzano-namespace-bobs-books"
		// GIVEN a Coherence application with logging enabled
		// WHEN the Elasticsearch index is retrieved
		// THEN verify that it is found
		It("Verify Elasticsearch index exists", func() {
			Eventually(func() bool {
				return pkg.LogIndexFound(indexName)
			}, shortWaitTimeout, shortPollingInterval).Should(BeTrue(), "Expected to find log index "+indexName)
		})
		pkg.Concurrently(
			// GIVEN a Coherence application with logging enabled
			// WHEN the log records are retrieved from the Elasticsearch index
			// THEN verify that a recent log record of roberts-coherence-0 stdout is found
			func() {
				It("Verify recent roberts-coherence-0 log record exists", func() {
					Eventually(func() bool {
						return pkg.LogRecordFound(indexName, time.Now().Add(-24*time.Hour), map[string]string{
							"kubernetes.labels.coherenceCluster":                "roberts-coherence",
							"kubernetes.labels.app_oam_dev\\/component.keyword": "robert-coh",
							"kubernetes.pod_name":                               "roberts-coherence-0",
							"kubernetes.container_name.keyword":                 "coherence",
						})
					}, shortWaitTimeout, shortPollingInterval).Should(BeTrue(), "Expected to find a recent log record")
				})
			},
			// GIVEN a Coherence application with logging enabled
			// WHEN the log records are retrieved from the Elasticsearch index
			// THEN verify that a recent log record of roberts-coherence-0 log file is found
			func() {
				It("Verify recent roberts-coherence-0 log record exists", func() {
					Eventually(func() bool {
						return pkg.FindLog(indexName,
							[]pkg.Match{
								{Key: "kubernetes.labels.app_oam_dev/component", Value: "robert-coh"},
								{Key: "kubernetes.labels.coherenceCluster", Value: "roberts-coherence"},
								{Key: "kubernetes.pod_name", Value: "roberts-coherence-0"},
								{Key: "product", Value: "Oracle Coherence"},
								{Key: "kubernetes.container_name", Value: "fluentd-stdout-sidecar"}},
							[]pkg.Match{ //MustNot
								{Key: "kubernetes.container_name", Value: "coherence"}})
					}, 5*time.Minute, 10*time.Second).Should(BeTrue(), "Expected to find a systemd log record")
				})
			},
			// GIVEN a Coherence application with logging enabled
			// WHEN the log records are retrieved from the Elasticsearch index
			// THEN verify that a recent log record of roberts-coherence-1 stdout is found
			func() {
				It("Verify recent roberts-coherence-1 log record exists", func() {
					Eventually(func() bool {
						return pkg.FindLog(indexName,
							[]pkg.Match{
								{Key: "kubernetes.labels.coherenceCluster", Value: "roberts-coherence"},
								{Key: "kubernetes.pod_name", Value: "roberts-coherence-1"},
								{Key: "kubernetes.container_name.keyword", Value: "coherence"}},
							[]pkg.Match{ //MustNot
								{Key: "kubernetes.container_name", Value: "fluentd-stdout-sidecar"},
							})
					}, 5*time.Minute, 10*time.Second).Should(BeTrue(), "Expected to find a systemd log record")

				})
			},
			// GIVEN a Coherence application with logging enabled
			// WHEN the log records are retrieved from the Elasticsearch index
			// THEN verify that a recent log record of roberts-coherence-1 log file is found
			func() {
				It("Verify recent roberts-coherence-1 log record exists", func() {
					Eventually(func() bool {
						return pkg.FindLog(indexName,
							[]pkg.Match{
								{Key: "kubernetes.labels.coherenceCluster", Value: "roberts-coherence"},
								{Key: "kubernetes.pod_name", Value: "roberts-coherence-1"},
								{Key: "product", Value: "Oracle Coherence"},
								{Key: "kubernetes.container_name", Value: "fluentd-stdout-sidecar"}},
							[]pkg.Match{})
					}, 5*time.Minute, 10*time.Second).Should(BeTrue(), "Expected to find a systemd log record")
				})
			},
			// GIVEN a Coherence application with logging enabled
			// WHEN the log records are retrieved from the Elasticsearch index
			// THEN verify that a recent log record of bobbys-coherence log file is found
			func() {
				It("Verify recent roberts-coherence-1 log record exists", func() {
					Eventually(func() bool {
						return pkg.FindLog(indexName,
							[]pkg.Match{
								{Key: "kubernetes.labels.app_oam_dev/component", Value: "bobby-coh"},
								{Key: "kubernetes.labels.coherenceCluster", Value: "bobbys-coherence"},
								{Key: "coherence.cluster.name", Value: "bobbys-coherence"},
								{Key: "product", Value: "Oracle Coherence"},
								{Key: "kubernetes.container_name", Value: "fluentd-stdout-sidecar"}},
							[]pkg.Match{})
					}, 5*time.Minute, 10*time.Second).Should(BeTrue(), "Expected to find a systemd log record")
				})
			},
		)
	})
})