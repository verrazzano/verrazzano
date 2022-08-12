// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package system

import (
	"fmt"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/verrazzano/verrazzano/pkg/k8sutil"
	"github.com/verrazzano/verrazzano/pkg/test/framework"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg"
	"strings"
	"time"
)

const (
	shortPollingInterval     = 10 * time.Second
	shortWaitTimeout         = 5 * time.Minute
)

const (
	jaegerOperatorSampleMetric   = "jaeger_operator_instances_managed"
	jaegerAgentSampleMetric      = "jaeger_agent_collector_proxy_total"
	jaegerQuerySampleMetric      = "jaeger_query_requests_total"
	jaegerCollectorSampleMetric  = "jaeger_collector_queue_capacity"
)

var (
	t                        = framework.NewTestFramework("jaeger")
	waitTimeout              = 10 * time.Minute
	pollingInterval          = 30 * time.Second
	failed                   = false
	beforeSuitePassed        = false
	start                    = time.Now()
)

func WhenJaegerOperatorEnabledIt(text string, args ...interface{}) {
	kubeconfig, err := k8sutil.GetKubeConfigLocation()
	if err != nil {
		t.It(text, func() {
			Fail(err.Error())
		})
	}
	if pkg.IsJaegerOperatorEnabled(kubeconfig) {
		t.ItMinimumVersion(text, "1.3.0", kubeconfig, args...)
	}
	t.Logs.Infof("Skipping spec, Jaeger Operator is disabled")
}



var _ = t.Describe("Verrazzano System traces with Jaeger", Label("f:jaeger.system-traces"), func() {
	t.Context("after successful installation", func() {

		// GIVEN the Jaeger Operator is enabled and istio tracing is enabled,
		// WHEN we query for traces from verrazzano system components,
		// THEN we are able to get the traces
		WhenJaegerOperatorEnabledIt("traces for the fluentd system service should be available when queried from Jaeger", func() {
			Eventually(func() (bool, error) {
				// Check if the service name is registered in Jaeger and traces are present for that service
				kubeconfigPath, err := k8sutil.GetKubeConfigLocation()
				if err != nil {
					return false, err
				}
				tracesFound := false
				servicesWithJaegerTraces := pkg.ListServicesInJaeger(kubeconfigPath)
				for _, serviceName := range servicesWithJaegerTraces {
					pkg.Log(pkg.Info, "Inspecting Service Name: " + serviceName)
					if strings.HasPrefix(serviceName, "fluentd.verrazzano-system") {
						traceIds := pkg.ListJaegerTraces(kubeconfigPath, serviceName)
						tracesFound = len(traceIds) > 0
						if !tracesFound {
							pkg.Log(pkg.Error, fmt.Sprintf("traces not found for service: %s", serviceName))
							return false, fmt.Errorf("traces not found for service: %s", serviceName)
						}
						break
					}
				}
				return tracesFound, nil
			}).WithPolling(shortPollingInterval).WithTimeout(shortWaitTimeout).Should(BeTrue())
		})

		// GIVEN the Jaeger Operator is enabled and istio tracing is enabled,
		// WHEN we check for traces from verrazzano system components in Opensearch Storage,
		// THEN we are able to get the traces
		WhenJaegerOperatorEnabledIt("traces for the fluentd system service should be available in the OS backend storage.", func() {
			Eventually(func() (bool, error) {
				kubeconfigPath, err := k8sutil.GetKubeConfigLocation()
				if err != nil {
					return false, err
				}
				return pkg.JaegerSpanRecordFoundInOpenSearch(kubeconfigPath, start, "fluentd.verrazzano-system")
			}).WithPolling(shortPollingInterval).WithTimeout(shortWaitTimeout).Should(BeTrue())
		})


		// GIVEN the Jaeger Operator component is enabled,
		// WHEN we check for metrics related to Jaeger operator
		// THEN we see that the metrics are present in prometheus
		WhenJaegerOperatorEnabledIt("metrics of jaeger operator are available in prometheus", func() {
			Eventually(func() bool {
				kubeconfigPath, err := k8sutil.GetKubeConfigLocation()
				if err != nil {
					return false
				}
				return pkg.IsJaegerMetricFound(kubeconfigPath, jaegerOperatorSampleMetric, nil)
			}).WithPolling(shortPollingInterval).WithTimeout(shortWaitTimeout).Should(BeTrue())

		})

		// GIVEN the Jaeger Operator component is installed with default Jaeger CR enabled
		// WHEN we check for metrics related to Jaeger Components (jaeger-query, jaeger-collector, jaeger-agent)
		// THEN we see that the metrics are present in prometheus
		WhenJaegerOperatorEnabledIt("metrics of jaeger components are available in prometheus", func() {
			Eventually(func() bool {
				kubeconfigPath, err := k8sutil.GetKubeConfigLocation()
				if err != nil {
					return false
				}
				return pkg.IsJaegerMetricFound(kubeconfigPath, jaegerCollectorSampleMetric, nil) &&
					pkg.IsJaegerMetricFound(kubeconfigPath, jaegerQuerySampleMetric, nil) &&
					pkg.IsJaegerMetricFound(kubeconfigPath, jaegerAgentSampleMetric, nil)
			}).WithPolling(shortPollingInterval).WithTimeout(shortWaitTimeout).Should(BeTrue())
		})
	})

})
