// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package monitor

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg"
)

var _ = t.Describe("Prometheus HA Monitoring", Label("f:platform-lcm:ha"), func() {
	RunningUntilShutdownIt("verifies Prometheus is ready and running", func() {
		Expect(pkg.VerifyPrometheusComponent(t.Logs, api, vzHTTPClient, vmiCredentials)).To(BeTrue())
	})
})
