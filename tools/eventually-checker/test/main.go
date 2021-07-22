// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
package main

import (
	"github.com/verrazzano/verrazzano/tools/eventually-checker/test/internal"

	. "github.com/onsi/ginkgo" //nolint
	. "github.com/onsi/gomega" //nolint
)

func main() {
	It("Test 1", func() {
		Eventually(func() (bool, error) {
			localFunc()
			return internal.DoSomething()
		})

		Expect(false).To(BeTrue())
	})

	It("Test 2", func() {
		Eventually(eventuallyFunc)
	})

	It("Test 3", func() {
		Eventually(internal.AnotherFunc)
	})
}

func eventuallyFunc() bool {
	Fail("FAIL!")
	return true
}

func unusedFunc() { //nolint
	internal.DoSomething()
}

func localFunc() {
}
