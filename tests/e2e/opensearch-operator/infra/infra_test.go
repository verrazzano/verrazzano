// Copyright (C) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package infra

import (
	"fmt"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg"
	dump "github.com/verrazzano/verrazzano/tests/e2e/pkg/test/clusterdump"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg/test/framework"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg/test/framework/metrics"
	"time"
)

const (
	longWaitTimeout          = 20 * time.Minute
	longPollingInterval      = 20 * time.Second
	shortPollingInterval     = 10 * time.Second
	shortWaitTimeout         = 5 * time.Minute
	imagePullWaitTimeout     = 30 * time.Minute
	imagePullPollingInterval = 30 * time.Second
)

var (
	t                        = framework.NewTestFramework("infra")
	namespace                = pkg.GenerateNamespace("hello-helidon")
	expectedPodsHelloHelidon = []string{"hello-helidon-deployment"}
)

var _ = t.AfterEach(func() {})

var beforeSuite = t.BeforeSuiteFunc(func() {
	start := time.Now()
	pkg.DeployHelloHelidonApplication(namespace, "", "enabled", "", "")
	metrics.Emit(t.Metrics.With("deployment_elapsed_time", time.Since(start).Milliseconds()))

	t.Logs.Info("Container image pull check")
	Eventually(func() bool {
		return pkg.ContainerImagePullWait(namespace, expectedPodsHelloHelidon)
	}, imagePullWaitTimeout, imagePullPollingInterval).Should(BeTrue())

	t.Logs.Info("Helidon Example: check expected pods are running")
	Eventually(func() bool {
		result, err := pkg.PodsRunning(namespace, expectedPodsHelloHelidon)
		if err != nil {
			AbortSuite(fmt.Sprintf("One or more pods are not running in the namespace: %v, error: %v", namespace, err))
		}
		return result
	}, longWaitTimeout, longPollingInterval).Should(BeTrue(), "Helidon Example Failed to Deploy: Pods are not ready")

	beforeSuitePassed = true
})
var _ = BeforeSuite(beforeSuite)

var beforeSuitePassed = false

var afterSuite = t.AfterSuiteFunc(func() {
	if !beforeSuitePassed {
		dump.ExecuteBugReport(namespace)
	}
	start := time.Now()
	pkg.UndeployHelloHelidonApplication(namespace, "", "")
	metrics.Emit(t.Metrics.With("undeployment_elapsed_time", time.Since(start).Milliseconds()))
})

var _ = t.Describe("Verify OpenSearch infra", func() {

	t.It("verrazzano-system index is present", func() {
		Eventually(func() bool {
			return pkg.LogIndexFound("verrazzano-system")
		}, shortWaitTimeout, shortPollingInterval).Should(BeTrue())
	})

	t.Context("hello-helidon application logs are present.", func() {
		var err error
		var indexName string
		Eventually(func() error {
			indexName, err = pkg.GetOpenSearchAppIndex(namespace)
			return err
		}, shortWaitTimeout, shortPollingInterval).Should(BeNil(), "Expected to get OpenSearch App Index")

		// GIVEN an application with logging enabled
		// WHEN the Opensearch index for hello-helidon namespace is retrieved
		// THEN verify that it is found
		t.It("Verify Opensearch index for Logging exists", func() {
			Eventually(func() bool {
				return pkg.LogIndexFound(indexName)
			}, longWaitTimeout, longPollingInterval).Should(BeTrue(), "Expected to find log index for hello-helidon-container")
		})
		pkg.Concurrently(
			func() {
				// GIVEN an application with logging enabled
				// WHEN the log records are retrieved from the Opensearch index for hello-helidon-container
				// THEN verify that at least one recent log record is found
				t.It("Verify recent Opensearch log record exists", func() {
					Eventually(func() bool {
						return pkg.LogRecordFound(indexName, time.Now().Add(-24*time.Hour), map[string]string{
							"kubernetes.labels.app_oam_dev\\/name": "hello-helidon",
							"kubernetes.container_name":            "hello-helidon-container"})
					}, longWaitTimeout, longPollingInterval).Should(BeTrue(), "Expected to find a recent log record for container hello-helidon-container")
				})
			},
			func() {
				// GIVEN an application with logging enabled
				// WHEN the log records are retrieved from the Openearch index for other-container
				// THEN verify that at least one recent log record is found
				t.It("Verify recent Opensearch log record of other-container exists", func() {
					Eventually(func() bool {
						return pkg.LogRecordFound(indexName, time.Now().Add(-24*time.Hour), map[string]string{
							"kubernetes.labels.app_oam_dev\\/name": "hello-helidon",
							"kubernetes.container_name":            "other-container"})
					}, longWaitTimeout, longPollingInterval).Should(BeTrue(), "Expected to find a recent log record for other-container")
				})
			},
		)
	})
})
