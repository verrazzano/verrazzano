// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package metrics_test

import (
	"fmt"
	. "github.com/onsi/ginkgo/v2"
	"github.com/verrazzano/verrazzano/pkg/test/framework"
	"github.com/verrazzano/verrazzano/pkg/test/framework/metrics"
)

var _ = Describe("Logger", func() {
	m, _ := metrics.NewForPackage("metrics_test")

	_ = framework.VzAfterEach(m, func() {})

	framework.VzIt(m, "Should do a thing", func() {
		fmt.Println("Ran a test!")

	})

	framework.VzIt(m, "Should do another thing", func() {
		fmt.Println("Second test!")

	})
})
