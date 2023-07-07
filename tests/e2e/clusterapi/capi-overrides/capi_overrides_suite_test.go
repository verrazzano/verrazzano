// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package capi_overrides

import (
	"testing"

	"github.com/onsi/ginkgo/v2"
)

func TestClusterAPIOverrides(test *testing.T) {
	t.RegisterFailHandler()
	ginkgo.RunSpecs(test, "Cluster API Overrides Test Suite")
}
