// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package defaultresource_test

import (
	"fmt"
	"os"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/verrazzano/pkg/k8sutil"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	waitTimeout     = 5 * time.Minute
	pollingInterval = 10 * time.Second
)

var expectedPodsKubeSystem = []string{
	"coredns",
	"kube-proxy"}

var _ = AfterSuite(func() {
	Eventually(func() error {
		return listPodsInKubeSystem()
	}, waitTimeout, pollingInterval).Should(BeNil())
})

var _ = Describe("Multi Cluster Install Validation",
	func() {
		It("has the expected namespaces", func() {
			kubeConfig := os.Getenv("KUBECONFIG")
			pkg.Log(pkg.Info, fmt.Sprintf("Kube config: %s", kubeConfig))
			namespaces, err := pkg.ListNamespaces(metav1.ListOptions{})
			Expect(err).ShouldNot(HaveOccurred())
			Expect(nsListContains(namespaces.Items, "default")).To(Equal(true))
			Expect(nsListContains(namespaces.Items, "kube-public")).To(Equal(true))
			Expect(nsListContains(namespaces.Items, "kube-system")).To(Equal(true))
			Expect(nsListContains(namespaces.Items, "kube-node-lease")).To(Equal(true))
		})

		Context("Expected pods are running.", func() {
			It("and waiting for expected pods must be running", func() {
				Eventually(func() bool {
					return pkg.PodsRunning("kube-system", expectedPodsKubeSystem)
				}, waitTimeout, pollingInterval).Should(BeTrue())
			})
		})
	})

func nsListContains(list []v1.Namespace, target string) bool {
	for i := range list {
		if list[i].Name == target {
			return true
		}
	}
	return false
}

func listPodsInKubeSystem() error {
	// Get the Kubernetes clientset and list pods in cluster
	clientset, err := k8sutil.GetKubernetesClientset()
	if err != nil {
		pkg.Log(pkg.Error, fmt.Sprintf("Error getting Kubernetes clientset: %v", err))
		return err
	}
	pods, err := pkg.ListPodsInCluster("kube-system", clientset)
	if err != nil {
		pkg.Log(pkg.Error, fmt.Sprintf("Error listing pods: %v", err))
		return err
	}
	for _, podInfo := range (*pods).Items {
		pkg.Log(pkg.Info, fmt.Sprintf("pods-name=%v\n", podInfo.Name))
		pkg.Log(pkg.Info, fmt.Sprintf("pods-status=%v\n", podInfo.Status.Phase))
		pkg.Log(pkg.Info, fmt.Sprintf("pods-condition=%v\n", podInfo.Status.Conditions))
	}
	return nil
}
