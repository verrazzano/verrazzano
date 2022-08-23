// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package jaeger

import (
	"time"

	"github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
	"github.com/verrazzano/verrazzano/pkg/k8sutil"
	"github.com/verrazzano/verrazzano/pkg/test/framework"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
)

const (
	shortPollingInterval     = 10 * time.Second
	shortWaitTimeout         = 5 * time.Minute
	imagePullWaitTimeout     = 40 * time.Minute
	imagePullPollingInterval = 30 * time.Second
	waitTimeout              = 10 * time.Minute
	pollingInterval          = 30 * time.Second
)

func WhenJaegerOperatorEnabledIt(t *framework.TestFramework, text string, args ...interface{}) {
	kubeconfig, err := k8sutil.GetKubeConfigLocation()
	if err != nil {
		t.It(text, func() {
			ginkgo.Fail(err.Error())
		})
	}
	if pkg.IsJaegerOperatorEnabled(kubeconfig) {
		t.ItMinimumVersion(text, "1.3.0", kubeconfig, args...)
	}
	t.Logs.Infof("Skipping spec, Jaeger Operator is disabled")
}

func DeployApplication(namespace, testAppComponentFilePath, testAppConfigurationFilePath string, expectedPods []string) {
	if !IsJaegerOperatorEnabled() {
		pkg.Log(pkg.Info, "Skipping Deploy as Jaeger operator component is disabled")
		return
	}
	gomega.Eventually(func() (*v1.Namespace, error) {
		nsLabels := map[string]string{
			"verrazzano-managed": "true",
			"istio-injection":    "enabled"}
		return pkg.CreateNamespace(namespace, nsLabels)
	}).WithPolling(shortPollingInterval).WithTimeout(shortWaitTimeout).ShouldNot(gomega.BeNil())

	gomega.Eventually(func() error {
		return pkg.CreateOrUpdateResourceFromFileInGeneratedNamespace(testAppComponentFilePath, namespace)
	}).WithPolling(shortPollingInterval).WithTimeout(shortWaitTimeout).ShouldNot(gomega.HaveOccurred())

	gomega.Eventually(func() error {
		return pkg.CreateOrUpdateResourceFromFileInGeneratedNamespace(testAppConfigurationFilePath, namespace)
	}).WithPolling(shortPollingInterval).WithTimeout(shortWaitTimeout).ShouldNot(gomega.HaveOccurred())

	gomega.Eventually(func() bool {
		return pkg.ContainerImagePullWait(namespace, expectedPods)
	}).WithPolling(imagePullPollingInterval).WithTimeout(imagePullWaitTimeout).Should(gomega.BeTrue())

	// Verify pods are running
	gomega.Eventually(func() bool {
		result, err := pkg.PodsRunning(namespace, expectedPods)
		if err != nil {
			return false
		}
		return result
	}).WithPolling(pollingInterval).WithTimeout(waitTimeout).Should(gomega.BeTrue())
}

func UndeployApplication(namespace, testAppComponentFilePath, testAppConfigurationFilePath string, expectedPods []string) {
	if !IsJaegerOperatorEnabled() {
		pkg.Log(pkg.Info, "Skipping Undeploy as Jaeger operator component is disabled")
		return
	}

	gomega.Eventually(func() error {
		return pkg.DeleteResourceFromFileInGeneratedNamespace(testAppComponentFilePath, namespace)
	}).WithPolling(shortPollingInterval).WithTimeout(shortWaitTimeout).ShouldNot(gomega.HaveOccurred())

	gomega.Eventually(func() error {
		return pkg.DeleteResourceFromFileInGeneratedNamespace(testAppConfigurationFilePath, namespace)
	}).WithPolling(shortPollingInterval).WithTimeout(shortWaitTimeout).ShouldNot(gomega.HaveOccurred())

	gomega.Eventually(func() bool {
		podsTerminated, _ := pkg.PodsNotRunning(namespace, expectedPods)
		return podsTerminated
	}).WithPolling(shortPollingInterval).WithTimeout(shortWaitTimeout).Should(gomega.BeTrue())

	gomega.Eventually(func() error {
		return pkg.DeleteNamespace(namespace)
	}).WithPolling(shortPollingInterval).WithTimeout(shortWaitTimeout).ShouldNot(gomega.HaveOccurred())

	gomega.Eventually(func() bool {
		return pkg.CheckNamespaceFinalizerRemoved(namespace)
	}).WithPolling(shortPollingInterval).WithTimeout(shortWaitTimeout).Should(gomega.BeTrue())

	gomega.Eventually(func() bool {
		_, err := pkg.GetNamespace(namespace)
		return err != nil && errors.IsNotFound(err)
	}).WithPolling(shortPollingInterval).WithTimeout(shortWaitTimeout).Should(gomega.BeTrue())

}

func IsJaegerOperatorEnabled() bool {
	kubeconfig, err := k8sutil.GetKubeConfigLocation()
	if err != nil {
		ginkgo.Fail(err.Error())
	}
	return pkg.IsJaegerOperatorEnabled(kubeconfig)
}
