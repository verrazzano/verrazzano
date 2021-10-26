// Copyright (c) 2020, 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package restapi_test

import (
	"fmt"
	"testing"

	"github.com/onsi/ginkgo"
	"github.com/onsi/ginkgo/reporters"
	"github.com/onsi/gomega"
)

func TestRestApi(t *testing.T) {
	gomega.RegisterFailHandler(ginkgo.Fail)
	junitReporter := reporters.NewJUnitReporter(fmt.Sprintf("rest-api-%d-test-result.xml", ginkgo.GinkgoParallelNode()))
	ginkgo.RunSpecsWithDefaultAndCustomReporters(t, "REST API Suite", []ginkgo.Reporter{junitReporter})
}
