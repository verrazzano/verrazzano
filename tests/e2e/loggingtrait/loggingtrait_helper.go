// Copyright (c) 2021, 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package loggingtrait

import (
	"fmt"
	"time"

	"github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
	"github.com/verrazzano/verrazzano/pkg/k8s/resource"
	"github.com/verrazzano/verrazzano/pkg/test/framework"
	"github.com/verrazzano/verrazzano/pkg/test/framework/metrics"
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

func DeployApplication(namespace, istionInjection, componentsPath, applicationPath, podName string, t *framework.TestFramework) {
	t.Logs.Info("Deploy test application")
	start := time.Now()

	t.Logs.Info("Create namespace")
	gomega.Eventually(func() (*v1.Namespace, error) {
		nsLabels := map[string]string{
			"verrazzano-managed": "true",
			"istio-injection":    istionInjection}
		return pkg.CreateNamespace(namespace, nsLabels)
	}, shortWaitTimeout, shortPollingInterval).ShouldNot(gomega.BeNil())

	t.Logs.Info("Create component resources")
	gomega.Eventually(func() error {
		file, err := pkg.FindTestDataFile(componentsPath)
		if err != nil {
			return err
		}
		return resource.CreateOrUpdateResourceFromFileInGeneratedNamespace(file, namespace)
	}, shortWaitTimeout, shortPollingInterval).ShouldNot(gomega.HaveOccurred())

	t.Logs.Info("Create application resources")
	gomega.Eventually(func() error {
		file, err := pkg.FindTestDataFile(applicationPath)
		if err != nil {
			return err
		}
		return resource.CreateOrUpdateResourceFromFileInGeneratedNamespace(file, namespace)
	}, shortWaitTimeout, shortPollingInterval).ShouldNot(gomega.HaveOccurred())

	t.Logs.Infof("Check pod %v is running", podName)
	gomega.Eventually(func() bool {
		result, err := pkg.PodsRunning(namespace, []string{podName})
		if err != nil {
			ginkgo.AbortSuite(fmt.Sprintf("Pod %v is not running in the namespace: %v, error: %v", podName, namespace, err))
		}
		return result
	}, longWaitTimeout, longPollingInterval).Should(gomega.BeTrue())

	metrics.Emit(t.Metrics.With("deployment_elapsed_time", time.Since(start).Milliseconds()))
}

func UndeployApplication(namespace string, componentsPath string, applicationPath string, configMapName string, t *framework.TestFramework) {
	t.Logs.Info("Delete application")
	start := time.Now()
	gomega.Eventually(func() error {
		file, err := pkg.FindTestDataFile(applicationPath)
		if err != nil {
			return err
		}
		return resource.DeleteResourceFromFileInGeneratedNamespace(file, namespace)
	}, shortWaitTimeout, shortPollingInterval).ShouldNot(gomega.HaveOccurred())

	t.Logs.Info("Delete components")
	gomega.Eventually(func() error {
		file, err := pkg.FindTestDataFile(componentsPath)
		if err != nil {
			return err
		}
		return resource.DeleteResourceFromFileInGeneratedNamespace(file, namespace)
	}, shortWaitTimeout, shortPollingInterval).ShouldNot(gomega.HaveOccurred())

	t.Logs.Info("Verify ConfigMap is Deleted")
	gomega.Eventually(func() bool {
		configMap, _ := pkg.GetConfigMap(configMapName, namespace)
		return (configMap == nil)
	}, shortWaitTimeout, shortPollingInterval).Should(gomega.BeTrue())

	t.Logs.Info("Delete namespace")
	gomega.Eventually(func() error {
		return pkg.DeleteNamespace(namespace)
	}, shortWaitTimeout, shortPollingInterval).ShouldNot(gomega.HaveOccurred())

	t.Logs.Info("Wait for Finalizer to be removed")
	gomega.Eventually(func() bool {
		return pkg.CheckNamespaceFinalizerRemoved(namespace)
	}, shortWaitTimeout, shortPollingInterval).Should(gomega.BeTrue())

	t.Logs.Info("Wait for namespace to be deleted")
	gomega.Eventually(func() bool {
		_, err := pkg.GetNamespace(namespace)
		return err != nil && errors.IsNotFound(err)
	}, shortWaitTimeout, shortPollingInterval).Should(gomega.BeTrue())
	metrics.Emit(t.Metrics.With("undeployment_elapsed_time", time.Since(start).Milliseconds()))
}
