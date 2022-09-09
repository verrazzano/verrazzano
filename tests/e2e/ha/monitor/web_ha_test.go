// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package monitor

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/verrazzano/verrazzano/tests/e2e/ha"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg"
)

var clusterDump = pkg.NewClusterDumpWrapper()

var _ = clusterDump.AfterEach(func() {}) // Dump cluster if spec fails

var _ = t.Describe("Web Access", Label("f:platform-lcm:ha"), func() {
	t.Context("Prometheus", func() {
		ha.RunningUntilShutdownIt(t, "can access Prometheus endpoint", web.clientset, runContinuous, func() {
			Expect(pkg.VerifyPrometheusComponent(t.Logs, nil, web.httpClient, web.users.verrazzano)).To(BeTrue())
		})
	})

	t.Context("OpenSearch", func() {
		ha.RunningUntilShutdownIt(t, "can access OpenSearch endpoint", web.clientset, runContinuous, func() {
			Expect(pkg.VerifyOpenSearchComponent(t.Logs, nil, web.httpClient, web.users.verrazzano)).To(BeTrue())
		})
	})

	t.Context("OpenSearch Dashboards", func() {
		ha.RunningUntilShutdownIt(t, "can access OpenSearch Dashboards endpoint", web.clientset, runContinuous, func() {
			Expect(pkg.VerifyOpenSearchDashboardsComponent(t.Logs, nil, web.httpClient, web.users.verrazzano)).To(BeTrue())
		})
	})

	t.Context("Rancher", func() {
		ha.RunningUntilShutdownIt(t, "can get a Rancher admin token", web.clientset, runContinuous, func() {
			Expect(pkg.GetRancherAdminToken(t.Logs, web.httpClient, web.hosts.rancher)).ShouldNot(BeEmpty())
		})
	})

	t.Context("Kiali", func() {
		ha.RunningUntilShutdownIt(t, "can access Kiali endpoint", web.clientset, runContinuous, func() {
			Expect(pkg.AssertBearerAuthorized(web.httpClient, web.hosts.kiali)).To(BeTrue())
		})
	})
})
