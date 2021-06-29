// Copyright (c) 2020, 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package verrazzano_test

import (
	"fmt"
	"testing"

	"github.com/onsi/ginkgo"
	"github.com/onsi/ginkgo/config"
	"github.com/onsi/gomega"
	vzreporters "github.com/verrazzano/verrazzano/tests/e2e/pkg/reporters"
)

func TestVerrazzano(t *testing.T) {
	gomega.RegisterFailHandler(ginkgo.Fail)
	junitReporter := vzreporters.NewJUnitReporter(fmt.Sprintf("verrazzano-%d-test-result.xml", config.GinkgoConfig.ParallelNode))
	ginkgo.RunSpecsWithDefaultAndCustomReporters(t, "Verrazzano Suite", []ginkgo.Reporter{junitReporter})
}
