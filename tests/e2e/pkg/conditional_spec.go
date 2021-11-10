// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
package pkg

import (
	"fmt"
	"github.com/onsi/ginkgo"
)

// ConditionalCheckFunc Test function for conditional specs
type ConditionalCheckFunc func() (bool, error)

// ConditionalSpec Executes the specified test spec/func when the condition function passes without error
func ConditionalSpec(description string, skipMessage string, condition ConditionalCheckFunc, specFunc interface{}) {
	checkPassed, err := condition()
	if err != nil {
		ginkgo.Fail(err.Error())
	}
	// Only run the target test function when the minimum version criteria is met
	if checkPassed {
		ginkgo.It(description, specFunc)
	} else {
		Log(Info, fmt.Sprintf("Skipping spec '%s', %s", description, skipMessage))
	}
}

// MinVersionSpec Executes the specified test spec/func when the Verrazzano version meets the minimum specified version
func MinVersionSpec(description string, minVersion string, specFunc interface{}) {
	ConditionalSpec(description, fmt.Sprintf("Min version not met: %s", minVersion), func() (bool, error) {
		return IsVerrazzanoMinVersion(minVersion)
	}, specFunc)
}
