// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package todo

import (
	"fmt"
	"time"

	"github.com/onsi/ginkgo"
	"github.com/onsi/gomega"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg/weblogic"
	"k8s.io/apimachinery/pkg/api/errors"
)

const (
	shortWaitTimeout     = 10 * time.Minute
	shortPollingInterval = 10 * time.Second
	longWaitTimeout      = 15 * time.Minute
	longPollingInterval  = 20 * time.Second
)

var _ = ginkgo.BeforeSuite(func() {
	deployToDoListExample()
})

var failed = false
var _ = ginkgo.AfterEach(func() {
	failed = failed || ginkgo.CurrentGinkgoTestDescription().Failed
})

var _ = ginkgo.AfterSuite(func() {
	if failed {
		pkg.ExecuteClusterDumpWithEnvVarConfig()
	}
	undeployToDoListExample()
})

func deployToDoListExample() {
	pkg.Log(pkg.Info, "Deploy ToDoList example")
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
	if _, err := pkg.CreateNamespace("todo-list", nsLabels); err != nil {
		ginkgo.Fail(fmt.Sprintf("Failed to create namespace: %v", err))
	}
	pkg.Log(pkg.Info, "Create Docker repository secret")
	if _, err := pkg.CreateDockerSecret("todo-list", "tododomain-repo-credentials", regServ, regUser, regPass); err != nil {
		ginkgo.Fail(fmt.Sprintf("Failed to create Docker registry secret: %v", err))
	}
	pkg.Log(pkg.Info, "Create WebLogic credentials secret")
	if _, err := pkg.CreateCredentialsSecret("todo-list", "tododomain-weblogic-credentials", wlsUser, wlsPass, nil); err != nil {
		ginkgo.Fail(fmt.Sprintf("Failed to create WebLogic credentials secret: %v", err))
	}
	pkg.Log(pkg.Info, "Create database credentials secret")
	if _, err := pkg.CreateCredentialsSecret("todo-list", "tododomain-jdbc-tododb", wlsUser, dbPass, map[string]string{"weblogic.domainUID": "tododomain"}); err != nil {
		ginkgo.Fail(fmt.Sprintf("Failed to create JDBC credentials secret: %v", err))
	}
	pkg.Log(pkg.Info, "Create component resources")
	if err := pkg.CreateOrUpdateResourceFromFile("examples/todo-list/todo-list-components.yaml"); err != nil {
		ginkgo.Fail(fmt.Sprintf("Failed to create ToDo List component resources: %v", err))
	}
	pkg.Log(pkg.Info, "Create application resources")
	gomega.Eventually(func() error {
		return pkg.CreateOrUpdateResourceFromFile("examples/todo-list/todo-list-application.yaml")
	},
		shortWaitTimeout, shortPollingInterval, "Failed to create application resource").Should(gomega.BeNil())
}

func undeployToDoListExample() {
	pkg.Log(pkg.Info, "Undeploy ToDoList example")
	pkg.Log(pkg.Info, "Delete application")
	if err := pkg.DeleteResourceFromFile("examples/todo-list/todo-list-application.yaml"); err != nil {
		pkg.Log(pkg.Error, fmt.Sprintf("Failed to delete application: %v", err))
	}
	pkg.Log(pkg.Info, "Delete components")
	if err := pkg.DeleteResourceFromFile("examples/todo-list/todo-list-components.yaml"); err != nil {
		pkg.Log(pkg.Error, fmt.Sprintf("Failed to delete components: %v", err))
	}
	pkg.Log(pkg.Info, "Delete namespace")
	if err := pkg.DeleteNamespace("todo-list"); err != nil {
		pkg.Log(pkg.Error, fmt.Sprintf("Failed to delete namespace: %v", err))
	}
	gomega.Eventually(func() bool {
		ns, err := pkg.GetNamespace("todo-list")
		return ns == nil && err != nil && errors.IsNotFound(err)
	}, 3*time.Minute, 15*time.Second).Should(gomega.BeFalse())

	// GIVEN the ToDoList app is undeployed
	// WHEN the app config secret generated to support secure gateways is fetched
	// THEN the secret should have been cleaned up
	gomega.Eventually(func() bool {
		s, err := pkg.GetSecret("istio-system", "todo-list-todo-appconf-cert-secret")
		return s == nil && err != nil && errors.IsNotFound(err)
	}, shortWaitTimeout, shortPollingInterval).Should(gomega.BeTrue())
}

var _ = ginkgo.Describe("Verify ToDo List example application.", func() {

	ginkgo.Context("Deployment.", func() {
		// GIVEN the ToDoList app is deployed
		// WHEN the running pods are checked
		// THEN the adminserver and mysql pods should be found running
		ginkgo.It("Verify 'tododomain-adminserver' and 'mysql' pods are running", func() {
			gomega.Eventually(func() bool {
				return pkg.PodsRunning("todo-list", []string{"mysql", "tododomain-adminserver"})
			}, longWaitTimeout, longPollingInterval).Should(gomega.BeTrue())
		})
		// GIVEN the ToDoList app is deployed
		// WHEN the app config secret generated to support secure gateways is fetched
		// THEN the secret should exist
		ginkgo.It("Verify 'todo-list-todo-appconf-cert-secret' has been created", func() {
			gomega.Eventually(func() bool {
				s, err := pkg.GetSecret("istio-system", "todo-list-todo-appconf-cert-secret")
				return s != nil && err == nil
			}, longWaitTimeout, longPollingInterval).Should(gomega.BeTrue())
		})
		// GIVEN the ToDoList app is deployed
		// WHEN the servers in the WebLogic domain is ready
		// THEN the domain.servers.status.health.overallHeath fields should be ok
		ginkgo.It("Verify 'todo-domain' overall health is ok", func() {
			gomega.Eventually(func() bool {
				domain, err := weblogic.GetDomain("todo-list", "todo-domain")
				if err != nil {
					return false
				}
				healths, err := weblogic.GetHealthOfServers(domain)
				if err != nil || healths[0] != weblogic.Healthy {
					return false
				}
				return true
			}, longWaitTimeout, longPollingInterval).Should(gomega.BeTrue())
		})

	})

	ginkgo.Context("Ingress.", func() {
		var host = ""
		// Get the host from the Istio gateway resource.
		// GIVEN the Istio gateway for the todo-list namespace
		// WHEN GetHostnameFromGateway is called
		// THEN return the host name found in the gateway.
		ginkgo.It("Get host from gateway.", func() {
			gomega.Eventually(func() string {
				host = pkg.GetHostnameFromGateway("todo-list", "")
				return host
			}, shortWaitTimeout, shortPollingInterval).Should(gomega.Not(gomega.BeEmpty()))
		})

		// Verify the application REST endpoint is working.
		// GIVEN the ToDoList app is deployed
		// WHEN the UI is accessed
		// THEN the expected returned page should contain an expected value.
		ginkgo.It("Verify '/todo' UI endpoint is working.", func() {
			gomega.Eventually(func() pkg.WebResponse {
				url := fmt.Sprintf("https://%s/todo/", host)
				status, content := pkg.GetWebPageWithCABundle(url, host)
				return pkg.WebResponse{
					Status:  status,
					Content: content,
				}
			}, shortWaitTimeout, shortPollingInterval).Should(gomega.And(pkg.HaveStatus(200), pkg.ContainContent("Derek")))
		})

		// Verify the application REST endpoint is working.
		// GIVEN the ToDoList app is deployed
		// WHEN the REST endpoint is accessed
		// THEN the expected results should be returned
		ginkgo.It("Verify '/todo/rest/items' REST endpoint is working.", func() {
			task := fmt.Sprintf("test-task-%s", time.Now().Format("20060102150405.0000"))
			gomega.Eventually(func() pkg.WebResponse {
				url := fmt.Sprintf("https://%s/todo/rest/items", host)
				status, content := pkg.GetWebPageWithCABundle(url, host)
				return pkg.WebResponse{
					Status:  status,
					Content: content,
				}
			}, shortWaitTimeout, shortPollingInterval).Should(gomega.And(pkg.HaveStatus(200), pkg.ContainContent("[")))
			gomega.Eventually(func() pkg.WebResponse {
				url := fmt.Sprintf("https://%s/todo/rest/item/%s", host, task)
				status, content := pkg.PutWithHostHeader(url, "application/json", host, nil)
				return pkg.WebResponse{
					Status:  status,
					Content: content,
				}
			}, shortWaitTimeout, shortPollingInterval).Should(pkg.HaveStatus(204))
			gomega.Eventually(func() pkg.WebResponse {
				url := fmt.Sprintf("https://%s/todo/rest/items", host)
				status, content := pkg.GetWebPageWithCABundle(url, host)
				return pkg.WebResponse{
					Status:  status,
					Content: content,
				}
			}, shortWaitTimeout, shortPollingInterval).Should(gomega.And(pkg.HaveStatus(200), pkg.ContainContent(task)))
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
						return pkg.MetricsExist("wls_jvm_process_cpu_load", "app_oam_dev_name", "todo-appconf")
					}, longWaitTimeout, longPollingInterval).Should(gomega.BeTrue(), "Expected to find metrics for todo-list")
				},
				func() {
					gomega.Eventually(func() bool {
						return pkg.MetricsExist("wls_scrape_mbeans_count_total", "app_oam_dev_name", "todo-appconf")
					}, longWaitTimeout, longPollingInterval).Should(gomega.BeTrue(), "Expected to find metrics for todo-list")
				},
			)
		})
	})

	ginkgo.Context("Logging.", func() {
		indexName := "verrazzano-namespace-todo-list"

		// GIVEN a WebLogic application with logging enabled
		// WHEN the Elasticsearch index is retrieved
		// THEN verify that it is found
		ginkgo.It("Verify Elasticsearch index exists", func() {
			gomega.Eventually(func() bool {
				return pkg.LogIndexFound(indexName)
			}, longWaitTimeout, longPollingInterval).Should(gomega.BeTrue(), "Expected to find log index for todo-list")
		})

		// GIVEN a WebLogic application with logging enabled
		// WHEN the log records are retrieved from the Elasticsearch index
		// THEN verify that at least one recent log record is found
		pkg.Concurrently(
			func() {
				ginkgo.It("Verify recent adminserver log record exists", func() {
					gomega.Eventually(func() bool {
						return pkg.LogRecordFound(indexName, time.Now().Add(-24*time.Hour), map[string]string{
							"kubernetes.labels.weblogic_domainUID":  "tododomain",
							"kubernetes.labels.app_oam_dev\\/name":  "todo-appconf",
							"kubernetes.labels.weblogic_serverName": "AdminServer",
							"kubernetes.container_name":             "weblogic-server",
						})
					}, longWaitTimeout, longPollingInterval).Should(gomega.BeTrue(), "Expected to find a recent log record")
				})
			},
			func() {
				ginkgo.It("Verify recent pattern-matched AdminServer log record exists", func() {
					gomega.Eventually(func() bool {
						return pkg.FindLog(indexName,
							[]pkg.Match{
								{Key: "kubernetes.container_name.keyword", Value: "fluentd-stdout-sidecar"},
								{Key: "messageID", Value: "BEA-"}, //matches BEA-*
								{Key: "serverName", Value: "tododomain-adminserver"},
								{Key: "serverName2", Value: "AdminServer"}},
							[]pkg.Match{})
					}, longWaitTimeout, longPollingInterval).Should(gomega.BeTrue(), "Expected to find a recent log record")
				})
			},
			func() {
				ginkgo.It("Verify recent pattern-matched WebLogic Server log record exists", func() {
					gomega.Eventually(func() bool {
						return pkg.FindLog(indexName,
							[]pkg.Match{
								{Key: "kubernetes.container_name.keyword", Value: "fluentd-stdout-sidecar"},
								{Key: "messageID", Value: "BEA-"},          //matches BEA-*
								{Key: "message", Value: "WebLogic Server"}, //contains WebLogic Server
								{Key: "subSystem", Value: "WebLogicServer"}},
							[]pkg.Match{})
					}, longWaitTimeout, longPollingInterval).Should(gomega.BeTrue(), "Expected to find a recent log record")
				})
			},
			func() {
				ginkgo.It("Verify recent pattern-matched Security log record exists", func() {
					gomega.Eventually(func() bool {
						return pkg.FindLog(indexName,
							[]pkg.Match{
								{Key: "kubernetes.container_name.keyword", Value: "fluentd-stdout-sidecar"},
								{Key: "messageID", Value: "BEA-"}, //matches BEA-*
								{Key: "serverName", Value: "tododomain-adminserver"},
								{Key: "subSystem.keyword", Value: "Security"}},
							[]pkg.Match{})
					}, longWaitTimeout, longPollingInterval).Should(gomega.BeTrue(), "Expected to find a recent log record")
				})
			},
			func() {
				ginkgo.It("Verify recent pattern-matched multi-lines log record exists", func() {
					gomega.Eventually(func() bool {
						return pkg.FindLog(indexName,
							[]pkg.Match{
								{Key: "kubernetes.container_name.keyword", Value: "fluentd-stdout-sidecar"},
								{Key: "messageID", Value: "BEA-"},         //matches BEA-*
								{Key: "message", Value: "Tunneling Ping"}, //"Tunneling Ping" in last line
								{Key: "serverName", Value: "tododomain-adminserver"},
								{Key: "subSystem.keyword", Value: "RJVM"}},
							[]pkg.Match{})
					}, longWaitTimeout, longPollingInterval).Should(gomega.BeTrue(), "Expected to find a recent log record")
				})
			},
			func() {
				ginkgo.It("Verify recent fluentd-stdout-sidecar log record exists", func() {
					gomega.Eventually(func() bool {
						return pkg.FindLog(indexName,
							[]pkg.Match{
								{Key: "kubernetes.container_name.keyword", Value: "fluentd-stdout-sidecar"},
								{Key: "stream", Value: "stdout"}},
							[]pkg.Match{
								{Key: "serverName.keyword", Value: "tododomain-adminserver"}})
					}, longWaitTimeout, longPollingInterval).Should(gomega.BeTrue(), "Expected to find a recent log record")
				})
			},
			// GIVEN a WebLogic application with logging enabled
			// WHEN the log records are retrieved from the Elasticsearch index
			// THEN verify that a recent pattern-matched log record of tododomain-adminserver stdout is found
			func() {
				ginkgo.It("Verify recent pattern-matched AdminServer log record exists", func() {
					gomega.Eventually(func() bool {
						return pkg.FindLog(indexName,
							[]pkg.Match{
								{Key: "kubernetes.container_name.keyword", Value: "fluentd-stdout-sidecar"},
								{Key: "subSystem.keyword", Value: "WorkManager"},
								{Key: "serverName.keyword", Value: "tododomain-adminserver"},
								{Key: "serverName2.keyword", Value: "AdminServer"},
								{Key: "message", Value: "standby threads"}},
							[]pkg.Match{})
					}, longWaitTimeout, longPollingInterval).Should(gomega.BeTrue(), "Expected to find a recent log record")
				})
			},
			// GIVEN a WebLogic application with logging enabled
			// WHEN the log records are retrieved from the Elasticsearch index
			// THEN verify that a recent pattern-matched log record of tododomain-adminserver stdout is found
			func() {
				ginkgo.It("Verify recent pattern-matched AdminServer log record exists", func() {
					gomega.Eventually(func() bool {
						return pkg.FindLog(indexName,
							[]pkg.Match{
								{Key: "kubernetes.container_name.keyword", Value: "fluentd-stdout-sidecar"},
								{Key: "subSystem", Value: "WorkManager"},
								{Key: "serverName", Value: "tododomain-adminserver"},
								{Key: "serverName2", Value: "AdminServer"},
								{Key: "message", Value: "Self-tuning"}},
							[]pkg.Match{})
					}, longWaitTimeout, longPollingInterval).Should(gomega.BeTrue(), "Expected to find a recent log record")
				})
			},
		)

		// GIVEN a WebLogic application with logging enabled
		// WHEN the log records are retrieved from the Elasticsearch index
		// THEN verify that no 'pattern not matched' log record of fluentd-stdout-sidecar is found
		ginkgo.It("Verify recent 'pattern not matched' log records do not exist", func() {
			gomega.Expect(pkg.NoLog(indexName,
				[]pkg.Match{
					{Key: "kubernetes.container_name.keyword", Value: "fluentd-stdout-sidecar"},
					{Key: "message", Value: "pattern not matched"}},
				[]pkg.Match{})).To(gomega.BeTrue())
		})
	})
})
