// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package prometheus

import (
	"os"
	"time"

	"github.com/verrazzano/verrazzano/pkg/test/framework/metrics"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	vzconst "github.com/verrazzano/verrazzano/pkg/constants"
	"github.com/verrazzano/verrazzano/pkg/test/framework"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg"
)

var waitTimeout = 15 * time.Minute
var pollingInterval = 30 * time.Second
var shortPollingInterval = 10 * time.Second

var t = framework.NewTestFramework("pre-upgrade")

var _ = t.BeforeSuite(func() {
	start := time.Now()
	recordConfigMapCreationTS()
	beforeSuitePassed = true
	metrics.Emit(t.Metrics.With("before_suite_elapsed_time", time.Since(start).Milliseconds()))
})

var failed = false
var beforeSuitePassed = false

var _ = t.AfterEach(func() {
	failed = failed || framework.VzCurrentGinkgoTestDescription().Failed()
})

var _ = t.AfterSuite(func() {
	start := time.Now()
	if failed || !beforeSuitePassed {
		pkg.ExecuteClusterDumpWithEnvVarConfig()
	}
	metrics.Emit(t.Metrics.With("after_suite_elapsed_time", time.Since(start).Milliseconds()))
})

func recordConfigMapCreationTS() {
	t.Logs.Info("Recording prometheus Cretaion timestamp")

	t.Logs.Info("Get Prometheus configmap creation timestamp")
	Eventually(func() (string, error) {
		configMap, err := pkg.GetConfigMap(vzconst.VmiPromConfigName, vzconst.VerrazzanoSystemNamespace)
		if err != nil {
			return "", err
		}

		creationTimestamp := configMap.CreationTimestamp.UTC().String()
		os.Setenv(vzconst.PromConfigMapCreationTimestamp, creationTimestamp)
		return creationTimestamp, nil
	}, waitTimeout, shortPollingInterval).ShouldNot(BeEmpty())
}

var _ = t.Describe("Record timestamp test,", Label("f:pre-upgrade"), func() {
	// Verify that prometheus configmap creation timestamp is set in an Environment variable
	// GIVEN the prometheus configmap is created
	// WHEN the upgrade has not started and vmo pod is not restarted
	// THEN the environment variable PROM_CONFIGMAP_CREATION_TIMESTAMP is populated
	t.Context("check PROM_CONFIGMAP_CREATION_TIMESTAMP env variable", func() {
		t.It("in foo namespace", func() {
			Eventually(func() string {
				return os.Getenv(vzconst.PromConfigMapCreationTimestamp)
			}, waitTimeout, pollingInterval).ShouldNot(BeEmpty())
		})
	})

})
