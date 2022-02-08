// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package helidon

import (
	"github.com/onsi/ginkgo/v2"
)

// VZDescribe annotates the test with the Verrazzano specific label for this test set.
func VZDescribe(description string, body func()) bool {
	return ginkgo.Describe(description, ginkgo.Label("f:app-lcm.oam", "f:app-lcm.helidon-workload"), body)
}
