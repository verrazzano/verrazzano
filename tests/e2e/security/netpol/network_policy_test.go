// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package netpol

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
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
	kialiMetricsPort             = 9090
)

var (
	expectedPods         = []string{"netpol-test"}
	waitTimeout          = 3 * time.Minute
	pollingInterval      = 30 * time.Second
	shortWaitTimeout     = 30 * time.Second
	shortPollingInterval = 10 * time.Second
)

var _ = BeforeSuite(func() {
	Eventually(func() (*corev1.Namespace, error) {
		nsLabels := map[string]string{}
		return pkg.CreateNamespace(testNamespace, nsLabels)
	}, waitTimeout, pollingInterval).ShouldNot(BeNil())

	Eventually(func() error {
		return pkg.CreateOrUpdateResourceFromFile("testdata/security/network-policies/netpol-test.yaml")
	}, waitTimeout, pollingInterval).ShouldNot(HaveOccurred())
})

var failed = false
var _ = AfterEach(func() {
	failed = failed || CurrentGinkgoTestDescription().Failed
})

var _ = AfterSuite(func() {
	// undeploy the application here
	Eventually(func() error {
		return pkg.DeleteResourceFromFile("testdata/security/network-policies/netpol-test.yaml")
	}, waitTimeout, pollingInterval).ShouldNot(HaveOccurred())

	Eventually(func() error {
		return pkg.DeleteNamespace(testNamespace)
	}, waitTimeout, pollingInterval).ShouldNot(HaveOccurred())

	if failed {
		err := pkg.ExecuteClusterDumpWithEnvVarConfig()
		if err != nil {
			pkg.Log(pkg.Error, fmt.Sprintf("Error dumping cluster %v", err))
		}
	}
})

var _ = Describe("Test Network Policies", func() {
	// Verify test pod is running
	// GIVEN netpol-test is deployed
	// WHEN the pod is created
	// THEN the expected pod must be running in the test namespace
	Describe("Verify test pod is running.", func() {
		It("and waiting for expected pod must be running", func() {
			Eventually(func() bool {
				return pkg.PodsRunning(testNamespace, expectedPods)
			}, waitTimeout, pollingInterval).Should(BeTrue())
		})
	})

	// GIVEN a Verrazzano deployment
	// WHEN access is attempted between pods within the ingress rules of the Verrazzano network policies
	// THEN the attempted access should succeed
	It("Test NetworkPolicy Rules", func() {
		pkg.Concurrently(
			func() {
				pkg.Log(pkg.Info, "Test rancher ingress rules")
				err := testAccess(metav1.LabelSelector{MatchLabels: map[string]string{"app": "rancher"}}, "cattle-system", metav1.LabelSelector{MatchLabels: map[string]string{"app": "rancher"}}, "cattle-system", 80, true)
				Expect(err).To(BeNil(), fmt.Sprintf("FAIL: Test rancher ingress failed: reason = %s", err))
				err = testAccess(metav1.LabelSelector{MatchLabels: map[string]string{"app.kubernetes.io/instance": "ingress-controller"}}, "ingress-nginx", metav1.LabelSelector{MatchLabels: map[string]string{"app": "rancher"}}, "cattle-system", 80, true)
				Expect(err).To(BeNil(), fmt.Sprintf("FAIL: Test rancher ingress failed: reason = %s", err))
			},
			func() {
				pkg.Log(pkg.Info, "Test rancher-webhook ingress rules")
				err := testAccess(metav1.LabelSelector{MatchLabels: map[string]string{"app": "rancher"}}, "cattle-system", metav1.LabelSelector{MatchLabels: map[string]string{"app": "rancher-webhook"}}, "cattle-system", 9443, true)
				Expect(err).To(BeNil(), fmt.Sprintf("FAIL: Test rancher-webhook ingress failed: reason = %s", err))
			},
			func() {
				pkg.Log(pkg.Info, "Test cert-manager ingress rules")
				err := testAccess(metav1.LabelSelector{MatchLabels: map[string]string{"app": "system-prometheus"}}, "verrazzano-system", metav1.LabelSelector{MatchLabels: map[string]string{"app": "cert-manager"}}, "cert-manager", 9402, true)
				Expect(err).To(BeNil(), fmt.Sprintf("FAIL: Test cert-manager ingress failed: reason = %s", err))
			},
			func() {
				pkg.Log(pkg.Info, "Test ingress-nginx-controller ingress rules")
				err := testAccess(metav1.LabelSelector{MatchLabels: map[string]string{"app": "system-prometheus"}}, "verrazzano-system", metav1.LabelSelector{MatchLabels: map[string]string{"app.kubernetes.io/component": "controller"}}, "ingress-nginx", 443, true)
				Expect(err).To(BeNil(), fmt.Sprintf("FAIL: Test ingress-nginx-controller ingress failed: reason = %s", err))
				err = testAccess(metav1.LabelSelector{MatchLabels: map[string]string{"app": "system-prometheus"}}, "verrazzano-system", metav1.LabelSelector{MatchLabels: map[string]string{"app.kubernetes.io/component": "controller"}}, "ingress-nginx", 80, true)
				Expect(err).To(BeNil(), fmt.Sprintf("FAIL: Test ingress-nginx-controller ingress failed: reason = %s", err))
				err = testAccess(metav1.LabelSelector{MatchLabels: map[string]string{"app": "system-prometheus"}}, "verrazzano-system", metav1.LabelSelector{MatchLabels: map[string]string{"app.kubernetes.io/component": "controller"}}, "ingress-nginx", ingressControllerMetricsPort, true)
				Expect(err).To(BeNil(), fmt.Sprintf("FAIL: Test ingress-nginx-controller ingress failed: reason = %s", err))
				err = testAccess(metav1.LabelSelector{MatchLabels: map[string]string{"app": "system-prometheus"}}, "verrazzano-system", metav1.LabelSelector{MatchLabels: map[string]string{"app.kubernetes.io/component": "controller"}}, "ingress-nginx", envoyStatsMetricsPort, true)
				Expect(err).To(BeNil(), fmt.Sprintf("FAIL: Test ingress-nginx-controller ingress failed: reason = %s", err))
			},
			func() {
				pkg.Log(pkg.Info, "Test ingress-nginx-default-backend ingress rules")
				err := testAccess(metav1.LabelSelector{MatchLabels: map[string]string{"app.kubernetes.io/component": "controller"}}, "ingress-nginx", metav1.LabelSelector{MatchLabels: map[string]string{"app.kubernetes.io/component": "default-backend"}}, "ingress-nginx", 8080, true)
				Expect(err).To(BeNil(), fmt.Sprintf("FAIL: Test ingress-nginx-default-backend ingress failed: reason = %s", err))
				err = testAccess(metav1.LabelSelector{MatchLabels: map[string]string{"app": "system-prometheus"}}, "verrazzano-system", metav1.LabelSelector{MatchLabels: map[string]string{"app.kubernetes.io/component": "default-backend"}}, "ingress-nginx", envoyStatsMetricsPort, true)
				Expect(err).To(BeNil(), fmt.Sprintf("FAIL: Test ingress-nginx-default-backend ingress failed: reason = %s", err))
			},
			func() {
				pkg.Log(pkg.Info, "Test istiod-verrazzano-system ingress rules")
				err := testAccess(metav1.LabelSelector{MatchLabels: map[string]string{"app": "system-prometheus"}}, "verrazzano-system", metav1.LabelSelector{MatchLabels: map[string]string{"app": "istiod"}}, "istio-system", 15012, true)
				Expect(err).To(BeNil(), fmt.Sprintf("FAIL: Test istiod-verrazzano-system ingress failed: reason = %s", err))
				err = testAccess(metav1.LabelSelector{MatchLabels: map[string]string{"app": "system-prometheus"}}, "verrazzano-system", metav1.LabelSelector{MatchLabels: map[string]string{"app": "istiod"}}, "istio-system", istiodMetricsPort, true)
				Expect(err).To(BeNil(), fmt.Sprintf("FAIL: Test istiod-verrazzano-system ingress failed: reason = %s", err))
			},
			func() {
				pkg.Log(pkg.Info, "Test keycloak ingress rules")
				err := testAccess(metav1.LabelSelector{MatchLabels: map[string]string{"app.kubernetes.io/instance": "ingress-controller"}}, "ingress-nginx", metav1.LabelSelector{MatchLabels: map[string]string{"app.kubernetes.io/name": "keycloak"}}, "keycloak", 8080, true)
				Expect(err).To(BeNil(), fmt.Sprintf("FAIL: Test keycloak ingress rules failed: reason = %s", err))
				err = testAccess(metav1.LabelSelector{MatchLabels: map[string]string{"app": "system-prometheus"}}, "verrazzano-system", metav1.LabelSelector{MatchLabels: map[string]string{"app.kubernetes.io/name": "keycloak"}}, "keycloak", envoyStatsMetricsPort, true)
				Expect(err).To(BeNil(), fmt.Sprintf("FAIL: Test keycloak ingress rules failed: reason = %s", err))
			},
			func() {
				pkg.Log(pkg.Info, "Test mysql ingress rules")
				err := testAccess(metav1.LabelSelector{MatchLabels: map[string]string{"app": "system-prometheus"}}, "verrazzano-system", metav1.LabelSelector{MatchLabels: map[string]string{"app": "mysql"}}, "keycloak", envoyStatsMetricsPort, true)
				Expect(err).To(BeNil(), fmt.Sprintf("FAIL: Test mysql ingress rules failed: reason = %s", err))
			},
			func() {
				pkg.Log(pkg.Info, "Test verrazzano-platform-operator ingress rules")
				err := testAccess(metav1.LabelSelector{MatchLabels: map[string]string{"app": "node-exporter"}}, "monitoring", metav1.LabelSelector{MatchLabels: map[string]string{"app": "verrazzano-platform-operator"}}, "verrazzano-install", 9443, true)
				Expect(err).To(BeNil(), fmt.Sprintf("FAIL: Test verrazzano-platform-operator ingress rules failed: reason = %s", err))
			},
			func() {
				pkg.Log(pkg.Info, "Test coherence-operator ingress rules")
				err := testAccess(metav1.LabelSelector{MatchLabels: map[string]string{"app": "netpol-test"}}, "netpol-test", metav1.LabelSelector{MatchLabels: map[string]string{"control-plane": "coherence"}}, "verrazzano-system", 9443, true)
				Expect(err).To(BeNil(), fmt.Sprintf("FAIL: Test coherence-operator ingress rules failed: reason = %s", err))
				err = testAccess(metav1.LabelSelector{MatchLabels: map[string]string{"app": "netpol-test"}}, "netpol-test", metav1.LabelSelector{MatchLabels: map[string]string{"control-plane": "coherence"}}, "verrazzano-system", 8000, true)
				Expect(err).To(BeNil(), fmt.Sprintf("FAIL: Test coherence-operator ingress rules failed: reason = %s", err))
			},
			func() {
				pkg.Log(pkg.Info, "Test verrazzano-application-operator ingress rules")
				err := testAccess(metav1.LabelSelector{MatchLabels: map[string]string{"app": "netpol-test"}}, "netpol-test", metav1.LabelSelector{MatchLabels: map[string]string{"app": "verrazzano-application-operator"}}, "verrazzano-system", 9443, true)
				Expect(err).To(BeNil(), fmt.Sprintf("FAIL: Test verrazzano-application-operator ingress rules failed: reason = %s", err))
				err = testAccess(metav1.LabelSelector{MatchLabels: map[string]string{"app": "node-exporter"}}, "monitoring", metav1.LabelSelector{MatchLabels: map[string]string{"app": "verrazzano-application-operator"}}, "verrazzano-system", 9443, true)
				Expect(err).To(BeNil(), fmt.Sprintf("FAIL: Test verrazzano-application-operator ingress rules failed: reason = %s", err))
			},
			func() {
				pkg.Log(pkg.Info, "Test verrazzano-authproxy ingress rules")
				err := testAccess(metav1.LabelSelector{MatchLabels: map[string]string{"app.kubernetes.io/instance": "ingress-controller"}}, "ingress-nginx", metav1.LabelSelector{MatchLabels: map[string]string{"app": "verrazzano-authproxy"}}, "verrazzano-system", 8775, true)
				Expect(err).To(BeNil(), fmt.Sprintf("FAIL: Test verrazzano-authproxy ingress rules failed: reason = %s", err))
				err = testAccess(metav1.LabelSelector{MatchLabels: map[string]string{"app": "fluentd"}}, "verrazzano-system", metav1.LabelSelector{MatchLabels: map[string]string{"app": "verrazzano-authproxy"}}, "verrazzano-system", 8775, true)
				Expect(err).To(BeNil(), fmt.Sprintf("FAIL: Test verrazzano-authproxy ingress rules failed: reason = %s", err))
				err = testAccess(metav1.LabelSelector{MatchLabels: map[string]string{"app": "system-prometheus"}}, "verrazzano-system", metav1.LabelSelector{MatchLabels: map[string]string{"app": "verrazzano-authproxy"}}, "verrazzano-system", envoyStatsMetricsPort, true)
				Expect(err).To(BeNil(), fmt.Sprintf("FAIL: Test verrazzano-authproxy ingress rules failed: reason = %s", err))
			},
			func() {
				pkg.Log(pkg.Info, "Test verrazzano-console ingress rules")
				err := testAccess(metav1.LabelSelector{MatchLabels: map[string]string{"app": "verrazzano-authproxy"}}, "verrazzano-system", metav1.LabelSelector{MatchLabels: map[string]string{"app": "verrazzano-console"}}, "verrazzano-system", 8000, true)
				Expect(err).To(BeNil(), fmt.Sprintf("FAIL: Test verrazzano-console ingress rules failed: reason = %s", err))
				err = testAccess(metav1.LabelSelector{MatchLabels: map[string]string{"app": "system-prometheus"}}, "verrazzano-system", metav1.LabelSelector{MatchLabels: map[string]string{"app": "verrazzano-console"}}, "verrazzano-system", envoyStatsMetricsPort, true)
				Expect(err).To(BeNil(), fmt.Sprintf("FAIL: Test verrazzano-console ingress rules failed: reason = %s", err))
			},
			func() {
				pkg.Log(pkg.Info, "Test node-exporter ingress rules")
				err := testAccess(metav1.LabelSelector{MatchLabels: map[string]string{"app": "system-prometheus"}}, "verrazzano-system", metav1.LabelSelector{MatchLabels: map[string]string{"app": "node-exporter"}}, "monitoring", nodeExporterMetricsPort, true)
				Expect(err).To(BeNil(), fmt.Sprintf("FAIL: Test node-exporter ingress rules failed: reason = %s", err))
			},
			func() {
				pkg.Log(pkg.Info, "Test istio-ingressgateway ingress rules")
				err := testAccess(metav1.LabelSelector{MatchLabels: map[string]string{"app": "system-prometheus"}}, "verrazzano-system", metav1.LabelSelector{MatchLabels: map[string]string{"app": "istio-ingressgateway"}}, "istio-system", envoyStatsMetricsPort, true)
				Expect(err).To(BeNil(), fmt.Sprintf("FAIL: Test istio-ingressgateway ingress rules failed: reason = %s", err))
			},
			func() {
				pkg.Log(pkg.Info, "Test istio-egressgateway ingress rules")
				err := testAccess(metav1.LabelSelector{MatchLabels: map[string]string{"app": "system-prometheus"}}, "verrazzano-system", metav1.LabelSelector{MatchLabels: map[string]string{"app": "istio-egressgateway"}}, "istio-system", envoyStatsMetricsPort, true)
				Expect(err).To(BeNil(), fmt.Sprintf("FAIL: Test istio-egressgateway ingress rules failed: reason = %s", err))
			},
			func() {
				pkg.Log(pkg.Info, "Test vmi-system-es-master ingress rules")
				err := testAccess(metav1.LabelSelector{MatchLabels: map[string]string{"app": "system-prometheus"}}, "verrazzano-system", metav1.LabelSelector{MatchLabels: map[string]string{"app": "system-es-master"}}, "verrazzano-system", envoyStatsMetricsPort, true)
				Expect(err).To(BeNil(), fmt.Sprintf("FAIL: Test vmi-system-es-master ingress rules failed: reason = %s", err))
				/* TODO:
				The following tests only work in verrazzano prod profile. There is a differnce in network policies used in prod and
				dev profile. Once that is resolved, the following lines can be uncommented. They have been tested to work in prod profile.
				*/
				// err = testAccess(metav1.LabelSelector{MatchLabels: map[string]string{"app": "system-es-data"}}, "verrazzano-system", metav1.LabelSelector{MatchLabels: map[string]string{"app": "system-es-master"}}, "verrazzano-system", 9300, true)
				// Expect(err).To(BeNil(), fmt.Sprintf("FAIL: Test vmi-system-es-master ingress rules failed: reason = %s", err))
				// err = testAccess(metav1.LabelSelector{MatchLabels: map[string]string{"app": "system-es-ingest"}}, "verrazzano-system", metav1.LabelSelector{MatchLabels: map[string]string{"app": "system-es-master"}}, "verrazzano-system", 9300, true)
				// Expect(err).To(BeNil(), fmt.Sprintf("FAIL: Test vmi-system-es-master ingress rules failed: reason = %s", err))
			},
			/* TODO:
			The following tests only work in verrazzano prod profile. There is a differnce in network policies used in prod and
			dev profile. Once that is resolved, the following lines can be uncommented. They have been tested to work in prod profile.
			*/
			// func() {
			// 	pkg.Log(pkg.Info, "Test vmi-system-es-data ingress rules")
			// 	err := testAccess(metav1.LabelSelector{MatchLabels: map[string]string{"app": "system-es-master"}}, "verrazzano-system", metav1.LabelSelector{MatchLabels: map[string]string{"app": "system-es-data"}}, "verrazzano-system", 9300, true)
			// 	Expect(err).To(BeNil(), fmt.Sprintf("FAIL: Test vmi-system-es-data ingress rules failed: reason = %s", err))
			// 	err = testAccess(metav1.LabelSelector{MatchLabels: map[string]string{"app": "system-es-ingest"}}, "verrazzano-system", metav1.LabelSelector{MatchLabels: map[string]string{"app": "system-es-data"}}, "verrazzano-system", 9200, true)
			// 	Expect(err).To(BeNil(), fmt.Sprintf("FAIL: Test vmi-system-es-data ingress rules failed: reason = %s", err))
			// 	err = testAccess(metav1.LabelSelector{MatchLabels: map[string]string{"app": "system-es-ingest"}}, "verrazzano-system", metav1.LabelSelector{MatchLabels: map[string]string{"app": "system-es-data"}}, "verrazzano-system", 9300, true)
			// 	Expect(err).To(BeNil(), fmt.Sprintf("FAIL: Test vmi-system-es-data ingress rules failed: reason = %s", err))
			// 	err = testAccess(metav1.LabelSelector{MatchLabels: map[string]string{"app": "system-kibana"}}, "verrazzano-system", metav1.LabelSelector{MatchLabels: map[string]string{"app": "system-es-data"}}, "verrazzano-system", 9200, true)
			// 	Expect(err).To(BeNil(), fmt.Sprintf("FAIL: Test vmi-system-es-data ingress rules failed: reason = %s", err))
			// 	err = testAccess(metav1.LabelSelector{MatchLabels: map[string]string{"app": "system-prometheus"}}, "verrazzano-system", metav1.LabelSelector{MatchLabels: map[string]string{"app": "system-es-data"}}, "verrazzano-system", envoyStatsMetricsPort, true)
			// 	Expect(err).To(BeNil(), fmt.Sprintf("FAIL: Test vmi-system-es-data ingress rules failed: reason = %s", err))
			// },
			// func() {
			// 	pkg.Log(pkg.Info, "Test vmi-system-es-ingest ingress rules")
			// 	err := testAccess(metav1.LabelSelector{MatchLabels: map[string]string{"app": "system-es-master"}}, "verrazzano-system", metav1.LabelSelector{MatchLabels: map[string]string{"app": "system-es-ingest"}}, "verrazzano-system", 9300, true)
			// 	Expect(err).To(BeNil(), fmt.Sprintf("FAIL: Test vmi-system-es-ingest ingress rules failed: reason = %s", err))
			// 	err = testAccess(metav1.LabelSelector{MatchLabels: map[string]string{"app": "system-es-data"}}, "verrazzano-system", metav1.LabelSelector{MatchLabels: map[string]string{"app": "system-es-ingest"}}, "verrazzano-system", 9300, true)
			// 	Expect(err).To(BeNil(), fmt.Sprintf("FAIL: Test vmi-system-es-ingest ingress rules failed: reason = %s", err))
			// 	err = testAccess(metav1.LabelSelector{MatchLabels: map[string]string{"app": "system-kibana"}}, "verrazzano-system", metav1.LabelSelector{MatchLabels: map[string]string{"app": "system-es-ingest"}}, "verrazzano-system", 9200, true)
			// 	Expect(err).To(BeNil(), fmt.Sprintf("FAIL: Test vmi-system-es-ingest ingress rules failed: reason = %s", err))
			// 	err = testAccess(metav1.LabelSelector{MatchLabels: map[string]string{"app": "system-prometheus"}}, "verrazzano-system", metav1.LabelSelector{MatchLabels: map[string]string{"app": "system-es-ingest"}}, "verrazzano-system", envoyStatsMetricsPort, true)
			// 	Expect(err).To(BeNil(), fmt.Sprintf("FAIL: Test vmi-system-es-ingest ingress rules failed: reason = %s", err))
			// },
			func() {
				pkg.Log(pkg.Info, "Test vmi-system-grafana ingress rules")
				err := testAccess(metav1.LabelSelector{MatchLabels: map[string]string{"app": "verrazzano-authproxy"}}, "verrazzano-system", metav1.LabelSelector{MatchLabels: map[string]string{"app": "system-grafana"}}, "verrazzano-system", 3000, true)
				Expect(err).To(BeNil(), fmt.Sprintf("FAIL: Test vmi-system-grafana ingress rules failed: reason = %s", err))
				err = testAccess(metav1.LabelSelector{MatchLabels: map[string]string{"app": "system-prometheus"}}, "verrazzano-system", metav1.LabelSelector{MatchLabels: map[string]string{"app": "system-grafana"}}, "verrazzano-system", envoyStatsMetricsPort, true)
				Expect(err).To(BeNil(), fmt.Sprintf("FAIL: Test vmi-system-grafana ingress rules failed: reason = %s", err))

			},
			func() {
				pkg.Log(pkg.Info, "Test vmi-system-kibana ingress rules")
				err := testAccess(metav1.LabelSelector{MatchLabels: map[string]string{"app": "verrazzano-authproxy"}}, "verrazzano-system", metav1.LabelSelector{MatchLabels: map[string]string{"app": "system-kibana"}}, "verrazzano-system", 5601, true)
				Expect(err).To(BeNil(), fmt.Sprintf("FAIL: Test vmi-system-kibana ingress rules failed: reason = %s", err))
				err = testAccess(metav1.LabelSelector{MatchLabels: map[string]string{"app": "system-prometheus"}}, "verrazzano-system", metav1.LabelSelector{MatchLabels: map[string]string{"app": "system-kibana"}}, "verrazzano-system", envoyStatsMetricsPort, true)
				Expect(err).To(BeNil(), fmt.Sprintf("FAIL: Test vmi-system-kibana ingress rules failed: reason = %s", err))
			},
			func() {
				pkg.Log(pkg.Info, "Test vmi-system-prometheus ingress rules")
				err := testAccess(metav1.LabelSelector{MatchLabels: map[string]string{"app": "verrazzano-authproxy"}}, "verrazzano-system", metav1.LabelSelector{MatchLabels: map[string]string{"app": "system-prometheus"}}, "verrazzano-system", 9090, true)
				Expect(err).To(BeNil(), fmt.Sprintf("FAIL: Test vmi-system-prometheus ingress rules failed: reason = %s", err))
				err = testAccess(metav1.LabelSelector{MatchLabels: map[string]string{"app": "system-grafana"}}, "verrazzano-system", metav1.LabelSelector{MatchLabels: map[string]string{"app": "system-prometheus"}}, "verrazzano-system", 9090, true)
				Expect(err).To(BeNil(), fmt.Sprintf("FAIL: Test vmi-system-prometheus ingress rules failed: reason = %s", err))
			},
			func() {
				pkg.Log(pkg.Info, "Test node-exporter ingress rules")
				err := testAccess(metav1.LabelSelector{MatchLabels: map[string]string{"app": "system-prometheus"}}, "verrazzano-system", metav1.LabelSelector{MatchLabels: map[string]string{"app": "node-exporter"}}, "monitoring", 9100, true)
				Expect(err).To(BeNil(), fmt.Sprintf("FAIL: Test node-exporter ingress rules failed: reason = %s", err))
			},
			func() {
				pkg.Log(pkg.Info, "Test weblogic-operator ingress rules")
				err := testAccess(metav1.LabelSelector{MatchLabels: map[string]string{"app": "istio-ingressgateway"}}, "istio-system", metav1.LabelSelector{MatchLabels: map[string]string{"app": "weblogic-operator"}}, "verrazzano-system", envoyStatsMetricsPort, true)
				Expect(err).To(BeNil(), fmt.Sprintf("FAIL: Test weblogic-operator ingress rules failed: reason = %s", err))
				err = testAccess(metav1.LabelSelector{MatchLabels: map[string]string{"app": "istio-egressgateway"}}, "istio-system", metav1.LabelSelector{MatchLabels: map[string]string{"app": "weblogic-operator"}}, "verrazzano-system", envoyStatsMetricsPort, true)
				Expect(err).To(BeNil(), fmt.Sprintf("FAIL: Test weblogic-operator ingress rules failed: reason = %s", err))
				err = testAccess(metav1.LabelSelector{MatchLabels: map[string]string{"app": "istiod"}}, "istio-system", metav1.LabelSelector{MatchLabels: map[string]string{"app": "weblogic-operator"}}, "verrazzano-system", envoyStatsMetricsPort, true)
				Expect(err).To(BeNil(), fmt.Sprintf("FAIL: Test weblogic-operator ingress rules failed: reason = %s", err))
				err = testAccess(metav1.LabelSelector{MatchLabels: map[string]string{"app": "system-prometheus"}}, "verrazzano-system", metav1.LabelSelector{MatchLabels: map[string]string{"app": "weblogic-operator"}}, "verrazzano-system", envoyStatsMetricsPort, true)
				Expect(err).To(BeNil(), fmt.Sprintf("FAIL: Test weblogic-operator ingress rules failed: reason = %s", err))
			},
			func() {
				pkg.Log(pkg.Info, "Test kiali ingress rules")
				err := testAccess(metav1.LabelSelector{MatchLabels: map[string]string{"app": "verrazzano-authproxy"}}, "verrazzano-system", metav1.LabelSelector{MatchLabels: map[string]string{"app": "kiali"}}, "verrazzano-system", 20001, true)
				Expect(err).To(BeNil(), fmt.Sprintf("FAIL: Test Kiali network ingress from verrazzano-authproxy failed: reason = %s", err))
				err = testAccess(metav1.LabelSelector{MatchLabels: map[string]string{"app": "system-prometheus"}}, "verrazzano-system", metav1.LabelSelector{MatchLabels: map[string]string{"app": "kiali"}}, "verrazzano-system", kialiMetricsPort, true)
				Expect(err).To(BeNil(), fmt.Sprintf("FAIL: Test Kiali network ingress from prometheus failed: reason = %s", err))
				err = testAccess(metav1.LabelSelector{MatchLabels: map[string]string{"app": "istiod"}}, "istio-system", metav1.LabelSelector{MatchLabels: map[string]string{"app": "kiali"}}, "verrazzano-system", envoyStatsMetricsPort, true)
				Expect(err).To(BeNil(), fmt.Sprintf("FAIL: Test Kiali network ingress from istiod failed: reason = %s", err))
			},
		)
	})

	// GIVEN a Verrazzano deployment
	// WHEN access is attempted between pods that violate the rules of the Verrazzano network policies
	// THEN the attempted access should fail
	It("Negative Test NetworkPolicy Rules", func() {
		pkg.Concurrently(
			func() {
				pkg.Log(pkg.Info, "Negative test rancher ingress rules")
				err := testAccess(metav1.LabelSelector{MatchLabels: map[string]string{"app": "netpol-test"}}, "netpol-test", metav1.LabelSelector{MatchLabels: map[string]string{"app": "rancher"}}, "cattle-system", 80, false)
				Expect(err).To(BeNil(), fmt.Sprintf("FAIL: Negative test  rancher ingress failed: reason = %s", err))
			},
			func() {
				pkg.Log(pkg.Info, "Negative test cert-manager ingress rules")
				err := testAccess(metav1.LabelSelector{MatchLabels: map[string]string{"app": "netpol-test"}}, "netpol-test", metav1.LabelSelector{MatchLabels: map[string]string{"app": "cert-manager"}}, "cert-manager", 9402, false)
				Expect(err).To(BeNil(), fmt.Sprintf("FAIL: Negative test cert-manager ingress failed: reason = %s", err))
			},
			func() {
				pkg.Log(pkg.Info, "Negative test ingress-nginx-controller ingress rules")
				err := testAccess(metav1.LabelSelector{MatchLabels: map[string]string{"app": "netpol-test"}}, "netpol-test", metav1.LabelSelector{MatchLabels: map[string]string{"app.kubernetes.io/component": "controller"}}, "ingress-nginx", 80, false)
				Expect(err).To(BeNil(), fmt.Sprintf("FAIL: Negative test ingress-nginx-controller ingress failed: reason = %s", err))
			},
			func() {
				pkg.Log(pkg.Info, "Negative test ingress-nginx-default-backend ingress rules")
				err := testAccess(metav1.LabelSelector{MatchLabels: map[string]string{"app": "netpol-test"}}, "netpol-test", metav1.LabelSelector{MatchLabels: map[string]string{"app.kubernetes.io/component": "default-backend"}}, "ingress-nginx", 8080, false)
				Expect(err).To(BeNil(), fmt.Sprintf("FAIL: Negative test ingress-nginx-default-backend ingress failed: reason = %s", err))
			},
			func() {
				pkg.Log(pkg.Info, "Negative test istio-egressgateway ingress rules")
				err := testAccess(metav1.LabelSelector{MatchLabels: map[string]string{"app": "netpol-test"}}, "netpol-test", metav1.LabelSelector{MatchLabels: map[string]string{"app": "istio-egressgateway"}}, "istio-system", 6443, false)
				Expect(err).To(BeNil(), fmt.Sprintf("FAIL: Negative testistio-egressgateway ingress failed: reason = %s", err))
			},
			func() {
				pkg.Log(pkg.Info, "Negative test istio-ingressgateway ingress rules")
				err := testAccess(metav1.LabelSelector{MatchLabels: map[string]string{"app": "netpol-test"}}, "netpol-test", metav1.LabelSelector{MatchLabels: map[string]string{"app": "istio-ingressgateway"}}, "istio-system", 6443, false)
				Expect(err).To(BeNil(), fmt.Sprintf("FAIL: Negative testistio-ingressgateway ingress failed: reason = %s", err))
			},
			func() {
				pkg.Log(pkg.Info, "Negative test istiod-verrazzano-system ingress rules")
				err := testAccess(metav1.LabelSelector{MatchLabels: map[string]string{"app": "netpol-test"}}, "netpol-test", metav1.LabelSelector{MatchLabels: map[string]string{"app": "istiod"}}, "istio-system", 15012, false)
				Expect(err).To(BeNil(), fmt.Sprintf("FAIL: Negative test istiod-verrazzano-system ingress failed: reason = %s", err))
			},
			func() {
				pkg.Log(pkg.Info, "Negative test keycloak ingress rules")
				err := testAccess(metav1.LabelSelector{MatchLabels: map[string]string{"app": "netpol-test"}}, "netpol-test", metav1.LabelSelector{MatchLabels: map[string]string{"app.kubernetes.io/name": "keycloak"}}, "keycloak", 8080, false)
				Expect(err).To(BeNil(), fmt.Sprintf("FAIL: Negative test keycloak ingress rules failed: reason = %s", err))
			},
			func() {
				pkg.Log(pkg.Info, "Negative test oam-kubernetes-runtime ingress rules")
				err := testAccess(metav1.LabelSelector{MatchLabels: map[string]string{"app": "netpol-test"}}, "netpol-test", metav1.LabelSelector{MatchLabels: map[string]string{"app.kubernetes.io/name": "oam-kubernetes-runtime"}}, "verrazzano-system", 8775, false)
				Expect(err).To(BeNil(), fmt.Sprintf("FAIL: Negative test oam-kubernetes-runtime ingress rules failed: reason = %s", err))
			},
			func() {
				pkg.Log(pkg.Info, "Negative test verrazzano-authproxy ingress rules")
				err := testAccess(metav1.LabelSelector{MatchLabels: map[string]string{"app": "netpol-test"}}, "netpol-test", metav1.LabelSelector{MatchLabels: map[string]string{"app": "verrazzano-authproxy"}}, "verrazzano-system", 8775, false)
				Expect(err).To(BeNil(), fmt.Sprintf("FAIL: Negative test verrazzano-authproxy ingress rules failed: reason = %s", err))
			},
			func() {
				pkg.Log(pkg.Info, "Negative test verrazzano-console ingress rules")
				err := testAccess(metav1.LabelSelector{MatchLabels: map[string]string{"app": "netpol-test"}}, "netpol-test", metav1.LabelSelector{MatchLabels: map[string]string{"app": "verrazzano-console"}}, "verrazzano-system", 8000, false)
				Expect(err).To(BeNil(), fmt.Sprintf("FAIL: Negative test verrazzano-console ingress rules failed: reason = %s", err))
			},
			func() {
				pkg.Log(pkg.Info, "Negative test verrazzano-monitoring-operator ingress rules")
				err := testAccess(metav1.LabelSelector{MatchLabels: map[string]string{"app": "netpol-test"}}, "netpol-test", metav1.LabelSelector{MatchLabels: map[string]string{"k8s-app": "verrazzano-monitoring-operator"}}, "verrazzano-system", 8000, false)
				Expect(err).To(BeNil(), fmt.Sprintf("FAIL: Negative test verrazzano-monitoring-operator ingress rules failed: reason = %s", err))
			},
			func() {
				pkg.Log(pkg.Info, "Negative test verrazzano-operator ingress rules")
				err := testAccess(metav1.LabelSelector{MatchLabels: map[string]string{"app": "netpol-test"}}, "netpol-test", metav1.LabelSelector{MatchLabels: map[string]string{"app": "verrazzano-operator"}}, "verrazzano-system", 8000, false)
				Expect(err).To(BeNil(), fmt.Sprintf("FAIL: Negative test verrazzano-operator ingress rules failed: reason = %s", err))
			},
			func() {
				pkg.Log(pkg.Info, "Negative test vmi-system-es-master ingress rules")
				err := testAccess(metav1.LabelSelector{MatchLabels: map[string]string{"app": "netpol-test"}}, "netpol-test", metav1.LabelSelector{MatchLabels: map[string]string{"app": "system-es-master"}}, "verrazzano-system", 9200, false)
				Expect(err).To(BeNil(), fmt.Sprintf("FAIL: Negative test vmi-system-es-master ingress rules failed: reason = %s", err))
			},
			/* TODO:
			The following tests only work in verrazzano prod profile. There is a differnce in network policies used in prod and
			dev profile. Once that is resolved, the following lines can be uncommented. They have been tested to work in prod profile.
			*/
			// func() {
			// 	pkg.Log(pkg.Info, "Negative test vmi-system-es-data ingress rules")
			// 	err := testAccess(metav1.LabelSelector{MatchLabels: map[string]string{"app": "netpol-test"}}, "netpol-test", metav1.LabelSelector{MatchLabels: map[string]string{"app": "system-es-data"}}, "verrazzano-system", 9200, false)
			// 	Expect(err).To(BeNil(), fmt.Sprintf("FAIL: Negative test vmi-system-es-data ingress rules failed: reason = %s", err))
			// },
			// func() {
			// 	pkg.Log(pkg.Info, "Negative test vmi-system-es-ingest ingress rules")
			// 	err := testAccess(metav1.LabelSelector{MatchLabels: map[string]string{"app": "netpol-test"}}, "netpol-test", metav1.LabelSelector{MatchLabels: map[string]string{"app": "system-es-ingest"}}, "verrazzano-system", 9200, false)
			// 	Expect(err).To(BeNil(), fmt.Sprintf("FAIL: Negative test vmi-system-es-ingest ingress rules failed: reason = %s", err))
			// },
			func() {
				pkg.Log(pkg.Info, "Negative test vmi-system-grafana ingress rules")
				err := testAccess(metav1.LabelSelector{MatchLabels: map[string]string{"app": "netpol-test"}}, "netpol-test", metav1.LabelSelector{MatchLabels: map[string]string{"app": "system-grafana"}}, "verrazzano-system", 3000, false)
				Expect(err).To(BeNil(), fmt.Sprintf("FAIL: Negative test vmi-system-grafana ingress rules failed: reason = %s", err))
			},
			func() {
				pkg.Log(pkg.Info, "Negative test vmi-system-kibana ingress rules")
				err := testAccess(metav1.LabelSelector{MatchLabels: map[string]string{"app": "netpol-test"}}, "netpol-test", metav1.LabelSelector{MatchLabels: map[string]string{"app": "system-kibana"}}, "verrazzano-system", 5601, false)
				Expect(err).To(BeNil(), fmt.Sprintf("FAIL: Negative test vmi-system-kibana ingress rules failed: reason = %s", err))
			},
			func() {
				pkg.Log(pkg.Info, "Negative test vmi-system-prometheus ingress rules")
				err := testAccess(metav1.LabelSelector{MatchLabels: map[string]string{"app": "netpol-test"}}, "netpol-test", metav1.LabelSelector{MatchLabels: map[string]string{"app": "system-prometheus"}}, "verrazzano-system", 9090, false)
				Expect(err).To(BeNil(), fmt.Sprintf("FAIL: Negative test vmi-system-prometheus ingress rules failed: reason = %s", err))
			},
			func() {
				pkg.Log(pkg.Info, "Negative test weblogic-operator ingress rules")
				err := testAccess(metav1.LabelSelector{MatchLabels: map[string]string{"app": "netpol-test"}}, "netpol-test", metav1.LabelSelector{MatchLabels: map[string]string{"app": "weblogic-operator"}}, "verrazzano-system", 8000, false)
				Expect(err).To(BeNil(), fmt.Sprintf("FAIL: Negative test weblogic-operator ingress rules failed: reason = %s", err))
			},
			func() {
				pkg.Log(pkg.Info, "Negative test kiali ingress rules")
				err := testAccess(metav1.LabelSelector{MatchLabels: map[string]string{"app": "netpol-test"}}, "netpol-test", metav1.LabelSelector{MatchLabels: map[string]string{"app": "kiali"}}, "verrazzano-system", kialiMetricsPort, false)
				Expect(err).To(BeNil(), fmt.Sprintf("FAIL: Negative test kiali ingress rules failed: reason = %s", err))
			},
		)
	})
})

// testAccess attempts to access a given pod from another pod on a given port and tests for the expected result
func testAccess(fromSelector metav1.LabelSelector, fromNamespace string, toSelector metav1.LabelSelector, toNamespace string, port int, expectAccess bool) error {
	// get the FROM pod
	var pods []corev1.Pod
	Eventually(func() error {
		var err error
		pods, err = pkg.GetPodsFromSelector(&fromSelector, fromNamespace)
		return err
	}, waitTimeout, pollingInterval).ShouldNot(HaveOccurred())
	mapFromSelector, _ := metav1.LabelSelectorAsMap(&fromSelector)
	jsonFromSelector, _ := json.Marshal(mapFromSelector)
	Expect(len(pods) > 0).To(BeTrue(), fmt.Sprintf("FAIL: Pod not found with label: %s in namespace: %s", jsonFromSelector, fromNamespace))
	fromPod := pods[0]

	// get the TO pod
	Eventually(func() error {
		var err error
		pods, err = pkg.GetPodsFromSelector(&toSelector, toNamespace)
		return err
	}, waitTimeout, pollingInterval).ShouldNot(HaveOccurred())
	mapToSelector, _ := metav1.LabelSelectorAsMap(&toSelector)
	jsonToSelector, _ := json.Marshal(mapToSelector)
	Expect(len(pods) > 0).To(BeTrue(), fmt.Sprintf("FAIL: Pod not found with label: %s in namespace: %s", jsonToSelector, toNamespace))
	toPod := pods[0]

	if expectAccess {
		Eventually(func() bool {
			return attemptConnection(&fromPod, &toPod, port, 10)
		}, waitTimeout, pollingInterval).Should(BeTrue(), fmt.Sprintf("Should be able to access pod %s from pod %s on port %d", toPod.Name, fromPod.Name, port))
	} else {
		Consistently(func() bool {
			return attemptConnection(&fromPod, &toPod, port, 10)
		}, shortWaitTimeout, shortPollingInterval).Should(BeFalse(), fmt.Sprintf("Should NOT be able to access pod %s from pod %s on port %d", toPod.Name, fromPod.Name, port))
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
