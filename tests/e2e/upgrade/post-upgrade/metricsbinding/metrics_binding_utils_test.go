// Copyright (c) 2021, 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package metricsbinding

import (
	"context"
	"fmt"
	"time"

	"github.com/verrazzano/verrazzano/pkg/k8s/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	. "github.com/onsi/ginkgo/v2"
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
	t.Logs.Infof("Verifying the namespace exists before attempting to delete resources")
	nsExists, err := pkg.DoesNamespaceExist(namespace)
	if err != nil {
		Fail(fmt.Sprintf("Failed to verify if the namespace %s exists", namespace))
	}
	if !nsExists {
		return
	}
	t.Logs.Info("Delete application")
	Eventually(func() error {
		file, err := pkg.FindTestDataFile(yamlPath)
		if err != nil {
			return err
		}
		err = resource.DeleteResourceFromFileInGeneratedNamespace(file, namespace)
		if err != nil {
			t.Logs.Errorf("Failed to delete the Application from file: %v", err)
		}
		return err
	}, shortWaitTimeout, shortPollingInterval).ShouldNot(HaveOccurred())

	t.Logs.Info("Delete namespace")
	Eventually(func() error {
		err := pkg.DeleteNamespace(namespace)
		if err != nil {
			t.Logs.Errorf("Failed to delete the Namespace %s from the cluster: %v", namespace, err)
		}
		return err
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

// verifyMetricsBindingsDeleted verifies that a metrics binding does not exist for a given namespace
func verifyMetricsBindingsDeleted(namespace string, t framework.TestFramework) {
	Eventually(func() (bool, error) {
		t.Logs.Infof("Verify no Metrics Bindings exist in the namespace %s", namespace)
		nsExists, err := pkg.DoesNamespaceExist(namespace)
		if err != nil {
			t.Logs.Errorf("Could not verify namespace %s exists", namespace)
			return false, err
		}
		if !nsExists {
			t.Logs.Infof("Namespace %s does not exist, no Metrics Binding check is occurring", namespace)
			return true, nil
		}
		clientset, err := pkg.GetVerrazzanoApplicationOperatorClientSet()
		if err != nil {
			t.Logs.Error("Could not get the Verrazzano Application Operator clientset")
			return false, err
		}
		bindingList, err := clientset.AppV1alpha1().MetricsBindings(namespace).List(context.TODO(), metav1.ListOptions{})
		if err != nil {
			t.Logs.Error("Could list Metrics Bindings in the namespace %s", namespace)
			return false, err
		}
		return len(bindingList.Items) == 0, nil
	}, shortWaitTimeout, shortPollingInterval).Should(BeTrue())
}
