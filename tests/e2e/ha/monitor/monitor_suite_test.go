// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package monitor

import (
	"flag"
	"testing"

	"github.com/onsi/ginkgo/v2"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg/test/framework"
)

var runContinuous bool
var t = framework.NewTestFramework("monitor")

func init() {
	flag.BoolVar(&runContinuous, "runContinuous", true, "run monitors continuously if set")
}

func TestHAMonitor(test *testing.T) {
	t.RegisterFailHandler()
	ginkgo.RunSpecs(test, "HA Monitoring Suite")
}
