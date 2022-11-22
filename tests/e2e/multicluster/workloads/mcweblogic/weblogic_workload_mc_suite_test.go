// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package mcweblogic

import (
	"testing"

	"github.com/onsi/ginkgo/v2"
)

func TestMultiClusterWebLogicApplication(test *testing.T) {
	t.RegisterFailHandler()
	ginkgo.RunSpecs(test, "Test Suite to validate the support for VerrazzanoWebLogicWorkload in multi-cluster environment")
}
