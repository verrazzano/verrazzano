// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package restapi_test

import (
	"fmt"
	"time"

	. "github.com/onsi/ginkgo/v2"
	"github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg/test/framework/metrics"
)

var _ = t.Describe("Argo CD", Label("f:infra-lcm",
	"f:ui.console"), func() {

	t.BeforeEach(func() {
		argoCD := vz.Status.Components["keycloak"]
		if argoCD == nil || argoCD.State == v1alpha1.CompStateDisabled {
			Skip("Argo CD disabled, skipping test")
		}

	})

	t.Context("is enabled", func() {
		t.It("Web URL and the applications page is accessible", func() {
			start := time.Now()
			// Verifying if the Argo CD Ingress URL is accessible
			err := pkg.VerifyArgoCDAccess(t.Logs)
			if err != nil {
				t.Logs.Error(fmt.Sprintf("Error verifying access to Argocd: %v", err))
				t.Fail(err.Error())
			}

			metrics.Emit(t.Metrics.With("argocd_access_elapsed_time", time.Since(start).Milliseconds()))

			start = time.Now()
			t.Logs.Info("Accessing the Argocd Applications")
			//Verifying if the Applications page is accessible after login
			err = pkg.VerifyArgoCDApplicationAccess(t.Logs)
			if err != nil {
				t.Logs.Error(fmt.Sprintf("Error verifying access to Argocd application: %v", err))
				t.Fail(err.Error())
			}

			metrics.Emit(t.Metrics.With("argocd_access_elapsed_time", time.Since(start).Milliseconds()))

		})
	})
})
