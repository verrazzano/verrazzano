// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package prometheus_test

import (
	"time"

	"github.com/onsi/ginkgo"
	"github.com/onsi/gomega"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg"
)

const (
	longPollingInterval = 20 * time.Second
	longWaitTimeout     = 10 * time.Minute
)

// Query Prometheus for the sample metrics from the default scraping jobs
var _ = ginkgo.Describe("Verify default component metrics", func() {
	isManagedClusterProfile := pkg.IsManagedClusterProfile()
	if !isManagedClusterProfile {
		ginkgo.Context("Verify metrics from NGINX ingress controller", func() {
			ginkgo.It("Verify sample NGINX metrics can be queried from Prometheus", func() {
					gomega.Eventually(func() bool {
						return pkg.MetricsExist("nginx_ingress_controller_success", "controller_namespace", "ingress-nginx")
					}, longWaitTimeout, longPollingInterval).Should(gomega.BeTrue())
			})
		})
	}

	ginkgo.Context("Verify metrics from Container Advisor", func() {
		ginkgo.It("Verify sample Container Advisor metrics can be queried from Prometheus", func() {
				gomega.Eventually(func() bool {
					return pkg.MetricsExist("container_start_time_seconds", "namespace", "verrazzano-system")
				}, longWaitTimeout, longPollingInterval).Should(gomega.BeTrue())
		})
	})

	ginkgo.Context("Verify metrics from Node Exporter", func() {
		ginkgo.It("Verify sample Node Exporter metrics can be queried from Prometheus", func() {
			gomega.Eventually(func() bool {
				return pkg.MetricsExist("go_gc_duration_seconds", "job", "node-exporter")
			}, longWaitTimeout, longPollingInterval).Should(gomega.BeTrue())
		})
	})

	ginkgo.Context("Verify metrics from Istio mesh and istiod", func() {
		ginkgo.It("Verify sample mesh metrics can be queried from Prometheus", func() {
			gomega.Eventually(func() bool {
				return pkg.MetricsExist("istio_tcp_connections_opened_total", "namespace", "verrazzano-system")
			}, longWaitTimeout, longPollingInterval).Should(gomega.BeTrue())
		})
		ginkgo.It("Verify sample istiod metrics can be queried from Prometheus", func() {
			gomega.Eventually(func() bool {
				return pkg.MetricsExist("sidecar_injection_requests_total", "app", "istiod")
			}, longWaitTimeout, longPollingInterval).Should(gomega.BeTrue())
		})
	})

	ginkgo.Context("Verify metrics from Prometheus", func() {
		ginkgo.It("Verify sample Prometheus metrics can be queried from Prometheus", func() {
			gomega.Eventually(func() bool {
				return pkg.MetricsExist("prometheus_target_interval_length_seconds", "job", "prometheus")
			}, longWaitTimeout, longPollingInterval).Should(gomega.BeTrue())
		})
	})

})
