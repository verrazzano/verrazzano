// Copyright (c) 2020, 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package verrazzano_test

import (
	"fmt"
	"testing"

	. "github.com/onsi/ginkgo"
	"github.com/onsi/ginkgo/config"
	. "github.com/onsi/gomega"
	vz "github.com/verrazzano/verrazzano/tests/e2e/pkg/ginkgo"
)

func TestVerrazzano(t *testing.T) {
	RegisterFailHandler(Fail)
	junitReporter := vz.NewJUnitReporter(fmt.Sprintf("verrazzano-%d-test-result.xml", config.GinkgoConfig.ParallelNode))
	vz.VZRunSpecsWithDefaultAndCustomReporters(t, "Verrazzano Suite", []Reporter{junitReporter}, "verrazzano.install")
	vz.CreateFeaturesXMLReport(fmt.Sprintf("verrazzano-%d-test-features.xml", config.GinkgoConfig.ParallelNode))
}
