// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package system

import (
	"os"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/verrazzano/verrazzano/pkg/test/framework"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg"
)

const (
	shortPollingInterval        = 10 * time.Second
	shortWaitTimeout            = 5 * time.Minute
	longPollingInterval         = 30 * time.Second
	longWaitTimeout             = 10 * time.Minute
	jaegerOperatorSampleMetric  = "jaeger_operator_instances_managed"
	jaegerCollectorSampleMetric = "jaeger_collector_queue_capacity"
)

var start time.Time

var t = framework.NewTestFramework("jaeger_mc_system_test")

var (
	adminKubeConfigPath = os.Getenv("ADMIN_KUBECONFIG")
	clusterName         = os.Getenv("CLUSTER_NAME")
	failed              = false
)

var _ = t.BeforeSuite(func() {
	// Allow 3hr allowance for the traces.
	start = time.Now().Add(-3 * time.Hour)
	if adminKubeConfigPath == "" {
		AbortSuite("Required env variable ADMIN_KUBECONFIG not set.")
	}
})

var _ = t.AfterEach(func() {
	failed = failed || CurrentSpecReport().Failed()
})

var _ = t.AfterSuite(func() {
	if failed {
		err := pkg.ExecuteClusterDumpWithEnvVarConfig()
		if err != nil {
			pkg.Log(pkg.Error, err.Error())
		}
	}
})

var _ = t.Describe("Multi Cluster Jaeger Validation", Label("f:platform-lcm.install"), func() {

	// GIVEN a multicluster setup with an admin and a manged cluster,
	// WHEN Jaeger operator is enabled in the admin cluster and the managed cluster is registered to it,
	// THEN system traces can be queried from the Jaeger UI in the admin cluster
	t.It("traces from verrazzano system components of managed cluster should be available when queried from Jaeger", func() {
		validatorFn := pkg.ValidateSystemTracesFuncInCluster(adminKubeConfigPath, start, getClusterName())
		Eventually(validatorFn).WithPolling(longPollingInterval).WithTimeout(longWaitTimeout).Should(BeTrue())
	})

	// GIVEN a multicluster setup with an admin and a manged cluster,
	// WHEN Jaeger operator is enabled in the admin cluster and the managed cluster is registered to it,
	// THEN we are able to query the metrics of Jaeger operator running in managed cluster
	//      from the prometheus service running admin cluster.
	t.It("metrics of jaeger operator running in managed cluster are available in prometheus of admin cluster", func() {
		Eventually(func() bool {
			return pkg.IsJaegerMetricFound(adminKubeConfigPath, jaegerOperatorSampleMetric, getClusterName(), nil)
		}).WithPolling(shortPollingInterval).WithTimeout(shortWaitTimeout).Should(BeTrue())
	})

	// GIVEN a multicluster setup with an admin and a manged cluster,
	// WHEN Jaeger operator is enabled in the admin cluster and the managed cluster is registered to it,
	// THEN we are able to query the metrics of Jaeger collector running in managed cluster
	//      from the prometheus service running admin cluster.
	t.It("metrics of jaeger collector running in managed cluster are available in prometheus of admin cluster", func() {
		Eventually(func() bool {
			return pkg.IsJaegerMetricFound(adminKubeConfigPath, jaegerCollectorSampleMetric, getClusterName(), nil)
		}).WithPolling(shortPollingInterval).WithTimeout(shortWaitTimeout).Should(BeTrue())
	})
})

func getClusterName() string {
	if clusterName == "admin" {
		return "local"
	}
	return clusterName
}
