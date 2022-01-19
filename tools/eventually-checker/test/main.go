// Copyright (c) 2021, 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
package main

import (
	"github.com/verrazzano/verrazzano/tools/eventually-checker/test/internal"

	. "github.com/onsi/ginkgo/v2" //nolint
	. "github.com/onsi/gomega"    //nolint
)

type testStruct struct {
}

func (t *testStruct) PointerReceiverThatCallsExpect() error {
	Expect(false).To(BeTrue())
	return nil
}

func (t testStruct) ValueReceiverThatCallsExpect() error {
	Expect(false).To(BeTrue())
	return nil
}

var ts testStruct

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

	It("Test 4", func() {
		Eventually(func() error {
			return ts.PointerReceiverThatCallsExpect()
		})
	})

	It("Test 5", func() {
		Eventually(func() error {
			return ts.ValueReceiverThatCallsExpect()
		})
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

// this common Ginkgo pattern is here to test a bug fix... prior to the fix, the "Fail"
// here would be associated with the preceding function declaration ("localFunc" in this
// case) and it would cause a false positive
var _ = Describe("Generic decl bug fix", func() {
	Fail("This is not in an eventually")
})
