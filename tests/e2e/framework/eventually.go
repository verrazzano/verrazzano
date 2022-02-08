// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package framework

import (
	"github.com/onsi/gomega"
)

// EventuallyBeTrue expects the condition to be true
func EventuallyBeTrue(actual interface{}, interval1 interface{}, interval2 interface{}, explain ...interface{}) {
	gomega.EventuallyWithOffset(1, actual, interval1, interval2).Should(gomega.BeTrue(), explain)
}

// EventuallyEqual expects the specified two are the same, otherwise an exception raises
func EventuallyEqual(actual interface{}, extra interface{}, explain ...interface{}) {
	gomega.EventuallyWithOffset(1, actual).Should(gomega.Equal(extra), explain...)
}

// EventuallyNotEqual expects the specified two are not the same, otherwise an exception raises
func EventuallyNotEqual(actual interface{}, extra interface{}, explain ...interface{}) {
	gomega.EventuallyWithOffset(1, actual).ShouldNot(gomega.Equal(extra), explain...)
}

// EventuallyError expects an error happens, otherwise an exception raises
func EventuallyError(err error, explain ...interface{}) {
	gomega.EventuallyWithOffset(1, err).Should(gomega.HaveOccurred(), explain...)
}

// EventuallyNoError checks if "err" is set, and if so, fails assertion while logging the error.
func EventuallyNoError(err error, explain ...interface{}) {
	EventuallyNoErrorWithOffset(1, err, explain...)
}

// EventuallyNoErrorWithOffset checks if "err" is set, and if so, fails assertion while logging the error at "offset" levels above its caller
// (for example, for call chain f -> g -> ExpectNoErrorWithOffset(1, ...) error would be logged for "f").
func EventuallyNoErrorWithOffset(offset int, err error, explain ...interface{}) {
	gomega.EventuallyWithOffset(1+offset, err).ShouldNot(gomega.HaveOccurred(), explain...)
}

// EventuallyConsistOf expects actual contains precisely the extra elements.  The ordering of the elements does not matter.
func EventuallyConsistOf(actual interface{}, extra interface{}, explain ...interface{}) {
	gomega.EventuallyWithOffset(1, actual).Should(gomega.ConsistOf(extra), explain...)
}

// EventuallyHaveKey expects the actual map has the key in the keyset
func EventuallyHaveKey(actual interface{}, key interface{}, explain ...interface{}) {
	gomega.EventuallyWithOffset(1, actual).Should(gomega.HaveKey(key), explain...)
}

// EventuallyEmpty expects actual is empty
func EventuallyEmpty(actual interface{}, explain ...interface{}) {
	gomega.EventuallyWithOffset(1, actual).Should(gomega.BeEmpty(), explain...)
}

// EventuallyNotEmpty expects actual is not empty
func EventuallyNotEmpty(actual interface{}, interval1 interface{}, interval2 interface{}, explain ...interface{}) {
	gomega.EventuallyWithOffset(1, actual, interval1, interval2).ShouldNot(gomega.BeEmpty(), explain)
}
