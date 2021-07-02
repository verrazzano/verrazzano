// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package ginkgo

import (
	"fmt"

	. "github.com/onsi/ginkgo"
)

// VZRunSpecsWithDefaultAndCustomReporters is wrapper around the Ginkgo function RunSpecsWithDefaultAndCustomReporters.
// This function takes an additional argument that identifies the features being tested by a test suite.
func VZRunSpecsWithDefaultAndCustomReporters(t GinkgoTestingT, description string, specReporters []Reporter, features []Feature) bool {
	fmt.Fprintln(GinkgoWriter, fmt.Sprintf("Features being testing: %v", features))
	checker, err := BuildChecker("../../../testdata/features/features.yaml", description)
	if err != nil {
		msg := fmt.Sprintf("- unable to build feature checker: %v", err)
		fmt.Fprintln(GinkgoWriter, msg)
		Fail(msg)
	}

	for _, feature := range features {
		found, _ := checker.Check(feature)
		if !found {
			msg := fmt.Sprintf("- invalid feature specified: %s", feature)
			fmt.Fprintln(GinkgoWriter, msg)
			Fail(msg)
		}
	}

	return RunSpecsWithDefaultAndCustomReporters(t, description, specReporters)
}
