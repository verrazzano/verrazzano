// Copyright (c) 2021, 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package metricsbinding

import (
	"fmt"
	"github.com/verrazzano/verrazzano/pkg/k8s/resource"
	v1 "k8s.io/api/core/v1"
	"time"

	"github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
	"github.com/verrazzano/verrazzano/pkg/test/framework"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg"
)

const (
	shortWaitTimeout     = 10 * time.Minute
	shortPollingInterval = 10 * time.Second
	longWaitTimeout      = 15 * time.Minute
	longPollingInterval  = 20 * time.Second
)

func createNamespace(namespace, istioInjection string, t framework.TestFramework) {
	t.Logs.Info("Create namespace")
	gomega.Eventually(func() (*v1.Namespace, error) {
		nsLabels := map[string]string{"verrazzano-managed": "true", "istio-injeciton": istioInjection}
		nsExists, err := pkg.DoesNamespaceExist(namespace)
		if err != nil {
			t.Logs.Errorf("Could not verify if namespace %s exists", namespace)
			return nil, err
		}
		if !nsExists {
			t.Logs.Infof("Namespace %s does not exist, creating now", namespace)
			nsObject, err := pkg.CreateNamespace(namespace, nsLabels)
			if err != nil {
				t.Logs.Errorf("Failed to create the Namespace %s in the cluster: %v", namespace, err)
			}
			return nsObject, err
		}
		nsObject, err := pkg.GetNamespace(namespace)
		if err != nil {
			t.Logs.Errorf("Failed to get the Namespace %s from the cluster: %v", namespace, err)
		}
		return nsObject, err
	}, shortWaitTimeout, shortPollingInterval).ShouldNot(gomega.BeNil())
}

// deployApplication deploys an application and namespace given the application parameters
func deployApplication(namespace, yamlPath, podPrefix string, t framework.TestFramework) {
	t.Logs.Info("Create application from yaml path")
	gomega.Eventually(func() error {
		err := resource.CreateOrUpdateResourceFromFileInGeneratedNamespace(yamlPath, namespace)
		if err != nil {
			t.Logs.Errorf("Failed to apply the Application from file: %v", err)
		}
		return err
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

// deployConfigMap deploys a ConfigMap from a file path
func deployConfigMap(namespace, configMapYamlPath string, t framework.TestFramework) {
	t.Logs.Info("Create ConfigMap resource")
	gomega.Eventually(func() error {
		err := resource.CreateOrUpdateResourceFromFileInGeneratedNamespace(configMapYamlPath, namespace)
		if err != nil {
			t.Logs.Errorf("Failed to apply the ConfigMap from file: %v", err)
		}
		return err
	}, shortWaitTimeout, shortPollingInterval).ShouldNot(gomega.HaveOccurred())
}

// deployTemplate deploys a Metrics Template from a file path
func deployTemplate(namespace, templateYamlPath string, t framework.TestFramework) {
	t.Logs.Info("Create template resource")
	gomega.Eventually(func() error {
		err := resource.CreateOrUpdateResourceFromFileInGeneratedNamespace(templateYamlPath, namespace)
		if err != nil {
			t.Logs.Errorf("Failed to apply the Metrics Template from file: %v", err)
		}
		return err
	}, shortWaitTimeout, shortPollingInterval).ShouldNot(gomega.HaveOccurred())
}
