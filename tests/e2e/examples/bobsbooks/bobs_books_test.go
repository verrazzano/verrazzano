// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package bobsbooks

import (
	"fmt"
	"time"

	"github.com/onsi/ginkgo"
	"github.com/onsi/gomega"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg"
	"k8s.io/apimachinery/pkg/api/errors"
)

const (
	shortWaitTimeout     = 10 * time.Minute
	shortPollingInterval = 10 * time.Second
	longWaitTimeout      = 20 * time.Minute
	longPollingInterval  = 20 * time.Second
)

var _ = ginkgo.BeforeSuite(func() {
	deployBobsBooksExample()
})

var _ = ginkgo.AfterSuite(func() {
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
	nsLabels := map[string]string{
		"verrazzano-managed": "true",
		"istio-injection":    "enabled"}
	if _, err := pkg.CreateNamespace("bobs-books", nsLabels); err != nil {
		ginkgo.Fail(fmt.Sprintf("Failed to create namespace: %v", err))
	}
	pkg.Log(pkg.Info, "Create Docker repository secret")
	if _, err := pkg.CreateDockerSecret("bobs-books", "bobs-books-repo-credentials", regServ, regUser, regPass); err != nil {
		ginkgo.Fail(fmt.Sprintf("Failed to create Docker registry secret: %v", err))
	}
	pkg.Log(pkg.Info, "Create Bobbys front end Weblogic credentials secret")
	if _, err := pkg.CreateCredentialsSecret("bobs-books", "bobbys-front-end-weblogic-credentials", wlsUser, wlsPass, nil); err != nil {
		ginkgo.Fail(fmt.Sprintf("Failed to create WebLogic credentials secret: %v", err))
	}
	pkg.Log(pkg.Info, "Create Bobbys front end runtime encrypt secret")
	if _, err := pkg.CreateCredentialsSecret("bobs-books", "bobbys-front-end-runtime-encrypt-secret", wlsUser, wlsPass, map[string]string{"weblogic.domainUID": "bobbys-front-end"}); err != nil {
		ginkgo.Fail(fmt.Sprintf("Failed to create WebLogic credentials secret: %v", err))
	}
	pkg.Log(pkg.Info, "Create Bobs Bookstore Weblogic credentials secret")
	if _, err := pkg.CreateCredentialsSecret("bobs-books", "bobs-bookstore-weblogic-credentials", wlsUser, wlsPass, nil); err != nil {
		ginkgo.Fail(fmt.Sprintf("Failed to create WebLogic credentials secret: %v", err))
	}
	pkg.Log(pkg.Info, "Create Bobs Bookstore runtime encrypt secret")
	if _, err := pkg.CreateCredentialsSecret("bobs-books", "bobs-bookstore-runtime-encrypt-secret", wlsUser, wlsPass, map[string]string{"weblogic.domainUID": "bobs-bookstore"}); err != nil {
		ginkgo.Fail(fmt.Sprintf("Failed to create WebLogic credentials secret: %v", err))
	}
	pkg.Log(pkg.Info, "Create database credentials secret")
	if _, err := pkg.CreateCredentialsSecretFromMap("bobs-books", "mysql-credentials",
		map[string]string{"password": dbPass, "username": wlsUser, "url": "jdbc:mysql://mysql.bobs-books.svc.cluster.local:3306/books"}, nil); err != nil {
		ginkgo.Fail(fmt.Sprintf("Failed to create WebLogic credentials secret: %v", err))
	}
	pkg.Log(pkg.Info, "Create logging scope resource")
	if err := pkg.CreateOrUpdateResourceFromFile("examples/bobs-books/bobs-books-logging-scope.yaml"); err != nil {
		ginkgo.Fail(fmt.Sprintf("Failed to create Bobs Books logging scope resource: %v", err))
	}
	pkg.Log(pkg.Info, "Create component resources")
	if err := pkg.CreateOrUpdateResourceFromFile("examples/bobs-books/bobs-books-comp.yaml"); err != nil {
		ginkgo.Fail(fmt.Sprintf("Failed to create Bobs Books component resources: %v", err))
	}
	pkg.Log(pkg.Info, "Create application resources")
	if err := pkg.CreateOrUpdateResourceFromFile("examples/bobs-books/bobs-books-app.yaml"); err != nil {
		ginkgo.Fail(fmt.Sprintf("Failed to create Bobs Books application resource: %v", err))
	}
}

func undeployBobsBooksExample() {
	pkg.Log(pkg.Info, "Undeploy BobsBooks example")
	pkg.Log(pkg.Info, "Delete application")
	if err := pkg.DeleteResourceFromFile("examples/bobs-books/bobs-books-app.yaml"); err != nil {
		pkg.Log(pkg.Error, fmt.Sprintf("Failed to delete application: %v", err))
	}
	pkg.Log(pkg.Info, "Delete components")
	if err := pkg.DeleteResourceFromFile("examples/bobs-books/bobs-books-comp.yaml"); err != nil {
		pkg.Log(pkg.Error, fmt.Sprintf("Failed to delete components: %v", err))
	}
	pkg.Log(pkg.Info, "Delete namespace")
	if err := pkg.DeleteNamespace("bobs-books"); err != nil {
		pkg.Log(pkg.Error, fmt.Sprintf("Failed to delete namespace: %v", err))
	}
	gomega.Eventually(func() bool {
		ns, err := pkg.GetNamespace("bobs-books")
		return ns == nil && err != nil && errors.IsNotFound(err)
	}, 3*time.Minute, 15*time.Second).Should(gomega.BeFalse())
}

var _ = ginkgo.Describe("Verify Bobs Books example application.", func() {
	ginkgo.Context("Deployment.", func() {
		// GIVEN the Bobs Books app is deployed
		// WHEN the running pods are checked
		// THEN the adminserver and managed server pods should be found running
		ginkgo.It("Verify expected pods are running", func() {
			gomega.Eventually(func() bool {
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
			}, longWaitTimeout, longPollingInterval).Should(gomega.BeTrue())
		})
	})
	var host = ""
	// Get the host from the Istio gateway resource.
	// GIVEN the Istio gateway for the bobs-books namespace
	// WHEN GetHostnameFromGateway is called
	// THEN return the host name found in the gateway.
	ginkgo.It("Get host from gateway.", func() {
		gomega.Eventually(func() string {
			host = pkg.GetHostnameFromGateway("bobs-books", "")
			return host
		}, shortWaitTimeout, shortPollingInterval).Should(gomega.Not(gomega.BeEmpty()))
	})
	ginkgo.Context("Ingress.", func() {
		// Verify the application endpoint is working.
		// GIVEN the Bobs Books app is deployed
		// WHEN the roberts-books UI is accessed
		// THEN the expected returned page should contain an expected value.
		ginkgo.It("Verify roberts-books UI endpoint is working.", func() {
			gomega.Eventually(func() pkg.WebResponse {
				url := fmt.Sprintf("https://%s", host)
				status, content := pkg.GetWebPageWithCABundle(url, host)
				return pkg.WebResponse{
					Status:  status,
					Content: content,
				}
			}, shortWaitTimeout, shortPollingInterval).Should(gomega.And(pkg.HaveStatus(200), pkg.ContainContent("Robert's Books")))
		})
		// Verify the application endpoint is working.
		// GIVEN the Bobs Books app is deployed
		// WHEN the bobbys-books UI is accessed
		// THEN the expected returned page should contain an expected value.
		ginkgo.It("Verify bobbys-books UI endpoint is working.", func() {
			gomega.Eventually(func() pkg.WebResponse {
				url := fmt.Sprintf("https://%s/bobbys-front-end/", host)
				status, content := pkg.GetWebPageWithCABundle(url, host)
				return pkg.WebResponse{
					Status:  status,
					Content: content,
				}
			}, shortWaitTimeout, shortPollingInterval).Should(gomega.And(pkg.HaveStatus(200), pkg.ContainContent("Bobby's Books")))
		})
		// Verify the application endpoint is working.
		// GIVEN the Bobs Books app is deployed
		// WHEN the bobs-orders UI is accessed
		// THEN the expected returned page should contain an expected value.
		ginkgo.It("Verify bobs-orders UI endpoint is working.", func() {
			gomega.Eventually(func() pkg.WebResponse {
				url := fmt.Sprintf("https://%s/bobs-bookstore-order-manager/orders", host)
				status, content := pkg.GetWebPageWithCABundle(url, host)
				return pkg.WebResponse{
					Status:  status,
					Content: content,
				}
			}, shortWaitTimeout, shortPollingInterval).Should(gomega.And(pkg.HaveStatus(200), pkg.ContainContent("Bob's Order Manager")))
		})
	})
	ginkgo.Context("Metrics.", func() {
		// Verify application Prometheus scraped metrics
		// GIVEN a deployed Bob's Books application
		// WHEN the application configuration uses a default metrics trait
		// THEN confirm that metrics are being collected
		ginkgo.It("Retrieve application Prometheus scraped metrics", func() {
			pkg.Concurrently(
				func() {
					gomega.Eventually(func() bool {
						return pkg.MetricsExist("base_jvm_uptime_seconds", "app", "bobbys-helidon-stock-application")
					}, shortWaitTimeout, shortPollingInterval).Should(gomega.BeTrue())
				},
				func() {
					gomega.Eventually(func() bool {
						return pkg.MetricsExist("base_jvm_uptime_seconds", "app", "robert-helidon")
					}, shortWaitTimeout, shortPollingInterval).Should(gomega.BeTrue())
				},
				func() {
					gomega.Eventually(func() bool {
						return pkg.MetricsExist("vendor_requests_count_total", "app_oam_dev_component", "bobby-helidon")
					}, shortWaitTimeout, shortPollingInterval).Should(gomega.BeTrue())
				},
				func() {
					gomega.Eventually(func() bool {
						return pkg.MetricsExist("vendor_requests_count_total", "app_oam_dev_component", "robert-helidon")
					}, longWaitTimeout, longPollingInterval).Should(gomega.BeTrue())
				},
				func() {
					gomega.Eventually(func() bool {
						return pkg.MetricsExist("wls_jvm_process_cpu_load", "weblogic_domainName", "bobbys-front-end")
					}, longWaitTimeout, longPollingInterval).Should(gomega.BeTrue())
				},
				func() {
					gomega.Eventually(func() bool {
						return pkg.MetricsExist("wls_jvm_process_cpu_load", "weblogic_domainName", "bobs-bookstore")
					}, shortWaitTimeout, shortPollingInterval).Should(gomega.BeTrue())
				},
			)
		})
		// Verify Istio Prometheus scraped metrics
		// GIVEN a deployed Bob's Books application
		// WHEN the application configuration is deployed
		// THEN confirm that Istio metrics are being collected
		ginkgo.It("Retrieve Istio Prometheus scraped metrics", func() {
			pkg.Concurrently(
				func() {
					gomega.Eventually(func() bool {
						return pkg.MetricsExist("istio_tcp_received_bytes_total", "destination_canonical_service", "bobbys-helidon-stock-application")
					}, shortWaitTimeout, shortPollingInterval).Should(gomega.BeTrue())
				},
				func() {
					gomega.Eventually(func() bool {
						return pkg.MetricsExist("istio_tcp_received_bytes_total", "destination_canonical_service", "robert-helidon")
					}, shortWaitTimeout, shortPollingInterval).Should(gomega.BeTrue())
				},
				func() {
					gomega.Eventually(func() bool {
						return pkg.MetricsExist("istio_tcp_received_bytes_total", "destination_canonical_service", "bobbys-front-end-adminserver")
					}, shortWaitTimeout, shortPollingInterval).Should(gomega.BeTrue())
				},
				func() {
					gomega.Eventually(func() bool {
						return pkg.MetricsExist("istio_tcp_received_bytes_total", "destination_canonical_service", "bobs-bookstore-adminserver")
					}, longWaitTimeout, longPollingInterval).Should(gomega.BeTrue())
				},
				func() {
					gomega.Eventually(func() bool {
						return pkg.MetricsExist("envoy_cluster_ssl_handshake", "pod_name", "bobbys-front-end-adminserver")
					}, longWaitTimeout, longPollingInterval).Should(gomega.BeTrue())
				},
				func() {
					gomega.Eventually(func() bool {
						return pkg.MetricsExist("envoy_cluster_ssl_handshake", "pod_name", "bobs-bookstore-adminserver")
					}, shortWaitTimeout, shortPollingInterval).Should(gomega.BeTrue())
				},
			)
		})
	})
	ginkgo.Context("WebLogic logging.", func() {
		bobbyIndexName := "bobs-books-bobs-books-bobby-wls"
		// GIVEN a WebLogic application with logging enabled via a logging scope
		// WHEN the Elasticsearch index is retrieved
		// THEN verify that it is found
		ginkgo.It("Verify Elasticsearch index exists", func() {
			gomega.Eventually(func() bool {
				return pkg.LogIndexFound(bobbyIndexName)
			}, shortWaitTimeout, shortPollingInterval).Should(gomega.BeTrue(), "Expected to find log index "+bobbyIndexName)
		})
		// GIVEN a WebLogic application with logging enabled via a logging scope
		// WHEN the log records are retrieved from the Elasticsearch index
		// THEN verify that at least one recent log record is found
		ginkgo.It("Verify recent Elasticsearch log record exists", func() {
			gomega.Eventually(func() bool {
				return pkg.LogRecordFound(bobbyIndexName, time.Now().Add(-24*time.Hour), map[string]string{
					"domainUID":  "bobbys-front-end",
					"serverName": "bobbys-front-end-adminserver"})
			}, shortWaitTimeout, shortPollingInterval).Should(gomega.BeTrue(), "Expected to find a recent log record")
		})
		bobIndexName := "bobs-books-bobs-books-bobs-orders-wls"
		// GIVEN a WebLogic application with logging enabled via a logging scope
		// WHEN the Elasticsearch index is retrieved
		// THEN verify that it is found
		ginkgo.It("Verify Elasticsearch index exists", func() {
			gomega.Eventually(func() bool {
				return pkg.LogIndexFound(bobIndexName)
			}, shortWaitTimeout, shortPollingInterval).Should(gomega.BeTrue(), "Expected to find log index "+bobIndexName)
		})
		// GIVEN a WebLogic application with logging enabled via a logging scope
		// WHEN the log records are retrieved from the Elasticsearch index
		// THEN verify that at least one recent log record is found
		ginkgo.It("Verify recent Elasticsearch log record exists", func() {
			gomega.Eventually(func() bool {
				return pkg.LogRecordFound(bobIndexName, time.Now().Add(-24*time.Hour), map[string]string{
					"domainUID":  "bobs-bookstore",
					"serverName": "bobs-bookstore-adminserver"})
			}, shortWaitTimeout, shortPollingInterval).Should(gomega.BeTrue(), "Expected to find a recent log record")
		})
	})
	ginkgo.Context("Coherence logging.", func() {
		indexName := "bobs-books-bobs-books-robert-coh"
		// GIVEN a Coherence application with logging enabled via a logging scope
		// WHEN the Elasticsearch index is retrieved
		// THEN verify that it is found
		ginkgo.It("Verify Elasticsearch index exists", func() {
			gomega.Eventually(func() bool {
				return pkg.LogIndexFound(indexName)
			}, shortWaitTimeout, shortPollingInterval).Should(gomega.BeTrue(), "Expected to find log index "+indexName)
		})
		// GIVEN a Coherence application with logging enabled via a logging scope
		// WHEN the log records are retrieved from the Elasticsearch index
		// THEN verify that at least one recent log record is found
		ginkgo.It("Verify recent Elasticsearch log record exists", func() {
			gomega.Eventually(func() bool {
				return pkg.LogRecordFound(indexName, time.Now().Add(-24*time.Hour), map[string]string{
					"coherence.cluster.name": "roberts-coherence"})
			}, shortWaitTimeout, shortPollingInterval).Should(gomega.BeTrue(), "Expected to find a recent log record")
		})
	})
})
