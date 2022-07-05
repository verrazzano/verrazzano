
// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package bomvalidator

import (
	"github.com/verrazzano/verrazzano/pkg/test/framework"
	. "github.com/onsi/ginkgo/v2"
)

var t = framework.NewTestFramework("bom validator")

var _ = t.AfterEach(func() {})

var _ = t.Describe("Bom Validator", Label("f:platform-lcm.install"), func() {
	t.Context("after successful validation", func() {

	})
})