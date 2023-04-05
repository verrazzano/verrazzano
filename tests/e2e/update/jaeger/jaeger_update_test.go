// Copyright (c) 2022, 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package jaeger

import (
	"fmt"
	"strings"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/verrazzano/verrazzano/pkg/k8sutil"
	"github.com/verrazzano/verrazzano/platform-operator/constants"
	"github.com/verrazzano/verrazzano/tests/e2e/jaeger"
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

var whenJaegerOperatorEnabledIt = t.WhenMeetsConditionFunc(jaeger.OperatorCondition, jaeger.IsJaegerEnabled)
var kubeconfigPath string
var metricsTest pkg.MetricsTest

var beforeSuite = t.BeforeSuiteFunc(func() {
	m := JaegerOperatorEnabledModifier{}
	update.UpdateCRWithRetries(m, pollingInterval, waitTimeout)

	// GIVEN a VZ custom resource in dev profile,
	// WHEN Jaeger operator is enabled,
	// THEN Jaeger operator and pods for jaeger-collector and jaeger-query components gets created.
	update.ValidatePods(jaegerOperatorLabelValue, jaegerComponentLabel, constants.VerrazzanoMonitoringNamespace, 1, false)
	update.ValidatePods(jaegerCollectorLabelValue, jaegerComponentLabel, constants.VerrazzanoMonitoringNamespace, 1, false)
	update.ValidatePods(jaegerQueryLabelValue, jaegerComponentLabel, constants.VerrazzanoMonitoringNamespace, 1, false)

	var err error
	kubeconfigPath, err = k8sutil.GetKubeConfigLocation()
	if err != nil {
		AbortSuite(fmt.Sprintf("Failed to find Kubeconfig location: %v", err))
	}
	metricsTest, err = pkg.NewMetricsTest([]string{kubeconfigPath}, kubeconfigPath, map[string]string{})
	if err != nil {
		AbortSuite(fmt.Sprintf("Failed to create the Metrics test object: %v", err))
	}
})

var _ = BeforeSuite(beforeSuite)

var _ = t.Describe("Update Jaeger", Label("f:platform-lcm.update"), func() {

	// GIVEN a VZ custom resource in dev profile,
	// WHEN Jaeger operator is enabled,
	// THEN Jaeger OpenSearch Index Cleaner cron job exists
	whenJaegerOperatorEnabledIt("should have a Jaeger OpenSearch Index Cleaner cron job", func() {
		validatorFn := pkg.ValidateEsIndexCleanerCronJobFunc()
		Eventually(validatorFn).WithPolling(pollingInterval).WithTimeout(waitTimeout).Should(BeTrue())
	})

	// GIVEN a VZ custom resource in dev profile,
	// WHEN Jaeger operator is enabled,
	// THEN we are able to get the traces
	whenJaegerOperatorEnabledIt("traces from verrazzano system components should be available when queried from Jaeger", func() {
		validatorFn := pkg.ValidateSystemTracesFuncInCluster(kubeconfigPath, start, "local")
		Eventually(validatorFn).WithPolling(shortPollingInterval).WithTimeout(shortWaitTimeout).Should(BeTrue())
	})

	// GIVEN a VZ custom resource in dev profile,
	// WHEN Jaeger operator is enabled,
	// THEN we are able to get the traces
	whenJaegerOperatorEnabledIt("traces from verrazzano system components should be available in the OS backend storage.", func() {
		validatorFn := pkg.ValidateSystemTracesInOSFunc(start)
		Eventually(validatorFn).WithPolling(shortPollingInterval).WithTimeout(shortWaitTimeout).Should(BeTrue())
	})

	// GIVEN a VZ custom resource in dev profile,
	// WHEN Jaeger operator is enabled,
	// THEN we see that the metrics of Jaeger operator are present in prometheus
	whenJaegerOperatorEnabledIt("metrics of jaeger operator are available in prometheus", func() {
		validatorFn := pkg.ValidateJaegerOperatorMetricFunc(metricsTest)
		Eventually(validatorFn).WithPolling(shortPollingInterval).WithTimeout(shortWaitTimeout).Should(BeTrue())
	})

	// GIVEN a VZ custom resource in dev profile,
	// WHEN Jaeger operator is enabled,
	// THEN we see that the metrics of Jaeger collector are present in prometheus
	whenJaegerOperatorEnabledIt("metrics of jaeger collector are available in prometheus", func() {
		validatorFn := pkg.ValidateJaegerCollectorMetricFunc(metricsTest)
		Eventually(validatorFn).WithPolling(shortPollingInterval).WithTimeout(shortWaitTimeout).Should(BeTrue())
	})

	// GIVEN a VZ custom resource in dev profile,
	// WHEN Jaeger operator is enabled,
	// THEN we see that the metrics of Jaeger query are present in prometheus
	whenJaegerOperatorEnabledIt("metrics of jaeger query are available in prometheus", func() {
		validatorFn := pkg.ValidateJaegerQueryMetricFunc(metricsTest)
		Eventually(validatorFn).WithPolling(shortPollingInterval).WithTimeout(shortWaitTimeout).Should(BeTrue())
	})

	// GIVEN a VZ custom resource in dev profile,
	// WHEN Jaeger operator is enabled,
	// THEN we see that the metrics of Jaeger agent are present in prometheus
	whenJaegerOperatorEnabledIt("metrics of jaeger agent are available in prometheus", func() {
		validatorFn := pkg.ValidateJaegerAgentMetricFunc(metricsTest)
		Eventually(validatorFn).WithPolling(shortPollingInterval).WithTimeout(shortWaitTimeout).Should(BeTrue())
	})

	// GIVEN a VZ custom resource in dev profile with Jaeger operator enabled,
	// WHEN user tries to disable it,
	// THEN the operation should be denied with an error
	whenJaegerOperatorEnabledIt("disabling previously enabled Jaeger operator should be disallowed", func() {
		m := JaegerOperatorCleanupModifier{}
		Eventually(func() bool {
			err := update.UpdateCR(m)
			foundExpectedErr := err != nil && strings.Contains(err.Error(), disableErrorMsg)
			return foundExpectedErr
		}).WithPolling(pollingInterval).WithTimeout(waitTimeout).Should(BeTrue())
	})
})
