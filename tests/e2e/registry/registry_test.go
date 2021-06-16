// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package registry

import (
	"fmt"
	"github.com/onsi/ginkgo"
	"github.com/onsi/gomega"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"os"
	"strings"
)

var registry = os.Getenv("REGISTRY")

// List of namespaces from which all the pods are queried to confirm the images are loaded from the target registry/repo
var listOfNamespaces = []string{
	"cattle-global-data",
	"cattle-global-data-nt",
	"cattle-system",
	"cert-manager",
	"default",
	"fleet-default",
	"fleet-local",
	"fleet-system",
	"ingress-nginx",
	"istio-system",
	"keycloak",
	"local",
	"monitoring",
	"rancher-operator-system",
	"verrazzano-install",
	"verrazzano-mc",
	"verrazzano-system",
}

var _ = ginkgo.Describe("Private Registry Verification",
	func() {
		ginkgo.It("All the pods in the cluster have the expected registry URLs",
			func() {
				var pod corev1.Pod
				for i, ns := range listOfNamespaces {
					pods, err := pkg.ListPods(ns, metav1.ListOptions{})
					if err != nil {
						ginkgo.Fail(fmt.Sprintf("Error listing pods in the namespace %s", ns))
					}
					for j := range pods.Items {
						pod = pods.Items[j]
						pkg.Log(pkg.Info, fmt.Sprintf("%d. Validating the registry url prefix for pod: %s in namespace: %s", i, pod.Name, ns))
						for k := range pod.Spec.Containers {
							gomega.Expect(strings.HasPrefix(pod.Spec.Containers[k].Image, registry)).To(gomega.BeTrue(),
								fmt.Sprintf("FAIL: The image for the pod %s in containers, doesn't starts with expected registry URL prefix %s", pod.Name, registry))
						}
						for k := range pod.Spec.InitContainers {
							gomega.Expect(strings.HasPrefix(pod.Spec.InitContainers[k].Image, registry)).To(gomega.BeTrue(),
								fmt.Sprintf("FAIL: The image for the pod %s in initContainers, doesn't starts with expected registry URL prefix %s", pod.Name, registry))
						}
					}
				}
			})
	})
