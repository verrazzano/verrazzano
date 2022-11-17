// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package e2e

import (
	"github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
	"testing"
)

func RunE2ETests(t *testing.T) {

	gomega.RegisterFailHandler(ginkgo.Fail)

	ginkgo.RunSpecs(test, "Verrazzano e2e tests")
}
