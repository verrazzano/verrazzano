// Copyright (c) 2021, 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
package internal

import "github.com/onsi/gomega"

func DoSomething() (bool, error) {
	return someNestedFunc(), nil
}

func someNestedFunc() bool {
	gomega.Expect(false).To(gomega.BeTrue())
	return true
}

func AnotherFunc() bool {
	return false
}

func DoCallExpect() bool {
	return gomega.Expect(true).To(gomega.BeTrue())
}

func DoCallEventually() (bool, error) {
	gomega.Eventually(func() (bool, error) {
		gomega.Expect(true).To(gomega.BeTrue())
		return true, nil
	})
	return true, nil
}
