// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package system

import (
	"fmt"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg"
	"time"
)

const (
	adminClusterName            = "local"
	managedClusterName          = "managed1"
	shortPollingInterval        = 10 * time.Second
	shortWaitTimeout            = 5 * time.Minute
	longPollingInterval         = 30 * time.Second
	longWaitTimeout             = 15 * time.Minute
	jaegerOperatorSampleMetric  = "jaeger_operator_instances_managed"
	jaegerCollectorSampleMetric = "jaeger_collector_queue_capacity"
	jaegerQuerySampleMetric     = "jaeger_query_requests_total"
)

var start time.Time
var _ = t.BeforeSuite(func() {
	start = time.Now()
})

var _ = t.Describe("Multi Cluster Jaeger Validation", Label("f:platform-lcm.install"),
	func() {

		// GIVEN a multicluster setup with an admin and a manged cluster,
		// WHEN Jaeger operator is enabled in the admin cluster and the managed cluster is registered to it,
		// THEN system traces can be queried from the Jaeger UI in the admin cluster
		t.It("traces from verrazzano system components of managed cluster should be available when queried from Jaeger", func() {
			validatorFn := pkg.ValidateSystemTracesFuncInCluster(adminKubeconfig, start, managedClusterName)
			Eventually(validatorFn).WithPolling(longPollingInterval).WithTimeout(longWaitTimeout).Should(BeTrue())
		})

		// GIVEN a multicluster setup with an admin and a manged cluster,
		// WHEN Jaeger operator is enabled in the admin cluster and the managed cluster is registered to it,
		// THEN only one Jaeger collector deployment is created in the managed cluster.
		t.It("traces from verrazzano system components running in managed cluster should be available in the OS backend storage.", func() {
			Eventually(func() bool {
				tracesFound := true
				systemServiceNames := pkg.GetJaegerSystemServicesInManagedCluster()
				for i := 0; i < len(systemServiceNames); i++ {
					pkg.Log(pkg.Info, fmt.Sprintf("Finding traces for service %s after %s", systemServiceNames[i], start.String()))
					if i == 0 {
						tracesFound = pkg.JaegerSpanRecordFoundInOpenSearch(adminKubeconfig, start, systemServiceNames[i], clusterName)
					} else {
						tracesFound = tracesFound && pkg.JaegerSpanRecordFoundInOpenSearch(adminKubeconfig, start, systemServiceNames[i], clusterName)
					}
					// return early and retry later
					if !tracesFound {
						return false
					}
				}
				return tracesFound
			}).WithPolling(longPollingInterval).WithTimeout(longWaitTimeout).Should(BeTrue())
		})

		// GIVEN a multicluster setup with an admin and a manged cluster,
		// WHEN Jaeger operator is enabled in the admin cluster and the managed cluster is registered to it,
		// THEN we are able to query the metrics of Jaeger operator running in managed cluster
		//      from the prometheus service running admin cluster.
		t.It("metrics of jaeger operator running in managed cluster are available in prometheus of admin cluster", func() {
			Eventually(func() bool {
				return pkg.IsJaegerMetricFound(adminKubeconfig, jaegerOperatorSampleMetric, managedClusterName, nil)
			}).WithPolling(shortPollingInterval).WithTimeout(shortWaitTimeout).Should(BeTrue())
		})

		// GIVEN a multicluster setup with an admin and a manged cluster,
		// WHEN Jaeger operator is enabled in the admin cluster and the managed cluster is registered to it,
		// THEN we are able to query the metrics of Jaeger collector running in managed cluster
		//      from the prometheus service running admin cluster.
		t.It("metrics of jaeger collector running in managed cluster are available in prometheus of admin cluster", func() {
			Eventually(func() bool {
				return pkg.IsJaegerMetricFound(adminKubeconfig, jaegerCollectorSampleMetric, managedClusterName, nil)
			}).WithPolling(shortPollingInterval).WithTimeout(shortWaitTimeout).Should(BeTrue())
		})

		// GIVEN a multicluster setup with an admin and a manged cluster,
		// WHEN Jaeger operator is enabled in the admin cluster and the managed cluster is registered to it,
		// THEN traces from the system services running in admin cluster can be queried from the Jaeger UI in the admin cluster
		t.It("traces from verrazzano system components of admin cluster should be available when queried from Jaeger", func() {
			validatorFn := pkg.ValidateSystemTracesFuncInCluster(adminKubeconfig, start, adminClusterName)
			Eventually(validatorFn).WithPolling(longPollingInterval).WithTimeout(longWaitTimeout).Should(BeTrue())
		})

		// GIVEN a multicluster setup with an admin and a manged cluster,
		// WHEN Jaeger operator is enabled in the admin cluster and the managed cluster is registered to it,
		// THEN traces from the system services running in the admin cluster is available in the OS of the admin cluster.
		t.It("traces from verrazzano system components running in admin cluster should be available in the OS backend storage.", func() {
			Eventually(func() bool {
				tracesFound := true
				systemServiceNames := pkg.GetJaegerSystemServicesInAdminCluster()
				for i := 0; i < len(systemServiceNames); i++ {
					pkg.Log(pkg.Info, fmt.Sprintf("Finding traces for service %s after %s", systemServiceNames[i], start.String()))
					if i == 0 {
						tracesFound = pkg.JaegerSpanRecordFoundInOpenSearch(adminKubeconfig, start, systemServiceNames[i], adminClusterName)
					} else {
						tracesFound = tracesFound && pkg.JaegerSpanRecordFoundInOpenSearch(adminKubeconfig, start, systemServiceNames[i], adminClusterName)
					}
					// return early and retry later
					if !tracesFound {
						return false
					}
				}
				return tracesFound
			}).WithPolling(longPollingInterval).WithTimeout(longWaitTimeout).Should(BeTrue())
		})

		// GIVEN a multicluster setup with an admin and a manged cluster,
		// WHEN Jaeger operator is enabled in the admin cluster and the managed cluster is registered to it,
		// THEN we are able to query the metrics of Jaeger operator running in admin cluster
		//      from the prometheus service running admin cluster.
		t.It("metrics of jaeger operator running in admin cluster are available in prometheus of admin cluster", func() {
			Eventually(func() bool {
				return pkg.IsJaegerMetricFound(adminKubeconfig, jaegerOperatorSampleMetric, adminClusterName, nil)
			}).WithPolling(shortPollingInterval).WithTimeout(shortWaitTimeout).Should(BeTrue())
		})

		// GIVEN a multicluster setup with an admin and a manged cluster,
		// WHEN Jaeger operator is enabled in the admin cluster and the managed cluster is registered to it,
		// THEN we are able to query the metrics of Jaeger collector running in admin cluster
		//      from the prometheus service running admin cluster.
		t.It("metrics of jaeger collector running in admin cluster are available in prometheus of admin cluster", func() {
			Eventually(func() bool {
				return pkg.IsJaegerMetricFound(adminKubeconfig, jaegerCollectorSampleMetric, adminClusterName, nil)
			}).WithPolling(shortPollingInterval).WithTimeout(shortWaitTimeout).Should(BeTrue())
		})

		// GIVEN a multicluster setup with an admin and a manged cluster,
		// WHEN Jaeger operator is enabled in the admin cluster and the managed cluster is registered to it,
		// THEN we are able to query the metrics of Jaeger query running in admin cluster
		//      from the prometheus service running admin cluster.
		t.It("metrics of jaeger collector running in admin cluster are available in prometheus of admin cluster", func() {
			Eventually(func() bool {
				return pkg.IsJaegerMetricFound(adminKubeconfig, jaegerQuerySampleMetric, adminClusterName, nil)
			}).WithPolling(shortPollingInterval).WithTimeout(shortWaitTimeout).Should(BeTrue())
		})

	})
