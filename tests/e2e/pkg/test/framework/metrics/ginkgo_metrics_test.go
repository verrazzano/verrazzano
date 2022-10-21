// Copyright (c) 2021, 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package metrics_test

import (
	"fmt"

	"github.com/verrazzano/verrazzano/tests/e2e/pkg/test/framework"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg/test/framework/metrics"
)

var t = framework.NewTestFramework("metrics_test")

var _ = t.Describe("Logger", func() {
	// Setup the Suite

	_ = t.AfterEach(func() {})

	t.It("Should do a thing", func() {
		fmt.Println("Ran a test!")
		// Emits a metric with key(foo), value(bar)
		metrics.Emit(t.Metrics.With("foo", "bar"))

	})

	t.It("Should do another thing", func() {
		fmt.Println("Second test!")

	})
})
