// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package jaeger

import (
	"strings"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/verrazzano/verrazzano/pkg/k8sutil"
	"github.com/verrazzano/verrazzano/platform-operator/constants"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg/test/framework"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg/update"
)

const (
	shortPollingInterval = 10 * time.Second
	shortWaitTimeout     = 5 * time.Minute
	disableErrorMsg      = "disabling component jaegerOperator is not allowed"
)

var (
	// Initialize the Test Framework
	t     = framework.NewTestFramework("update jaeger operator")
	start = time.Now()
)

var _ = t.BeforeSuite(func() {
	m := JaegerOperatorEnabledModifier{}
	update.UpdateCRWithRetries(m, pollingInterval, waitTimeout)
})

var _ = t.Describe("Update Jaeger", Label("f:platform-lcm.update"), func() {

	// GIVEN a VZ custom resource in dev profile,
	// WHEN Jaeger operator is enabled,
	// THEN Jaeger operator and pods for jaeger-collector and jaeger-query components gets created.
	t.It("Jaeger enable post install", func() {
		update.ValidatePods(jaegerOperatorLabelValue, jaegerComponentLabel, constants.VerrazzanoMonitoringNamespace, 1, false)
		update.ValidatePods(jaegerCollectorLabelValue, jaegerComponentLabel, constants.VerrazzanoMonitoringNamespace, 1, false)
		update.ValidatePods(jaegerQueryLabelValue, jaegerComponentLabel, constants.VerrazzanoMonitoringNamespace, 1, false)
	})

	// GIVEN a VZ custom resource in dev profile,
	// WHEN Jaeger operator is enabled,
	// THEN Jaeger OpenSearch Index Cleaner cron job exists
	t.It("should have a Jaeger OpenSearch Index Cleaner cron job", func() {
		validatorFn := pkg.ValidateEsIndexCleanerCronJobFunc()
		Eventually(validatorFn).WithPolling(pollingInterval).WithTimeout(waitTimeout).Should(BeTrue())
	})

	// GIVEN a VZ custom resource in dev profile,
	// WHEN Jaeger operator is enabled,
	// THEN we are able to get the traces
	t.It("traces from verrazzano system components should be available when queried from Jaeger", func() {
		kubeconfigPath, err := k8sutil.GetKubeConfigLocation()
		if err != nil {
			Fail(err.Error())
		}
		validatorFn := pkg.ValidateSystemTracesFuncInCluster(kubeconfigPath, start, "local")
		Eventually(validatorFn).WithPolling(shortPollingInterval).WithTimeout(shortWaitTimeout).Should(BeTrue())
	})

	// GIVEN a VZ custom resource in dev profile,
	// WHEN Jaeger operator is enabled,
	// THEN we are able to get the traces
	t.It("traces from verrazzano system components should be available in the OS backend storage.", func() {
		validatorFn := pkg.ValidateSystemTracesInOSFunc(start)
		Eventually(validatorFn).WithPolling(shortPollingInterval).WithTimeout(shortWaitTimeout).Should(BeTrue())
	})

	// GIVEN a VZ custom resource in dev profile,
	// WHEN Jaeger operator is enabled,
	// THEN we see that the metrics of Jaeger operator are present in prometheus
	t.It("metrics of jaeger operator are available in prometheus", func() {
		validatorFn := pkg.ValidateJaegerOperatorMetricFunc()
		Eventually(validatorFn).WithPolling(shortPollingInterval).WithTimeout(shortWaitTimeout).Should(BeTrue())
	})

	// GIVEN a VZ custom resource in dev profile,
	// WHEN Jaeger operator is enabled,
	// THEN we see that the metrics of Jaeger collector are present in prometheus
	t.It("metrics of jaeger collector are available in prometheus", func() {
		validatorFn := pkg.ValidateJaegerCollectorMetricFunc()
		Eventually(validatorFn).WithPolling(shortPollingInterval).WithTimeout(shortWaitTimeout).Should(BeTrue())
	})

	// GIVEN a VZ custom resource in dev profile,
	// WHEN Jaeger operator is enabled,
	// THEN we see that the metrics of Jaeger query are present in prometheus
	t.It("metrics of jaeger query are available in prometheus", func() {
		validatorFn := pkg.ValidateJaegerQueryMetricFunc()
		Eventually(validatorFn).WithPolling(shortPollingInterval).WithTimeout(shortWaitTimeout).Should(BeTrue())
	})

	// GIVEN a VZ custom resource in dev profile,
	// WHEN Jaeger operator is enabled,
	// THEN we see that the metrics of Jaeger agent are present in prometheus
	t.It("metrics of jaeger agent are available in prometheus", func() {
		validatorFn := pkg.ValidateJaegerAgentMetricFunc()
		Eventually(validatorFn).WithPolling(shortPollingInterval).WithTimeout(shortWaitTimeout).Should(BeTrue())
	})

	// GIVEN a VZ custom resource in dev profile with Jaeger operator enabled,
	// WHEN user tries to disable it,
	// THEN the operation should be denied with an error
	t.It("disabling previously enabled Jaeger operator should be disallowed", func() {
		m := JaegerOperatorCleanupModifier{}
		Eventually(func() bool {
			err := update.UpdateCR(m)
			foundExpectedErr := err != nil && strings.Contains(err.Error(), disableErrorMsg)
			return foundExpectedErr
		}).Should(BeTrue(), pollingInterval, waitTimeout)
	})
})
