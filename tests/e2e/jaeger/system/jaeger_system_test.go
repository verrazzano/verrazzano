// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package system

import (
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/verrazzano/verrazzano/pkg/k8sutil"
	"github.com/verrazzano/verrazzano/pkg/test/framework"
	"github.com/verrazzano/verrazzano/tests/e2e/jaeger"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg"
)

const (
	shortPollingInterval = 10 * time.Second
	shortWaitTimeout     = 5 * time.Minute
	longPollingInterval  = 30 * time.Second
	longWaitTimeout      = 15 * time.Minute
)

var (
	t     = framework.NewTestFramework("jaeger")
	start = time.Now()
)

var _ = t.Describe("Verrazzano System traces with Jaeger", Label("f:jaeger.system-traces"), func() {
	t.Context("after successful installation", func() {

		// GIVEN the Jaeger Operator is enabled and istio tracing is enabled,
		// WHEN we query for traces from verrazzano system components,
		// THEN we are able to get the traces
		jaeger.WhenJaegerOperatorEnabledIt(t, "traces from verrazzano system components should be available when queried from Jaeger", func() {
			kubeconfigPath, err := k8sutil.GetKubeConfigLocation()
			if err != nil {
				Fail(err.Error())
			}
			validatorFn := pkg.ValidateSystemTracesFuncInCluster(kubeconfigPath, start, "local")
			Eventually(validatorFn).WithPolling(longPollingInterval).WithTimeout(longWaitTimeout).Should(BeTrue())
		})

		// GIVEN the Jaeger Operator is enabled and istio tracing is enabled,
		// WHEN we check for traces from verrazzano system components in Opensearch Storage,
		// THEN we are able to get the traces
		jaeger.WhenJaegerOperatorEnabledIt(t, "traces from verrazzano system components should be available in the OS backend storage.", func() {
			validatorFn := pkg.ValidateSystemTracesInOSFunc(start)
			Eventually(validatorFn).WithPolling(longPollingInterval).WithTimeout(longWaitTimeout).Should(BeTrue())
		})

		// GIVEN the Jaeger Operator component is enabled,
		// WHEN we query for metrics related to Jaeger operator
		// THEN we see that the metrics are present in prometheus
		jaeger.WhenJaegerOperatorEnabledIt(t, "metrics of jaeger operator are available in prometheus", func() {
			validatorFn := pkg.ValidateJaegerOperatorMetricFunc()
			Eventually(validatorFn).WithPolling(shortPollingInterval).WithTimeout(shortWaitTimeout).Should(BeTrue())
		})

		// GIVEN the Jaeger Operator component is installed with default Jaeger CR enabled
		// WHEN we query for metrics related to Jaeger collector
		// THEN we see that the metrics are present in prometheus
		jaeger.WhenJaegerOperatorEnabledIt(t, "metrics of jaeger collector are available in prometheus", func() {
			validatorFn := pkg.ValidateJaegerCollectorMetricFunc()
			Eventually(validatorFn).WithPolling(shortPollingInterval).WithTimeout(shortWaitTimeout).Should(BeTrue())
		})

		// GIVEN the Jaeger Operator component is installed with default Jaeger CR enabled
		// WHEN we query for metrics related to Jaeger collector
		// THEN we see that the metrics are present in prometheus
		jaeger.WhenJaegerOperatorEnabledIt(t, "metrics of jaeger query are available in prometheus", func() {
			validatorFn := pkg.ValidateJaegerQueryMetricFunc()
			Eventually(validatorFn).WithPolling(shortPollingInterval).WithTimeout(shortWaitTimeout).Should(BeTrue())
		})

		// GIVEN the Jaeger Operator component is installed with default Jaeger CR enabled
		// WHEN we query for metrics related to Jaeger collector
		// THEN we see that the metrics are present in prometheus
		jaeger.WhenJaegerOperatorEnabledIt(t, "metrics of jaeger agent are available in prometheus", func() {
			validatorFn := pkg.ValidateJaegerAgentMetricFunc()
			Eventually(validatorFn).WithPolling(shortPollingInterval).WithTimeout(shortWaitTimeout).Should(BeTrue())
		})
	})

})
