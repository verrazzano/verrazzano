// Copyright (c) 2021, 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package restapi_test

import (
	. "github.com/onsi/ginkgo/v2"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg"
)

var _ = t.Describe("keycloak", Label("f:infra-lcm",
	"f:ui.console"), func() {

	t.Context("test to", func() {
		t.It("Verify Keycloak access", func() {
			pkg.VerifyKeycloakAccess(t)
		})
	})
})

var _ = t.AfterEach(func() {})
