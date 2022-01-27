// Copyright (c) 2021, 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package registry

import (
	"fmt"
	"github.com/verrazzano/verrazzano/pkg/test/framework"
	"os"
	"strings"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	waitTimeout     = 2 * time.Minute
	pollingInterval = 10 * time.Second
)

var registry = os.Getenv("REGISTRY")
var privateRepo = os.Getenv("PRIVATE_REPO")

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

var t = framework.NewTestFramework("registry")

var _ = t.BeforeSuite(func() {})
var _ = t.AfterSuite(func() {})
var _ = t.AfterEach(func() {})

var _ = t.Describe("Private Registry Verification", Label("f:platform-lcm.private-registry"),
	func() {
		t.It("All the pods in the cluster have the expected registry URLs",
			func() {
				var pod corev1.Pod
				imagePrefix := "ghcr.io"
				if len(registry) > 0 {
					imagePrefix = registry
				}
				if len(privateRepo) > 0 {
					imagePrefix += "/" + privateRepo
				}
				for i, ns := range listOfNamespaces {
					var pods *corev1.PodList
					Eventually(func() (*corev1.PodList, error) {
						var err error
						pods, err = pkg.ListPods(ns, metav1.ListOptions{})
						return pods, err
					}, waitTimeout, pollingInterval).ShouldNot(BeNil(), fmt.Sprintf("Error listing pods in the namespace %s", ns))

					for j := range pods.Items {
						pod = pods.Items[j]
						pkg.Log(pkg.Info, fmt.Sprintf("%d. Validating the registry url prefix for pod: %s in namespace: %s", i, pod.Name, ns))
						for k := range pod.Spec.Containers {
							Expect(strings.HasPrefix(pod.Spec.Containers[k].Image, imagePrefix)).To(BeTrue(),
								fmt.Sprintf("FAIL: The image for the pod %s in containers, doesn't starts with expected registry URL prefix %s, image name %s", pod.Name, registry, pod.Spec.Containers[k].Image))
						}
						for k := range pod.Spec.InitContainers {
							Expect(strings.HasPrefix(pod.Spec.InitContainers[k].Image, imagePrefix)).To(BeTrue(),
								fmt.Sprintf("FAIL: The image for the pod %s in initContainers, doesn't starts with expected registry URL prefix %s, image name %s", pod.Name, registry, pod.Spec.InitContainers[k].Image))
						}
					}
				}
			})
	})
