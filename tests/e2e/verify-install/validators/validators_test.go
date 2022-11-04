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
		t.It("Component Validator v1beta1 Negative Test", runValidatorTestV1Beta1)
		t.It("Component Validator v1alpha1 negative test", runValidatorTestV1Alpha1)

		// GIVEN A valid verrazzano installation
		// WHEN An attempt to make a mysql PodSepc modification configuration edit is made
		// THEN The MySQL podspec webhook issues a warning
		t.It("Run MySQL podspec warning v1beta1 negative test", runMySQLPodspecEditWarningTestV1Beta1)
		t.It("Run MySQL podspec warning v1alpha1 negative test", runMySQLPodspecEditWarningTestV1Alpha1)
	})
})
