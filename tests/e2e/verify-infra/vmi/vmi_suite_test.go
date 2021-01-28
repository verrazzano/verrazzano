// Copyright (c) 2020, 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package vmi_test

import (
	"fmt"
	"github.com/onsi/ginkgo/config"
	"github.com/onsi/ginkgo/reporters"
	"testing"

	"github.com/onsi/ginkgo"
	"github.com/onsi/gomega"
)

func TestVmi(t *testing.T) {
	gomega.RegisterFailHandler(ginkgo.Fail)
	junitReporter := reporters.NewJUnitReporter(fmt.Sprintf("vmi-%d-test-result.xml", config.GinkgoConfig.ParallelNode))
	ginkgo.RunSpecsWithDefaultAndCustomReporters(t, "VMI Suite", []ginkgo.Reporter{junitReporter})
}
