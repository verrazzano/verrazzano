// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package framework

import "github.com/onsi/ginkgo"

// VzIt - wrapper function for ginkgo It
func VzIt(text string, body interface{}, timeout ...float64) bool {
	ginkgo.It(text, body, timeout...)
	return true
}

// VzBeforeEach - wrapper function for ginkgo BeforeEach
func VzBeforeEach(body interface{}, timeout ...float64) bool {
	ginkgo.BeforeEach(body, timeout...)
	return true
}
