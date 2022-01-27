// Copyright (c) 2020, 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package kubernetes_test

import (
	"fmt"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/verrazzano/verrazzano/pkg/k8sutil"
	"github.com/verrazzano/verrazzano/pkg/test/framework"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"time"
)

const waitTimeout = 15 * time.Minute
const pollingInterval = 30 * time.Second
const timeout5Min = 5 * time.Minute

var t = framework.NewTestFramework("kubernetes")

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

var _ = t.AfterEach(func() {})

var _ = t.Describe("In the Kubernetes Cluster", Label("f:platform-lcm.install"),
	func() {
		isManagedClusterProfile := pkg.IsManagedClusterProfile()
		isProdProfile := pkg.IsProdProfile()

		t.It("the expected number of nodes exist", func() {
			Eventually(func() (bool, error) {
				nodes, err := pkg.ListNodes()
				return nodes != nil && len(nodes.Items) >= 1, err
			}, timeout5Min, pollingInterval).Should(BeTrue())
		})

		t.It("the expected namespaces exist", func() {
			var namespaces *v1.NamespaceList
			Eventually(func() (*v1.NamespaceList, error) {
				var err error
				namespaces, err = pkg.ListNamespaces(metav1.ListOptions{})
				return namespaces, err
			}, timeout5Min, pollingInterval).ShouldNot(BeNil())

			if isManagedClusterProfile {
				Expect(nsListContains(namespaces.Items, "cattle-global-data")).To(BeFalse())
				Expect(nsListContains(namespaces.Items, "cattle-global-nt")).To(BeFalse())
				// Even though we do not install Rancher on managed clusters, we do create the namespace
				// so we can create network policies. Rancher will run pods in this namespace once
				// the managed cluster manifest YAML is applied to the managed cluster.
				Expect(nsListContains(namespaces.Items, "cattle-system")).To(BeTrue())
				Expect(nsListContains(namespaces.Items, "local")).To(BeFalse())
			} else {
				Expect(nsListContains(namespaces.Items, "cattle-system")).To(BeTrue())
				Expect(nsListContains(namespaces.Items, "cattle-global-data")).To(BeTrue())
				Expect(nsListContains(namespaces.Items, "cattle-global-nt")).To(BeTrue())
				Expect(nsListContains(namespaces.Items, "local")).To(BeTrue())
			}
			Expect(nsListContains(namespaces.Items, "istio-system")).To(BeTrue())
			Expect(nsListContains(namespaces.Items, "gitlab")).To(BeFalse())
			if isManagedClusterProfile {
				Expect(nsListContains(namespaces.Items, "keycloak")).To(BeFalse())
			} else {
				Expect(nsListContains(namespaces.Items, "keycloak")).To(BeTrue())
			}
			Expect(nsListContains(namespaces.Items, "verrazzano-system")).To(BeTrue())
			Expect(nsListContains(namespaces.Items, "verrazzano-mc")).To(BeTrue())
			Expect(nsListContains(namespaces.Items, "cert-manager")).To(BeTrue())
			Expect(nsListContains(namespaces.Items, "ingress-nginx")).To(BeTrue())
		})

		kubeconfigPath, _ := k8sutil.GetKubeConfigLocation()

		t.DescribeTable("Verrazzano components are deployed,",
			func(name string, expected bool) {
				Eventually(func() (bool, error) {
					return vzComponentPresent(name, "verrazzano-system")
				}, waitTimeout, pollingInterval).Should(Equal(expected))
			},
			t.Entry("includes verrazzano-operator", "verrazzano-operator", true),
			t.Entry("does not include verrazzano-web", "verrazzano-web", false),
			t.Entry("includes verrazzano-console", "verrazzano-console", !isManagedClusterProfile),
			t.Entry("does not include verrazzano-ldap", "verrazzano-ldap", false),
			t.Entry("does not include verrazzano-cluster-operator", "verrazzano-cluster-operator", false),
			t.Entry("includes verrazzano-monitoring-operator", "verrazzano-monitoring-operator", true),
			t.Entry("Check weblogic-operator deployment", "weblogic-operator", pkg.IsWebLogicOperatorEnabled(kubeconfigPath)),
			t.Entry("Check coherence-operator deployment", "coherence-operator", pkg.IsCoherenceOperatorEnabled(kubeconfigPath)),
		)

		t.DescribeTable("cert-manager components are deployed,",
			func(name string, expected bool) {
				Eventually(func() (bool, error) {
					return vzComponentPresent(name, "cert-manager")
				}, waitTimeout, pollingInterval).Should(Equal(expected))
			},
			t.Entry("includes cert-manager", "cert-manager", true),
			t.Entry("does include cert-manager-cainjector", "cert-manager-cainjector", true),
		)

		t.DescribeTable("ingress components are deployed,",
			func(name string, expected bool) {
				Eventually(func() (bool, error) {
					return vzComponentPresent(name, "ingress-nginx")
				}, waitTimeout, pollingInterval).Should(Equal(expected))
			},
			t.Entry("includes ingress-controller-ingress-nginx-controller", "ingress-controller-ingress-nginx-controller", true),
		)

		t.DescribeTable("keycloak components are not deployed,",
			func(name string, expected bool) {
				Eventually(func() (bool, error) {
					return vzComponentPresent(name, "keycloak")
				}, waitTimeout, pollingInterval).Should(Equal(expected))
			},
			t.Entry("includes ssoproxycontroller", "ssoproxycontroller", false),
		)

		if isManagedClusterProfile {
			t.DescribeTable("rancher components are not deployed,",
				func(name string, expected bool) {
					Eventually(func() (bool, error) {
						return vzComponentPresent(name, "cattle-system")
					}, waitTimeout, pollingInterval).Should(Equal(expected))
				},
				t.Entry("includes rancher", "rancher", false),
			)
		} else {
			t.DescribeTable("rancher components are deployed,",
				func(name string, expected bool) {
					Eventually(func() (bool, error) {
						return vzComponentPresent(name, "cattle-system")
					}, waitTimeout, pollingInterval).Should(Equal(expected))
				},
				t.Entry("includes rancher", "rancher", true),
			)
		}

		t.DescribeTable("VMI components are deployed,",
			func(name string, expected bool) {
				Eventually(func() (bool, error) {
					return vzComponentPresent(name, "verrazzano-system")
				}, waitTimeout, pollingInterval).Should(Equal(expected))
			},
			t.Entry("includes prometheus", "vmi-system-prometheus", true),
			t.Entry("includes prometheus-gw", "vmi-system-prometheus-gw", false),
			t.Entry("includes es-ingest", "vmi-system-es-ingest", isProdProfile),
			t.Entry("includes es-data", "vmi-system-es-data", isProdProfile),
			t.Entry("includes es-master", "vmi-system-es-master", !isManagedClusterProfile),
			t.Entry("includes es-kibana", "vmi-system-kibana", !isManagedClusterProfile),
			t.Entry("includes es-grafana", "vmi-system-grafana", !isManagedClusterProfile),
			t.Entry("includes verrazzano-console", "verrazzano-console", !isManagedClusterProfile),
		)

		// Test components that may not exist for older versions
		t.DescribeTable("VMI components that don't exist in older versions are deployed,",
			func(name string, expected bool) {
				Eventually(func() (bool, error) {
					ok, _ := pkg.IsVerrazzanoMinVersion("1.1.0")
					if !ok {
						// skip test
						fmt.Printf("Skipping Kiali check since version < 1.1.0")
						return expected, nil
					}
					return vzComponentPresent(name, "verrazzano-system")
				}, waitTimeout, pollingInterval).Should(Equal(expected))
			},
			t.Entry("includes kiali", "vmi-system-kiali", !isManagedClusterProfile),
		)

		t.It("the expected pods are running", func() {
			pkg.Concurrently(
				func() {
					// Rancher pods do not run on the managed cluster at install time (they do get started later when the managed
					// cluster is registered)
					if !isManagedClusterProfile {
						Eventually(func() bool { return pkg.PodsRunning("cattle-system", expectedPodsCattleSystem) }, waitTimeout, pollingInterval).
							Should(BeTrue())
					}
				},
				func() {
					if !isManagedClusterProfile {
						Eventually(func() bool { return pkg.PodsRunning("keycloak", expectedPodsKeycloak) }, waitTimeout, pollingInterval).
							Should(BeTrue())
					}
				},
				func() {
					Eventually(func() bool { return pkg.PodsRunning("cert-manager", expectedPodsCertManager) }, waitTimeout, pollingInterval).
						Should(BeTrue())
				},
				func() {
					Eventually(func() bool { return pkg.PodsRunning("ingress-nginx", expectedPodsIngressNginx) }, waitTimeout, pollingInterval).
						Should(BeTrue())
				},
				func() {
					Eventually(func() bool { return pkg.PodsRunning("verrazzano-system", expectedNonVMIPodsVerrazzanoSystem) }, waitTimeout, pollingInterval).
						Should(BeTrue())
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

func vzComponentPresent(name string, namespace string) (bool, error) {
	return pkg.DoesPodExist(namespace, name)
}
