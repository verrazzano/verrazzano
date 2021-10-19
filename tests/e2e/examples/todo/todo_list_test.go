// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package todo

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/verrazzano/verrazzano/pkg/k8sutil"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg/weblogic"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
)

const (
	shortWaitTimeout     = 10 * time.Minute
	shortPollingInterval = 10 * time.Second
	longWaitTimeout      = 15 * time.Minute
	longPollingInterval  = 20 * time.Second
)

var _ = BeforeSuite(func() {
	if skipInstall != "true" {
		deployToDoListExample()
	}
})

var failed = false
var _ = AfterEach(func() {
	failed = failed || CurrentGinkgoTestDescription().Failed
})

var _ = AfterSuite(func() {
	if failed {
		pkg.ExecuteClusterDumpWithEnvVarConfig()
	}
	if skipUninstall != "true" {
		undeployToDoListExample()
	}
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
	Eventually(func() (*v1.Namespace, error) {
		nsLabels := map[string]string{
			"verrazzano-managed": "true",
			"istio-injection":    "enabled"}
		return pkg.CreateNamespace("todo-list", nsLabels)
	}, shortWaitTimeout, shortPollingInterval).ShouldNot(BeNil())

	pkg.Log(pkg.Info, "Create Docker repository secret")
	Eventually(func() (*v1.Secret, error) {
		return pkg.CreateDockerSecret("todo-list", "tododomain-repo-credentials", regServ, regUser, regPass)
	}, shortWaitTimeout, shortPollingInterval).ShouldNot(BeNil())

	pkg.Log(pkg.Info, "Create WebLogic credentials secret")
	Eventually(func() (*v1.Secret, error) {
		return pkg.CreateCredentialsSecret("todo-list", "tododomain-weblogic-credentials", wlsUser, wlsPass, nil)
	}, shortWaitTimeout, shortPollingInterval).ShouldNot(BeNil())

	pkg.Log(pkg.Info, "Create database credentials secret")
	Eventually(func() (*v1.Secret, error) {
		return pkg.CreateCredentialsSecret("todo-list", "tododomain-jdbc-tododb", wlsUser, dbPass, map[string]string{"weblogic.domainUID": "tododomain"})
	}, shortWaitTimeout, shortPollingInterval).ShouldNot(BeNil())

	pkg.Log(pkg.Info, "Create component resources")
	Eventually(func() error {
		return pkg.CreateOrUpdateResourceFromFile("examples/todo-list/todo-list-components.yaml")
	}, shortWaitTimeout, shortPollingInterval).ShouldNot(HaveOccurred())

	pkg.Log(pkg.Info, "Create application resources")
	Eventually(func() error {
		return pkg.CreateOrUpdateResourceFromFile("examples/todo-list/todo-list-application.yaml")
	}, shortWaitTimeout, shortPollingInterval, "Failed to create application resource").ShouldNot(HaveOccurred())
}

func undeployToDoListExample() {
	pkg.Log(pkg.Info, "Undeploy ToDoList example")
	pkg.Log(pkg.Info, "Delete application")
	Eventually(func() error {
		return pkg.DeleteResourceFromFile("examples/todo-list/todo-list-application.yaml")
	}, shortWaitTimeout, shortPollingInterval).ShouldNot(HaveOccurred())

	pkg.Log(pkg.Info, "Delete components")
	Eventually(func() error {
		return pkg.DeleteResourceFromFile("examples/todo-list/todo-list-components.yaml")
	}, shortWaitTimeout, shortPollingInterval).ShouldNot(HaveOccurred())

	pkg.Log(pkg.Info, "Delete namespace")
	Eventually(func() error {
		return pkg.DeleteNamespace("todo-list")
	}, shortWaitTimeout, shortPollingInterval).ShouldNot(HaveOccurred())

	pkg.Log(pkg.Info, "Deleted namespace check")
	Eventually(func() bool {
		_, err := pkg.GetNamespace("todo-list")
		return err != nil && errors.IsNotFound(err)
	}, shortWaitTimeout, shortPollingInterval).Should(BeTrue())

	// GIVEN the ToDoList app is undeployed
	// WHEN the app config certificate generated to support secure gateways is fetched
	// THEN the certificate should have been cleaned up
	pkg.Log(pkg.Info, "Deleted certificate check")
	Eventually(func() bool {
		_, err := pkg.GetCertificate("istio-system", "todo-list-todo-appconf-cert")
		return err != nil && errors.IsNotFound(err)
	}, shortWaitTimeout, shortPollingInterval).Should(BeTrue())

	// GIVEN the ToDoList app is undeployed
	// WHEN the app config secret generated to support secure gateways is fetched
	// THEN the secret should have been cleaned up
	pkg.Log(pkg.Info, "Waiting for secret containing certificate to be deleted")
	var secret *v1.Secret
	var err error
	for i := 0; i < 30; i++ {
		secret, err = pkg.GetSecret("istio-system", "todo-list-todo-appconf-cert-secret")
		if err != nil && errors.IsNotFound(err) {
			pkg.Log(pkg.Info, "Secret deleted")
			return
		}
		if err != nil {
			pkg.Log(pkg.Error, fmt.Sprintf("Error attempting to get secret: %v", err))
		}
		time.Sleep(shortPollingInterval)
	}

	pkg.Log(pkg.Error, "Secret could not be deleted. Secret data:")
	if secret != nil {
		if b, err := json.Marshal(secret); err == nil {
			pkg.Log(pkg.Info, string(b))
		}
	}
	pkg.ExecuteClusterDumpWithEnvVarConfig()
	Fail("Unable to delete secret")
}

var _ = Describe("Verify ToDo List example application.", func() {

	Context("Deployment.", func() {
		// GIVEN the ToDoList app is deployed
		// WHEN the running pods are checked
		// THEN the adminserver and mysql pods should be found running
		It("Verify 'tododomain-adminserver' and 'mysql' pods are running", func() {
			Eventually(func() bool {
				return pkg.PodsRunning("todo-list", []string{"mysql", "tododomain-adminserver"})
			}, longWaitTimeout, longPollingInterval).Should(BeTrue())
		})
		// GIVEN the ToDoList app is deployed
		// WHEN the app config secret generated to support secure gateways is fetched
		// THEN the secret should exist
		It("Verify 'todo-list-todo-appconf-cert-secret' has been created", func() {
			Eventually(func() (*v1.Secret, error) {
				return pkg.GetSecret("istio-system", "todo-list-todo-appconf-cert-secret")
			}, longWaitTimeout, longPollingInterval).ShouldNot(BeNil())
		})
		// GIVEN the ToDoList app is deployed
		// WHEN the servers in the WebLogic domain is ready
		// THEN the domain.servers.status.health.overallHeath fields should be ok
		It("Verify 'todo-domain' overall health is ok", func() {
			Eventually(func() bool {
				domain, err := weblogic.GetDomain("todo-list", "todo-domain")
				if err != nil {
					return false
				}
				healths, err := weblogic.GetHealthOfServers(domain)
				if err != nil || healths[0] != weblogic.Healthy {
					return false
				}
				return true
			}, longWaitTimeout, longPollingInterval).Should(BeTrue())
		})

	})

	Context("Ingress.", func() {
		var host = ""
		var err error
		// Get the host from the Istio gateway resource.
		// GIVEN the Istio gateway for the todo-list namespace
		// WHEN GetHostnameFromGateway is called
		// THEN return the host name found in the gateway.
		It("Get host from gateway.", func() {
			Eventually(func() (string, error) {
				host, err = k8sutil.GetHostnameFromGateway("todo-list", "")
				return host, err
			}, shortWaitTimeout, shortPollingInterval).Should(Not(BeEmpty()))
		})

		// Verify the application REST endpoint is working.
		// GIVEN the ToDoList app is deployed
		// WHEN the UI is accessed
		// THEN the expected returned page should contain an expected value.
		It("Verify '/todo' UI endpoint is working.", func() {
			Eventually(func() (*pkg.HTTPResponse, error) {
				url := fmt.Sprintf("https://%s/todo/", host)
				return pkg.GetWebPage(url, host)
			}, shortWaitTimeout, shortPollingInterval).Should(And(pkg.HasStatus(http.StatusOK), pkg.BodyContains("Derek")))
		})

		// Verify the application REST endpoint is working.
		// GIVEN the ToDoList app is deployed
		// WHEN the REST endpoint is accessed
		// THEN the expected results should be returned
		It("Verify '/todo/rest/items' REST endpoint is working.", func() {
			task := fmt.Sprintf("test-task-%s", time.Now().Format("20060102150405.0000"))
			Eventually(func() (*pkg.HTTPResponse, error) {
				url := fmt.Sprintf("https://%s/todo/rest/items", host)
				return pkg.GetWebPage(url, host)
			}, shortWaitTimeout, shortPollingInterval).Should(And(pkg.HasStatus(http.StatusOK), pkg.BodyContains("[")))
			Eventually(func() (*pkg.HTTPResponse, error) {
				url := fmt.Sprintf("https://%s/todo/rest/item/%s", host, task)
				return pkg.PutWithHostHeader(url, "application/json", host, nil)
			}, shortWaitTimeout, shortPollingInterval).Should(pkg.HasStatus(204))
			Eventually(func() (*pkg.HTTPResponse, error) {
				url := fmt.Sprintf("https://%s/todo/rest/items", host)
				return pkg.GetWebPage(url, host)
			}, shortWaitTimeout, shortPollingInterval).Should(And(pkg.HasStatus(http.StatusOK), pkg.BodyContains(task)))
		})
	})

	Context("Metrics.", func() {
		// Verify Prometheus scraped metrics
		// GIVEN a deployed WebLogic application
		// WHEN the application configuration uses a default metrics trait
		// THEN confirm that metrics are being collected
		It("Retrieve Prometheus scraped metrics", func() {
			pkg.Concurrently(
				func() {
					Eventually(func() bool {
						return pkg.MetricsExist("wls_jvm_process_cpu_load", "app_oam_dev_name", "todo-appconf")
					}, longWaitTimeout, longPollingInterval).Should(BeTrue(), "Expected to find metrics for todo-list")
				},
				func() {
					Eventually(func() bool {
						return pkg.MetricsExist("wls_scrape_mbeans_count_total", "app_oam_dev_name", "todo-appconf")
					}, longWaitTimeout, longPollingInterval).Should(BeTrue(), "Expected to find metrics for todo-list")
				},
			)
		})
	})

	Context("Logging.", func() {
		indexName := "verrazzano-namespace-todo-list"

		// GIVEN a WebLogic application with logging enabled
		// WHEN the Elasticsearch index is retrieved
		// THEN verify that it is found
		It("Verify Elasticsearch index exists", func() {
			Eventually(func() bool {
				return pkg.LogIndexFound(indexName)
			}, longWaitTimeout, longPollingInterval).Should(BeTrue(), "Expected to find log index for todo-list")
		})

		// GIVEN a WebLogic application with logging enabled
		// WHEN the log records are retrieved from the Elasticsearch index
		// THEN verify that at least one recent log record is found
		pkg.Concurrently(
			func() {
				It("Verify recent adminserver log record exists", func() {
					Eventually(func() bool {
						return pkg.LogRecordFound(indexName, time.Now().Add(-24*time.Hour), map[string]string{
							"kubernetes.labels.weblogic_domainUID":  "tododomain",
							"kubernetes.labels.app_oam_dev\\/name":  "todo-appconf",
							"kubernetes.labels.weblogic_serverName": "AdminServer",
							"kubernetes.container_name":             "weblogic-server",
						})
					}, longWaitTimeout, longPollingInterval).Should(BeTrue(), "Expected to find a recent log record")
				})
			},
			func() {
				It("Verify recent pattern-matched AdminServer log record exists", func() {
					Eventually(func() bool {
						return pkg.FindLog(indexName,
							[]pkg.Match{
								{Key: "kubernetes.container_name.keyword", Value: "fluentd-stdout-sidecar"},
								{Key: "messageID", Value: "BEA-"}, //matches BEA-*
								{Key: "serverName", Value: "tododomain-adminserver"},
								{Key: "serverName2", Value: "AdminServer"}},
							[]pkg.Match{})
					}, longWaitTimeout, longPollingInterval).Should(BeTrue(), "Expected to find a recent log record")
				})
			},
			func() {
				It("Verify recent pattern-matched WebLogic Server log record exists", func() {
					Eventually(func() bool {
						return pkg.FindLog(indexName,
							[]pkg.Match{
								{Key: "kubernetes.container_name.keyword", Value: "fluentd-stdout-sidecar"},
								{Key: "messageID", Value: "BEA-"},          //matches BEA-*
								{Key: "message", Value: "WebLogic Server"}, //contains WebLogic Server
								{Key: "subSystem", Value: "WebLogicServer"}},
							[]pkg.Match{})
					}, longWaitTimeout, longPollingInterval).Should(BeTrue(), "Expected to find a recent log record")
				})
			},
			func() {
				It("Verify recent pattern-matched Security log record exists", func() {
					Eventually(func() bool {
						return pkg.FindLog(indexName,
							[]pkg.Match{
								{Key: "kubernetes.container_name.keyword", Value: "fluentd-stdout-sidecar"},
								{Key: "messageID", Value: "BEA-"}, //matches BEA-*
								{Key: "serverName", Value: "tododomain-adminserver"},
								{Key: "subSystem.keyword", Value: "Security"}},
							[]pkg.Match{})
					}, longWaitTimeout, longPollingInterval).Should(BeTrue(), "Expected to find a recent log record")
				})
			},
			func() {
				It("Verify recent pattern-matched multi-lines log record exists", func() {
					Eventually(func() bool {
						return pkg.FindLog(indexName,
							[]pkg.Match{
								{Key: "kubernetes.container_name.keyword", Value: "fluentd-stdout-sidecar"},
								{Key: "messageID", Value: "BEA-"},         //matches BEA-*
								{Key: "message", Value: "Tunneling Ping"}, //"Tunneling Ping" in last line
								{Key: "serverName", Value: "tododomain-adminserver"},
								{Key: "subSystem.keyword", Value: "RJVM"}},
							[]pkg.Match{})
					}, longWaitTimeout, longPollingInterval).Should(BeTrue(), "Expected to find a recent log record")
				})
			},
			func() {
				It("Verify recent fluentd-stdout-sidecar server log record exists", func() {
					Eventually(func() bool {
						return pkg.FindLog(indexName,
							[]pkg.Match{
								{Key: "kubernetes.container_name.keyword", Value: "fluentd-stdout-sidecar"},
								{Key: "wls_log_stream", Value: "server_log"},
								{Key: "stream", Value: "stdout"}},
							[]pkg.Match{})
					}, longWaitTimeout, longPollingInterval).Should(BeTrue(), "Expected to find a recent server log record")
				})
			},
			func() {
				It("Verify recent fluentd-stdout-sidecar domain log record exists", func() {
					Eventually(func() bool {
						return pkg.FindLog(indexName,
							[]pkg.Match{
								{Key: "kubernetes.container_name.keyword", Value: "fluentd-stdout-sidecar"},
								{Key: "wls_log_stream", Value: "domain_log"},
								{Key: "stream", Value: "stdout"}},
							[]pkg.Match{})
					}, longWaitTimeout, longPollingInterval).Should(BeTrue(), "Expected to find a recent domain log record")
				})
			},
			func() {
				It("Verify recent fluentd-stdout-sidecar server nodemanager log record exists", func() {
					Eventually(func() bool {
						return pkg.FindLog(indexName,
							[]pkg.Match{
								{Key: "kubernetes.container_name.keyword", Value: "fluentd-stdout-sidecar"},
								{Key: "wls_log_stream", Value: "server_nodemanager_log"},
								{Key: "stream", Value: "stdout"}},
							[]pkg.Match{})
					}, longWaitTimeout, longPollingInterval).Should(BeTrue(), "Expected to find a recent server nodemanager log record")
				})
			},
			// GIVEN a WebLogic application with logging enabled
			// WHEN the log records are retrieved from the Elasticsearch index
			// THEN verify that a recent pattern-matched log record of tododomain-adminserver stdout is found
			func() {
				It("Verify recent pattern-matched AdminServer log record exists", func() {
					Eventually(func() bool {
						return pkg.FindLog(indexName,
							[]pkg.Match{
								{Key: "kubernetes.container_name.keyword", Value: "fluentd-stdout-sidecar"},
								{Key: "subSystem.keyword", Value: "WorkManager"},
								{Key: "serverName.keyword", Value: "tododomain-adminserver"},
								{Key: "serverName2.keyword", Value: "AdminServer"},
								{Key: "message", Value: "standby threads"}},
							[]pkg.Match{})
					}, longWaitTimeout, longPollingInterval).Should(BeTrue(), "Expected to find a recent log record")
				})
			},
			// GIVEN a WebLogic application with logging enabled
			// WHEN the log records are retrieved from the Elasticsearch index
			// THEN verify that a recent pattern-matched log record of tododomain-adminserver stdout is found
			func() {
				It("Verify recent pattern-matched AdminServer log record exists", func() {
					Eventually(func() bool {
						return pkg.FindLog(indexName,
							[]pkg.Match{
								{Key: "kubernetes.container_name.keyword", Value: "fluentd-stdout-sidecar"},
								{Key: "subSystem", Value: "WorkManager"},
								{Key: "serverName", Value: "tododomain-adminserver"},
								{Key: "serverName2", Value: "AdminServer"},
								{Key: "message", Value: "Self-tuning"}},
							[]pkg.Match{})
					}, longWaitTimeout, longPollingInterval).Should(BeTrue(), "Expected to find a recent log record")
				})
			},
		)

		// GIVEN a WebLogic application with logging enabled
		// WHEN the log records are retrieved from the Elasticsearch index
		// THEN verify that no 'pattern not matched' log record of fluentd-stdout-sidecar is found
		It("Verify recent 'pattern not matched' log records do not exist", func() {
			Expect(pkg.NoLog(indexName,
				[]pkg.Match{
					{Key: "kubernetes.container_name.keyword", Value: "fluentd-stdout-sidecar"},
					{Key: "message", Value: "pattern not matched"}},
				[]pkg.Match{})).To(BeTrue())
		})
	})
})
