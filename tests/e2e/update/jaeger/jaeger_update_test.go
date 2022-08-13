// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package jaeger

import (
	"fmt"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/verrazzano/verrazzano/pkg/k8sutil"
	"github.com/verrazzano/verrazzano/platform-operator/constants"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg"
	"github.com/verrazzano/verrazzano/tests/e2e/update"
	"strings"
	"time"
)

const (
	shortPollingInterval = 10 * time.Second
	shortWaitTimeout     = 5 * time.Minute
	disableErrorMsg      = "disabling component jaegerOperator is not allowed"
)

const (
	jaegerOperatorSampleMetric  = "jaeger_operator_instances_managed"
	jaegerAgentSampleMetric     = "jaeger_agent_collector_proxy_total"
	jaegerQuerySampleMetric     = "jaeger_query_requests_total"
	jaegerCollectorSampleMetric = "jaeger_collector_queue_capacity"
	jaegerESIndexCleanerJob     = "jaeger-operator-jaeger-es-index-cleaner"
)

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
	WhenJaegerOperatorEnabledIt("should have a Jaeger OpenSearch Index Cleaner cron job", func() {
		Eventually(func() (bool, error) {
			create, err := pkg.IsJaegerInstanceCreated()
			if err != nil {
				pkg.Log(pkg.Error, fmt.Sprintf("Error checking if Jaeger CR is available %s", err.Error()))
			}
			if create {
				return pkg.DoesCronJobExist(constants.VerrazzanoMonitoringNamespace, jaegerESIndexCleanerJob)
			}
			return false, nil
		}, waitTimeout, pollingInterval).Should(BeTrue())
	})

	// GIVEN a VZ custom resource in dev profile,
	// WHEN Jaeger operator is enabled,
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
				pkg.Log(pkg.Info, fmt.Sprintf("Inspecting traces for service: %s", serviceName))
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

	// GIVEN a VZ custom resource in dev profile,
	// WHEN Jaeger operator is enabled,
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

	// GIVEN a VZ custom resource in dev profile,
	// WHEN Jaeger operator is enabled,
	// THEN we see that the metrics of Jaeger operator are present in prometheus
	WhenJaegerOperatorEnabledIt("metrics of jaeger operator are available in prometheus", func() {
		Eventually(func() bool {
			kubeconfigPath, err := k8sutil.GetKubeConfigLocation()
			if err != nil {
				return false
			}
			return pkg.IsJaegerMetricFound(kubeconfigPath, jaegerOperatorSampleMetric, nil)
		}).WithPolling(shortPollingInterval).WithTimeout(shortWaitTimeout).Should(BeTrue())

	})

	// GIVEN a VZ custom resource in dev profile,
	// WHEN Jaeger operator is enabled,
	// THEN we see that the metrics of Jaeger Components (jaeger-query, jaeger-collector, jaeger-agent) are present in prometheus
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

	// GIVEN a VZ custom resource in dev profile with Jaeger operator enabled,
	// WHEN user tries to disable it,
	// THEN the operation should be denied with an error
	WhenJaegerOperatorEnabledIt("disabling previously enabled Jaeger operator should be disallowed", func() {
		Expect(func() bool {
			m := JaegerOperatorCleanupModifier{}
			err := update.UpdateCR(m)
			return err != nil && strings.Contains(err.Error(), disableErrorMsg)
		}).Should(BeTrue())
	})
})
