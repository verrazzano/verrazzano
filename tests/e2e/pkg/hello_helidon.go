// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package pkg

import (
	"fmt"
	"time"

	"github.com/onsi/gomega"
	v1 "k8s.io/api/core/v1"
)

const (
	helidonPollingInterval = 10 * time.Second
	helidonWaitTimeout     = 5 * time.Minute

	helidonComponentYaml = "../../../examples/hello-helidon/hello-helidon-comp.yaml"
	helidonAppYaml       = "../../../examples/hello-helidon/hello-helidon-app.yaml"
)

// DeployHelloHelidonApplication deploys the Hello Helidon example application. It accepts an optional
// OCI Log ID that is added as an annotation on the namespace to test the OCI Logging service integration.
func DeployHelloHelidonApplication(namespace string, ociLogID string) {
	Log(Info, "Deploy Hello Helidon Application")
	Log(Info, fmt.Sprintf("Create namespace %s", namespace))
	gomega.Eventually(func() (*v1.Namespace, error) {
		nsLabels := map[string]string{
			"verrazzano-managed": "true",
			"istio-injection":    "enabled"}

		var annotations map[string]string
		if len(ociLogID) > 0 {
			annotations = make(map[string]string)
			annotations["verrazzano.io/oci-log-id"] = ociLogID
		}

		return CreateNamespaceWithAnnotations(namespace, nsLabels, annotations)
	}, helidonWaitTimeout, helidonPollingInterval).ShouldNot(gomega.BeNil(), fmt.Sprintf("Failed to create namespace %s", namespace))

	Log(Info, "Create Hello Helidon component resource")
	gomega.Eventually(func() error {
		return CreateOrUpdateResourceFromFileInGeneratedNamespace(helidonComponentYaml, namespace)
	}, helidonWaitTimeout, helidonPollingInterval).ShouldNot(gomega.HaveOccurred(), "Failed to create hello-helidon component resource")

	Log(Info, "Create Hello Helidon application resource")
	gomega.Eventually(func() error {
		return CreateOrUpdateResourceFromFileInGeneratedNamespace(helidonAppYaml, namespace)
	}, helidonWaitTimeout, helidonPollingInterval).ShouldNot(gomega.HaveOccurred(), "Failed to create hello-helidon application resource")
}

// UndeployHelloHelidonApplication undeploys the Hello Helidon example application.
func UndeployHelloHelidonApplication(namespace string) {
	Log(Info, "Undeploy Hello Helidon Application")
	if exists, _ := DoesNamespaceExist(namespace); exists {
		Log(Info, "Delete Hello Helidon application")
		gomega.Eventually(func() error {
			return DeleteResourceFromFileInGeneratedNamespace(helidonAppYaml, namespace)
		}, helidonWaitTimeout, helidonPollingInterval).ShouldNot(gomega.HaveOccurred(), "Failed to create hello-helidon application resource")

		Log(Info, "Delete Hello Helidon components")
		gomega.Eventually(func() error {
			return DeleteResourceFromFileInGeneratedNamespace(helidonComponentYaml, namespace)
		}, helidonWaitTimeout, helidonPollingInterval).ShouldNot(gomega.HaveOccurred(), "Failed to create hello-helidon component resource")

		Log(Info, fmt.Sprintf("Delete namespace %s", namespace))
		gomega.Eventually(func() error {
			return DeleteNamespace(namespace)
		}, helidonWaitTimeout, helidonPollingInterval).ShouldNot(gomega.HaveOccurred(), fmt.Sprintf("Failed to deleted namespace %s", namespace))
	}
}
