// Copyright (c) 2020, 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package kubernetes_test

import (
	"fmt"
	"time"

	"github.com/onsi/ginkgo"
	ginkgoExt "github.com/onsi/ginkgo/extensions/table"
	"github.com/onsi/gomega"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var waitTimeout = 15 * time.Minute
var pollingInterval = 30 * time.Second
var expectedPodsCattleSystem = []string{
	"rancher"}

var expectedPodsKeycloak = []string{
	"mysql",
	"keycloak"}

var expectedPodsCertManager = []string{
	"cert-manager"}

var expectedPodsIngressNginx = []string{
	"ingress-controller-ingress-nginx-controller",
	"ingress-controller-ingress-nginx-defaultbackend"}

var expectedNonVMIPodsVerrazzanoSystem = []string{
	"verrazzano-monitoring-operator",
	"verrazzano-operator",
}

// comment out while debugging so it does not break master
//"vmi-system-prometheus",
//"vmi-system-prometheus-gw"}

var _ = ginkgo.Describe("Kubernetes Cluster",
	func() {
		isManagedClusterProfile := pkg.IsManagedClusterProfile()
		isProdProfile := pkg.IsProdProfile()

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

		ginkgo.It("has the expected namespaces",
			func() {
				namespaces, err := pkg.ListNamespaces(metav1.ListOptions{})
				if err != nil {
					ginkgo.Fail(fmt.Sprintf("Failed to get namespaces with error: %v", err))
				}
				if isManagedClusterProfile {
					gomega.Expect(nsListContains(namespaces.Items, "cattle-global-data")).To(gomega.BeFalse())
					gomega.Expect(nsListContains(namespaces.Items, "cattle-global-nt")).To(gomega.BeFalse())
					// Even though we do not install Rancher on managed clusters, we do create the namespace
					// so we can create network policies. Rancher will run pods in this namespace once
					// the managed cluster manifest YAML is applied to the managed cluster.
					gomega.Expect(nsListContains(namespaces.Items, "cattle-system")).To(gomega.BeTrue())
					gomega.Expect(nsListContains(namespaces.Items, "local")).To(gomega.BeFalse())
				} else {
					gomega.Expect(nsListContains(namespaces.Items, "cattle-global-data")).To(gomega.BeTrue())
					gomega.Expect(nsListContains(namespaces.Items, "cattle-global-nt")).To(gomega.BeTrue())
					gomega.Expect(nsListContains(namespaces.Items, "cattle-system")).To(gomega.BeTrue())
					gomega.Expect(nsListContains(namespaces.Items, "local")).To(gomega.BeTrue())
				}
				gomega.Expect(nsListContains(namespaces.Items, "istio-system")).To(gomega.Equal(true))
				gomega.Expect(nsListContains(namespaces.Items, "gitlab")).To(gomega.Equal(false))
				if isManagedClusterProfile {
					gomega.Expect(nsListContains(namespaces.Items, "keycloak")).To(gomega.Equal(false))
				} else {
					gomega.Expect(nsListContains(namespaces.Items, "keycloak")).To(gomega.Equal(true))
				}
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
			ginkgoExt.Entry("includes verrazzano-console", "verrazzano-console", !isManagedClusterProfile),
			ginkgoExt.Entry("does not include verrazzano-ldap", "verrazzano-ldap", false),
			ginkgoExt.Entry("does not include verrazzano-cluster-operator", "verrazzano-cluster-operator", false),
			ginkgoExt.Entry("includes verrazzano-monitoring-operator", "verrazzano-monitoring-operator", true),
		)

		ginkgoExt.DescribeTable("deployed cert-manager components",
			func(name string, expected bool) {
				gomega.Expect(vzComponentPresent(name, "cert-manager")).To(gomega.Equal(expected))
			},
			ginkgoExt.Entry("includes cert-manager", "cert-manager", true),
			ginkgoExt.Entry("does include cert-manager-cainjector", "cert-manager-cainjector", true),
		)

		ginkgoExt.DescribeTable("deployed ingress components",
			func(name string, expected bool) {
				gomega.Expect(vzComponentPresent(name, "ingress-nginx")).To(gomega.Equal(expected))
			},
			ginkgoExt.Entry("includes ingress-controller-ingress-nginx-controller", "ingress-controller-ingress-nginx-controller", true),
		)

		if isManagedClusterProfile {
			ginkgoExt.DescribeTable("keycloak components are not deployed",
				func(name string, expected bool) {
					gomega.Expect(vzComponentPresent(name, "keycloak")).To(gomega.BeFalse())
				},
				ginkgoExt.Entry("includes ssoproxycontroller", "ssoproxycontroller", false),
			)
		} else {
			ginkgoExt.DescribeTable("deployed keycloak components",
				func(name string, expected bool) {
					gomega.Expect(vzComponentPresent(name, "keycloak")).To(gomega.Equal(expected))
				},
				ginkgoExt.Entry("includes ssoproxycontroller", "ssoproxycontroller", false),
			)
		}

		if isManagedClusterProfile {
			ginkgoExt.DescribeTable("rancher components are not deployed",
				func(name string, expected bool) {
					gomega.Expect(vzComponentPresent(name, "cattle-system")).To(gomega.BeFalse())
				},
				ginkgoExt.Entry("includes rancher", "rancher", false),
			)
		} else {
			ginkgoExt.DescribeTable("deployed rancher components",
				func(name string, expected bool) {
					gomega.Expect(vzComponentPresent(name, "cattle-system")).To(gomega.Equal(expected))
				},
				ginkgoExt.Entry("includes rancher", "rancher", true),
			)
		}

		ginkgoExt.DescribeTable("deployed VMI components",
			func(name string, expected bool) {
				gomega.Expect(vzComponentPresent(name, "verrazzano-system")).To(gomega.Equal(expected))
			},
			ginkgoExt.Entry("includes prometheus", "vmi-system-prometheus", true),
			ginkgoExt.Entry("includes prometheus-gw", "vmi-system-prometheus-gw", false),
			ginkgoExt.Entry("includes es-ingest", "vmi-system-es-ingest", isProdProfile),
			ginkgoExt.Entry("includes es-data", "vmi-system-es-data", isProdProfile),
			ginkgoExt.Entry("includes es-master", "vmi-system-es-master", !isManagedClusterProfile),
			ginkgoExt.Entry("includes es-kibana", "vmi-system-kibana", !isManagedClusterProfile),
			ginkgoExt.Entry("includes es-grafana", "vmi-system-grafana", !isManagedClusterProfile),
			ginkgoExt.Entry("includes verrazzano-console", "verrazzano-console", !isManagedClusterProfile),
		)

		ginkgo.It("Expected pods are running",
			func() {
				pkg.Concurrently(
					func() {
						// Rancher pods do not run on the managed cluster at install time (they do get started later when the managed
						// cluster is registered)
						if !isManagedClusterProfile {
							gomega.Eventually(func() bool { return pkg.PodsRunning("cattle-system", expectedPodsCattleSystem) }, waitTimeout, pollingInterval).
								Should(gomega.BeTrue())
						}
					},
					func() {
						if !isManagedClusterProfile {
							gomega.Eventually(func() bool { return pkg.PodsRunning("keycloak", expectedPodsKeycloak) }, waitTimeout, pollingInterval).
								Should(gomega.BeTrue())
						}
					},
					func() {
						gomega.Eventually(func() bool { return pkg.PodsRunning("cert-manager", expectedPodsCertManager) }, waitTimeout, pollingInterval).
							Should(gomega.BeTrue())
					},
					func() {
						gomega.Eventually(func() bool { return pkg.PodsRunning("ingress-nginx", expectedPodsIngressNginx) }, waitTimeout, pollingInterval).
							Should(gomega.BeTrue())
					},
					func() {
						gomega.Eventually(func() bool { return pkg.PodsRunning("verrazzano-system", expectedNonVMIPodsVerrazzanoSystem) }, waitTimeout, pollingInterval).
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
