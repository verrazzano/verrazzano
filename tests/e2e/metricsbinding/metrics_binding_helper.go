// Copyright (c) 2021, 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package metricsbinding

import (
	"fmt"
	"time"

	"github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
	"github.com/verrazzano/verrazzano/pkg/test/framework"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
)

const (
	shortWaitTimeout     = 10 * time.Minute
	shortPollingInterval = 10 * time.Second
	longWaitTimeout      = 15 * time.Minute
	longPollingInterval  = 20 * time.Second
)

func DeployApplication(namespace, yamlPath, podPrefix string, istioEnabled bool, t framework.TestFramework) {
	t.Logs.Info("Deploy test application")

	t.Logs.Info("Create namespace")
	gomega.Eventually(func() (*v1.Namespace, error) {
		nsLabels := map[string]string{"verrazzano-managed": "true"}
		if istioEnabled {
			nsLabels["istio-injection"] = "enabled"
		}
		return pkg.CreateNamespace(namespace, nsLabels)
	}, shortWaitTimeout, shortPollingInterval).ShouldNot(gomega.BeNil())

	t.Logs.Info("Create helidon resources")
	gomega.Eventually(func() error {
		return pkg.CreateOrUpdateResourceFromFileInGeneratedNamespace(yamlPath, namespace)
	}, shortWaitTimeout, shortPollingInterval).ShouldNot(gomega.HaveOccurred())

	t.Logs.Info("Check application pods are running")
	gomega.Eventually(func() bool {
		result, err := pkg.PodsRunning(namespace, []string{podPrefix})
		if err != nil {
			ginkgo.AbortSuite(fmt.Sprintf("One or more pods are not running in the namespace: %v, error: %v", namespace, err))
		}
		return result
	}, longWaitTimeout, longPollingInterval).Should(gomega.BeTrue())
}

func UndeployApplication(namespace string, yamlPath string, promConfigJobName string, t framework.TestFramework) {
	t.Logs.Info("Delete application")
	gomega.Eventually(func() error {
		return pkg.DeleteResourceFromFileInGeneratedNamespace(yamlPath, namespace)
	}, shortWaitTimeout, shortPollingInterval).ShouldNot(gomega.HaveOccurred())

	t.Logs.Info("Remove application from Prometheus Config")
	gomega.Eventually(func() bool {
		return pkg.IsAppInPromConfig(promConfigJobName)
	}, shortWaitTimeout, shortPollingInterval).Should(gomega.BeFalse(), "Expected application to be removed from Prometheus config")

	t.Logs.Info("Delete namespace")
	gomega.Eventually(func() error {
		return pkg.DeleteNamespace(namespace)
	}, shortWaitTimeout, shortPollingInterval).ShouldNot(gomega.HaveOccurred())

	t.Logs.Info("Wait for namespace finalizer to be removed")
	gomega.Eventually(func() bool {
		return pkg.CheckNamespaceFinalizerRemoved(namespace)
	}, shortWaitTimeout, shortPollingInterval).Should(gomega.BeTrue())

	t.Logs.Info("Wait for namespace to be deleted")
	gomega.Eventually(func() bool {
		_, err := pkg.GetNamespace(namespace)
		return err != nil && errors.IsNotFound(err)
	}, shortWaitTimeout, shortPollingInterval).Should(gomega.BeTrue())
}

func DeployApplicationAndTemplate(namespace, appYamlPath, templateYamlPath, podPrefix string, nsAnnotations map[string]string, t framework.TestFramework) {
	t.Logs.Info("Deploy test application")

	t.Logs.Info("Create namespace")
	gomega.Eventually(func() (*v1.Namespace, error) {
		nsLabels := map[string]string{
			"verrazzano-managed": "true",
			"istio-injection":    "enabled"}
		return pkg.CreateNamespaceWithAnnotations(namespace, nsLabels, nsAnnotations)
	}, shortWaitTimeout, shortPollingInterval).ShouldNot(gomega.BeNil())

	t.Logs.Info("Create template resource")
	gomega.Eventually(func() error {
		return pkg.CreateOrUpdateResourceFromFileInGeneratedNamespace(templateYamlPath, namespace)
	}, shortWaitTimeout, shortPollingInterval).ShouldNot(gomega.HaveOccurred())

	t.Logs.Info("Create helidon resources")
	gomega.Eventually(func() error {
		return pkg.CreateOrUpdateResourceFromFileInGeneratedNamespace(appYamlPath, namespace)
	}, shortWaitTimeout, shortPollingInterval).ShouldNot(gomega.HaveOccurred())

	t.Logs.Info("Check application pods are running")
	gomega.Eventually(func() bool {
		result, err := pkg.PodsRunning(namespace, []string{podPrefix})
		if err != nil {
			ginkgo.AbortSuite(fmt.Sprintf("One or more pods are not running in the namespace: %v, error: %v", namespace, err))
		}
		return result
	}, longWaitTimeout, longPollingInterval).Should(gomega.BeTrue())
}
