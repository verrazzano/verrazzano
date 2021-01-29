// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package todo_list

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/onsi/ginkgo"
	"github.com/onsi/gomega"
	"github.com/onsi/gomega/types"
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
	wlsPass := getRequiredEnvVarOrFail("WEBLOGIC_PSW")
	dbPass := getRequiredEnvVarOrFail("DATABASE_PSW")
	regServ := getRequiredEnvVarOrFail("OCR_REPO")
	regUser := getRequiredEnvVarOrFail("OCR_CREDS_USR")
	regPass := getRequiredEnvVarOrFail("OCR_CREDS_PSW")

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
	gomega.Eventually(func () bool {
		ns, err := pkg.GetNamespace("todo-list")
		return ns == nil && err != nil && errors.IsNotFound(err)
	}, 3*time.Minute, 15*time.Second).Should(gomega.BeFalse())
}

type WebResponse struct {
	status  int
	content string
}

// HaveStatus asserts that a WebResponse has a given status.
func HaveStatus(expected int) types.GomegaMatcher {
	return gomega.WithTransform(func(response WebResponse) int { return response.status }, gomega.Equal(expected))
}

// ContainContent asserts that a WebResponse contains a given substring.
func ContainContent(expected string) types.GomegaMatcher {
	return gomega.WithTransform(func(response WebResponse) string { return response.content }, gomega.ContainSubstring(expected))
}

var _ = ginkgo.Describe("Verify ToDo List example application.", func() {

	ginkgo.Context("Deployment.", func() {
		// GIVEN the ToDoList app is deployed
		// WHEN the running pods are checked
		// THEN the adminserver and mysql pods should be found running
		ginkgo.It("Verify 'tododomain-adminserver' and 'mysql' pods are running", func() {
			gomega.Eventually(func () bool {
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
			gomega.Eventually(func() WebResponse {
				ingress := pkg.Ingress()
				pkg.Log(pkg.Info, fmt.Sprintf("Ingress: %s", ingress))
				url := fmt.Sprintf("http://%s/todo/", ingress)
				host := "todo.example.com"
				status, content := pkg.GetWebPageWithCABundle(url, host)
				return WebResponse{
					status:  status,
					content: content,
				}
			}, shortWaitTimeout, shortPollingInterval).Should(gomega.And(HaveStatus(200), ContainContent("Derek")))
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
			gomega.Eventually(func() WebResponse {
				url := fmt.Sprintf("http://%s/todo/rest/items", ingress)
				status, content := pkg.GetWebPageWithCABundle(url, host)
				return WebResponse{
					status:  status,
					content: content,
				}
			}, shortWaitTimeout, shortPollingInterval).Should(gomega.And(HaveStatus(200), ContainContent("[")))
			gomega.Eventually(func() WebResponse {
				url := fmt.Sprintf("http://%s/todo/rest/item/%s", ingress, task)
				status, content := pkg.PutWithHostHeader(url, "application/json", host, nil)
				return WebResponse{
					status:  status,
					content: content,
				}
			}, shortWaitTimeout, shortPollingInterval).Should(HaveStatus(204))
			gomega.Eventually(func() WebResponse {
				url := fmt.Sprintf("http://%s/todo/rest/items", ingress)
				status, content := pkg.GetWebPageWithCABundle(url, host)
				return WebResponse{
					status:  status,
					content: content,
				}
			}, shortWaitTimeout, shortPollingInterval).Should(gomega.And(HaveStatus(200), ContainContent(task)))
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
		indexNamePrefix := "oam-todo-list--"

		// GIVEN a WebLogic application with logging enabled via a logging scope
		// WHEN the Elasticsearch index is retrieved
		// THEN verify that it is found
		ginkgo.It("Verify Elasticsearch index exists", func() {
			gomega.Eventually(func() bool {
				return getLogIndex(indexNamePrefix) != ""
			}, longWaitTimeout, longPollingInterval).Should(gomega.BeTrue(), "Expected to find log index for todo-list")
		})

		// GIVEN a WebLogic application with logging enabled via a logging scope
		// WHEN the log records are retrieved from the Elasticsearch index
		// THEN verify that at least one recent log record is found
		ginkgo.It("Verify recent Elasticsearch log record exists", func() {
			gomega.Eventually(func() bool {
				return logRecordFound(getLogIndex(indexNamePrefix), time.Now().Add(-24*time.Hour), map[string]string{
					"domainUID":  "tododomain",
					"serverName": "tododomain-adminserver"})
			}, longWaitTimeout, longPollingInterval).Should(gomega.BeTrue(), "Expected to find a recent log record")
		})
	})
})

// appMetricsExists confirms that a specific application metrics can be found.
func appMetricsExists() bool {
	return metricExist("wls_scrape_mbeans_count_total", "app_oam_dev_name", "todo")
}

// findMetric confirms a metric with the key and value can be found in a list of metrics.
func findMetric(metrics []interface{}, key, value string) bool {
	for _, metric := range metrics {
		if pkg.Jq(metric, "metric", key) == value {
			return true
		}
	}
	return false
}

// metricExist confirms that a metric with the key and value can be found.
func metricExist(metricsName, key, value string) bool {
	metrics := pkg.JTq(pkg.QueryMetric(metricsName), "data", "result").([]interface{})
	if metrics != nil {
		return findMetric(metrics, key, value)
	} else {
		return false
	}
}

// getLogIndex returns an index whose name has specified prefix.
func getLogIndex(indexNamePrefix string) string {
	for _, name := range pkg.ListSystemElasticSearchIndices() {
		if strings.HasPrefix(name, indexNamePrefix) {
			return name
		}
	}
	pkg.Log(pkg.Error, fmt.Sprintf("Expected to find log index prefixed by %s", indexNamePrefix))
	return ""
}

// logRecordFound confirms a recent log record for the index with matching fields can be found.
func logRecordFound(indexName string, after time.Time, fields map[string]string) bool {
	searchResult := pkg.QuerySystemElasticSearch(indexName, fields)
	hits := pkg.Jq(searchResult, "hits", "hits")
	if hits == nil {
		pkg.Log(pkg.Info, "Expected to find hits in log record query results")
		return false
	}
	pkg.Log(pkg.Info, fmt.Sprintf("Found %d records", len(hits.([]interface{}))))
	if len(hits.([]interface{})) == 0 {
		pkg.Log(pkg.Info, "Expected log record query results to contain at least one hit")
		return false
	}
	for _, hit := range hits.([]interface{}) {
		timestamp := pkg.Jq(hit, "_source", "@timestamp")
		t, err := time.Parse(ISO8601Layout, timestamp.(string))
		if err != nil {
			pkg.Log(pkg.Error, fmt.Sprintf("Failed to parse timestamp: %s", timestamp))
			return false
		}
		if t.After(after) {
			pkg.Log(pkg.Info, fmt.Sprintf("Found recent record: %s", timestamp))
			return true
		}
		pkg.Log(pkg.Info, fmt.Sprintf("Found old record: %s", timestamp))
	}
	pkg.Log(pkg.Error, fmt.Sprintf("Failed to find recent log record for index %s", indexName))
	return false
}

// getRequiredEnvVarOrFail returns the values of the provided environment variable name or fails.
func getRequiredEnvVarOrFail(name string) string {
	value, found := os.LookupEnv(name)
	if !found {
		ginkgo.Fail(fmt.Sprintf("Environment variable '%s' required.", name))
	}
	return value
}
