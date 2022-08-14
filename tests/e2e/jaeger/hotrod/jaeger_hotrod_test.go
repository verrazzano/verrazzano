// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package hotrod

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
	testAppComponentFilePath     = "testdata/jaeger/hotrod/hotrod-tracing-comp.yaml"
	testAppConfigurationFilePath = "testdata/jaeger/hotrod/hotrod-tracing-app.yaml"
)

var (
	t                  = framework.NewTestFramework("jaeger")
	generatedNamespace = pkg.GenerateNamespace("hotrod-tracing")
	expectedPodsHotrod = []string{"hotrod-workload"}
	waitTimeout        = 10 * time.Minute
	pollingInterval    = 30 * time.Second
	failed             = false
	beforeSuitePassed  = false
	start              = time.Now()
	hotrodServiceName  = fmt.Sprintf("hotrod.%s", generatedNamespace)
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

var _ = t.BeforeSuite(func() {
	kubeconfig, err := k8sutil.GetKubeConfigLocation()
	if err != nil {
		Fail(err.Error())
	}
	if !pkg.IsJaegerOperatorEnabled(kubeconfig) {
		pkg.Log(pkg.Info, "Skipping BeforeSuite as Jaeger Operator is disabled.")
		return
	}

	start = time.Now()
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
		return pkg.ContainerImagePullWait(namespace, expectedPodsHotrod)
	}).WithPolling(imagePullPollingInterval).WithTimeout(imagePullWaitTimeout).Should(BeTrue())

	// Verify hotrod-workload pod is running
	Eventually(hotrodPodsRunning(), waitTimeout, pollingInterval).Should(BeTrue())
	beforeSuitePassed = true
	metrics.Emit(t.Metrics.With("deployment_elapsed_time", time.Since(start).Milliseconds()))
})

var _ = t.AfterSuite(func() {
	kubeconfig, err := k8sutil.GetKubeConfigLocation()
	if err != nil {
		Fail(err.Error())
	}
	if !pkg.IsJaegerOperatorEnabled(kubeconfig) {
		pkg.Log(pkg.Info, "Skipping BeforeSuite as Jaeger Operator is disabled.")
		return
	}
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
		podsTerminated, _ := pkg.PodsNotRunning(namespace, expectedPodsHotrod)
		return podsTerminated
	}).WithPolling(shortPollingInterval).WithTimeout(shortWaitTimeout).Should(BeTrue())

	t.Logs.Info("Delete namespace")
	Eventually(func() error {
		return pkg.DeleteNamespace(namespace)
	}).WithPolling(shortPollingInterval).WithTimeout(shortWaitTimeout).ShouldNot(HaveOccurred())

	t.Logs.Info("Wait for Finalizer to be removed")
	Eventually(func() bool {
		return pkg.CheckNamespaceFinalizerRemoved(namespace)
	}).WithPolling(shortPollingInterval).WithTimeout(shortWaitTimeout).Should(BeTrue())

	t.Logs.Info("Wait for namespace to be deleted")
	Eventually(func() bool {
		_, err := pkg.GetNamespace(namespace)
		return err != nil && errors.IsNotFound(err)
	}).WithPolling(shortPollingInterval).WithTimeout(shortWaitTimeout).Should(BeTrue())

	metrics.Emit(t.Metrics.With("undeployment_elapsed_time", time.Since(start).Milliseconds()))
})

var _ = t.Describe("Hotrod App with Jaeger Traces", Label("f:jaeger.hotrod-workload"), func() {
	t.Context("after successful installation", func() {
		// GIVEN the Jaeger Operator is enabled and a sample application is installed,
		// WHEN we check for traces for that service,
		// THEN we are able to get the traces
		WhenJaegerOperatorEnabledIt("traces for the hotrod app should be available when queried from Jaeger", func() {
			Eventually(func() (bool, error) {
				// Check if the service name is registered in Jaeger and traces are present for that service
				kubeconfigPath, err := k8sutil.GetKubeConfigLocation()
				if err != nil {
					return false, err
				}
				tracesFound := false
				servicesWithJaegerTraces := pkg.ListServicesInJaeger(kubeconfigPath)
				for _, serviceName := range servicesWithJaegerTraces {
					if strings.HasPrefix(serviceName, hotrodServiceName) {
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

		// GIVEN the Jaeger Operator component is enabled,
		// WHEN a sample application is installed,
		// THEN the traces are found in OpenSearch Backend
		WhenJaegerOperatorEnabledIt("traces for the hotrod app should be available in the OS backend storage.", func() {
			Eventually(func() (bool, error) {
				kubeconfigPath, err := k8sutil.GetKubeConfigLocation()
				if err != nil {
					return false, err
				}
				return pkg.JaegerSpanRecordFoundInOpenSearch(kubeconfigPath, start, hotrodServiceName)
			}).WithPolling(shortPollingInterval).WithTimeout(shortWaitTimeout).Should(BeTrue())
		})
	})

})

//hotrodPodsRunning checks if the hotrod pods are running
func hotrodPodsRunning() bool {
	result, err := pkg.PodsRunning(namespace, expectedPodsHotrod)
	if err != nil {
		AbortSuite(fmt.Sprintf("One or more pods are not running in the namespace: %v, error: %v", namespace, err))
	}
	return result
}
