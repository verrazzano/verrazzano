// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package opensearch

import (
	"fmt"
	"github.com/verrazzano/verrazzano/pkg/k8sutil"
	"os"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/verrazzano/verrazzano/pkg/test/framework"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg"
)

const (
	waitTimeout     = 3 * time.Minute
	pollingInterval = 10 * time.Second
	documentFile    = "testdata/upgrade/opensearch/document1.json"
)

var t = framework.NewTestFramework("opensearch")

var _ = t.Describe("Pre Upgrade OpenSearch", Label("f:observability.logging.es"), func() {
	// GIVEN the OpenSearch pod
	// THEN verify that the data can be written to indices successfully
	It("OpenSearch Write data", func() {
		Eventually(func() bool {
			kubeConfigPath, _ := k8sutil.GetKubeConfigLocation()
			isOSEnabled, err := pkg.IsOpenSearchEnabled(kubeConfigPath)
			if err != nil {
				pkg.Log(pkg.Error, err.Error())
				return false
			}
			if isOSEnabled {
				indexName, err := pkg.GetOpenSearchSystemIndex(pkg.VerrazzanoNamespace)
				if err != nil {
					pkg.Log(pkg.Error, fmt.Sprintf("error getting the system index: %v", err))
					return false
				}
				file, err := pkg.FindTestDataFile(documentFile)
				if err != nil {
					pkg.Log(pkg.Error, fmt.Sprintf("failed to find test data file: %v", err))
					return false
				}
				data, err := os.ReadFile(file)
				if err != nil {
					pkg.Log(pkg.Error, fmt.Sprintf("failed to read test data file: %v", err))
					return false
				}
				resp, err := pkg.PostElasticsearch(fmt.Sprintf("%s/_doc", indexName), string(data))
				if err != nil {
					pkg.Log(pkg.Error, fmt.Sprintf("Failed to write to OpenSearch: %v", err))
					return false
				}
				if resp.StatusCode != 201 {
					pkg.Log(pkg.Error, fmt.Sprintf("Failed to write to OpenSearch: status=%d: body=%s", resp.StatusCode, string(resp.Body)))
					return false
				}
			}
			return true
		}).WithPolling(pollingInterval).WithTimeout(waitTimeout).Should(BeTrue(), "Expected not to fail while writing data to OpenSearch")
	})
})
