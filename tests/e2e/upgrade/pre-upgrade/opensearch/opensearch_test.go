// Copyright (c) 2022, 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package opensearch

import (
	"fmt"
	"github.com/verrazzano/verrazzano/pkg/k8sutil"
	"os"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg/test/framework"
)

const (
	waitTimeout     = 3 * time.Minute
	pollingInterval = 10 * time.Second
	documentFile    = "testdata/upgrade/opensearch/document1.json"
	ismTemplateFile = "testdata/upgrade/opensearch/policy.json"
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
				resp, err := pkg.PostOpensearch(fmt.Sprintf("%s/_doc", indexName), string(data))
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

	kubeConfigPath, err := k8sutil.GetKubeConfigLocation()
	if err != nil {
		Expect(err).To(BeNil(), fmt.Sprintf(pkg.KubeConfigErrorFmt, err))
	}
	t.ItMinimumVersion("Verify OpenSearch plugins have been installed", "1.6.0", kubeConfigPath, func() {
		pkg.TestOpenSearchPlugins(pollingInterval, waitTimeout)
	})
})
var _ = t.Describe("Pre Upgrade OpenSearch", Label("f:observability.logging.es"), func() {
	// GIVEN the OpenSearch pod
	// THEN verify that the ism policy  can be written to successfully
	It("OpenSearch ISM policy", func() {
		Eventually(func() bool {
			kubeConfigPath, _ := k8sutil.GetKubeConfigLocation()
			isOSEnabled, err := pkg.IsOpenSearchEnabled(kubeConfigPath)
			if err != nil {
				pkg.Log(pkg.Error, err.Error())
				return false
			}
			if isOSEnabled {
				file, err := pkg.FindTestDataFile(ismTemplateFile)
				if err != nil {
					pkg.Log(pkg.Error, fmt.Sprintf("failed to find test data file: %v", err))
					return false
				}
				data, err := os.ReadFile(file)
				if err != nil {
					pkg.Log(pkg.Error, fmt.Sprintf("failed to read test data file: %v", err))
					return false
				}
				resp, err := pkg.PutISMPolicy(string(data), "vz-custom")
				if err != nil {
					pkg.Log(pkg.Error, fmt.Sprintf("Failed to create to ISM: %v", err))
					return false
				}
				if resp.StatusCode != 201 {
					pkg.Log(pkg.Error, fmt.Sprintf("Failed to write to OpenSearch: status=%d: body=%s", resp.StatusCode, string(resp.Body)))
					return false
				}
			}
			return true
		}).WithPolling(pollingInterval).WithTimeout(waitTimeout).Should(BeTrue(), "Expected not to fail when creating ISM policies in OpenSearch")
	})
})
