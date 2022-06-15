// Copyright (c) 2021, 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package metricsbinding

import (
	"time"

	"github.com/onsi/gomega"
	"github.com/verrazzano/verrazzano/pkg/test/framework"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg"
	"k8s.io/apimachinery/pkg/api/errors"
)

const (
	shortWaitTimeout     = 10 * time.Minute
	shortPollingInterval = 10 * time.Second
	longWaitTimeout      = 15 * time.Minute
	longPollingInterval  = 20 * time.Second
)

// undeployApplication removes the application and namespace from the cluster
func undeployApplication(namespace string, yamlPath string, promConfigJobName string, t framework.TestFramework) {
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
