// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package netpol

import (
	"fmt"
	"testing"

	"github.com/onsi/ginkgo"
	"github.com/onsi/ginkgo/config"
	"github.com/onsi/ginkgo/reporters"
	"github.com/onsi/gomega"
)

func TestSecurityNetworkPolicies(t *testing.T) {
	gomega.RegisterFailHandler(ginkgo.Fail)
	junitReporter := reporters.NewJUnitReporter(fmt.Sprintf("security-network-policy-%d-test-result.xml", config.GinkgoConfig.ParallelNode))
	ginkgo.RunSpecsWithDefaultAndCustomReporters(t, "Verrazzano Network Policy Suite", []ginkgo.Reporter{junitReporter})
}
