// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package pkg

import (
	"fmt"
	"time"

	"github.com/verrazzano/verrazzano/pkg/k8s/resource"

	"github.com/onsi/gomega"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
)

//const SpringbootNamespace = "springboot"

const (
	springbootPollingInterval = 10 * time.Second
	springbootWaitTimeout     = 5 * time.Minute

	springbootComponentYaml = "examples/springboot-app/springboot-comp.yaml"
	springbootAppYaml       = "examples/springboot-app/springboot-app.yaml"
)

var expectedPodsSpringBootApp = []string{"springboot-workload"}

// DeploySpringBootApplication deploys the Springboot example application.
func DeploySpringBootApplication(namespace string, istioInjection string) {
	Log(Info, "Deploy Spring Boot Application")
	Log(Info, fmt.Sprintf("Create namespace %s", namespace))
	gomega.Eventually(func() (*v1.Namespace, error) {
		nsLabels := map[string]string{
			"verrazzano-managed": "true",
			"istio-injection":    istioInjection}
		return CreateNamespace(namespace, nsLabels)
	}, springbootWaitTimeout, springbootPollingInterval).ShouldNot(gomega.BeNil())

	Log(Info, "Create Spring Boot component resource")
	gomega.Eventually(func() error {
		file, err := FindTestDataFile(springbootComponentYaml)
		if err != nil {
			return err
		}
		return resource.CreateOrUpdateResourceFromFileInGeneratedNamespace(file, namespace)
	}, springbootWaitTimeout, springbootPollingInterval).ShouldNot(gomega.HaveOccurred())

	Log(Info, "Create Spring Boot application resource")
	gomega.Eventually(func() error {
		file, err := FindTestDataFile(springbootAppYaml)
		if err != nil {
			return err
		}
		return resource.CreateOrUpdateResourceFromFileInGeneratedNamespace(file, namespace)
	}, springbootWaitTimeout, springbootPollingInterval).ShouldNot(gomega.HaveOccurred())
}

// UndeploySpringBootApplication undeploys the Springboot example application.
func UndeploySpringBootApplication(namespace string) {
	Log(Info, "Undeploy Spring Boot Application")
	if exists, _ := DoesNamespaceExist(namespace); exists {
		Log(Info, "Delete Spring Boot application")
		gomega.Eventually(func() error {
			file, err := FindTestDataFile(springbootAppYaml)
			if err != nil {
				return err
			}
			return resource.DeleteResourceFromFileInGeneratedNamespace(file, namespace)
		}, springbootWaitTimeout, springbootPollingInterval).ShouldNot(gomega.HaveOccurred())

		Log(Info, "Delete Spring Boot components")
		gomega.Eventually(func() error {
			file, err := FindTestDataFile(springbootComponentYaml)
			if err != nil {
				return err
			}
			return resource.DeleteResourceFromFileInGeneratedNamespace(file, namespace)
		}, springbootWaitTimeout, springbootPollingInterval).ShouldNot(gomega.HaveOccurred())

		Log(Info, "Wait for application pods to terminate")
		gomega.Eventually(func() bool {
			podsTerminated, _ := PodsNotRunning(namespace, expectedPodsSpringBootApp)
			return podsTerminated
		}, springbootWaitTimeout, springbootPollingInterval).Should(gomega.BeTrue())

		Log(Info, fmt.Sprintf("Delete namespace %s", namespace))
		gomega.Eventually(func() error {
			return DeleteNamespace(namespace)
		}, springbootWaitTimeout, springbootPollingInterval).ShouldNot(gomega.HaveOccurred())

		Log(Info, "Wait for namespace finalizer to be removed")
		gomega.Eventually(func() bool {
			return CheckNamespaceFinalizerRemoved(namespace)
		}, springbootWaitTimeout, springbootPollingInterval).Should(gomega.BeTrue())

		Log(Info, "Wait for namespace to be deleted")
		gomega.Eventually(func() bool {
			_, err := GetNamespace(namespace)
			return err != nil && errors.IsNotFound(err)
		}, springbootWaitTimeout, springbootPollingInterval).Should(gomega.BeTrue())
	}
}
