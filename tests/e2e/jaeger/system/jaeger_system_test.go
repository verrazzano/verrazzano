// Copyright (c) 2022, 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package system

import (
	"fmt"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/verrazzano/verrazzano/pkg/k8sutil"
	"github.com/verrazzano/verrazzano/tests/e2e/jaeger"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg/test/framework"
)

const (
	shortPollingInterval = 10 * time.Second
	shortWaitTimeout     = 5 * time.Minute
	longPollingInterval  = 30 * time.Second
	longWaitTimeout      = 15 * time.Minute
)

var (
	t = framework.NewTestFramework("jaeger-system-traces")
	// Allow 3 hour allowance in start time to find the system traces faster
	start = time.Now().Add(-3 * time.Hour)

	kubeconfigPath string
	metricsTest    pkg.MetricsTest
)

var beforeSuite = t.BeforeSuiteFunc(func() {
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

var whenJaegerOperatorEnabledIt = t.WhenMeetsConditionFunc(jaeger.OperatorCondition, jaeger.IsJaegerEnabled)

var _ = t.Describe("Verrazzano System traces with Jaeger", Label("f:jaeger.system-traces"), func() {
	t.Context("after successful installation", func() {

		// GIVEN the Jaeger Operator is enabled and istio tracing is enabled,
		// WHEN we query for traces from verrazzano system components,
		// THEN we are able to get the traces
		whenJaegerOperatorEnabledIt("traces from verrazzano system components should be available when queried from Jaeger", func() {
			validatorFn := pkg.ValidateSystemTracesFuncInCluster(kubeconfigPath, start, "local")
			Eventually(validatorFn).WithPolling(longPollingInterval).WithTimeout(longWaitTimeout).Should(BeTrue())
		})

		// GIVEN the Jaeger Operator is enabled and istio tracing is enabled,
		// WHEN we check for traces from verrazzano system components in Opensearch Storage,
		// THEN we are able to get the traces
		whenJaegerOperatorEnabledIt("traces from verrazzano system components should be available in the OS backend storage.", func() {
			validatorFn := pkg.ValidateSystemTracesInOSFunc(start)
			Eventually(validatorFn).WithPolling(longPollingInterval).WithTimeout(longWaitTimeout).Should(BeTrue())
		})

		// GIVEN the Jaeger Operator component is enabled,
		// WHEN we query for metrics related to Jaeger operator
		// THEN we see that the metrics are present in prometheus
		whenJaegerOperatorEnabledIt("metrics of jaeger operator are available in prometheus", func() {
			validatorFn := pkg.ValidateJaegerOperatorMetricFunc(metricsTest)
			Eventually(validatorFn).WithPolling(shortPollingInterval).WithTimeout(shortWaitTimeout).Should(BeTrue())
		})

		// GIVEN the Jaeger Operator component is installed with default Jaeger CR enabled
		// WHEN we query for metrics related to Jaeger collector
		// THEN we see that the metrics are present in prometheus
		whenJaegerOperatorEnabledIt("metrics of jaeger collector are available in prometheus", func() {
			validatorFn := pkg.ValidateJaegerCollectorMetricFunc(metricsTest)
			Eventually(validatorFn).WithPolling(shortPollingInterval).WithTimeout(shortWaitTimeout).Should(BeTrue())
		})

		// GIVEN the Jaeger Operator component is installed with default Jaeger CR enabled
		// WHEN we query for metrics related to Jaeger collector
		// THEN we see that the metrics are present in prometheus
		whenJaegerOperatorEnabledIt("metrics of jaeger query are available in prometheus", func() {
			validatorFn := pkg.ValidateJaegerQueryMetricFunc(metricsTest)
			Eventually(validatorFn).WithPolling(shortPollingInterval).WithTimeout(shortWaitTimeout).Should(BeTrue())
		})

		// GIVEN the Jaeger Operator component is installed with default Jaeger CR enabled
		// WHEN we query for metrics related to Jaeger collector
		// THEN we see that the metrics are present in prometheus
		whenJaegerOperatorEnabledIt("metrics of jaeger agent are available in prometheus", func() {
			validatorFn := pkg.ValidateJaegerAgentMetricFunc(metricsTest)
			Eventually(validatorFn).WithPolling(shortPollingInterval).WithTimeout(shortWaitTimeout).Should(BeTrue())
		})
	})

})
