// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package opensearch

import (
	"fmt"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/verrazzano/verrazzano/pkg/k8sutil"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg/test/framework"
)

const (
	shortPollingInterval = 10 * time.Second
	shortWaitTimeout     = 1 * time.Minute
	longWaitTimeout      = 15 * time.Minute
	indexDocumentURL     = "%s/_doc"
)

var t = framework.NewTestFramework("field-mappings")

var failed = false
var _ = t.AfterEach(func() {
	failed = failed || CurrentSpecReport().Failed()
})

var _ = t.AfterSuite(func() {
	if failed {
		pkg.ExecuteBugReport()
	}
	pkg.DeleteApplicationDataStream("verrazzano-application-test")
})

var _ = t.Describe("OpenSearch field mappings", Label("f:observability.logging.es"), func() {
	// It Wrapper to only run spec if component is supported on the current Verrazzano installation
	MinimumVerrazzanoIt := func(description string, f func()) {
		kubeconfigPath, err := k8sutil.GetKubeConfigLocation()
		if err != nil {
			t.It(description, func() {
				Fail(fmt.Sprintf("Failed to get default kubeconfig path: %s", err.Error()))
			})
		}
		supported, err := pkg.IsVerrazzanoMinVersion("1.3.0", kubeconfigPath)
		if err != nil {
			t.It(description, func() {
				Fail(err.Error())
			})
		}
		// Only run tests if Verrazzano is at least version 1.3.0
		if supported {
			t.It(description, f)
		} else {
			pkg.Log(pkg.Info, fmt.Sprintf("Skipping check '%v', Verrazzano is not at version 1.3.0", description))
		}
	}
	MinimumVerrazzanoIt("Documents with non-object fields get stored as strings", func() {
		// GIVEN OpenSearch verrazzano application index
		// WHEN the documents with same field name but different data types are written
		// THEN verify that both the docs are written successfully
		var indexName string
		var err error
		Eventually(func() error {
			indexName, err = pkg.GetOpenSearchAppIndex("test")
			return err
		}, shortWaitTimeout, shortPollingInterval).Should(BeNil(), "Expected to get OpenSearch App Index")

		Eventually(func() bool {
			doc1 := `{"key":2,"@timestamp":"2022-03-15T19:55:54Z"}`
			resp, err := pkg.PostElasticsearch(fmt.Sprintf(indexDocumentURL, indexName), doc1)
			if err != nil {
				pkg.Log(pkg.Error, fmt.Sprintf("Failed to write to OpenSearch: %v", err))
				return false
			}
			if resp.StatusCode != 201 {
				pkg.Log(pkg.Error, fmt.Sprintf("Failed to write to OpenSearch: status=%d: body=%s", resp.StatusCode,
					string(resp.Body)))
				return false
			}
			kubeconfigPath, err := k8sutil.GetKubeConfigLocation()
			if err != nil {
				pkg.Log(pkg.Error, fmt.Sprintf("Failed to get default kubeconfig path: %s", err.Error()))
				return false
			}
			supported, err := pkg.IsVerrazzanoMinVersion("1.3.0", kubeconfigPath)
			if err != nil {
				pkg.Log(pkg.Error, fmt.Sprintf("Error getting Verrazzano version: %v", err))
				return false
			}
			doc2 := `{"key":"text","@timestamp":"2022-03-15T19:55:54Z"}`
			resp, err = pkg.PostElasticsearch(fmt.Sprintf(indexDocumentURL, indexName), doc2)
			if err != nil {
				pkg.Log(pkg.Error, fmt.Sprintf("Failed to write a document to OpenSearch: %v", err))
				return false
			}
			if supported {
				if resp.StatusCode != 201 {
					pkg.Log(pkg.Error, fmt.Sprintf("Failed to write to OpenSearch with a different data type field: "+
						"'status=%d: body=%s", resp.StatusCode, string(resp.Body)))
					return false
				}
			} else {
				if resp.StatusCode != 400 {
					pkg.Log(pkg.Error, fmt.Sprintf("Excepted to fail to write to OpenSearch: status=%d: body=%s",
						resp.StatusCode, string(resp.Body)))
					return false
				}
			}
			return true
		}).WithPolling(shortPollingInterval).WithTimeout(longWaitTimeout).Should(BeTrue(), "Expected to write data successfully to OpenSearch with different data types")
	})

	MinimumVerrazzanoIt("Documents with object fields get stored as objects", func() {
		// GIVEN OpenSearch verrazzano application index
		// WHEN the documents with same field name but one with object and the other one with concrete value are written
		// THEN verify that the second document insertion fails
		var indexName string
		var err error
		Eventually(func() error {
			indexName, err = pkg.GetOpenSearchAppIndex("test")
			return err
		}, shortWaitTimeout, shortPollingInterval).Should(BeNil(), "Expected to get OpenSearch App Index")

		Eventually(func() bool {
			doc1 := `{"keyObject":{"name":"unit-test-cluster"},"@timestamp":"2022-03-15T19:55:54Z"}`
			resp, err := pkg.PostElasticsearch(fmt.Sprintf(indexDocumentURL, indexName), doc1)
			if err != nil {
				pkg.Log(pkg.Error, fmt.Sprintf("Failed to write to OpenSearch: %v", err))
				return false
			}
			if resp.StatusCode != 201 {
				pkg.Log(pkg.Error, fmt.Sprintf("Failed to write to OpenSearch: status=%d: body=%s", resp.StatusCode,
					string(resp.Body)))
				return false
			}
			doc2 := `{"keyObject":"text","@timestamp":"2022-03-15T19:55:54Z"}`
			resp, err = pkg.PostElasticsearch(fmt.Sprintf(indexDocumentURL, indexName), doc2)
			if err != nil {
				pkg.Log(pkg.Error, fmt.Sprintf("Failed to write another document to OpenSearch: %v", err))
				return false
			}
			if resp.StatusCode != 400 {
				pkg.Log(pkg.Error, fmt.Sprintf("Excepted to fail to write to OpenSearch: status=%d: body=%s",
					resp.StatusCode, string(resp.Body)))
				return false
			}
			return true
		}, longWaitTimeout, shortPollingInterval).Should(BeTrue(), "Expected to fail writing data with concrete value for object field in OpenSearch")
	})
})
