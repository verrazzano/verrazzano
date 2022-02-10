// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package poc

import "github.com/onsi/ginkgo"

// Describe annotates the test with the SIG label.
func Describe(text string, body func()) bool {
	return ginkgo.Describe("[verify_vz] "+text, body)
}
