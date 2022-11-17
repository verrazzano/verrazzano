// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package mcnshelidon

import (
	"testing"

	"github.com/onsi/ginkgo/v2"
)

func TestMultiClusterHelidonExample(test *testing.T) {
	t.RegisterFailHandler()
	ginkgo.RunSpecs(test, "Multi-cluster Hello Helidon NS Ops Suite")
}
