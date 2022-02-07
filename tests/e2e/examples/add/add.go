// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package add

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// Add function that sums two integers
func Add(a, b int) int {
	return a + b
}

var _ = Describe("Adding", func() {
	Describe("Add", func() {
		It("adds two numbers together to form a sum", func() {
			sum := Add(2, 2)
			Expect(sum).To(Equal(4))
		})
	})
})
