// Copyright (c) 2021, 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package todo

import (
	"fmt"
	"net/http"
	"time"

	"github.com/verrazzano/verrazzano/tests/e2e/pkg/test/framework"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg/test/framework/metrics"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/verrazzano/verrazzano/pkg/k8sutil"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg/weblogic"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
)

const (
	shortWaitTimeout         = 10 * time.Minute
	shortPollingInterval     = 10 * time.Second
	longWaitTimeout          = 25 * time.Minute
	longPollingInterval      = 20 * time.Second
	imagePullWaitTimeout     = 40 * time.Minute
	imagePullPollingInterval = 30 * time.Second
)

var (
	t                  = framework.NewTestFramework("todo")
	generatedNamespace = pkg.GenerateNamespace("todo-list")
	clusterDump        = pkg.NewClusterDumpWrapper(generatedNamespace)
)

var _ = clusterDump.BeforeSuite(func() {
	if !skipDeploy {
		start := time.Now()
		deployToDoListExample(namespace)
		metrics.Emit(t.Metrics.With("deployment_elapsed_time", time.Since(start).Milliseconds()))
	}
	t.Logs.Info("Container image pull check")
	Eventually(func() bool {
		return pkg.ContainerImagePullWait(namespace, []string{"mysql", "tododomain-adminserver"})
	}, imagePullWaitTimeout, imagePullPollingInterval).Should(BeTrue())
	// GIVEN the ToDoList app is deployed
	// WHEN the running pods are checked
	// THEN the tododomain-adminserver and mysql pods should be found running
	Eventually(func() bool {
		result, err := pkg.PodsRunning(namespace, []string{"mysql", "tododomain-adminserver"})
		if err != nil {
			AbortSuite(fmt.Sprintf("One or more pods are not running in the namespace: %v, error: %v", namespace, err))
		}
		return result
	}, longWaitTimeout, longPollingInterval).Should(BeTrue())
})

var _ = clusterDump.AfterEach(func() {}) // Dump cluster if spec fails
var _ = clusterDump.AfterSuite(func() {  // Dump cluster if aftersuite fails
	if !skipUndeploy {
		undeployToDoListExample()
	}
})

func deployToDoListExample(namespace string) {
	t.Logs.Info("Deploy ToDoList example")
	wlsUser := "weblogic"
	wlsPass := pkg.GetRequiredEnvVarOrFail("WEBLOGIC_PSW")
	dbPass := pkg.GetRequiredEnvVarOrFail("DATABASE_PSW")
	regServ := pkg.GetRequiredEnvVarOrFail("OCR_REPO")
	regUser := pkg.GetRequiredEnvVarOrFail("OCR_CREDS_USR")
	regPass := pkg.GetRequiredEnvVarOrFail("OCR_CREDS_PSW")

	t.Logs.Info("Create namespace")
	start := time.Now()
	Eventually(func() (*v1.Namespace, error) {
		nsLabels := map[string]string{
			"verrazzano-managed": "true",
			"istio-injection":    istioInjection}
		return pkg.CreateNamespace(namespace, nsLabels)
	}, shortWaitTimeout, shortPollingInterval).ShouldNot(BeNil())

	t.Logs.Info("Create Docker repository secret")
	Eventually(func() (*v1.Secret, error) {
		return pkg.CreateDockerSecret(namespace, "tododomain-repo-credentials", regServ, regUser, regPass)
	}, shortWaitTimeout, shortPollingInterval).ShouldNot(BeNil())

	t.Logs.Info("Create WebLogic credentials secret")
	Eventually(func() (*v1.Secret, error) {
		return pkg.CreateCredentialsSecret(namespace, "tododomain-weblogic-credentials", wlsUser, wlsPass, nil)
	}, shortWaitTimeout, shortPollingInterval).ShouldNot(BeNil())

	t.Logs.Info("Create database credentials secret")
	Eventually(func() (*v1.Secret, error) {
		return pkg.CreateCredentialsSecret(namespace, "tododomain-jdbc-tododb", wlsUser, dbPass, map[string]string{"weblogic.domainUID": "tododomain"})
	}, shortWaitTimeout, shortPollingInterval).ShouldNot(BeNil())

	t.Logs.Info("Create component resources")
	Eventually(func() error {
		return pkg.CreateOrUpdateResourceFromFileInGeneratedNamespace("examples/todo-list/todo-list-components.yaml", namespace)
	}, shortWaitTimeout, shortPollingInterval).ShouldNot(HaveOccurred())

	t.Logs.Info("Create application resources")
	Eventually(func() error {
		return pkg.CreateOrUpdateResourceFromFileInGeneratedNamespace("examples/todo-list/todo-list-application.yaml", namespace)
	}, shortWaitTimeout, shortPollingInterval, "Failed to create application resource").ShouldNot(HaveOccurred())
	metrics.Emit(t.Metrics.With("deployment_elapsed_time", time.Since(start).Milliseconds()))
}

func undeployToDoListExample() {
	t.Logs.Info("Undeploy ToDoList example")
	t.Logs.Info("Delete application")
	start := time.Now()
	Eventually(func() error {
		return pkg.DeleteResourceFromFileInGeneratedNamespace("examples/todo-list/todo-list-application.yaml", namespace)
	}, shortWaitTimeout, shortPollingInterval).ShouldNot(HaveOccurred(), "deleting todo-list-application")

	t.Logs.Info("Delete components")
	Eventually(func() error {
		return pkg.DeleteResourceFromFileInGeneratedNamespace("examples/todo-list/todo-list-components.yaml", namespace)
	}, shortWaitTimeout, shortPollingInterval).ShouldNot(HaveOccurred(), "deleting todo-list-components")

	t.Logs.Info("Wait for pods to terminate")
	Eventually(func() bool {
		podsNotRunning, _ := pkg.PodsNotRunning(namespace, []string{"mysql", "tododomain-adminserver"})
		return podsNotRunning
	}, shortWaitTimeout, shortPollingInterval).Should(BeTrue(), "pods deleted")

	t.Logs.Info("Delete namespace")
	Eventually(func() error {
		return pkg.DeleteNamespace(namespace)
	}, shortWaitTimeout, shortPollingInterval).ShouldNot(HaveOccurred(), "deleting namespace")

	t.Logs.Info("Wait for finalizer to be removed")
	Eventually(func() bool {
		return pkg.CheckNamespaceFinalizerRemoved(namespace)
	}, shortWaitTimeout, shortPollingInterval).Should(BeTrue(), "namespace finalizer deleted")

	t.Logs.Info("Deleted namespace check")
	Eventually(func() bool {
		_, err := pkg.GetNamespace(namespace)
		return err != nil && errors.IsNotFound(err)
	}, shortWaitTimeout, shortPollingInterval).Should(BeTrue(), "namespace deleted")

	// GIVEN the ToDoList app is undeployed
	// WHEN the app config certificate generated to support secure gateways is fetched
	// THEN the certificate should have been cleaned up
	t.Logs.Info("Deleted certificate check")
	Eventually(func() bool {
		_, err := pkg.GetCertificate("istio-system", namespace+"-todo-domain-ingress-cert")
		return err != nil && errors.IsNotFound(err)
	}, shortWaitTimeout, shortPollingInterval).Should(BeTrue(), "ingress trait cert deleted")

	// GIVEN the ToDoList app is undeployed
	// WHEN the app config secret generated to support secure gateways is fetched
	// THEN the secret should have been cleaned up
	t.Logs.Info("Waiting for secret containing certificate to be deleted")
	Eventually(func() bool {
		_, err := pkg.GetSecret("istio-system", namespace+"-todo-domain-ingress-cert-secret")
		if err != nil && errors.IsNotFound(err) {
			t.Logs.Info("Secret deleted")
			return true
		}
		if err != nil {
			t.Logs.Errorf("Error attempting to get secret: %v", err)
		}
		return false
	}, shortWaitTimeout, shortPollingInterval).Should(BeTrue(), "ingress trait cert secret deleted")
	metrics.Emit(t.Metrics.With("undeployment_elapsed_time", time.Since(start).Milliseconds()))
}

var _ = t.AfterEach(func() {})

var _ = t.Describe("ToDo List test", Label("f:app-lcm.oam",
	"f:app-lcm.weblogic-workload"), func() {

	t.Context("application Deployment.", func() {
		// GIVEN the ToDoList app is deployed
		// WHEN the app config secret generated to support secure gateways is fetched
		// THEN the secret should exist
		t.It("Verify cert secret for todo-list has been created", Label("f:cert-mgmt"), func() {
			Eventually(func() (*v1.Secret, error) {
				return pkg.GetSecret("istio-system", namespace+"-todo-domain-ingress-cert-secret")
			}, longWaitTimeout, longPollingInterval).ShouldNot(BeNil())
		})
		// GIVEN the ToDoList app is deployed
		// WHEN the servers in the WebLogic domain is ready
		// THEN the domain.servers.status.health.overallHeath fields should be ok
		t.It("Verify 'todo-domain' overall health is ok", func() {
			Eventually(func() bool {
				domain, err := weblogic.GetDomain(namespace, "todo-domain")
				if err != nil {
					t.Logs.Errorf("Error attempting to get domain: %v", err)
					return false
				}
				healths, err := weblogic.GetHealthOfServers(domain)
				if err != nil {
					t.Logs.Errorf("Error attempting to get health of servers: %v", err)
					return false
				}

				if healths[0] != weblogic.Healthy {
					t.Logs.Errorf("server not healthy")
					return false
				}
				return true
			}, longWaitTimeout, longPollingInterval).Should(BeTrue())
		})

	})

	t.Context("Ingress.", Label("f:mesh.ingress"), func() {
		var host = ""
		var err error
		// Get the host from the Istio gateway resource.
		// GIVEN the Istio gateway for the todo-list namespace
		// WHEN GetHostnameFromGateway is called
		// THEN return the host name found in the gateway.
		t.BeforeEach(func() {
			Eventually(func() (string, error) {
				host, err = k8sutil.GetHostnameFromGateway(namespace, "")
				return host, err
			}, shortWaitTimeout, shortPollingInterval).Should(Not(BeEmpty()))
		})

		// Verify the application REST endpoint is working.
		// GIVEN the ToDoList app is deployed
		// WHEN the UI is accessed
		// THEN the expected returned page should contain an expected value.
		t.It("Verify '/todo' UI endpoint is working.", func() {
			Eventually(func() (*pkg.HTTPResponse, error) {
				url := fmt.Sprintf("https://%s/todo/", host)
				return pkg.GetWebPage(url, host)
			}, shortWaitTimeout, shortPollingInterval).Should(And(pkg.HasStatus(http.StatusOK), pkg.BodyContains("Derek")))
		})

		// Verify the application REST endpoint is working.
		// GIVEN the ToDoList app is deployed
		// WHEN the REST endpoint is accessed
		// THEN the expected results should be returned
		t.It("Verify '/todo/rest/items' REST endpoint is working.", func() {
			task := fmt.Sprintf("test-task-%s", time.Now().Format("20060102150405.0000"))
			Eventually(func() (*pkg.HTTPResponse, error) {
				url := fmt.Sprintf("https://%s/todo/rest/items", host)
				return pkg.GetWebPage(url, host)
			}, shortWaitTimeout, shortPollingInterval).Should(And(pkg.HasStatus(http.StatusOK), pkg.BodyContains("[")))
			Eventually(func() bool {
				return putGetTodoTask(host, task)

			}, shortWaitTimeout, shortPollingInterval).Should(BeTrue())
		})
	})

	t.Context("Metrics.", Label("f:observability.monitoring.prom"), func() {
		// Verify Prometheus scraped metrics
		// GIVEN a deployed WebLogic application
		// WHEN the application configuration uses a default metrics trait
		// THEN confirm that metrics are being collected
		t.It("Retrieve Prometheus scraped metrics", func() {
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

	t.Context("Logging.", Label("f:observability.logging.es"), func() {
		indexName, err := pkg.GetOpenSearchAppIndex(namespace)
		Expect(err).To(BeNil())
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
				t.It("Verify recent adminserver log record exists", func() {
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
				t.It("Verify recent pattern-matched AdminServer log record exists", func() {
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
				t.It("Verify recent pattern-matched WebLogic Server log record exists", func() {
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
				t.It("Verify recent pattern-matched Security log record exists", func() {
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
				t.It("Verify recent pattern-matched multi-lines log record exists", func() {
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
				pkg.MinVersionSpec("Verify recent fluentd-stdout-sidecar server log record exists", "1.1.0",
					func() {
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
				pkg.MinVersionSpec("Verify recent fluentd-stdout-sidecar domain log record exists", "1.1.0",
					func() {
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
				pkg.MinVersionSpec("Verify recent fluentd-stdout-sidecar nodemanager log record exists", "1.1.0",
					func() {
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
				t.It("Verify recent pattern-matched AdminServer log record exists", func() {
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
				t.It("Verify recent pattern-matched AdminServer log record exists", func() {
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
	})
})

// function to pair a put and get for a given task item
func putGetTodoTask(host string, task string) bool {
	url := fmt.Sprintf("https://%s/todo/rest/item/%s", host, task)
	resp, err := pkg.PutWithHostHeader(url, "application/json", host, nil)
	if err != nil {
		t.Logs.Errorf("PUT failed with error: %v", err)
		return false
	}
	if resp.StatusCode != http.StatusNoContent {
		t.Logs.Errorf("Put status code is: %d", resp.StatusCode)
		return false
	}
	url = fmt.Sprintf("https://%s/todo/rest/items", host)
	resp, err = pkg.GetWebPage(url, host)
	if err != nil {
		t.Logs.Errorf("GET failed with error: %v", err)
		return false
	}
	if resp.StatusCode == http.StatusOK && resp.Body != nil {
		return true
	}
	t.Logs.Errorf("Get status code is: %d", resp.StatusCode)
	return false
}
