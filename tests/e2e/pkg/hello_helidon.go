// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package pkg

import (
	"time"

	"github.com/onsi/gomega"
	v1 "k8s.io/api/core/v1"
)

const HelloHelidonNamespace = "hello-helidon"

const (
	helidonPollingInterval = 10 * time.Second
	helidonWaitTimeout     = 5 * time.Minute

	helidonComponentYaml = "examples/hello-helidon/hello-helidon-comp.yaml"
	helidonAppYaml       = "examples/hello-helidon/hello-helidon-app.yaml"
)

func DeployHelloHelidonApplication(ociLogId string) {
	gomega.Eventually(func() (*v1.Namespace, error) {
		nsLabels := map[string]string{
			"verrazzano-managed": "true",
			"istio-injection":    "enabled"}

		var annotations map[string]string
		if len(ociLogId) > 0 {
			annotations = make(map[string]string)
			annotations["verrazzano.io/oci-log-id"] = ociLogId
		}

		return CreateNamespaceWithAnnotations(HelloHelidonNamespace, nsLabels, annotations)
	}, helidonWaitTimeout, helidonPollingInterval).ShouldNot(gomega.BeNil())

	gomega.Eventually(func() error {
		return CreateOrUpdateResourceFromFile(helidonComponentYaml)
	}, helidonWaitTimeout, helidonPollingInterval).ShouldNot(gomega.HaveOccurred())

	gomega.Eventually(func() error {
		return CreateOrUpdateResourceFromFile(helidonAppYaml)
	}, helidonWaitTimeout, helidonPollingInterval).ShouldNot(gomega.HaveOccurred(), "Failed to create hello-helidon application resource")
}

func UndeployHelloHelidonApplication() {
	gomega.Eventually(func() error {
		return DeleteResourceFromFile(helidonAppYaml)
	}, helidonWaitTimeout, helidonPollingInterval).ShouldNot(gomega.HaveOccurred())

	gomega.Eventually(func() error {
		return DeleteResourceFromFile(helidonComponentYaml)
	}, helidonWaitTimeout, helidonPollingInterval).ShouldNot(gomega.HaveOccurred())

	gomega.Eventually(func() error {
		return DeleteNamespace(HelloHelidonNamespace)
	}, helidonWaitTimeout, helidonPollingInterval).ShouldNot(gomega.HaveOccurred())
}
