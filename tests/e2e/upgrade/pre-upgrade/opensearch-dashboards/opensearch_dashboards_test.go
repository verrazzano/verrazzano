// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package dashboards

import (
	"bufio"
	"fmt"
	"github.com/verrazzano/verrazzano/pkg/k8sutil"
	"os"
	"strings"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg/test/framework"
)

const (
	waitTimeout                 = 3 * time.Minute
	pollingInterval             = 10 * time.Second
	oldPatternsTestDataFile     = "testdata/upgrade/opensearch-dashboards/old-index-patterns.txt"
	updatedPatternsTestDataFile = "testdata/upgrade/opensearch-dashboards/updated-index-patterns.txt"
)

var t = framework.NewTestFramework("opensearch-dashboards")

var _ = t.Describe("Pre Upgrade OpenSearch Dashboards Setup", Label("f:observability.logging.kibana"), func() {
	// GIVEN the OpenSearchDashboards pod
	// WHEN the index patterns are created
	// THEN verify that they are created successfully
	It("Create Index Patterns", func() {
		Eventually(func() bool {
			kubeConfigPath, _ := k8sutil.GetKubeConfigLocation()
			if pkg.IsOpenSearchDashboardsEnabled(kubeConfigPath) {
				isVersionAbove1_3_0, err := pkg.IsVerrazzanoMinVersion("1.3.0", kubeConfigPath)
				if err != nil {
					pkg.Log(pkg.Error, fmt.Sprintf("failed to find the verrazzano version: %v", err))
					return false
				}
				patternFile := oldPatternsTestDataFile
				if isVersionAbove1_3_0 {
					patternFile = updatedPatternsTestDataFile
				}
				file, err := pkg.FindTestDataFile(patternFile)
				if err != nil {
					pkg.Log(pkg.Error, fmt.Sprintf("failed to find test data file: %v", err))
					return false
				}
				found, err := os.Open(file)
				if err != nil {
					pkg.Log(pkg.Error, fmt.Sprintf("failed to open test data file: %v", err))
					return false
				}
				reader := bufio.NewScanner(found)
				reader.Split(bufio.ScanLines)
				defer found.Close()

				for reader.Scan() {
					line := strings.TrimSpace(reader.Text())
					// skip empty lines
					if len(line) == 0 {
						continue
					}
					// ignore lines starting with "#"
					if strings.HasPrefix(line, "#") {
						continue
					}
					// Create index pattern
					result := pkg.CreateIndexPattern(line)
					if result == nil {
						pkg.Log(pkg.Error, fmt.Sprintf("failed to create index pattern %s: %v", line, err))
						return false
					}
				}
			}
			return true
		}).WithPolling(pollingInterval).WithTimeout(waitTimeout).Should(BeTrue(), "Expected not to fail creation of index patterns")
	})
})
