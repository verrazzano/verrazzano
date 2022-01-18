// Copyright (c) 2021, 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package loggingtrait

import (
	"github.com/onsi/gomega"
	"github.com/verrazzano/verrazzano/pkg/test/framework"
	"github.com/verrazzano/verrazzano/pkg/test/framework/metrics"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"time"
)

const (
	shortWaitTimeout     = 10 * time.Minute
	shortPollingInterval = 10 * time.Second
)

func DeployApplication(namespace string, componentsPath string, applicationPath string, t *framework.TestFramework) {
	pkg.Log(pkg.Info, "Deploy test application")
	start := time.Now()
	// Wait for namespace to finish deletion possibly from a prior run.
	t.Eventually(func() bool {
		_, err := pkg.GetNamespace(namespace)
		return err != nil && errors.IsNotFound(err)
	}, shortWaitTimeout, shortPollingInterval).Should(gomega.BeTrue())

	pkg.Log(pkg.Info, "Create namespace")
	t.Eventually(func() (*v1.Namespace, error) {
		nsLabels := map[string]string{
			"verrazzano-managed": "true",
			"istio-injection":    "enabled"}
		return pkg.CreateNamespace(namespace, nsLabels)
	}, shortWaitTimeout, shortPollingInterval).ShouldNot(gomega.BeNil())

	pkg.Log(pkg.Info, "Create component resources")
	t.Eventually(func() error {
		return pkg.CreateOrUpdateResourceFromFile(componentsPath)
	}, shortWaitTimeout, shortPollingInterval).ShouldNot(gomega.HaveOccurred())

	pkg.Log(pkg.Info, "Create application resources")
	t.Eventually(func() error {
		return pkg.CreateOrUpdateResourceFromFile(applicationPath)
	}, shortWaitTimeout, shortPollingInterval).ShouldNot(gomega.HaveOccurred())
	metrics.Emit(t.Metrics.With("deployment_elapsed_time", time.Since(start).Milliseconds()))
}

func UndeployApplication(namespace string, componentsPath string, applicationPath string, configMapName string, t *framework.TestFramework) {
	pkg.Log(pkg.Info, "Delete application")
	start := time.Now()
	t.Eventually(func() error {
		return pkg.DeleteResourceFromFile(applicationPath)
	}, shortWaitTimeout, shortPollingInterval).ShouldNot(gomega.HaveOccurred())

	pkg.Log(pkg.Info, "Delete components")
	t.Eventually(func() error {
		return pkg.DeleteResourceFromFile(componentsPath)
	}, shortWaitTimeout, shortPollingInterval).ShouldNot(gomega.HaveOccurred())

	pkg.Log(pkg.Info, "Verify ConfigMap is Deleted")
	t.Eventually(func() bool {
		configMap, _ := pkg.GetConfigMap(configMapName, namespace)
		return (configMap == nil)
	}, shortWaitTimeout, shortPollingInterval).Should(gomega.BeTrue())

	pkg.Log(pkg.Info, "Delete namespace")
	t.Eventually(func() error {
		return pkg.DeleteNamespace(namespace)
	}, shortWaitTimeout, shortPollingInterval).ShouldNot(gomega.HaveOccurred())

	t.Eventually(func() bool {
		_, err := pkg.GetNamespace(namespace)
		return err != nil && errors.IsNotFound(err)
	}, shortWaitTimeout, shortPollingInterval).Should(gomega.BeTrue())
	metrics.Emit(t.Metrics.With("undeployment_elapsed_time", time.Since(start).Milliseconds()))
}
