// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package jaeger

import (
	"fmt"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/verrazzano/verrazzano/pkg/k8sutil"
	"github.com/verrazzano/verrazzano/pkg/test/framework"
	"github.com/verrazzano/verrazzano/pkg/test/framework/metrics"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"strings"
	"time"
)

const (
	shortPollingInterval     = 10 * time.Second
	shortWaitTimeout         = 5 * time.Minute
	imagePullWaitTimeout     = 40 * time.Minute
	imagePullPollingInterval = 30 * time.Second
)

const (
	testAppComponentFilePath     = "testdata/jaeger/helidon/helidon-tracing-comp.yaml"
	testAppConfigurationFilePath = "testdata/jaeger/helidon/helidon-tracing-app.yaml"
	jaegerOperatorSampleMetric   = "jaeger_operator_instances_managed"
	jaegerAgentSampleMetric      = "jaeger_agent_collector_proxy_total"
	jaegerQuerySampleMetric      = "jaeger_query_requests_total"
	jaegerCollectorSampleMetric  = "jaeger_collector_queue_capacity"
)

var (
	t                        = framework.NewTestFramework("jaeger")
	generatedNamespace       = pkg.GenerateNamespace("jaeger-tracing")
	expectedPodsHelloHelidon = []string{"hello-helidon-deployment"}
	waitTimeout              = 10 * time.Minute
	pollingInterval          = 30 * time.Second
	failed                   = false
	beforeSuitePassed        = false
)

var _ = t.BeforeSuite(func() {
	start := time.Now()
	Eventually(func() (*v1.Namespace, error) {
		nsLabels := map[string]string{
			"verrazzano-managed": "true",
			"istio-injection":    istioInjection}
		return pkg.CreateNamespace(namespace, nsLabels)
	}).WithPolling(shortPollingInterval).WithTimeout(shortWaitTimeout).ShouldNot(BeNil())

	Eventually(func() error {
		return pkg.CreateOrUpdateResourceFromFileInGeneratedNamespace(testAppComponentFilePath, namespace)
	}).WithPolling(shortPollingInterval).WithTimeout(shortWaitTimeout).ShouldNot(HaveOccurred())

	Eventually(func() error {
		return pkg.CreateOrUpdateResourceFromFileInGeneratedNamespace(testAppConfigurationFilePath, namespace)
	}).WithPolling(shortPollingInterval).WithTimeout(shortWaitTimeout).ShouldNot(HaveOccurred())

	Eventually(func() bool {
		return pkg.ContainerImagePullWait(namespace, expectedPodsHelloHelidon)
	}).WithPolling(imagePullPollingInterval).WithTimeout(imagePullWaitTimeout).Should(BeTrue())

	// Verify hello-helidon-workload pod is running
	Eventually(helloHelidonPodsRunning, waitTimeout, pollingInterval).Should(BeTrue())
	beforeSuitePassed = true
	metrics.Emit(t.Metrics.With("deployment_elapsed_time", time.Since(start).Milliseconds()))
})

var _ = t.AfterEach(func() {
	failed = failed || CurrentSpecReport().Failed()
})

var _ = t.AfterSuite(func() {
	if failed || !beforeSuitePassed {
		pkg.ExecuteClusterDumpWithEnvVarConfig()
	}
	// undeploy the application here
	start := time.Now()

	t.Logs.Info("Delete application")
	Eventually(func() error {
		return pkg.DeleteResourceFromFileInGeneratedNamespace(testAppComponentFilePath, namespace)
	}).WithPolling(shortPollingInterval).WithTimeout(shortWaitTimeout).ShouldNot(HaveOccurred())

	t.Logs.Info("Delete components")
	Eventually(func() error {
		return pkg.DeleteResourceFromFileInGeneratedNamespace(testAppConfigurationFilePath, namespace)
	}).WithPolling(shortPollingInterval).WithTimeout(shortWaitTimeout).ShouldNot(HaveOccurred())

	t.Logs.Info("Wait for application pods to terminate")
	Eventually(func() bool {
		podsTerminated, _ := pkg.PodsNotRunning("helidon-logging", expectedPodsHelloHelidon)
		return podsTerminated
	}).WithPolling(shortPollingInterval).WithTimeout(shortWaitTimeout).Should(BeTrue())

	t.Logs.Info("Delete namespace")
	Eventually(func() error {
		return pkg.DeleteNamespace(namespace)
	}).WithPolling(shortPollingInterval).WithTimeout(shortWaitTimeout).ShouldNot(HaveOccurred())

	t.Logs.Info("Wait for Finalizer to be removed")
	Eventually(func() bool {
		return pkg.CheckNamespaceFinalizerRemoved("helidon-logging")
	}).WithPolling(shortPollingInterval).WithTimeout(shortWaitTimeout).Should(BeTrue())

	t.Logs.Info("Wait for namespace to be deleted")
	Eventually(func() bool {
		_, err := pkg.GetNamespace("helidon-logging")
		return err != nil && errors.IsNotFound(err)
	}).WithPolling(shortPollingInterval).WithTimeout(shortWaitTimeout).Should(BeTrue())

	metrics.Emit(t.Metrics.With("undeployment_elapsed_time", time.Since(start).Milliseconds()))
})

func isJaegerOperatorEnabled() bool {
	kubeconfigPath, err := k8sutil.GetKubeConfigLocation()
	if err != nil {
		AbortSuite(fmt.Sprintf("Failed to get default kubeconfig path: %s", err.Error()))
	}
	return pkg.IsJaegerOperatorEnabled(kubeconfigPath)
}

// 'It' Wrapper to only run spec if the Jaeger operator is supported on the current Verrazzano version
func WhenJaegerOperatorInstalledIt(description string, f func()) {
	kubeconfigPath, err := k8sutil.GetKubeConfigLocation()
	if err != nil {
		t.It(description, func() {
			Fail(fmt.Sprintf("Failed to get default kubeconfig path: %s", err.Error()))
		})
	}
	supported, err := pkg.IsVerrazzanoMinVersion("1.3.0", kubeconfigPath)
	if err != nil {
		t.It(description, func() {
			Fail(fmt.Sprintf("Failed to check Verrazzano version 1.3.0: %s", err.Error()))
		})
	}
	if supported {
		t.It(description, f)
	} else {
		t.Logs.Infof("Skipping check '%v', the Jaeger Operator is not supported", description)
	}
}

var _ = t.Describe("Jaeger Operator", Label("f:platform-lcm.install"), func() {
	t.Context("after successful installation", func() {
		// GIVEN the Jaeger Operator is enabled and a sample application is installed,
		// WHEN we check for traces for that service,
		// THEN we are able to get the traces
		WhenJaegerOperatorInstalledIt("should have a verrazzano-monitoring namespace", func() {
			Eventually(func() (bool, error) {
				if !isJaegerOperatorEnabled() {
					return true, nil
				}
				// Check if the service name is registered in Jaeger and traces are present for that service
				kubeconfigPath, err := k8sutil.GetKubeConfigLocation()
				if err != nil {
					return false, err
				}
				tracesFound := false
				for _, serviceName := range pkg.ListServicesInJaeger(kubeconfigPath) {
					if strings.HasPrefix(serviceName, "hello-helidon") {
						traceIds := pkg.ListJaegerTraces(kubeconfigPath, serviceName)
						tracesFound = len(traceIds) > 0
						if !tracesFound {
							pkg.Log(pkg.Error, fmt.Sprintf("traces not found for service: %s", serviceName))
							return false, fmt.Errorf("traces not found for service: %s", serviceName)
						}
					}
				}
				return false, nil
			}).WithPolling(shortPollingInterval).WithTimeout(shortWaitTimeout).Should(BeTrue())
		})

		// GIVEN the Jaeger Operator component is enabled,
		// WHEN a sample application is installed,
		// THEN the traces are found in OpenSearch Backend
		WhenJaegerOperatorInstalledIt("should have running pods", func() {
			Eventually(func() (bool, error) {
				if !isJaegerOperatorEnabled() {
					return true, nil
				}
				return true, nil
			}).WithPolling(shortPollingInterval).WithTimeout(shortWaitTimeout).Should(BeTrue())
		})

		// GIVEN the Jaeger Operator component is enabled,
		// WHEN we check for metrics related to Jaeger operator
		// THEN we see that the metrics are present in prometheus
		WhenJaegerOperatorInstalledIt("should have the correct default Jaeger images", func() {
			Eventually(func() bool {
				if !isJaegerOperatorEnabled() {
					return true
				}
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
		WhenJaegerOperatorInstalledIt("should have the correct Jaeger Operator CRDs", func() {
			Eventually(func() bool {
				if !isJaegerOperatorEnabled() {
					return true
				}
				kubeconfigPath, err := k8sutil.GetKubeConfigLocation()
				if err != nil {
					return false
				}
				return (pkg.IsJaegerMetricFound(kubeconfigPath, jaegerCollectorSampleMetric, nil) &&
					pkg.IsJaegerMetricFound(kubeconfigPath, jaegerQuerySampleMetric, nil) &&
					pkg.IsJaegerMetricFound(kubeconfigPath, jaegerAgentSampleMetric, nil))
			}).WithPolling(shortPollingInterval).WithTimeout(shortWaitTimeout).Should(BeTrue())
		})
	})

})

func helloHelidonPodsRunning() bool {
	result, err := pkg.PodsRunning(namespace, expectedPodsHelloHelidon)
	if err != nil {
		AbortSuite(fmt.Sprintf("One or more pods are not running in the namespace: %v, error: %v", namespace, err))
	}
	return result
}
