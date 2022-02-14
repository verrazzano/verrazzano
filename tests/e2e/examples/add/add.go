// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package add

import (
	"github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
	"github.com/verrazzano/verrazzano/tests/e2e/framework"
)

// Add function that sums two integers
func Add(a, b int) int {
	return a + b
}

var f = framework.NewDefaultFramework("add")

var _ = ginkgo.Describe("Adding", func() {
	ginkgo.Describe("Add", func() {
		ginkgo.It("adds two numbers together to form a sum", func() {
			sum := Add(2, 2)
			gomega.Expect(sum).To(gomega.Equal(4))
		})
	})
	framework.Emit(f.Metrics.With(framework.Duration, framework.DurationMillis()))
})
