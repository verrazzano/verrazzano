// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package monitor

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg"
)

var _ = t.Describe("Prometheus REST API", Label("f:platform-lcm:ha"), func() {
	RunningUntilShutdownIt("can access Prometheus over HTTPS", func() {
		Expect(pkg.VerifyPrometheusComponent(t.Logs, api, vzHTTPClient, vmiCredentials)).To(BeTrue())
	})
})

var _ = t.Describe("Grafana REST API", Label("f:platform-lcm:ha"), func() {
	RunningUntilShutdownIt("can access Grafana over HTTPS", func() {
		Expect(pkg.VerifyGrafanaComponent(t.Logs, api, vzHTTPClient, vmiCredentials)).To(BeTrue())
	})
})

var _ = t.Describe("OpenSearch REST API", Label("f:platform-lcm:ha"), func() {
	RunningUntilShutdownIt("can access OpenSearch over HTTPS", func() {
		Expect(pkg.VerifyOpenSearchComponent(t.Logs, api, vzHTTPClient, vmiCredentials)).To(BeTrue())
	})
})
