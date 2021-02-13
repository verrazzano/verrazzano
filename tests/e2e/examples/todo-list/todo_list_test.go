// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package todo_list

import (
	"fmt"
	"time"

	"github.com/onsi/ginkgo"
	"github.com/onsi/gomega"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg"
	"k8s.io/apimachinery/pkg/api/errors"
)

const (
	ISO8601Layout        = "2006-01-02T15:04:05.999999999-07:00"
	shortWaitTimeout     = 5 * time.Minute
	shortPollingInterval = 10 * time.Second
	longWaitTimeout      = 10 * time.Minute
	longPollingInterval  = 20 * time.Second
)

var _ = ginkgo.BeforeSuite(func() {
	deployToDoListExample()
})

var _ = ginkgo.AfterSuite(func() {
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
	if _, err := pkg.CreateNamespace("todo-list", map[string]string{"verrazzano-managed": "true"}); err != nil {
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
	pkg.Log(pkg.Info, "Create encryption credentials secret")
	if _, err := pkg.CreatePasswordSecret("todo-list", "tododomain-runtime-encrypt-secret", wlsPass, map[string]string{"weblogic.domainUID": "tododomain"}); err != nil {
		ginkgo.Fail(fmt.Sprintf("Failed to create encryption secret: %v", err))
	}
	pkg.Log(pkg.Info, "Create logging scope resource")
	if err := pkg.CreateOrUpdateResourceFromFile("examples/todo-list/todo-list-logging-scope.yaml"); err != nil {
		ginkgo.Fail(fmt.Sprintf("Failed to create ToDo List logging scope resource: %v", err))
	}
	pkg.Log(pkg.Info, "Create component resources")
	if err := pkg.CreateOrUpdateResourceFromFile("examples/todo-list/todo-list-components.yaml"); err != nil {
		ginkgo.Fail(fmt.Sprintf("Failed to create ToDo List component resources: %v", err))
	}
	pkg.Log(pkg.Info, "Create application resources")
	if err := pkg.CreateOrUpdateResourceFromFile("examples/todo-list/todo-list-application.yaml"); err != nil {
		ginkgo.Fail(fmt.Sprintf("Failed to create ToDo List application resource: %v", err))
	}
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
	pkg.Log(pkg.Info, "Delete logging scope")
	if err := pkg.DeleteResourceFromFile("examples/todo-list/todo-list-logging-scope.yaml"); err != nil {
		pkg.Log(pkg.Error, fmt.Sprintf("Failed to delete logging scope: %v", err))
	}
	pkg.Log(pkg.Info, "Delete namespace")
	if err := pkg.DeleteNamespace("todo-list"); err != nil {
		pkg.Log(pkg.Error, fmt.Sprintf("Failed to delete namespace: %v", err))
	}
	gomega.Eventually(func() bool {
		ns, err := pkg.GetNamespace("todo-list")
		return ns == nil && err != nil && errors.IsNotFound(err)
	}, 3*time.Minute, 15*time.Second).Should(gomega.BeFalse())
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
	})

	ginkgo.Context("Ingress.", func() {
		// Verify the application REST endpoint is working.
		// GIVEN the ToDoList app is deployed
		// WHEN the UI is accessed
		// THEN the expected returned page should contain an expected value.
		ginkgo.It("Verify '/todo' UI endpoint is working.", func() {
			gomega.Eventually(func() pkg.WebResponse {
				ingress := pkg.Ingress()
				pkg.Log(pkg.Info, fmt.Sprintf("Ingress: %s", ingress))
				url := fmt.Sprintf("http://%s/todo/", ingress)
				host := "todo.example.com"
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
			ingress := pkg.Ingress()
			pkg.Log(pkg.Info, fmt.Sprintf("Ingress: %s", ingress))
			host := "todo.example.com"
			task := fmt.Sprintf("test-task-%s", time.Now().Format("20060102150405.0000"))
			gomega.Eventually(func() pkg.WebResponse {
				url := fmt.Sprintf("http://%s/todo/rest/items", ingress)
				status, content := pkg.GetWebPageWithCABundle(url, host)
				return pkg.WebResponse{
					Status:  status,
					Content: content,
				}
			}, shortWaitTimeout, shortPollingInterval).Should(gomega.And(pkg.HaveStatus(200), pkg.ContainContent("[")))
			gomega.Eventually(func() pkg.WebResponse {
				url := fmt.Sprintf("http://%s/todo/rest/item/%s", ingress, task)
				status, content := pkg.PutWithHostHeader(url, "application/json", host, nil)
				return pkg.WebResponse{
					Status:  status,
					Content: content,
				}
			}, shortWaitTimeout, shortPollingInterval).Should(pkg.HaveStatus(204))
			gomega.Eventually(func() pkg.WebResponse {
				url := fmt.Sprintf("http://%s/todo/rest/items", ingress)
				status, content := pkg.GetWebPageWithCABundle(url, host)
				return pkg.WebResponse{
					Status:  status,
					Content: content,
				}
			}, shortWaitTimeout, shortPollingInterval).Should(gomega.And(pkg.HaveStatus(200), pkg.ContainContent(task)))
		})
	})

	// The ToDoList example application currently does not include a metrics exporter.
	// This test has been disabled until that issue is resolved.
	//ginkgo.Context("Metrics.", func() {
	//	// Verify Prometheus scraped metrics
	//	// GIVEN a deployed WebLogic application
	//	// WHEN the application configuration uses a default metrics trait
	//	// THEN confirm that metrics are being collected
	//	ginkgo.It("Retrieve Prometheus scraped metrics", func() {
	//		pkg.Concurrently(
	//			func() {
	//				gomega.Eventually(appMetricsExists, longWaitTimeout, longPollingInterval).Should(gomega.BeTrue())
	//			},
	//		)
	//	})
	//})

	ginkgo.Context("Logging.", func() {
		indexName := "todo-list-todo-appconf-todo-domain"

		// GIVEN a WebLogic application with logging enabled via a logging scope
		// WHEN the Elasticsearch index is retrieved
		// THEN verify that it is found
		ginkgo.It("Verify Elasticsearch index exists", func() {
			gomega.Eventually(func() bool {
				return pkg.LogIndexFound(indexName)
			}, longWaitTimeout, longPollingInterval).Should(gomega.BeTrue(), "Expected to find log index for todo-list")
		})

		// GIVEN a WebLogic application with logging enabled via a logging scope
		// WHEN the log records are retrieved from the Elasticsearch index
		// THEN verify that at least one recent log record is found
		ginkgo.It("Verify recent Elasticsearch log record exists", func() {
			gomega.Eventually(func() bool {
				return pkg.LogRecordFound(indexName, time.Now().Add(-24*time.Hour), map[string]string{
					"domainUID":  "tododomain",
					"serverName": "tododomain-adminserver"})
			}, longWaitTimeout, longPollingInterval).Should(gomega.BeTrue(), "Expected to find a recent log record")
		})
	})
})

// appMetricsExists confirms that a specific application metrics can be found.
func appMetricsExists() bool {
	return pkg.MetricsExist("wls_scrape_mbeans_count_total", "app_oam_dev_name", "todo")
}
