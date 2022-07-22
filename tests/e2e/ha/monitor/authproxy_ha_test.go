// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package monitor

import (
	. "github.com/onsi/ginkgo/v2"
	"time"
)

var _ = t.Describe("AuthProxy HA Monitoring", Label("f:platform-lcm:ha"), func() {
	RunningUntilShutdownIt("verifies AuthProxy is ready and running", func() {
		t.Logs.Info("running authproxy tests!")
		time.Sleep(5 * time.Second)
	})
})
