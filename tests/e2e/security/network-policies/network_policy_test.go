// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package network_policy_test

import (
	"fmt"
	"github.com/onsi/gomega"
	"strings"
	"time"

	"github.com/onsi/ginkgo"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	connectTestCmdFmt = "timeout %d curl -v http://%s:%d 2>&1"
	connectedFmt      = "Connected to %s (%s) port %d"
	curlCode52        = "exit code 52"
	testNamespace     = "netpol-test"

	// Constants for various ports to scrape metrics
	ingressControllerMetricsPort = 10254
	envoyStatsMetricsPort        = 15090
	istiodMetricsPort            = 15014
	nodeExporterMetricsPort      = 9100
)

var (
	expectedPods    = []string{"netpol-test"}
	waitTimeout     = 3 * time.Minute
	pollingInterval = 30 * time.Second
)

var _ = ginkgo.BeforeSuite(func() {
	nsLabels := map[string]string{}
	if _, err := pkg.CreateNamespace(testNamespace, nsLabels); err != nil {
		ginkgo.Fail(fmt.Sprintf("Failed to create namespace: %v", err))
	}
	if err := pkg.CreateOrUpdateResourceFromFile("testdata/security/network-policies/netpol-test.yaml"); err != nil {
		ginkgo.Fail(fmt.Sprintf("Failed to create network policy test pod: %v", err))
	}
})

var failed = false
var _ = ginkgo.AfterEach(func() {
	failed = failed || ginkgo.CurrentGinkgoTestDescription().Failed
})

var _ = ginkgo.AfterSuite(func() {
	// undeploy the application here
	err := pkg.DeleteResourceFromFile("testdata/security/network-policies/netpol-test.yaml")
	if err != nil {
		ginkgo.Fail(fmt.Sprintf("Could not delete network policy test pod: %v\n", err.Error()))
	}
	err = pkg.DeleteNamespace(testNamespace)
	if err != nil {
		ginkgo.Fail(fmt.Sprintf("Could not delete namespace: %v\n", err.Error()))
	}
	if failed {
		err := pkg.ExecuteClusterDumpWithEnvVarConfig()
		if err != nil {
			pkg.Log(pkg.Error, fmt.Sprintf("Error dumping cluster %v", err))
		}
	}
})

var _ = ginkgo.Describe("Test Network Policies", func() {
	// Verify test pod is running
	// GIVEN netpol-test is deployed
	// WHEN the pod is created
	// THEN the expected pod must be running in the test namespace
	ginkgo.Describe("Verify test pod is running.", func() {
		ginkgo.It("and waiting for expected pod must be running", func() {
			gomega.Eventually(func() bool {
				return pkg.PodsRunning(testNamespace, expectedPods)
			}, waitTimeout, pollingInterval).Should(gomega.BeTrue())
		})
	})

	// GIVEN a Verrazzano deployment
	// WHEN access is attempted between pods within the ingress/egress rules of the Verrazzano network policies
	// THEN the attempted access should succeed
	ginkgo.It("Test NetworkPolicy Rules", func() {
		pkg.Concurrently(
			func() {
				pkg.Log(pkg.Info, "Test cattle-cluster-agent egress rules")
				err := testAccess(metav1.LabelSelector{MatchLabels: map[string]string{"app": "cattle-cluster-agent"}}, "cattle-system", metav1.LabelSelector{MatchLabels: map[string]string{"app": "node-exporter"}}, "monitoring", 6443, true)
				gomega.Expect(err).To(gomega.BeNil(), fmt.Sprintf("FAIL: Test cattle-cluster-agent egress failed: reason = %s", err))
				err = testAccess(metav1.LabelSelector{MatchLabels: map[string]string{"app": "cattle-cluster-agent"}}, "cattle-system", metav1.LabelSelector{MatchLabels: map[string]string{"app.kubernetes.io/instance": "ingress-controller"}}, "ingress-nginx", 443, true)
				gomega.Expect(err).To(gomega.BeNil(), fmt.Sprintf("FAIL: Test cattle-cluster-agent egress failed: reason = %s", err))
				err = testAccess(metav1.LabelSelector{MatchLabels: map[string]string{"app": "cattle-cluster-agent"}}, "cattle-system", metav1.LabelSelector{MatchLabels: map[string]string{"k8s-app": "kube-dns"}}, "kube-system", 53, true)
				gomega.Expect(err).To(gomega.BeNil(), fmt.Sprintf("FAIL: Test cattle-cluster-agent egress failed: reason = %s", err))
				err = testAccess(metav1.LabelSelector{MatchLabels: map[string]string{"app": "cattle-cluster-agent"}}, "cattle-system", metav1.LabelSelector{MatchLabels: map[string]string{"app.kubernetes.io/name": "keycloak"}}, "keycloak", 8080, true)
				gomega.Expect(err).To(gomega.BeNil(), fmt.Sprintf("FAIL: Test cattle-cluster-agent egress failed: reason = %s", err))
			},
			func() {
				pkg.Log(pkg.Info, "Test rancher ingress rules")
				err := testAccess(metav1.LabelSelector{MatchLabels: map[string]string{"app": "rancher"}}, "cattle-system", metav1.LabelSelector{MatchLabels: map[string]string{"app": "rancher"}}, "cattle-system", 80, true)
				gomega.Expect(err).To(gomega.BeNil(), fmt.Sprintf("FAIL: Test rancher egress failed: reason = %s", err))
				err = testAccess(metav1.LabelSelector{MatchLabels: map[string]string{"app.kubernetes.io/instance": "ingress-controller"}}, "ingress-nginx", metav1.LabelSelector{MatchLabels: map[string]string{"app": "rancher"}}, "cattle-system", 80, true)
				gomega.Expect(err).To(gomega.BeNil(), fmt.Sprintf("FAIL: Test rancher egress failed: reason = %s", err))
			},
			func() {
				pkg.Log(pkg.Info, "Test rancher egress rules")
				err := testAccess(metav1.LabelSelector{MatchLabels: map[string]string{"app": "rancher"}}, "cattle-system", metav1.LabelSelector{MatchLabels: map[string]string{"app": "node-exporter"}}, "monitoring", 6443, true)
				gomega.Expect(err).To(gomega.BeNil(), fmt.Sprintf("FAIL: Test rancher egress failed: reason = %s", err))
			},
			func() {
				pkg.Log(pkg.Info, "Test rancher-webhook ingress rules")
				err := testAccess(metav1.LabelSelector{MatchLabels: map[string]string{"app": "rancher"}}, "cattle-system", metav1.LabelSelector{MatchLabels: map[string]string{"app": "rancher-webhook"}}, "cattle-system", 9443, true)
				gomega.Expect(err).To(gomega.BeNil(), fmt.Sprintf("FAIL: Test rancher-webhook egress failed: reason = %s", err))
			},
			func() {
				pkg.Log(pkg.Info, "Test cert-manager ingress rules")
				err := testAccess(metav1.LabelSelector{MatchLabels: map[string]string{"app": "system-prometheus"}}, "verrazzano-system", metav1.LabelSelector{MatchLabels: map[string]string{"app": "cert-manager"}}, "cert-manager", 9402, true)
				gomega.Expect(err).To(gomega.BeNil(), fmt.Sprintf("FAIL: Test cert-manager ingress failed: reason = %s", err))
			},
			func() {
				pkg.Log(pkg.Info, "Test cert-manager egress rules")
				err := testAccess(metav1.LabelSelector{MatchLabels: map[string]string{"app": "cert-manager"}}, "cert-manager", metav1.LabelSelector{MatchLabels: map[string]string{"app": "node-exporter"}}, "monitoring", 6443, true)
				gomega.Expect(err).To(gomega.BeNil(), fmt.Sprintf("FAIL: Test cert-manager egress rules failed: reason = %s", err))
			},
			func() {
				pkg.Log(pkg.Info, "Test ingress-nginx-controller ingress rules")
				err := testAccess(metav1.LabelSelector{MatchLabels: map[string]string{"app": "system-prometheus"}}, "verrazzano-system", metav1.LabelSelector{MatchLabels: map[string]string{"app.kubernetes.io/component": "controller"}}, "ingress-nginx", 443, true)
				gomega.Expect(err).To(gomega.BeNil(), fmt.Sprintf("FAIL: Test ingress-nginx-controller ingress failed: reason = %s", err))
				err = testAccess(metav1.LabelSelector{MatchLabels: map[string]string{"app": "system-prometheus"}}, "verrazzano-system", metav1.LabelSelector{MatchLabels: map[string]string{"app.kubernetes.io/component": "controller"}}, "ingress-nginx", 80, true)
				gomega.Expect(err).To(gomega.BeNil(), fmt.Sprintf("FAIL: Test ingress-nginx-controller ingress failed: reason = %s", err))
				err = testAccess(metav1.LabelSelector{MatchLabels: map[string]string{"app": "system-prometheus"}}, "verrazzano-system", metav1.LabelSelector{MatchLabels: map[string]string{"app.kubernetes.io/component": "controller"}}, "ingress-nginx", ingressControllerMetricsPort, true)
				gomega.Expect(err).To(gomega.BeNil(), fmt.Sprintf("FAIL: Test ingress-nginx-controller ingress failed: reason = %s", err))
				err = testAccess(metav1.LabelSelector{MatchLabels: map[string]string{"app": "system-prometheus"}}, "verrazzano-system", metav1.LabelSelector{MatchLabels: map[string]string{"app.kubernetes.io/component": "controller"}}, "ingress-nginx", envoyStatsMetricsPort, true)
				gomega.Expect(err).To(gomega.BeNil(), fmt.Sprintf("FAIL: Test ingress-nginx-controller ingress failed: reason = %s", err))
			},
			func() {
				pkg.Log(pkg.Info, "Test ingress-nginx-controller egress rules")
				err := testAccess(metav1.LabelSelector{MatchLabels: map[string]string{"app.kubernetes.io/component": "controller"}}, "ingress-nginx", metav1.LabelSelector{MatchLabels: map[string]string{"app": "node-exporter"}}, "monitoring", 6443, true)
				gomega.Expect(err).To(gomega.BeNil(), fmt.Sprintf("FAIL: Test ingress-nginx-controller egress failed: reason = %s", err))
				err = testAccess(metav1.LabelSelector{MatchLabels: map[string]string{"app.kubernetes.io/component": "controller"}}, "ingress-nginx", metav1.LabelSelector{MatchLabels: map[string]string{"k8s-app": "kube-dns"}}, "kube-system", 53, true)
				gomega.Expect(err).To(gomega.BeNil(), fmt.Sprintf("FAIL: Test ingress-nginx-controller egress failed: reason = %s", err))
				err = testAccess(metav1.LabelSelector{MatchLabels: map[string]string{"app.kubernetes.io/component": "controller"}}, "ingress-nginx", metav1.LabelSelector{MatchLabels: map[string]string{"app": "verrazzano-api"}}, "verrazzano-system", 8775, true)
				gomega.Expect(err).To(gomega.BeNil(), fmt.Sprintf("FAIL: Test ingress-nginx-controller egress failed: reason = %s", err))
				err = testAccess(metav1.LabelSelector{MatchLabels: map[string]string{"app.kubernetes.io/component": "controller"}}, "ingress-nginx", metav1.LabelSelector{MatchLabels: map[string]string{"app": "verrazzano-console"}}, "verrazzano-system", 8000, true)
				gomega.Expect(err).To(gomega.BeNil(), fmt.Sprintf("FAIL: Test ingress-nginx-controller egress failed: reason = %s", err))
				err = testAccess(metav1.LabelSelector{MatchLabels: map[string]string{"app.kubernetes.io/component": "controller"}}, "ingress-nginx", metav1.LabelSelector{MatchLabels: map[string]string{"app": "rancher"}}, "cattle-system", 80, true)
				gomega.Expect(err).To(gomega.BeNil(), fmt.Sprintf("FAIL: Test ingress-nginx-controller egress failed: reason = %s", err))
				err = testAccess(metav1.LabelSelector{MatchLabels: map[string]string{"app.kubernetes.io/component": "controller"}}, "ingress-nginx", metav1.LabelSelector{MatchLabels: map[string]string{"app.kubernetes.io/name": "keycloak"}}, "keycloak", 8080, true)
				gomega.Expect(err).To(gomega.BeNil(), fmt.Sprintf("FAIL: Test ingress-nginx-controller egress failed: reason = %s", err))
			},
			func() {
				pkg.Log(pkg.Info, "Test ingress-nginx-default-backend ingress rules")
				err := testAccess(metav1.LabelSelector{MatchLabels: map[string]string{"app.kubernetes.io/component": "controller"}}, "ingress-nginx", metav1.LabelSelector{MatchLabels: map[string]string{"app.kubernetes.io/component": "default-backend"}}, "ingress-nginx", 8080, true)
				gomega.Expect(err).To(gomega.BeNil(), fmt.Sprintf("FAIL: Test ingress-nginx-default-backend ingress failed: reason = %s", err))
				err = testAccess(metav1.LabelSelector{MatchLabels: map[string]string{"app": "system-prometheus"}}, "verrazzano-system", metav1.LabelSelector{MatchLabels: map[string]string{"app.kubernetes.io/component": "default-backend"}}, "ingress-nginx", envoyStatsMetricsPort, true)
				gomega.Expect(err).To(gomega.BeNil(), fmt.Sprintf("FAIL: Test ingress-nginx-default-backend ingress failed: reason = %s", err))
			},
			func() {
				pkg.Log(pkg.Info, "Test istiod-verrazzano-system ingress rules")
				err := testAccess(metav1.LabelSelector{MatchLabels: map[string]string{"app": "system-prometheus"}}, "verrazzano-system", metav1.LabelSelector{MatchLabels: map[string]string{"app": "istiod"}}, "istio-system", 15012, true)
				gomega.Expect(err).To(gomega.BeNil(), fmt.Sprintf("FAIL: Test istiod-verrazzano-system ingress failed: reason = %s", err))
				err = testAccess(metav1.LabelSelector{MatchLabels: map[string]string{"app": "system-prometheus"}}, "verrazzano-system", metav1.LabelSelector{MatchLabels: map[string]string{"app": "istiod"}}, "istio-system", istiodMetricsPort, true)
				gomega.Expect(err).To(gomega.BeNil(), fmt.Sprintf("FAIL: Test istiod-verrazzano-system ingress failed: reason = %s", err))
			},
			func() {
				pkg.Log(pkg.Info, "Test keycloak ingress rules")
				err := testAccess(metav1.LabelSelector{MatchLabels: map[string]string{"app.kubernetes.io/instance": "ingress-controller"}}, "ingress-nginx", metav1.LabelSelector{MatchLabels: map[string]string{"app.kubernetes.io/name": "keycloak"}}, "keycloak", 8080, true)
				gomega.Expect(err).To(gomega.BeNil(), fmt.Sprintf("FAIL: Test keycloak ingress rules failed: reason = %s", err))
				err = testAccess(metav1.LabelSelector{MatchLabels: map[string]string{"app": "system-prometheus"}}, "verrazzano-system", metav1.LabelSelector{MatchLabels: map[string]string{"app.kubernetes.io/name": "keycloak"}}, "keycloak", envoyStatsMetricsPort, true)
				gomega.Expect(err).To(gomega.BeNil(), fmt.Sprintf("FAIL: Test keycloak ingress rules failed: reason = %s", err))
			},
			func() {
				pkg.Log(pkg.Info, "Test mysql ingress rules")
				err := testAccess(metav1.LabelSelector{MatchLabels: map[string]string{"app": "system-prometheus"}}, "verrazzano-system", metav1.LabelSelector{MatchLabels: map[string]string{"app": "mysql"}}, "keycloak", envoyStatsMetricsPort, true)
				gomega.Expect(err).To(gomega.BeNil(), fmt.Sprintf("FAIL: Test mysql ingress rules failed: reason = %s", err))
			},
			func() {
				pkg.Log(pkg.Info, "Test keycloak egress rules")
				err := testAccess(metav1.LabelSelector{MatchLabels: map[string]string{"app.kubernetes.io/name": "keycloak"}}, "keycloak", metav1.LabelSelector{MatchLabels: map[string]string{"k8s-app": "kube-dns"}}, "kube-system", 53, true)
				gomega.Expect(err).To(gomega.BeNil(), fmt.Sprintf("FAIL: Test keycloak egress rules failed: reason = %s", err))
				err = testAccess(metav1.LabelSelector{MatchLabels: map[string]string{"app.kubernetes.io/name": "keycloak"}}, "keycloak", metav1.LabelSelector{MatchLabels: map[string]string{"app": "mysql"}}, "keycloak", 3306, true)
				gomega.Expect(err).To(gomega.BeNil(), fmt.Sprintf("FAIL: Test keycloak egress rules failed: reason = %s", err))
			},
			func() {
				pkg.Log(pkg.Info, "Test monitoring egress rules")
				err := testAccess(metav1.LabelSelector{MatchLabels: map[string]string{"verrazzano.io/namespace": "monitoring"}}, "monitoring", metav1.LabelSelector{MatchLabels: map[string]string{"app": "system-prometheus"}}, "verrazzano-system", nodeExporterMetricsPort, true)
				gomega.Expect(err).To(gomega.BeNil(), fmt.Sprintf("FAIL: Test monitoring egress rules failed: reason = %s", err))
			},
			func() {
				pkg.Log(pkg.Info, "Test verrazzano-platform-operator ingress rules")
				err := testAccess(metav1.LabelSelector{MatchLabels: map[string]string{"app": "node-exporter"}}, "monitoring", metav1.LabelSelector{MatchLabels: map[string]string{"app": "verrazzano-platform-operator"}}, "verrazzano-install", 9443, true)
				gomega.Expect(err).To(gomega.BeNil(), fmt.Sprintf("FAIL: Test verrazzano-platform-operator ingress rules failed: reason = %s", err))
			},
			func() {
				pkg.Log(pkg.Info, "Test verrazzano-platform-operator egress rules")
				err := testAccess(metav1.LabelSelector{MatchLabels: map[string]string{"app": "verrazzano-platform-operator"}}, "verrazzano-install", metav1.LabelSelector{MatchLabels: map[string]string{"app": "node-exporter"}}, "monitoring", 6443, true)
				gomega.Expect(err).To(gomega.BeNil(), fmt.Sprintf("FAIL: Test verrazzano-platform-operator egress rules failed: reason = %s", err))
				err = testAccess(metav1.LabelSelector{MatchLabels: map[string]string{"app": "verrazzano-platform-operator"}}, "verrazzano-install", metav1.LabelSelector{MatchLabels: map[string]string{"k8s-app": "kube-dns"}}, "kube-system", 53, true)
				gomega.Expect(err).To(gomega.BeNil(), fmt.Sprintf("FAIL: Test verrazzano-platform-operator egress rules failed: reason = %s", err))
				err = testAccess(metav1.LabelSelector{MatchLabels: map[string]string{"app": "verrazzano-platform-operator"}}, "verrazzano-install", metav1.LabelSelector{MatchLabels: map[string]string{"app.kubernetes.io/instance": "ingress-controller"}}, "ingress-nginx", 443, true)
				gomega.Expect(err).To(gomega.BeNil(), fmt.Sprintf("FAIL: Test verrazzano-platform-operator egress rules failed: reason = %s", err))
			},
			func() {
				pkg.Log(pkg.Info, "Test oam-kubernetes-runtime egress rules")
				err := testAccess(metav1.LabelSelector{MatchLabels: map[string]string{"app.kubernetes.io/name": "oam-kubernetes-runtime"}}, "verrazzano-system", metav1.LabelSelector{MatchLabels: map[string]string{"app": "node-exporter"}}, "monitoring", 6443, true)
				gomega.Expect(err).To(gomega.BeNil(), fmt.Sprintf("FAIL: Test oam-kubernetes-runtime egress rules failed: reason = %s", err))
			},
			func() {
				pkg.Log(pkg.Info, "Test verrazzano-api ingress rules")
				err := testAccess(metav1.LabelSelector{MatchLabels: map[string]string{"app.kubernetes.io/instance": "ingress-controller"}}, "ingress-nginx", metav1.LabelSelector{MatchLabels: map[string]string{"app": "verrazzano-api"}}, "verrazzano-system", 8775, true)
				gomega.Expect(err).To(gomega.BeNil(), fmt.Sprintf("FAIL: Test verrazzano-api ingress rules failed: reason = %s", err))
				err = testAccess(metav1.LabelSelector{MatchLabels: map[string]string{"app": "system-prometheus"}}, "verrazzano-system", metav1.LabelSelector{MatchLabels: map[string]string{"app": "verrazzano-api"}}, "verrazzano-system", envoyStatsMetricsPort, true)
				gomega.Expect(err).To(gomega.BeNil(), fmt.Sprintf("FAIL: Test verrazzano-api ingress rules failed: reason = %s", err))
			},
			func() {
				pkg.Log(pkg.Info, "Test verrazzano-application-operator ingress rules")
				err := testAccess(metav1.LabelSelector{MatchLabels: map[string]string{"app": "node-exporter"}}, "monitoring", metav1.LabelSelector{MatchLabels: map[string]string{"app": "verrazzano-application-operator"}}, "verrazzano-system", 9443, true)
				gomega.Expect(err).To(gomega.BeNil(), fmt.Sprintf("FAIL: Test verrazzano-application-operator ingress rules failed: reason = %s", err))
			},
			func() {
				pkg.Log(pkg.Info, "Test verrazzano-application-operator egress rules")
				err := testAccess(metav1.LabelSelector{MatchLabels: map[string]string{"app": "verrazzano-application-operator"}}, "verrazzano-system", metav1.LabelSelector{MatchLabels: map[string]string{"app": "node-exporter"}}, "monitoring", 6443, true)
				gomega.Expect(err).To(gomega.BeNil(), fmt.Sprintf("FAIL: Test verrazzano-application-operator egress rules failed: reason = %s", err))
				err = testAccess(metav1.LabelSelector{MatchLabels: map[string]string{"app": "verrazzano-application-operator"}}, "verrazzano-system", metav1.LabelSelector{MatchLabels: map[string]string{"app.kubernetes.io/instance": "ingress-controller"}}, "ingress-nginx", 443, true)
				gomega.Expect(err).To(gomega.BeNil(), fmt.Sprintf("FAIL: Test verrazzano-application-operator egress rules failed: reason = %s", err))
			},
			func() {
				pkg.Log(pkg.Info, "Test verrazzano-console ingress rules")
				err := testAccess(metav1.LabelSelector{MatchLabels: map[string]string{"app.kubernetes.io/instance": "ingress-controller"}}, "ingress-nginx", metav1.LabelSelector{MatchLabels: map[string]string{"app": "verrazzano-console"}}, "verrazzano-system", 8000, true)
				gomega.Expect(err).To(gomega.BeNil(), fmt.Sprintf("FAIL: Test verrazzano-console ingress rules failed: reason = %s", err))
				err = testAccess(metav1.LabelSelector{MatchLabels: map[string]string{"app": "system-prometheus"}}, "verrazzano-system", metav1.LabelSelector{MatchLabels: map[string]string{"app": "verrazzano-console"}}, "verrazzano-system", envoyStatsMetricsPort, true)
				gomega.Expect(err).To(gomega.BeNil(), fmt.Sprintf("FAIL: Test verrazzano-console ingress rules failed: reason = %s", err))
			},
			func() {
				pkg.Log(pkg.Info, "Test node-exporter ingress rules")
				err := testAccess(metav1.LabelSelector{MatchLabels: map[string]string{"app": "system-prometheus"}}, "verrazzano-system", metav1.LabelSelector{MatchLabels: map[string]string{"verrazzano.io/namespace": "monitoring"}}, "monitoring", nodeExporterMetricsPort, true)
				gomega.Expect(err).To(gomega.BeNil(), fmt.Sprintf("FAIL: Test node-exporter ingress rules failed: reason = %s", err))
				err = testAccess(metav1.LabelSelector{MatchLabels: map[string]string{"app": "system-prometheus"}}, "verrazzano-system", metav1.LabelSelector{MatchLabels: map[string]string{"verrazzano.io/namespace": "monitoring"}}, "monitoring", envoyStatsMetricsPort, true)
				gomega.Expect(err).To(gomega.BeNil(), fmt.Sprintf("FAIL: Test node-exporter ingress rules failed: reason = %s", err))
			},
			func() {
				pkg.Log(pkg.Info, "Test istio-ingressgateway ingress rules")
				err := testAccess(metav1.LabelSelector{MatchLabels: map[string]string{"app": "system-prometheus"}}, "verrazzano-system", metav1.LabelSelector{MatchLabels: map[string]string{"app": "istio-ingressgateway"}}, "istio-system", envoyStatsMetricsPort, true)
				gomega.Expect(err).To(gomega.BeNil(), fmt.Sprintf("FAIL: Test istio-ingressgateway ingress rules failed: reason = %s", err))
			},
			func() {
				pkg.Log(pkg.Info, "Test istio-egressgateway ingress rules")
				err := testAccess(metav1.LabelSelector{MatchLabels: map[string]string{"app": "system-prometheus"}}, "verrazzano-system", metav1.LabelSelector{MatchLabels: map[string]string{"app": "istio-egressgateway"}}, "istio-system", envoyStatsMetricsPort, true)
				gomega.Expect(err).To(gomega.BeNil(), fmt.Sprintf("FAIL: Test istio-egressgateway ingress rules failed: reason = %s", err))
			},
			func() {
				pkg.Log(pkg.Info, "Test verrazzano-monitoring-operator egress rules")
				err := testAccess(metav1.LabelSelector{MatchLabels: map[string]string{"k8s-app": "verrazzano-monitoring-operator"}}, "verrazzano-system", metav1.LabelSelector{MatchLabels: map[string]string{"app": "node-exporter"}}, "monitoring", 6443, true)
				gomega.Expect(err).To(gomega.BeNil(), fmt.Sprintf("FAIL: Test verrazzano-monitoring-operator egress rules failed: reason = %s", err))
			},
			func() {
				pkg.Log(pkg.Info, "Test verrazzano-operator egress rules")
				err := testAccess(metav1.LabelSelector{MatchLabels: map[string]string{"app": "verrazzano-operator"}}, "verrazzano-system", metav1.LabelSelector{MatchLabels: map[string]string{"app": "node-exporter"}}, "monitoring", 6443, true)
				gomega.Expect(err).To(gomega.BeNil(), fmt.Sprintf("FAIL: Test verrazzano-operator egress rules failed: reason = %s", err))
			},
			func() {
				pkg.Log(pkg.Info, "Test vmi-system-es-master ingress rules")
				err := testAccess(metav1.LabelSelector{MatchLabels: map[string]string{"app.kubernetes.io/instance": "ingress-controller"}}, "ingress-nginx", metav1.LabelSelector{MatchLabels: map[string]string{"app": "system-es-master"}}, "verrazzano-system", 8775, true)
				gomega.Expect(err).To(gomega.BeNil(), fmt.Sprintf("FAIL: Test vmi-system-es-master ingress rules failed: reason = %s", err))
				err = testAccess(metav1.LabelSelector{MatchLabels: map[string]string{"app": "system-kibana"}}, "ingress-nginx", metav1.LabelSelector{MatchLabels: map[string]string{"app": "system-es-master"}}, "verrazzano-system", 9200, true)
				gomega.Expect(err).To(gomega.BeNil(), fmt.Sprintf("FAIL: Test vmi-system-es-master ingress rules failed: reason = %s", err))
				err = testAccess(metav1.LabelSelector{MatchLabels: map[string]string{"app": "fluentd"}}, "ingress-nginx", metav1.LabelSelector{MatchLabels: map[string]string{"app": "system-es-master"}}, "verrazzano-system", 8775, true)
				gomega.Expect(err).To(gomega.BeNil(), fmt.Sprintf("FAIL: Test vmi-system-es-master ingress rules failed: reason = %s", err))
				err = testAccess(metav1.LabelSelector{MatchLabels: map[string]string{"verrazzano.binding": "system"}}, "ingress-nginx", metav1.LabelSelector{MatchLabels: map[string]string{"app": "system-es-master"}}, "verrazzano-system", 9200, true)
				gomega.Expect(err).To(gomega.BeNil(), fmt.Sprintf("FAIL: Test vmi-system-es-master ingress rules failed: reason = %s", err))
				err = testAccess(metav1.LabelSelector{MatchLabels: map[string]string{"app": "system-prometheus"}}, "verrazzano-system", metav1.LabelSelector{MatchLabels: map[string]string{"app": "system-es-master"}}, "verrazzano-system", envoyStatsMetricsPort, true)
				gomega.Expect(err).To(gomega.BeNil(), fmt.Sprintf("FAIL: Test vmi-system-es-master ingress rules failed: reason = %s", err))
			},
			func() {
				pkg.Log(pkg.Info, "Test vmi-system-grafana ingress rules")
				err := testAccess(metav1.LabelSelector{MatchLabels: map[string]string{"app.kubernetes.io/instance": "ingress-controller"}}, "ingress-nginx", metav1.LabelSelector{MatchLabels: map[string]string{"app": "system-grafana"}}, "verrazzano-system", 8775, true)
				gomega.Expect(err).To(gomega.BeNil(), fmt.Sprintf("FAIL: Test vmi-system-grafana ingress rules failed: reason = %s", err))
				err = testAccess(metav1.LabelSelector{MatchLabels: map[string]string{"app": "system-prometheus"}}, "verrazzano-system", metav1.LabelSelector{MatchLabels: map[string]string{"app": "system-grafana"}}, "verrazzano-system", envoyStatsMetricsPort, true)
				gomega.Expect(err).To(gomega.BeNil(), fmt.Sprintf("FAIL: Test vmi-system-grafana ingress rules failed: reason = %s", err))

			},
			func() {
				pkg.Log(pkg.Info, "Test vmi-system-grafana egress rules")
				err := testAccess(metav1.LabelSelector{MatchLabels: map[string]string{"app": "system-grafana"}}, "verrazzano-system", metav1.LabelSelector{MatchLabels: map[string]string{"app": "system-prometheus"}}, "verrazzano-system", 9090, true)
				gomega.Expect(err).To(gomega.BeNil(), fmt.Sprintf("FAIL: Test vmi-system-grafana egress rules failed: reason = %s", err))
				err = testAccess(metav1.LabelSelector{MatchLabels: map[string]string{"app": "system-grafana"}}, "verrazzano-system", metav1.LabelSelector{MatchLabels: map[string]string{"k8s-app": "kube-dns"}}, "kube-system", 53, true)
				gomega.Expect(err).To(gomega.BeNil(), fmt.Sprintf("FAIL: Test vmi-system-grafana egress rules failed: reason = %s", err))
				err = testAccess(metav1.LabelSelector{MatchLabels: map[string]string{"app": "system-grafana"}}, "verrazzano-system", metav1.LabelSelector{MatchLabels: map[string]string{"app.kubernetes.io/name": "keycloak"}}, "keycloak", 8080, true)
				gomega.Expect(err).To(gomega.BeNil(), fmt.Sprintf("FAIL: Test vmi-system-grafana egress rules failed: reason = %s", err))
			},
			func() {
				pkg.Log(pkg.Info, "Test vmi-system-kibana ingress rules")
				err := testAccess(metav1.LabelSelector{MatchLabels: map[string]string{"app.kubernetes.io/instance": "ingress-controller"}}, "ingress-nginx", metav1.LabelSelector{MatchLabels: map[string]string{"app": "system-kibana"}}, "verrazzano-system", 8775, true)
				gomega.Expect(err).To(gomega.BeNil(), fmt.Sprintf("FAIL: Test vmi-system-kibana ingress rules failed: reason = %s", err))
				err = testAccess(metav1.LabelSelector{MatchLabels: map[string]string{"app": "system-prometheus"}}, "verrazzano-system", metav1.LabelSelector{MatchLabels: map[string]string{"app": "system-kibana"}}, "verrazzano-system", envoyStatsMetricsPort, true)
				gomega.Expect(err).To(gomega.BeNil(), fmt.Sprintf("FAIL: Test vmi-system-kibana ingress rules failed: reason = %s", err))
			},
			func() {
				pkg.Log(pkg.Info, "Test vmi-system-prometheus rules")
				err := testAccess(metav1.LabelSelector{MatchLabels: map[string]string{"app.kubernetes.io/instance": "ingress-controller"}}, "ingress-nginx", metav1.LabelSelector{MatchLabels: map[string]string{"app": "system-prometheus"}}, "verrazzano-system", 8775, true)
				gomega.Expect(err).To(gomega.BeNil(), fmt.Sprintf("FAIL: Test vmi-system-prometheus ingress rules failed: reason = %s", err))
			},
			func() {
				pkg.Log(pkg.Info, "Test weblogic-operator egress rules")
				err := testAccess(metav1.LabelSelector{MatchLabels: map[string]string{"app": "weblogic-operator"}}, "verrazzano-system", metav1.LabelSelector{MatchLabels: map[string]string{"app": "node-exporter"}}, "monitoring", 6443, true)
				gomega.Expect(err).To(gomega.BeNil(), fmt.Sprintf("FAIL: Test weblogic-operator egress rules failed: reason = %s", err))
			},
			func() {
				pkg.Log(pkg.Info, "Test weblogic-operator ingress rules")
				err := testAccess(metav1.LabelSelector{MatchLabels: map[string]string{"app": "system-prometheus"}}, "verrazzano-system", metav1.LabelSelector{MatchLabels: map[string]string{"app": "weblogic-operator"}}, "verrazzano-system", envoyStatsMetricsPort, true)
				gomega.Expect(err).To(gomega.BeNil(), fmt.Sprintf("FAIL: Test weblogic-operator ingress rules failed: reason = %s", err))
			},
		)
	})

	// GIVEN a Verrazzano deployment
	// WHEN access is attempted between pods that violate the ingress/egress rules of the Verrazzano network policies
	// THEN the attempted access should fail
	ginkgo.It("Negative Test NetworkPolicy Rules", func() {
		pkg.Concurrently(
			func() {
				pkg.Log(pkg.Info, "Negative test  rancher ingress rules")
				err := testAccess(metav1.LabelSelector{MatchLabels: map[string]string{"app": "netpol-test"}}, "netpol-test", metav1.LabelSelector{MatchLabels: map[string]string{"app": "rancher"}}, "cattle-system", 80, false)
				gomega.Expect(err).To(gomega.BeNil(), fmt.Sprintf("FAIL: Negative test  rancher egress failed: reason = %s", err))
			},
			func() {
				pkg.Log(pkg.Info, "Negative test cert-manager ingress rules")
				err := testAccess(metav1.LabelSelector{MatchLabels: map[string]string{"app": "netpol-test"}}, "netpol-test", metav1.LabelSelector{MatchLabels: map[string]string{"app": "cert-manager"}}, "cert-manager", 9402, false)
				gomega.Expect(err).To(gomega.BeNil(), fmt.Sprintf("FAIL: Negative test cert-manager ingress failed: reason = %s", err))
			},
			func() {
				pkg.Log(pkg.Info, "Negative test ingress-nginx-controller ingress rules")
				err := testAccess(metav1.LabelSelector{MatchLabels: map[string]string{"app": "netpol-test"}}, "netpol-test", metav1.LabelSelector{MatchLabels: map[string]string{"app.kubernetes.io/component": "controller"}}, "ingress-nginx", 80, false)
				gomega.Expect(err).To(gomega.BeNil(), fmt.Sprintf("FAIL: Negative test ingress-nginx-controller ingress failed: reason = %s", err))
			},
			func() {
				pkg.Log(pkg.Info, "Negative test ingress-nginx-default-backend ingress rules")
				err := testAccess(metav1.LabelSelector{MatchLabels: map[string]string{"app": "netpol-test"}}, "netpol-test", metav1.LabelSelector{MatchLabels: map[string]string{"app.kubernetes.io/component": "default-backend"}}, "ingress-nginx", 8080, false)
				gomega.Expect(err).To(gomega.BeNil(), fmt.Sprintf("FAIL: Negative test ingress-nginx-default-backend ingress failed: reason = %s", err))
			},
			func() {
				pkg.Log(pkg.Info, "Negative testistio-egressgateway ingress rules")
				err := testAccess(metav1.LabelSelector{MatchLabels: map[string]string{"app": "netpol-test"}}, "netpol-test", metav1.LabelSelector{MatchLabels: map[string]string{"app": "istio-egressgateway"}}, "istio-system", 6443, false)
				gomega.Expect(err).To(gomega.BeNil(), fmt.Sprintf("FAIL: Negative testistio-egressgateway ingress failed: reason = %s", err))
			},
			func() {
				pkg.Log(pkg.Info, "Negative testistio-ingressgateway ingress rules")
				err := testAccess(metav1.LabelSelector{MatchLabels: map[string]string{"app": "netpol-test"}}, "netpol-test", metav1.LabelSelector{MatchLabels: map[string]string{"app": "istio-ingressgateway"}}, "istio-system", 6443, false)
				gomega.Expect(err).To(gomega.BeNil(), fmt.Sprintf("FAIL: Negative testistio-ingressgateway ingress failed: reason = %s", err))
			},
			func() {
				pkg.Log(pkg.Info, "Negative test istiod-verrazzano-system ingress rules")
				err := testAccess(metav1.LabelSelector{MatchLabels: map[string]string{"app": "netpol-test"}}, "netpol-test", metav1.LabelSelector{MatchLabels: map[string]string{"app": "istiod"}}, "istio-system", 15012, false)
				gomega.Expect(err).To(gomega.BeNil(), fmt.Sprintf("FAIL: Negative test istiod-verrazzano-system ingress failed: reason = %s", err))
			},
			func() {
				pkg.Log(pkg.Info, "Negative test keycloak ingress rules")
				err := testAccess(metav1.LabelSelector{MatchLabels: map[string]string{"app": "netpol-test"}}, "netpol-test", metav1.LabelSelector{MatchLabels: map[string]string{"app.kubernetes.io/name": "keycloak"}}, "keycloak", 8080, false)
				gomega.Expect(err).To(gomega.BeNil(), fmt.Sprintf("FAIL: Negative test keycloak ingress rules failed: reason = %s", err))
				err = testAccess(metav1.LabelSelector{MatchLabels: map[string]string{"app": "netpol-test"}}, "netpol-test", metav1.LabelSelector{MatchLabels: map[string]string{"app.kubernetes.io/name": "keycloak"}}, "keycloak", 8080, false)
				gomega.Expect(err).To(gomega.BeNil(), fmt.Sprintf("FAIL: Negative test keycloak ingress rules failed: reason = %s", err))
			},
			func() {
				pkg.Log(pkg.Info, "Negative test verrazzano-api ingress rules")
				err := testAccess(metav1.LabelSelector{MatchLabels: map[string]string{"app": "netpol-test"}}, "netpol-test", metav1.LabelSelector{MatchLabels: map[string]string{"app": "verrazzano-api"}}, "verrazzano-system", 8775, false)
				gomega.Expect(err).To(gomega.BeNil(), fmt.Sprintf("FAIL: Negative test verrazzano-api ingress rules failed: reason = %s", err))
			},
			func() {
				pkg.Log(pkg.Info, "Negative test verrazzano-console ingress rules")
				err := testAccess(metav1.LabelSelector{MatchLabels: map[string]string{"app": "netpol-test"}}, "netpol-test", metav1.LabelSelector{MatchLabels: map[string]string{"app": "verrazzano-console"}}, "verrazzano-system", 8000, false)
				gomega.Expect(err).To(gomega.BeNil(), fmt.Sprintf("FAIL: Negative test verrazzano-console ingress rules failed: reason = %s", err))
			},
			func() {
				pkg.Log(pkg.Info, "Negative test vmi-system-es-master ingress rules")
				err := testAccess(metav1.LabelSelector{MatchLabels: map[string]string{"app": "netpol-test"}}, "netpol-test", metav1.LabelSelector{MatchLabels: map[string]string{"app": "system-es-master"}}, "verrazzano-system", 8775, false)
				gomega.Expect(err).To(gomega.BeNil(), fmt.Sprintf("FAIL: Negative test vmi-system-es-master ingress rules failed: reason = %s", err))
				err = testAccess(metav1.LabelSelector{MatchLabels: map[string]string{"app": "netpol-test"}}, "netpol-test", metav1.LabelSelector{MatchLabels: map[string]string{"app": "system-es-master"}}, "verrazzano-system", 9200, false)
				gomega.Expect(err).To(gomega.BeNil(), fmt.Sprintf("FAIL: Negative test vmi-system-es-master ingress rules failed: reason = %s", err))
				err = testAccess(metav1.LabelSelector{MatchLabels: map[string]string{"app": "netpol-test"}}, "netpol-test", metav1.LabelSelector{MatchLabels: map[string]string{"app": "system-es-master"}}, "verrazzano-system", 8775, false)
				gomega.Expect(err).To(gomega.BeNil(), fmt.Sprintf("FAIL: Negative test vmi-system-es-master ingress rules failed: reason = %s", err))
			},
			func() {
				pkg.Log(pkg.Info, "Negative test vmi-system-grafana ingress rules")
				err := testAccess(metav1.LabelSelector{MatchLabels: map[string]string{"app": "netpol-test"}}, "netpol-test", metav1.LabelSelector{MatchLabels: map[string]string{"app": "system-grafana"}}, "verrazzano-system", 8775, false)
				gomega.Expect(err).To(gomega.BeNil(), fmt.Sprintf("FAIL: Negative test vmi-system-grafana ingress rules failed: reason = %s", err))
			},
			func() {
				pkg.Log(pkg.Info, "Negative test vmi-system-kibana ingress rules")
				err := testAccess(metav1.LabelSelector{MatchLabels: map[string]string{"app": "netpol-test"}}, "netpol-test", metav1.LabelSelector{MatchLabels: map[string]string{"app": "system-kibana"}}, "verrazzano-system", 8775, false)
				gomega.Expect(err).To(gomega.BeNil(), fmt.Sprintf("FAIL: Negative test vmi-system-kibana ingress rules failed: reason = %s", err))
			},
			func() {
				pkg.Log(pkg.Info, "Negative test vmi-system-prometheus ingress rules")
				err := testAccess(metav1.LabelSelector{MatchLabels: map[string]string{"app": "netpol-test"}}, "netpol-test", metav1.LabelSelector{MatchLabels: map[string]string{"app": "system-prometheus"}}, "verrazzano-system", 8775, false)
				gomega.Expect(err).To(gomega.BeNil(), fmt.Sprintf("FAIL: Negative test vmi-system-prometheus ingress rules failed: reason = %s", err))
			},
		)
	})
})

// testAccess attempts to access a given pod from another pod on a given port and tests for the expected result
func testAccess(fromSelector metav1.LabelSelector, fromNamespace string, toSelector metav1.LabelSelector, toNamespace string, port int, expectAccess bool) error {
	// get the FROM pod
	pods, err := pkg.GetPodsFromSelector(&fromSelector, fromNamespace)
	if err != nil {
		return err
	}
	if len(pods) > 0 {
		fromPod := pods[0]
		// get the TO pod
		pods, err = pkg.GetPodsFromSelector(&toSelector, toNamespace)
		if err != nil {
			return err
		}
		if len(pods) > 0 {
			toPod := pods[0]
			access := attemptConnection(&fromPod, &toPod, port, 10)
			if access && !expectAccess {
				return fmt.Errorf(fmt.Sprintf("Should NOT be able to access pod %s from pod %s on port %d", toPod.Name, fromPod.Name, port))
			} else if !access && expectAccess {
				return fmt.Errorf(fmt.Sprintf("Should be able to access pod %s from pod %s on port %d", toPod.Name, fromPod.Name, port))
			}
		}
	}
	return nil
}

// attemptConnection attempts to access a given pod from another pod on a given port
func attemptConnection(fromPod, toPod *corev1.Pod, port int, duration time.Duration) bool {
	command := fmt.Sprintf(connectTestCmdFmt, duration, toPod.Status.PodIP, port)
	pkg.Log(pkg.Info, fmt.Sprintf("Executing command on pod %s.%s (%s)", fromPod.Namespace, fromPod.Name, command))
	stdout, _, err := pkg.Execute(fromPod.Name, fromPod.Spec.Containers[0].Name, fromPod.Namespace, []string{"sh", "-c", command})
	// check response for 'Connected' message; fail on error except for 'curl: (52) Empty reply from server'
	connected := strings.Contains(stdout, fmt.Sprintf(connectedFmt, toPod.Status.PodIP, toPod.Status.PodIP, port)) &&
		(err == nil || strings.Contains(fmt.Sprintf("%q", err), curlCode52))

	if connected {
		pkg.Log(pkg.Info, fmt.Sprintf("Connected from pod %s.%s to %s.%s on port %d", fromPod.Namespace, fromPod.Name, toPod.Namespace, toPod.Name, port))
	} else {
		pkg.Log(pkg.Info, fmt.Sprintf("Can NOT connect from pod %s.%s to %s.%s on port %d", fromPod.Namespace, fromPod.Name, toPod.Namespace, toPod.Name, port))
	}
	return connected
}
