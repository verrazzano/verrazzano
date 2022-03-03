// Copyright (c) 2021, 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package metricsbinding

import (
	"fmt"
	"time"

	"github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
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

func DeployApplication(namespace, yamlPath, podPrefix string) {
	pkg.Log(pkg.Info, "Deploy test application")
	// Wait for namespace to finish deletion possibly from a prior run.
	gomega.Eventually(func() bool {
		_, err := pkg.GetNamespace(namespace)
		return err != nil && errors.IsNotFound(err)
	}, shortWaitTimeout, shortPollingInterval).Should(gomega.BeTrue())

	pkg.Log(pkg.Info, "Create namespace")
	gomega.Eventually(func() (*v1.Namespace, error) {
		nsLabels := map[string]string{
			"verrazzano-managed": "true",
			"istio-injection":    "enabled"}
		return pkg.CreateNamespace(namespace, nsLabels)
	}, shortWaitTimeout, shortPollingInterval).ShouldNot(gomega.BeNil())

	pkg.Log(pkg.Info, "Create helidon resources")
	gomega.Eventually(func() error {
		return pkg.CreateOrUpdateResourceFromFile(yamlPath)
	}, shortWaitTimeout, shortPollingInterval).ShouldNot(gomega.HaveOccurred())

	pkg.Log(pkg.Info, "Check application pods are running")
	gomega.Eventually(func() bool {
		result, err := pkg.PodsRunning(namespace, []string{podPrefix})
		if err != nil {
			ginkgo.AbortSuite(fmt.Sprintf("One or more pods are not running in the namespace: %v, error: %v", namespace, err))
		}
		return result
	}, longWaitTimeout, longPollingInterval).Should(gomega.BeTrue())
}

func UndeployApplication(namespace string, yamlPath string, promConfigJobName string) {
	pkg.Log(pkg.Info, "Delete application")
	gomega.Eventually(func() error {
		return pkg.DeleteResourceFromFile(yamlPath)
	}, shortWaitTimeout, shortPollingInterval).ShouldNot(gomega.HaveOccurred())

	pkg.Log(pkg.Info, "Remove application from Prometheus Config")
	gomega.Eventually(func() bool {
		return pkg.IsAppInPromConfig(promConfigJobName)
	}, shortWaitTimeout, shortPollingInterval).Should(gomega.BeFalse(), "Expected application to be removed from Prometheus config")

	pkg.Log(pkg.Info, "Delete namespace")
	gomega.Eventually(func() error {
		return pkg.DeleteNamespace(namespace)
	}, shortWaitTimeout, shortPollingInterval).ShouldNot(gomega.HaveOccurred())

	gomega.Eventually(func() bool {
		_, err := pkg.GetNamespace(namespace)
		return err != nil && errors.IsNotFound(err)
	}, shortWaitTimeout, shortPollingInterval).Should(gomega.BeTrue())
}

func DeployApplicationAndTemplate(namespace, appYamlPath, templateYamlPath, podPrefix string, nsAnnotations map[string]string) {
	pkg.Log(pkg.Info, "Deploy test application")
	// Wait for namespace to finish deletion possibly from a prior run.
	gomega.Eventually(func() bool {
		_, err := pkg.GetNamespace(namespace)
		return err != nil && errors.IsNotFound(err)
	}, shortWaitTimeout, shortPollingInterval).Should(gomega.BeTrue())

	pkg.Log(pkg.Info, "Create namespace")
	gomega.Eventually(func() (*v1.Namespace, error) {
		nsLabels := map[string]string{
			"verrazzano-managed": "true",
			"istio-injection":    "enabled"}
		return pkg.CreateNamespaceWithAnnotations(namespace, nsLabels, nsAnnotations)
	}, shortWaitTimeout, shortPollingInterval).ShouldNot(gomega.BeNil())

	pkg.Log(pkg.Info, "Create template resource")
	gomega.Eventually(func() error {
		return pkg.CreateOrUpdateResourceFromFile(templateYamlPath)
	}, shortWaitTimeout, shortPollingInterval).ShouldNot(gomega.HaveOccurred())

	pkg.Log(pkg.Info, "Create helidon resources")
	gomega.Eventually(func() error {
		return pkg.CreateOrUpdateResourceFromFile(appYamlPath)
	}, shortWaitTimeout, shortPollingInterval).ShouldNot(gomega.HaveOccurred())

	pkg.Log(pkg.Info, "Check application pods are running")
	gomega.Eventually(func() bool {
		result, err := pkg.PodsRunning(namespace, []string{podPrefix})
		if err != nil {
			ginkgo.AbortSuite(fmt.Sprintf("One or more pods are not running in the namespace: %v, error: %v", namespace, err))
		}
		return result
	}, longWaitTimeout, longPollingInterval).Should(gomega.BeTrue())
}
