// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package thanos

import (
	"testing"

	"github.com/onsi/ginkgo/v2"
)

func TestThanos(test *testing.T) {
	t.RegisterFailHandler()
	ginkgo.RunSpecs(test, "Thanos Test Suite")
}
