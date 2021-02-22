// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package bobs_books

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
	if _, err := pkg.CreateNamespace("bobs-books", map[string]string{"verrazzano-managed": "true"}); err != nil {
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
	ginkgo.Context("Ingress.", func() {
		// Verify the application endpoint is working.
		// GIVEN the Bobs Books app is deployed
		// WHEN the roberts-books UI is accessed
		// THEN the expected returned page should contain an expected value.
		ginkgo.It("Verify roberts-books UI endpoint is working.", func() {
			gomega.Eventually(func() pkg.WebResponse {
				ingress := pkg.Ingress()
				pkg.Log(pkg.Info, fmt.Sprintf("Ingress: %s", ingress))
				url := fmt.Sprintf("http://%s", ingress)
				host := pkg.GetHostnameFromGateway("bobs-books", "robert")
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
				ingress := pkg.Ingress()
				pkg.Log(pkg.Info, fmt.Sprintf("Ingress: %s", ingress))
				url := fmt.Sprintf("http://%s/bobbys-front-end/", ingress)
				host := pkg.GetHostnameFromGateway("bobs-books", "bobby-front-end")
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
				ingress := pkg.Ingress()
				pkg.Log(pkg.Info, fmt.Sprintf("Ingress: %s", ingress))
				url := fmt.Sprintf("http://%s/bobs-bookstore-order-manager/orders", ingress)
				host := pkg.GetHostnameFromGateway("bobs-books", "bobs-orders-wls")
				status, content := pkg.GetWebPageWithCABundle(url, host)
				return pkg.WebResponse{
					Status:  status,
					Content: content,
				}
			}, shortWaitTimeout, shortPollingInterval).Should(gomega.And(pkg.HaveStatus(200), pkg.ContainContent("Bob's Order Manager")))
		})
	})
	ginkgo.Context("Metrics.", func() {
		// Verify Prometheus scraped metrics
		// GIVEN a deployed WebLogic application
		// WHEN the application configuration uses a default metrics trait
		// THEN confirm that metrics are being collected
		ginkgo.It("Retrieve Prometheus scraped metrics", func() {
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
	})
	// disabling this test until we can fix the bug caused by the fact that we write a single FLUENTD configmap
	// which causes a Coherence configmap to get loaded by WebLogic pods depending on timing of writing the configmap
	// ginkgo.Context("Logging.", func() {
	// 	indexName := "bobs-books-bobby-front-end-bobby-wls"
	// 	// GIVEN a WebLogic application with logging enabled via a logging scope
	// 	// WHEN the Elasticsearch index is retrieved
	// 	// THEN verify that it is found
	// 	ginkgo.It("Verify Elasticsearch index exists", func() {
	// 		gomega.Eventually(func() bool {
	// 			return pkg.LogIndexFound(indexName)
	// 		}, shortWaitTimeout, shortPollingInterval).Should(gomega.BeTrue(), "Expected to find log index for bobs-books")
	// 	})
	// 	// GIVEN a WebLogic application with logging enabled via a logging scope
	// 	// WHEN the log records are retrieved from the Elasticsearch index
	// 	// THEN verify that at least one recent log record is found
	// 	ginkgo.It("Verify recent Elasticsearch log record exists", func() {
	// 		gomega.Eventually(func() bool {
	// 			return pkg.LogRecordFound(indexName, time.Now().Add(-24*time.Hour), map[string]string{
	// 				"domainUID":  "bobbys-front-end",
	// 				"serverName": "bobbys-front-end-adminserver"})
	// 		}, shortWaitTimeout, shortPollingInterval).Should(gomega.BeTrue(), "Expected to find a recent log record")
	// 	})
	// })
})
