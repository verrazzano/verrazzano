// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package loggingtrait

import (
	. "github.com/onsi/gomega"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"time"
)

const (
	shortWaitTimeout     = 10 * time.Minute
	shortPollingInterval = 10 * time.Second
)

func DeployApplication(namespace string, componentsPath string, applicationPath string) {
	pkg.Log(pkg.Info, "Deploy test application")
	// Wait for namespace to finish deletion possibly from a prior run.
	Eventually(func() bool {
		_, err := pkg.GetNamespace(namespace)
		return err != nil && errors.IsNotFound(err)
	}, shortWaitTimeout, shortPollingInterval).Should(BeTrue())

	pkg.Log(pkg.Info, "Create namespace")
	Eventually(func() (*v1.Namespace, error) {
		nsLabels := map[string]string{
			"verrazzano-managed": "true",
			"istio-injection":    "enabled"}
		return pkg.CreateNamespace(namespace, nsLabels)
	}, shortWaitTimeout, shortPollingInterval).ShouldNot(BeNil())

	pkg.Log(pkg.Info, "Create component resources")
	Eventually(func() error {
		return pkg.CreateOrUpdateResourceFromFile(componentsPath)
	}, shortWaitTimeout, shortPollingInterval).ShouldNot(HaveOccurred())

	pkg.Log(pkg.Info, "Create application resources")
	Eventually(func() error {
		return pkg.CreateOrUpdateResourceFromFile(applicationPath)
	}, shortWaitTimeout, shortPollingInterval).ShouldNot(HaveOccurred())
}

func UndeployApplication(namespace string, componentsPath string, applicationPath string, configMapName string) {
	pkg.Log(pkg.Info, "Delete application")
	Eventually(func() error {
		return pkg.DeleteResourceFromFile(applicationPath)
	}, shortWaitTimeout, shortPollingInterval).ShouldNot(HaveOccurred())

	pkg.Log(pkg.Info, "Delete components")
	Eventually(func() error {
		return pkg.DeleteResourceFromFile(componentsPath)
	}, shortWaitTimeout, shortPollingInterval).ShouldNot(HaveOccurred())

	pkg.Log(pkg.Info, "Verify ConfigMap is Deleted")
	Eventually(func() bool {
		configMap, _ := pkg.GetConfigMap(configMapName, namespace)
		return (configMap == nil)
	}, shortWaitTimeout, shortPollingInterval).Should(BeTrue())

	pkg.Log(pkg.Info, "Delete namespace")
	Eventually(func() error {
		return pkg.DeleteNamespace(namespace)
	}, shortWaitTimeout, shortPollingInterval).ShouldNot(HaveOccurred())

	Eventually(func() bool {
		_, err := pkg.GetNamespace(namespace)
		return err != nil && errors.IsNotFound(err)
	}, shortWaitTimeout, shortPollingInterval).Should(BeTrue())
}
