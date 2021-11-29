// Copyright (C) 2020, 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package integ_test

import (
	"fmt"
	"testing"

	"github.com/onsi/ginkgo/v2/reporters"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestInteg(t *testing.T) {
	RegisterFailHandler(Fail)
	junitReporter := reporters.NewJUnitReporter(fmt.Sprintf("integ-%d-test-result.xml", GinkgoParallelProcess()))
	RunSpecsWithDefaultAndCustomReporters(t, "Integration Test Suite", []Reporter{junitReporter})
}
