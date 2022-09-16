// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package monitor

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/verrazzano/verrazzano/pkg/k8sutil"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg"
	hacommon "github.com/verrazzano/verrazzano/tests/e2e/pkg/ha"
	"time"
)

const (
	waitTimeout     = 3 * time.Minute
	pollingInterval = 10 * time.Second
)

var (
	clusterDump = pkg.NewClusterDumpWrapper()
	clientset   = k8sutil.GetKubernetesClientsetOrDie()
)

var _ = clusterDump.AfterEach(func() {}) // Dump cluster if spec fails

var _ = t.Describe("Web Access", Label("f:platform-lcm:ha"), func() {
	t.Context("Prometheus", func() {
		hacommon.RunningUntilShutdownIt(t, "can access Prometheus endpoint", clientset, runContinuous, func() {
			Eventually(func() bool {
				return pkg.VerifyPrometheusComponent(t.Logs, nil, web.httpClient, web.users.verrazzano)
			}, waitTimeout, pollingInterval).Should(BeTrue())
		})
	})

	t.Context("OpenSearch", func() {
		hacommon.RunningUntilShutdownIt(t, "can access OpenSearch endpoint", clientset, runContinuous, func() {
			Eventually(func() bool {
				return pkg.VerifyOpenSearchComponent(t.Logs, nil, web.httpClient, web.users.verrazzano)
			}, waitTimeout, pollingInterval).Should(BeTrue())
		})
	})

	t.Context("OpenSearch Dashboards", func() {
		hacommon.RunningUntilShutdownIt(t, "can access OpenSearch Dashboards endpoint", clientset, runContinuous, func() {
			Eventually(func() bool {
				return pkg.VerifyOpenSearchDashboardsComponent(t.Logs, nil, web.httpClient, web.users.verrazzano)
			}, waitTimeout, pollingInterval).Should(BeTrue())
		})
	})

	t.Context("Rancher", func() {
		hacommon.RunningUntilShutdownIt(t, "can get a Rancher admin token", clientset, runContinuous, func() {
			Eventually(func() string {
				return pkg.GetRancherAdminToken(t.Logs, web.httpClient, web.hosts.rancher)
			}, waitTimeout, pollingInterval).ShouldNot(BeEmpty())
		})
	})

	t.Context("Kiali", func() {
		hacommon.RunningUntilShutdownIt(t, "can access Kiali endpoint", clientset, runContinuous, func() {
			Eventually(func() bool {
				return pkg.AssertBearerAuthorized(web.httpClient, web.hosts.kiali)
			}, waitTimeout, pollingInterval).Should(BeTrue())
		})
	})
})
