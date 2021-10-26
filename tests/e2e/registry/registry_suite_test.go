// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package registry

import (
	"fmt"
	"testing"

	"github.com/onsi/ginkgo"
	"github.com/onsi/ginkgo/reporters"
	"github.com/onsi/gomega"
)

func TestKubernetes(t *testing.T) {
	gomega.RegisterFailHandler(ginkgo.Fail)
	junitReporter := reporters.NewJUnitReporter(fmt.Sprintf("registry-%d-test-result.xml", ginkgo.GinkgoParallelNode()))
	ginkgo.RunSpecsWithDefaultAndCustomReporters(t, "Private Registry Suite", []ginkgo.Reporter{junitReporter})
}
