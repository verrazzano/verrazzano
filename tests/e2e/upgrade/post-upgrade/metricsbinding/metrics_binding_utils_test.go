// Copyright (c) 2021, 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package metricsbinding

import (
	"context"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"time"

	. "github.com/onsi/gomega"
	"github.com/verrazzano/verrazzano/pkg/test/framework"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg"
	"k8s.io/apimachinery/pkg/api/errors"
)

const (
	shortWaitTimeout     = 10 * time.Minute
	shortPollingInterval = 10 * time.Second
)

// undeployApplication removes the application and namespace from the cluster
func undeployApplication(namespace string, yamlPath string, t framework.TestFramework) {
	t.Logs.Info("Delete application")
	Eventually(func() error {
		return pkg.DeleteResourceFromFileInGeneratedNamespace(yamlPath, namespace)
	}, shortWaitTimeout, shortPollingInterval).ShouldNot(HaveOccurred())

	t.Logs.Info("Delete namespace")
	Eventually(func() error {
		return pkg.DeleteNamespace(namespace)
	}, shortWaitTimeout, shortPollingInterval).ShouldNot(HaveOccurred())

	t.Logs.Info("Wait for namespace finalizer to be removed")
	Eventually(func() bool {
		return pkg.CheckNamespaceFinalizerRemoved(namespace)
	}, shortWaitTimeout, shortPollingInterval).Should(BeTrue())

	t.Logs.Info("Wait for namespace to be deleted")
	Eventually(func() bool {
		_, err := pkg.GetNamespace(namespace)
		return err != nil && errors.IsNotFound(err)
	}, shortWaitTimeout, shortPollingInterval).Should(BeTrue())
}

func verifyMetricsBindingsDeleted(namespace string) {
	Eventually(func() (bool, error) {
		nsExists, err := pkg.DoesNamespaceExist(namespace)
		if err != nil {
			return false, err
		}
		if !nsExists {
			return true, nil
		}
		clientset, err := pkg.GetVerrazzanoApplicationOperatorClientSet()
		if err != nil {
			return false, err
		}
		bindingList, err := clientset.AppV1alpha1().MetricsBindings(namespace).List(context.TODO(), metav1.ListOptions{})
		if err != nil {
			return false, err
		}
		return len(bindingList.Items) > 0, nil
	}, shortWaitTimeout, shortPollingInterval).Should(BeTrue())
}
