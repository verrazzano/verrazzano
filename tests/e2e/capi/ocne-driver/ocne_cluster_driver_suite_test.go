// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package ocnedriver

import (
	"testing"

	"github.com/onsi/ginkgo/v2"
)

func TestOCNEClusterDriver(test *testing.T) {
	t.RegisterFailHandler()
	ginkgo.RunSpecs(test, "OCNE Cluster Driver Suite")
}
