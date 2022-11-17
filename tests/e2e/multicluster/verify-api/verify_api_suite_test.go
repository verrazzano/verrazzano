// Copyright (c) 2021, 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package api_test

import (
	"testing"

	"github.com/onsi/ginkgo/v2"
)

func TestVerifyRegister(test *testing.T) {
	t.RegisterFailHandler()
	ginkgo.RunSpecs(test, "Multi-cluster api test Suite")
}
