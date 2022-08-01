// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package monitor

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg"
)

var _ = t.Describe("Web Access", Label("f:platform-lcm:ha"), func() {
	t.Context("Prometheus", func() {
		RunningUntilShutdownIt("can access Prometheus endpoint", func() {
			Expect(pkg.VerifyPrometheusComponent(t.Logs, web.api, web.httpClient, web.users.verrazzano)).To(BeTrue())
		})
	})

	t.Context("Grafana", func() {
		RunningUntilShutdownIt("can access Grafana endpoint", func() {
			Expect(pkg.VerifyGrafanaComponent(t.Logs, web.api, web.httpClient, web.users.verrazzano)).To(BeTrue())
		})
	})

	t.Context("OpenSearch", func() {
		RunningUntilShutdownIt("can access OpenSearch endpoint", func() {
			Expect(pkg.VerifyOpenSearchComponent(t.Logs, web.api, web.httpClient, web.users.verrazzano)).To(BeTrue())
		})
	})

	t.Context("OpenSearch Dashboards", func() {
		RunningUntilShutdownIt("can access OpenSearch Dashboards endpoint", func() {
			Expect(pkg.VerifyOpenSearchDashboardsComponent(t.Logs, web.api, web.httpClient, web.users.verrazzano)).To(BeTrue())
		})
	})

	t.Context("Rancher", func() {
		RunningUntilShutdownIt("can get a Rancher admin token", func() {
			Expect(pkg.GetRancherAdminToken(t.Logs, web.httpClient, web.hosts.rancher)).ShouldNot(BeEmpty())
		})
	})

	t.Context("Kiali", func() {
		RunningUntilShutdownIt("can access Kiali endpoint", func() {
			Expect(pkg.AssertBearerAuthorized(web.httpClient, web.hosts.kiali)).To(BeTrue())
		})
	})
})
