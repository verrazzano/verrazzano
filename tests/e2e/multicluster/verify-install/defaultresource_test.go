// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package defaultresource_test

import (
	"fmt"
	"os"
	"time"

	"github.com/onsi/ginkgo"
	"github.com/onsi/gomega"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg"
	v1 "k8s.io/api/core/v1"
)

var waitTimeout = 30 * time.Minute
var pollingInterval = 30 * time.Second

var expectedPodsKubeSystem = []string{
	"coredns",
	"kube-proxy"}

var _ = ginkgo.AfterSuite(func() {
	listPodsInKubeSystem()
})

var _ = ginkgo.Describe("Multi Cluster Install Validation",
	func() {
		ginkgo.It("has the expected namespaces", func() {
			kubeConfig := os.Getenv("KUBECONFIG")
			fmt.Println("Kube config ", kubeConfig)
			namespaces := pkg.ListNamespaces()
			gomega.Expect(nsListContains(namespaces.Items, "default")).To(gomega.Equal(true))
			gomega.Expect(nsListContains(namespaces.Items, "kube-public")).To(gomega.Equal(true))
			gomega.Expect(nsListContains(namespaces.Items, "kube-system")).To(gomega.Equal(true))
			gomega.Expect(nsListContains(namespaces.Items, "kube-node-lease")).To(gomega.Equal(true))

			// dump out namespace data to file
			logData := ""
			for i := range namespaces.Items {
				logData = logData + namespaces.Items[i].Name + "\n"
			}
		})

		ginkgo.Context("Expected pods are running.", func() {
			ginkgo.It("and waiting for expected pods must be running", func() {
				gomega.Eventually(func() bool {
					return pkg.PodsRunning("kube-system", expectedPodsKubeSystem)
				}, waitTimeout, pollingInterval).Should(gomega.BeTrue())
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

func listPodsInKubeSystem() {
	pods := pkg.ListPods("kube-system")
	for _, podInfo := range (*pods).Items {
		fmt.Printf("pods-name=%v\n", podInfo.Name)
		fmt.Printf("pods-status=%v\n", podInfo.Status.Phase)
		fmt.Printf("pods-condition=%v\n", podInfo.Status.Conditions)
	}
}
