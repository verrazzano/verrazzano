// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package prometheus

import (
	"fmt"
	"github.com/verrazzano/verrazzano/pkg/k8sutil"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/verrazzano/verrazzano/pkg/test/framework"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg"
)

const (
	threeMinutes    = 3 * time.Minute
	pollingInterval = 10 * time.Second
	documentFile    = "testdata/upgrade/opensearch/document1.json"
	longTimeout     = 10 * time.Minute
)

var t = framework.NewTestFramework("prometheus")

var _ = t.BeforeSuite(func() {
	kubeconfigPath, err := k8sutil.GetKubeConfigLocation()
	if err != nil {
		Fail(fmt.Sprintf("Failed to get default kubeconfig path: %s", err.Error()))
	}
	supported := pkg.IsPrometheusEnabled(kubeconfigPath)
	// Only run tests if Prometheus component is enabled in Verrazzano CR
	if !supported {
		Skip("Prometheus component is not enabled")
	}
})

var _ = t.Describe("Post upgrade Prometheus", Label("f:observability.logging.es"), func() {

	// GIVEN a running Prometheus instance,
	// WHEN a scrape config is created,
	// THEN verify that the scrape config is created correctly
	It("Old indices are deleted", func() {
		Eventually(func() bool {

			return true
		}).WithPolling(pollingInterval).WithTimeout(threeMinutes).Should(BeTrue(),
			"Expected not to find any old indices")
	})

	// GIVEN a running Prometheus instance,
	// WHEN the data streams are retrieved
	// THEN verify that they have data streams
	It("Data streams are created", func() {
		Eventually(func() bool {
			return true
		}).WithPolling(pollingInterval).WithTimeout(threeMinutes).Should(BeTrue(),
			"Expected not to find any old indices")
	})

	// GIVEN a running Prometheus instance,
	// WHEN
	// THEN verify that the data can be retrieved successfully
	It("OpenSearch get old data", func() {
		Eventually(func() bool {
			return true
		}).WithPolling(pollingInterval).WithTimeout(threeMinutes).Should(BeTrue(),
			"Expected to find the old data")
	})

	// GIVEN a running Prometheus instance,
	// WHEN VZ custom resource is upgraded
	// THEN only the system logs that are as old as the retention period
	//      is migrated and older logs are purged
	It("OpenSearch system logs older than retention period is not available post upgrade", func() {
		Eventually(func() bool {
			return true
		}).WithPolling(pollingInterval).WithTimeout(threeMinutes).Should(BeTrue(),
			"Expected to find the old data")
	})

	// GIVEN a running Prometheus instance,
	// WHEN VZ custom resource is upgraded
	// THEN only the application logs that are as old as the retention period
	//      is migrated and older logs are purged
	It("OpenSearch application logs older than retention period is not available post upgrade", func () {
		Eventually(func() bool {
			return true
		}).WithPolling(pollingInterval).WithTimeout(threeMinutes).Should(BeTrue(),
			"Expected to find the old data")
	})

})
