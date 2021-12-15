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
	// Setup the Suite
	m, _ := metrics.NewMetricsLogger("metrics")
	_ = framework.AfterEachM(m, func() {})

	framework.ItM(m, "Should do a thing", func() {
		fmt.Println("Ran a test!")
		// Emits a metric with key(foo), value(bar)
		metrics.Emit(m.With("foo", "bar"))

	})

	framework.ItM(m, "Should do another thing", func() {
		fmt.Println("Second test!")

	})
})
