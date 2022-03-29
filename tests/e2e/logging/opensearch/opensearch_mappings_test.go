// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package opensearch

import (
	"fmt"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/verrazzano/verrazzano/pkg/k8sutil"
	"github.com/verrazzano/verrazzano/pkg/test/framework"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg"
	"time"
)

const (
	shortPollingInterval = 10 * time.Second
	shortWaitTimeout     = 1 * time.Minute
)

var t = framework.NewTestFramework("field-mappings")

var _ = t.BeforeSuite(func() {})

var failed = false
var _ = t.AfterEach(func() {
	failed = failed || CurrentSpecReport().Failed()
})

var _ = t.AfterSuite(func() {
	if failed {
		pkg.ExecuteClusterDumpWithEnvVarConfig()
	}
	pkg.DeleteApplicationDataStream("verrazzano-application-test")
})

var _ = t.Describe("OpenSearch field mappings", Label("f:observability.logging.es"), func() {
	t.It("Documents with non-object fields get stored as strings", func() {
		// GIVEN OpenSearch verrazzano application index
		// WHEN the documents with same field name but different data types are written
		// THEN verify that both the docs are written successfully
		indexName := pkg.GetOpenSearchAppIndex("test")
		Eventually(func() bool {
			doc1 := `{"key":2,"@timestamp":"2022-03-15T19:55:54Z"}`
			resp, err := pkg.PostElasticsearch(fmt.Sprintf("%s/_doc", indexName), doc1)
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
				Fail(fmt.Sprintf("Failed to get default kubeconfig path: %s", err.Error()))
				return false
			}
			supported, err := pkg.IsVerrazzanoMinVersion("1.3.0", kubeconfigPath)
			if err != nil {
				pkg.Log(pkg.Error, fmt.Sprintf("Error getting Verrazzano version: %v", err))
				return false
			}
			doc2 := `{"key":"text","@timestamp":"2022-03-15T19:55:54Z"}`
			resp, err = pkg.PostElasticsearch(fmt.Sprintf("%s/_doc", indexName), doc2)
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
		}, shortWaitTimeout, shortPollingInterval).Should(BeTrue(), "Expected to write data successfully to OpenSearch with different data types")
	})

	t.It("Documents with object fields get stored as objects", func() {
		// GIVEN OpenSearch verrazzano application index
		// WHEN the documents with same field name but one with object and the other one with concrete value are written
		// THEN verify that the second document insertion fails
		indexName := pkg.GetOpenSearchAppIndex("test")
		Eventually(func() bool {
			doc1 := `{"keyObject":{"name":"unit-test-cluster"},"@timestamp":"2022-03-15T19:55:54Z"}`
			resp, err := pkg.PostElasticsearch(fmt.Sprintf("%s/_doc", indexName), doc1)
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
			resp, err = pkg.PostElasticsearch(fmt.Sprintf("%s/_doc", indexName), doc2)
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
		}, shortWaitTimeout, shortPollingInterval).Should(BeTrue(), "Expected to fail writing data with concrete value for object field in OpenSearch")
	})
})
