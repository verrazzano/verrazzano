// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package restapi_test

import (
	"fmt"
	"time"

	. "github.com/onsi/ginkgo/v2"
	"github.com/verrazzano/verrazzano/pkg/k8sutil"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg/test/framework/metrics"
)

var _ = t.Describe("argocd", Label("f:infra-lcm",
	"f:ui.console"), func() {
	t.Context("test to", func() {
		t.It("Verify argocd access and configuration", func() {
			kubeconfigPath, err := k8sutil.GetKubeConfigLocation()
			if err != nil {
				t.Logs.Error(fmt.Sprintf("Error getting kubeconfig: %v", err))
				t.Fail(err.Error())
			}
			if pkg.IsArgoCDEnabled(kubeconfigPath) {

				start := time.Now()
				err = pkg.VerifyArgoCDAccess(t.Logs, kubeconfigPath)
				if err != nil {
					t.Logs.Error(fmt.Sprintf("Error verifying access to Argocd: %v", err))
					t.Fail(err.Error())
				}

				metrics.Emit(t.Metrics.With("argocd_access_elapsed_time", time.Since(start).Milliseconds()))

				start = time.Now()
				t.Logs.Info("Accessing the Argocd Applications")
				err = pkg.VerifyArgocdApplicationAccess(t.Logs, kubeconfigPath)
				if err != nil {
					t.Logs.Error(fmt.Sprintf("Error verifying access to Argocd: %v", err))
					t.Fail(err.Error())
				}

				metrics.Emit(t.Metrics.With("argocd_access_elapsed_time", time.Since(start).Milliseconds()))

			}

		})
	})
})
