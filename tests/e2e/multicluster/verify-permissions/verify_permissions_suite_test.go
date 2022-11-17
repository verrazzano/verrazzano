// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
package permissions_test

import (
	"github.com/onsi/ginkgo/v2"

	"testing"
)

func TestVerifyPermissions(test *testing.T) {
	t.RegisterFailHandler()
	ginkgo.RunSpecs(test, "Verify kubeconfig permissions multi-cluster Suite")
}
