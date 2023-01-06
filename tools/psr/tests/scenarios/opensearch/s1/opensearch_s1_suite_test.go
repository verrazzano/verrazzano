// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package s1

import (
	"flag"
	"testing"

	"github.com/onsi/ginkgo/v2"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg/test/framework"
)

var t = framework.NewTestFramework("psr_ops_s1")

var skipStartScenario bool
var skipStopScenario bool

func init() {
	flag.BoolVar(&skipStartScenario, "skipStartScenario", false, "skipStartScenario skips the call to start the scenario")
	flag.BoolVar(&skipStopScenario, "skipStopScenario", false, "skipStopScenario skips the call to stop the scenario")
}

func TestOpenSearchScenarios(test *testing.T) {
	t.RegisterFailHandler()
	ginkgo.RunSpecs(test, "Opensearch ops-s1 Suite")
}
