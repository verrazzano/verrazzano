// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
package validators

import (
	. "github.com/onsi/ginkgo/v2"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg/test/framework"
)

var t = framework.NewTestFramework("validators")

var _ = t.Describe("Verrazzano Validators Test Suite", Label("f:platform-lcm.install"), func() {
	t.Context("Validator tests", func() {
		// GIVEN A valid verrazzano installation
		// WHEN An attempt to make an illegal jaegerOperator configuration edit is made
		// THEN The validating webhook catches it and rejects it
		t.It("Run Validator Negative Test", runValidatorTest)
	})
})
