// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package coherencelogging

import (
	"fmt"
	"testing"

	"github.com/onsi/ginkgo"
	"github.com/onsi/ginkgo/reporters"
	"github.com/onsi/gomega"
)

// TestCoherenceLoggingTrait tests an ingress trait setup for console access.
func TestCoherenceLoggingTrait(t *testing.T) {
	gomega.RegisterFailHandler(ginkgo.Fail)
	junitReporter := reporters.NewJUnitReporter(fmt.Sprintf("coherence-loggingtrait-%d-test-result.xml", ginkgo.GinkgoParallelNode()))
	ginkgo.RunSpecsWithDefaultAndCustomReporters(t, "Coherence Logging Trait Test Suite", []ginkgo.Reporter{junitReporter})
}
