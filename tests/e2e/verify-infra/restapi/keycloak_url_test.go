// Copyright (c) 2021, 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package restapi_test

import (
	"fmt"
	"time"

	. "github.com/onsi/ginkgo/v2"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg/test/framework/metrics"
)

var _ = t.Describe("keycloak", Label("f:infra-lcm",
	"f:ui.console"), func() {

	t.Context("test to", func() {
		t.It("Verify Keycloak access", func() {
			start := time.Now()
			err := pkg.VerifyKeycloakAccess(t.Logs)
			metrics.Emit(t.Metrics.With("verify_keycloak_access_response_time", time.Since(start).Milliseconds()))
			if err != nil {
				t.Logs.Error(fmt.Sprintf("Error verifying keycloak access: %v", err))
				t.Fail(err.Error())
			}
		})
	})
})

var _ = t.AfterEach(func() {})
