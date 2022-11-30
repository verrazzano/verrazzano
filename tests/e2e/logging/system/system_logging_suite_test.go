// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package system

import (
	"testing"

	"github.com/onsi/ginkgo/v2"
)

// TestSystemLogging tests the logging from Verrazzano system components.
func TestSystemLogging(test *testing.T) {
	t.RegisterFailHandler()
	ginkgo.RunSpecs(test, "System Logging Suite")
}
