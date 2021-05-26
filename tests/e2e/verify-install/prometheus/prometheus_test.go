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

var _ = ginkgo.Describe("Verify Verrazzano component metrics", func() {
	ginkgo.Context("Verify NGINX ingress controller metrics", func() {
		isManagedClusterProfile := pkg.IsManagedClusterProfile()
		ginkgo.It("Verify NGINX metrics exist on admin cluster", func() {
			if !isManagedClusterProfile {
				gomega.Eventually(func() bool {
					return pkg.MetricsExist("nginx_ingress_controller_ingress_upstream_latency_seconds", "app_kubernetes_io_instance", "ingress-controller")
				}, longWaitTimeout, longPollingInterval).Should(gomega.BeTrue())
			}

		})
	})
})
