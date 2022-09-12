// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package monitor

import (
	"flag"
	"github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
	"github.com/verrazzano/verrazzano/pkg/test/framework"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg/ha"
	"testing"
)

var runContinuous bool
var t = framework.NewTestFramework("monitor")

func init() {
	flag.BoolVar(&runContinuous, "runContinuous", true, "run monitors continuously if set")
}

func TestHAMonitor(t *testing.T) {
	gomega.RegisterFailHandler(ginkgo.Fail)
	ginkgo.RunSpecs(t, "HA Monitoring Suite")
}

func RunningUntilShutdownIt(description string, test func()) {
	t.It(description, func() {
		for {
			test()
			// break out of the loop if we are not running the suite continuously,
			// or the shutdown signal is set
			if !runContinuous || ha.IsShutdownSignalSet(web.clientset) {
				t.Logs.Info("Shutting down...")
				break
			}
		}
	})
}
