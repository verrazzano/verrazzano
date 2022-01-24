// Copyright (c) 2021, 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
package internal

import . "github.com/onsi/gomega" //nolint

func DoSomething() (bool, error) {
	return someNestedFunc(), nil
}

func someNestedFunc() bool {
	Expect(false).To(BeTrue())
	return true
}

func AnotherFunc() bool {
	return false
}

func DoCallExpect() bool {
	return Expect(true).To(BeTrue())
}

func DoCallEventually() (bool, error) {
	Eventually(func() (bool, error) {
		Expect(true).To(BeTrue())
		return true, nil
	})
	return true, nil
}
