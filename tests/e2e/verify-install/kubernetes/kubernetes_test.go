// Copyright (c) 2020, 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package kubernetes_test

import (
	"time"

	"github.com/onsi/ginkgo"
	ginkgoExt "github.com/onsi/ginkgo/extensions/table"
	"github.com/onsi/gomega"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg"
	v1 "k8s.io/api/core/v1"
)

var waitTimeout = 15 * time.Minute
var pollingInterval = 30 * time.Second
var expectedPodsCattleSystem = []string{
	"rancher",
	"cattle-node-agent",
	"cattle-cluster-agent"}

var expectedPodsKeycloak = []string{
	"mysql",
	"keycloak"}

var expectedPodsCertManager = []string{
	"cert-manager"}

var expectedPodsIngressNginx = []string{
	"ingress-controller-ingress-nginx-controller",
	"ingress-controller-ingress-nginx-defaultbackend"}

var expectedPodsVerrazzanoSystemMinimal = []string{
	"verrazzano-cluster-operator",
	"verrazzano-console",
	"verrazzano-monitoring-operator",
	"verrazzano-operator",
	"vmi-system-api",
	"vmi-system-es-master",
	"vmi-system-grafana",
	"vmi-system-kibana",
	"vmi-system-prometheus",
	"vmi-system-prometheus-gw"}

var _ = ginkgo.Describe("Kubernetes Cluster",
	func() {
		ginkgo.It("has the expected number of nodes",
			func() {
				nodes := pkg.ListNodes()
				gomega.Expect(len(nodes.Items)).To(gomega.BeNumerically(">=", 1))

				// dump out node data to file
				logData := ""
				for i := range nodes.Items {
					logData = logData + nodes.Items[i].ObjectMeta.Name + "\n"
				}
			})

		ginkgo.It("is a target Kubernetes version",
			func() {
				clientset := pkg.GetKubernetesClientset()
				versionInfo, _ := clientset.ServerVersion()
				gomega.Expect(versionInfo.GitVersion).Should(gomega.MatchRegexp(`(v1\.1[5-8]\.*)`))
			})

		ginkgo.It("has the expected namespaces",
			func() {
				namespaces := pkg.ListNamespaces()
				gomega.Expect(nsListContains(namespaces.Items, "cattle-global-data")).To(gomega.Equal(true))
				gomega.Expect(nsListContains(namespaces.Items, "cattle-global-nt")).To(gomega.Equal(true))
				gomega.Expect(nsListContains(namespaces.Items, "cattle-system")).To(gomega.Equal(true))
				gomega.Expect(nsListContains(namespaces.Items, "istio-system")).To(gomega.Equal(true))
				gomega.Expect(nsListContains(namespaces.Items, "gitlab")).To(gomega.Equal(false))
				gomega.Expect(nsListContains(namespaces.Items, "keycloak")).To(gomega.Equal(true))
				gomega.Expect(nsListContains(namespaces.Items, "local")).To(gomega.Equal(true))
				gomega.Expect(nsListContains(namespaces.Items, "verrazzano-system")).To(gomega.Equal(true))
				gomega.Expect(nsListContains(namespaces.Items, "verrazzano-mc")).To(gomega.Equal(true))
				gomega.Expect(nsListContains(namespaces.Items, "cert-manager")).To(gomega.Equal(true))
				gomega.Expect(nsListContains(namespaces.Items, "ingress-nginx")).To(gomega.Equal(true))

				// dump out namespace data to file
				logData := ""
				for i := range namespaces.Items {
					logData = logData + namespaces.Items[i].Name + "\n"
				}
			})

		ginkgoExt.DescribeTable("deployed Verrazzano components",
			func(name string, expected bool) {
				gomega.Expect(vzComponentPresent(name, "verrazzano-system")).To(gomega.Equal(expected))
			},
			ginkgoExt.Entry("includes verrazzano-operator", "verrazzano-operator", true),
			ginkgoExt.Entry("does not include verrazzano-web", "verrazzano-web", false),
			ginkgoExt.Entry("includes verrazzano-console", "verrazzano-console", true),
			ginkgoExt.Entry("does not include verrazzano-ldap", "verrazzano-ldap", false),
			ginkgoExt.Entry("includes verrazzano-cluster-operator", "verrazzano-cluster-operator", true),
			ginkgoExt.Entry("includes verrazzano-monitoring-operator", "verrazzano-monitoring-operator", true),
		)

		ginkgoExt.DescribeTable("deployed cert-manager components",
			func(name string, expected bool) {
				gomega.Expect(vzComponentPresent(name, "cert-manager")).To(gomega.Equal(expected))
			},
			ginkgoExt.Entry("includes cert-manager", "cert-manager", true),
			ginkgoExt.Entry("does not include cert-manager-cainjector", "cert-manager-cainjector", false),
		)

		ginkgoExt.DescribeTable("deployed ingress components",
			func(name string, expected bool) {
				gomega.Expect(vzComponentPresent(name, "ingress-nginx")).To(gomega.Equal(expected))
			},
			ginkgoExt.Entry("includes ingress-controller-ingress-nginx-controller", "ingress-controller-ingress-nginx-controller", true),
		)

		ginkgoExt.DescribeTable("deployed keycloak components",
			func(name string, expected bool) {
				gomega.Expect(vzComponentPresent(name, "keycloak")).To(gomega.Equal(expected))
			},
			ginkgoExt.Entry("includes ssoproxycontroller", "ssoproxycontroller", false),
		)

		ginkgoExt.DescribeTable("deployed rancher components",
			func(name string, expected bool) {
				gomega.Expect(vzComponentPresent(name, "cattle-system")).To(gomega.Equal(expected))
			},
			ginkgoExt.Entry("includes rancher", "rancher", true),
			ginkgoExt.Entry("includes rancher-agent", "cattle-node-agent", true),
			ginkgoExt.Entry("includes rancher-agent", "cattle-cluster-agent", true),
		)

		ginkgo.It("Expected pods are running",
			func() {
				pkg.Concurrently(
					func() {
						gomega.Eventually(pkg.PodsRunning("cattle-system", expectedPodsCattleSystem), waitTimeout, pollingInterval).
							Should(gomega.BeTrue())
					},
					func() {
						gomega.Eventually(pkg.PodsRunning("keycloak", expectedPodsKeycloak), waitTimeout, pollingInterval).
							Should(gomega.BeTrue())
					},
					func() {
						gomega.Eventually(pkg.PodsRunning("cert-manager", expectedPodsCertManager), waitTimeout, pollingInterval).
							Should(gomega.BeTrue())
					},
					func() {
						gomega.Eventually(pkg.PodsRunning("ingress-nginx", expectedPodsIngressNginx), waitTimeout, pollingInterval).
							Should(gomega.BeTrue())
					},
					func() {
						gomega.Eventually(pkg.PodsRunning("verrazzano-system", expectedPodsVerrazzanoSystemMinimal), waitTimeout, pollingInterval).
							Should(gomega.BeTrue())
					},
				)
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

func vzComponentPresent(name string, namespace string) bool {
	return pkg.DoesPodExist(namespace, name)
}
