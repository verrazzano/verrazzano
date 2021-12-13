// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package scrapegenerator

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

func DeployApplication(namespace string, yamlPath string) {
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

	pkg.Log(pkg.Info, "Create helidon resources")
	Eventually(func() error {
		return pkg.CreateOrUpdateResourceFromFile(yamlPath)
	}, shortWaitTimeout, shortPollingInterval).ShouldNot(HaveOccurred())
}

func UndeployApplication(namespace string, yamlPath string) {
	pkg.Log(pkg.Info, "Delete application")
	Eventually(func() error {
		return pkg.DeleteResourceFromFile(yamlPath)
	}, shortWaitTimeout, shortPollingInterval).ShouldNot(HaveOccurred())

	pkg.Log(pkg.Info, "Delete namespace")
	Eventually(func() error {
		return pkg.DeleteNamespace(namespace)
	}, shortWaitTimeout, shortPollingInterval).ShouldNot(HaveOccurred())

	Eventually(func() bool {
		_, err := pkg.GetNamespace(namespace)
		return err != nil && errors.IsNotFound(err)
	}, shortWaitTimeout, shortPollingInterval).Should(BeTrue())
}




