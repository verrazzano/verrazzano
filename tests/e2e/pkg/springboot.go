// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package pkg

import (
	"fmt"
	"time"

	"github.com/onsi/gomega"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
)

const SpringbootNamespace = "springboot"

const (
	springbootPollingInterval = 10 * time.Second
	springbootWaitTimeout     = 5 * time.Minute

	springbootComponentYaml = "examples/springboot-app/springboot-comp.yaml"
	springbootAppYaml       = "examples/springboot-app/springboot-app.yaml"
)

// DeploySpringBootApplication deploys the Springboot example application.
func DeploySpringBootApplication() {
	Log(Info, "Deploy Spring Boot Application")
	Log(Info, fmt.Sprintf("Create namespace %s", SpringbootNamespace))
	t.Eventually(func() (*v1.Namespace, error) {
		nsLabels := map[string]string{
			"verrazzano-managed": "true",
			"istio-injection":    "enabled"}
		return CreateNamespace(SpringbootNamespace, nsLabels)
	}, springbootWaitTimeout, springbootPollingInterval).ShouldNot(gomega.BeNil())

	Log(Info, "Create Spring Boot component resource")
	t.Eventually(func() error {
		return CreateOrUpdateResourceFromFile(springbootComponentYaml)
	}, springbootWaitTimeout, springbootPollingInterval).ShouldNot(gomega.HaveOccurred())

	Log(Info, "Create Spring Boot application resource")
	t.Eventually(func() error {
		return CreateOrUpdateResourceFromFile(springbootAppYaml)
	}, springbootWaitTimeout, springbootPollingInterval).ShouldNot(gomega.HaveOccurred())
}

// UndeploySpringBootApplication undeploys the Springboot example application.
func UndeploySpringBootApplication() {
	Log(Info, "Undeploy Spring Boot Application")
	if exists, _ := DoesNamespaceExist(SpringbootNamespace); exists {
		Log(Info, "Delete Spring Boot application")
		t.Eventually(func() error {
			return DeleteResourceFromFile(springbootAppYaml)
		}, springbootWaitTimeout, springbootPollingInterval).ShouldNot(gomega.HaveOccurred())

		Log(Info, "Delete Spring Boot components")
		t.Eventually(func() error {
			return DeleteResourceFromFile(springbootComponentYaml)
		}, springbootWaitTimeout, springbootPollingInterval).ShouldNot(gomega.HaveOccurred())

		Log(Info, fmt.Sprintf("Delete namespace %s", SpringbootNamespace))
		t.Eventually(func() error {
			return DeleteNamespace(SpringbootNamespace)
		}, springbootWaitTimeout, springbootPollingInterval).ShouldNot(gomega.HaveOccurred())

		t.Eventually(func() bool {
			_, err := GetNamespace(SpringbootNamespace)
			return err != nil && errors.IsNotFound(err)
		}, springbootWaitTimeout, springbootPollingInterval).Should(gomega.BeTrue())
	}
}
