// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package todo_list

import (
	"fmt"
	"os"
	"time"

	"github.com/onsi/ginkgo"
	"github.com/onsi/gomega"
	"github.com/onsi/gomega/types"
	"github.com/verrazzano/verrazzano/tests/e2e/util"
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

var _ = ginkgo.AfterSuite(func () {
	undeployToDoListExample()
})

func deployToDoListExample() {
	util.Log(util.Info, "Deploy ToDoList example")
	wlsUser := "weblogic"
	wlsPass := getRequiredEnvVarOrFail("WEBLOGIC_PSW")
	dbPass := getRequiredEnvVarOrFail("DATABASE_PSW")
	regServ := getRequiredEnvVarOrFail("OCR_REPO")
	regUser := getRequiredEnvVarOrFail("OCR_CREDS_USR")
	regPass := getRequiredEnvVarOrFail("OCR_CREDS_PSW")

	util.Log(util.Info, "Create namespace")
	if _, err := util.CreateNamespace("todo-list", map[string]string{"verrazzano-managed": "true"}); err != nil {
		ginkgo.Fail(fmt.Sprintf("Failed to create namespace: %v", err))
	}
	util.Log(util.Info, "Create Docker repository secret")
	if _, err := util.CreateDockerSecret("todo-list", "tododomain-repo-credentials", regServ, regUser, regPass); err != nil {
		ginkgo.Fail(fmt.Sprintf("Failed to create Docker registry secret: %v", err))
	}
	util.Log(util.Info, "Create WebLogic credentials secret")
	if _, err := util.CreateCredentialsSecret("todo-list", "tododomain-weblogic-credentials", wlsUser, wlsPass, nil); err != nil {
		ginkgo.Fail(fmt.Sprintf("Failed to create WebLogic credentials secret: %v", err))
	}
	util.Log(util.Info, "Create database credentials secret")
	if _, err := util.CreateCredentialsSecret("todo-list", "tododomain-jdbc-tododb", wlsUser, dbPass, map[string]string{"weblogic.domainUID": "tododomain"}); err != nil {
		ginkgo.Fail(fmt.Sprintf("Failed to create JDBC credentials secret: %v", err))
	}
	util.Log(util.Info, "Create encryption credentials secret")
	if _, err := util.CreatePasswordSecret("todo-list", "tododomain-runtime-encrypt-secret", wlsPass, map[string]string{"weblogic.domainUID": "tododomain"}); err != nil {
		ginkgo.Fail(fmt.Sprintf("Failed to create encryption secret: %v", err))
	}
	util.Log(util.Info, "Create logging scope resource")
	if err := util.CreateOrUpdateResourceFromFile("examples/todo-list/todo-list-logging-scope.yaml"); err != nil {
		ginkgo.Fail(fmt.Sprintf("Failed to create ToDo List logging scope resource: %v", err))
	}
	util.Log(util.Info, "Create component resources")
	if err := util.CreateOrUpdateResourceFromFile("examples/todo-list/todo-list-components.yaml"); err != nil {
		ginkgo.Fail(fmt.Sprintf("Failed to create ToDo List component resources: %v", err))
	}
	util.Log(util.Info, "Create application resources")
	if err := util.CreateOrUpdateResourceFromFile("examples/todo-list/todo-list-application.yaml"); err != nil {
		ginkgo.Fail(fmt.Sprintf("Failed to create ToDo List application resource: %v", err))
	}
}

func undeployToDoListExample() {
	util.Log(util.Info, "Undeploy ToDoList example")
	util.Log(util.Info, "Delete application")
	if err := util.DeleteResourceFromFile("examples/todo-list/todo-list-application.yaml"); err != nil {
		util.Log(util.Error, fmt.Sprintf("Failed to delete application: %v", err))
	}
	util.Log(util.Info, "Delete components")
	if err := util.DeleteResourceFromFile("examples/todo-list/todo-list-components.yaml"); err != nil {
		util.Log(util.Error, fmt.Sprintf("Failed to delete components: %v", err))
	}
	util.Log(util.Info, "Delete logging scope")
	if err := util.DeleteResourceFromFile("examples/todo-list/todo-list-logging-scope.yaml"); err != nil {
		util.Log(util.Error, fmt.Sprintf("Failed to delete logging scope: %v", err))
	}
	util.Log(util.Info, "Delete namespace")
	if err := util.DeleteNamespace("todo-list"); err != nil {
		util.Log(util.Error, fmt.Sprintf("Failed to delete namespace: %v", err))
	}
	gomega.Eventually(func () bool {
		ns, err := util.GetNamespace("todo-list")
		return ns == nil && err != nil && errors.IsNotFound(err)
	}, 3*time.Minute, 15*time.Second).Should(gomega.BeFalse())
}

type WebResponse struct {
	status int
	content string
}

// HaveStatus asserts that a WebResponse has a given status.
func HaveStatus(expected int) types.GomegaMatcher {
	return gomega.WithTransform(func (response WebResponse) int { return response.status }, gomega.Equal(expected))
}

// ContainContent asserts that a WebResponse contains a given substring.
func ContainContent(expected string) types.GomegaMatcher {
	return gomega.WithTransform(func(response WebResponse) string { return response.content }, gomega.ContainSubstring(expected))
}

var _ = ginkgo.Describe("Verify ToDo List example application.", func() {

	ginkgo.Context("Deployment.", func () {
		// GIVEN the ToDoList app is deployed
		// WHEN the running pods are checked
		// THEN the adminserver and mysql pods should be found running
		ginkgo.It("Verify 'tododomain-adminserver' and 'mysql' pods are running", func() {
			gomega.Eventually(func () bool {
				return util.PodsRunning("todo-list", []string{"mysql", "tododomain-adminserver"})
			}, longWaitTimeout, longPollingInterval).Should(gomega.BeTrue())
		})
	})

	ginkgo.Context("Ingress.", func () {
		// Verify the application REST endpoint is working.
		// GIVEN the ToDoList app is deployed
		// WHEN the UI is accessed
		// THEN the expected returned page should contain an expected value.
		ginkgo.It("Verify '/todo' UI endpoint is working.", func() {
			gomega.Eventually(func() WebResponse {
				ingress := util.Ingress()
				util.Log(util.Info, fmt.Sprintf("Ingress: %s", ingress))
				url := fmt.Sprintf("http://%s/todo/", ingress)
				host := "todo.example.com"
				status, content := util.GetWebPageWithCABundle(url, host)
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
			gomega.Eventually(func() WebResponse {
				ingress := util.Ingress()
				util.Log(util.Info, fmt.Sprintf("Ingress: %s", ingress))
				url := fmt.Sprintf("http://%s/todo/rest/items", ingress)
				host := "todo.example.com"
				status, content := util.GetWebPageWithCABundle(url, host)
				return WebResponse{
					status:  status,
					content: content,
				}
			}, shortWaitTimeout, shortPollingInterval).Should(gomega.And(HaveStatus(200), ContainContent("[")))
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
	//		util.Concurrently(
	//			func() {
	//				gomega.Eventually(appMetricsExists, longWaitTimeout, longPollingInterval).Should(gomega.BeTrue())
	//			},
	//		)
	//	})
	//})

	ginkgo.Context("Logging.", func() {
		// GIVEN a WebLogic application with logging enabled via a logging scope
		// WHEN the ELASTICSEARCH index is retrieved
		// THEN verify that it is found
		ginkgo.It("Verify tododomain ELASTICSEARCH index exists", func() {
			gomega.Eventually(func() bool {
				return logIndexFound("tododomain")
			}, longWaitTimeout, longPollingInterval).Should(gomega.BeTrue(), "Expected to find log index tododomain")
		})

		// GIVEN a WebLogic application with logging enabled via a logging scope
		// WHEN the log records are retrieved from the ELASTICSEARCH index tododomain
		// THEN verify that at least one recent log record is found
		ginkgo.It("Verify recent tododomain ELASTICSEARCH log record exists", func() {
			gomega.Eventually(func() bool {
				return logRecordFound("tododomain", time.Now().Add(-24*time.Hour), map[string]string{
					"domainUID": "tododomain",
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
		if util.Jq(metric, "metric", key) == value {
			return true
		}
	}
	return false
}

// metricExist confirms that a metric with the key and value can be found.
func metricExist(metricsName, key, value string) bool {
	metrics := util.JTq(util.QueryMetric(metricsName), "data", "result").([]interface{})
	if metrics != nil {
		return findMetric(metrics, key, value)
	} else {
		return false
	}
}

// logIndexFound confirms a named index can be found.
func logIndexFound(indexName string) bool {
	for _, name := range util.ListSystemElasticSearchIndices() {
		if name == indexName {
			return true
		}
	}
	util.Log(util.Error, fmt.Sprintf("Expected to find log index %s", indexName))
	return false
}

// logRecordFound confirms a recent log record for the index with matching fields can be found.
func logRecordFound(indexName string, after time.Time, fields map[string]string) bool {
	searchResult := util.QuerySystemElasticSearch(indexName, fields)
	hits := util.Jq(searchResult, "hits", "hits")
	if hits == nil {
		util.Log(util.Info, "Expected to find hits in log record query results")
		return false
	}
	util.Log(util.Info, fmt.Sprintf("Found %d records", len(hits.([]interface{}))))
	if len(hits.([]interface{})) == 0 {
		util.Log(util.Info, "Expected log record query results to contain at least one hit")
		return false
	}
	for _, hit := range hits.([]interface{}) {
		timestamp := util.Jq(hit, "_source", "@timestamp")
		t, err := time.Parse(ISO8601Layout, timestamp.(string))
		if err != nil {
			util.Log(util.Error, fmt.Sprintf("Failed to parse timestamp: %s", timestamp))
			return false
		}
		if t.After(after) {
			util.Log(util.Info, fmt.Sprintf("Found recent record: %s", timestamp))
			return true
		}
		util.Log(util.Info, fmt.Sprintf("Found old record: %s", timestamp))
	}
	util.Log(util.Error, fmt.Sprintf("Failed to find recent log record for index %s", indexName))
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
