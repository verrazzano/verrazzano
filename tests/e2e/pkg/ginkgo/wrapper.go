// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package ginkgo

import (
	"fmt"

	. "github.com/onsi/ginkgo"
	ginkgoExt "github.com/onsi/ginkgo/extensions/table"
)

var checker Checker

// VZRunSpecsWithDefaultAndCustomReporters is wrapper around the Ginkgo function RunSpecsWithDefaultAndCustomReporters.
// This function takes an additional argument that identifies the features being tested by a test suite.
func VZRunSpecsWithDefaultAndCustomReporters(t GinkgoTestingT, description string, specReporters []Reporter, features ...Feature) bool {
	initBuildFeaureChecker(description)
	checkFeature(features...)
	return RunSpecsWithDefaultAndCustomReporters(t, description, specReporters)
}

func VzDescribe(text string, body func(), features ...Feature) bool {
	initBuildFeaureChecker(text)
	checkFeature(features...)
	return Describe(text, body)
}

func VzDescribeTable(description string, itBody interface{}, feature Feature, entries ...ginkgoExt.TableEntry) bool {
	initBuildFeaureChecker(description)
	checkFeature(feature)
	return ginkgoExt.DescribeTable(description, itBody, entries...)
}

func initBuildFeaureChecker(text string) {
	if checker != nil {
		return
	}
	var err error
	checker, err = BuildFeatureChecker("../../../testdata/features/features.yaml", text)
	if err != nil {
		msg := fmt.Sprintf("--- ERROR: unable to build feature checker: %v", err)
		fmt.Fprintln(GinkgoWriter, msg)
		Fail(msg)
	}
}

func checkFeature(features ...Feature) {
	for _, feature := range features {
		//		fmt.Fprintln(GinkgoWriter, fmt.Sprintf("Testing feature: %s", feature))
		found, _ := checker.Check(feature)
		if !found {
			msg := fmt.Sprintf("--- ERROR: invalid feature specified: %s", feature)
			fmt.Fprintln(GinkgoWriter, msg)
			Fail(msg)
		}
	}
}
