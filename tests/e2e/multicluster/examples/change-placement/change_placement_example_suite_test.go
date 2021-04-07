// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package change_placement

import (
	"fmt"
	"testing"

	"github.com/onsi/ginkgo"
	"github.com/onsi/ginkgo/config"
	"github.com/onsi/ginkgo/reporters"
	"github.com/onsi/gomega"
)

func TestMultiClusterChangePlacementExample(t *testing.T) {
	gomega.RegisterFailHandler(ginkgo.Fail)
	junitReporter := reporters.NewJUnitReporter(fmt.Sprintf("mc-change-placement-%d-test-result.xml", config.GinkgoConfig.ParallelNode))
	ginkgo.RunSpecsWithDefaultAndCustomReporters(t, "Multi-cluster Change Placement Suite", []ginkgo.Reporter{junitReporter})
}
