// Copyright (c) 2021, 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
package main

import (
	"github.com/verrazzano/verrazzano/tests/e2e/pkg/test/framework"
	"github.com/verrazzano/verrazzano/tools/eventually-checker/testdata/internal"

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

var t = framework.NewTestFramework("main")

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

// Tests for the Ginkgo wrapped functions
var _ = t.Describe("Wrapper for the Ginkgo Describe node", func() {
	t.It("Test 6", func() {
		Eventually(func() (bool, error) {
			return true, nil
		})
	})

	t.It("Test 7, sample test with Expect inside Eventually", func() {
		Eventually(func() (bool, error) {
			// Linter should catch this as an issue
			Expect(true).To(BeTrue())
			return true, nil
		})
	})

	t.It("Test 8, sample test with Fail inside Eventually", func() {
		Eventually(func() (bool, error) {
			// Linter should catch this as an issue
			Fail("There is a failure")
			return true, nil
		})
	})

	t.It("Test 9, the function called from Eventually has Expect inside it", func() {
		Eventually(func() (bool, error) {
			return internal.DoCallExpect(), nil
		})

		Expect(false).To(BeTrue())
	})

	t.It("Test 10, call a function having Eventually has Expect inside it", func() {
		internal.DoCallEventually()
	})

	// The following calls are good
	Expect(true).To(BeTrue())
	Fail("This Fail is not in an eventually")
})
