// Copyright (c) 2021, 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package install

import (
	"testing"

	"github.com/onsi/ginkgo/v2"
)

func TestPrometheus(test *testing.T) {
	t.RegisterFailHandler()
	ginkgo.RunSpecs(test, "Install Script Suite")
}
