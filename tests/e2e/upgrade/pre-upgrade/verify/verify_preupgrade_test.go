// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package verify

import (
	"fmt"
	"io/ioutil"
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

var t = framework.NewTestFramework("verify")

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
			pkg.Log(pkg.Error, fmt.Sprintf("Failed getting configmap: %v", err))
			return "", err
		}

		creationTimestamp := configMap.CreationTimestamp.UTC().String()
		f, err := os.Create(fmt.Sprintf("../../%s", vzconst.PromConfigMapCreationTimestampFile))
		if err != nil {
			pkg.Log(pkg.Error, fmt.Sprintf("Failed creating timestamp file: %v", err))
			return "", err
		}
		defer f.Close()

		_, err = f.WriteString(creationTimestamp)
		if err != nil {
			pkg.Log(pkg.Error, fmt.Sprintf("Failed writing to timestamp file: %v", err))
			return "", err
		}

		err = f.Sync()
		if err != nil {
			pkg.Log(pkg.Error, fmt.Sprintf("Failed saving timestamp file: %v", err))
			return "", err
		}

		pkg.Log(pkg.Info, fmt.Sprintf("Wrote configmap timestamp to : %v", f))
		return creationTimestamp, nil
	}, waitTimeout, shortPollingInterval).ShouldNot(BeEmpty())
}

var _ = t.Describe("Record prometheus configmap timestamp", Label("f:pre-upgrade"), func() {
	// Verify that prometheus configmap creation timestamp is set in an Environment variable
	// GIVEN the prometheus configmap is created
	// WHEN the upgrade has not started and vmo pod is not restarted
	// THEN the file contining configmap timestamp is populated
	t.Context("check prometheus configmap timestamp", func() {
		t.It("before upgrade", func() {
			Eventually(func() string {
				data, err := ioutil.ReadFile(vzconst.PromConfigMapCreationTimestampFile)
				if err != nil {
					return ""
				}
				return string(data)
			}, waitTimeout, pollingInterval).ShouldNot(BeEmpty())
		})
	})

})
