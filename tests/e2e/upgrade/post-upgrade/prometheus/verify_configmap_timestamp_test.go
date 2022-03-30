// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package prometheus

import (
	"os"
	"time"

	vzconst "github.com/verrazzano/verrazzano/pkg/constants"
	"github.com/verrazzano/verrazzano/pkg/test/framework/metrics"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/verrazzano/verrazzano/pkg/test/framework"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg"
)

var waitTimeout = 15 * time.Minute
var shortPollingInterval = 10 * time.Second

var t = framework.NewTestFramework("post-upgrade")

var failed = false

var _ = t.AfterEach(func() {
	failed = failed || framework.VzCurrentGinkgoTestDescription().Failed()
})

var _ = t.AfterSuite(func() {
	start := time.Now()
	if failed {
		pkg.ExecuteClusterDumpWithEnvVarConfig()
	}
	metrics.Emit(t.Metrics.With("after_suite_elapsed_time", time.Since(start).Milliseconds()))
})

var _ = t.Describe("Verify timestamp test,", Label("f:pre-upgrade"), func() {
	var actualConfigMapCreationTimestamp string
	var expectedConfigMapCreationTimestamp = os.Getenv(vzconst.PromConfigMapCreationTimestamp)

	t.BeforeEach(func() {
		Eventually(func() (string, error) {
			configMap, err := pkg.GetConfigMap(vzconst.VmiPromConfigName, vzconst.VerrazzanoSystemNamespace)
			if err != nil {
				return "", err
			}

			actualConfigMapCreationTimestamp = configMap.CreationTimestamp.UTC().String()
			return actualConfigMapCreationTimestamp, nil
		}, waitTimeout, shortPollingInterval).Should(Not(BeEmpty()), "Failed to get creation timestamp of prometheus configmap")
	})

	// Verify prometheus configmap is not deleted
	// GIVEN upgrade has completed
	// WHEN the vmo pod is restarted
	// THEN the creation timestamp on prometheus configmap should be same
	t.It("Verify prometheus configmap is not deleted.", func() {
		Expect(actualConfigMapCreationTimestamp).To(Equal(expectedConfigMapCreationTimestamp))
	})

})
