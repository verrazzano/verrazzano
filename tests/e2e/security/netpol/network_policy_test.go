// Copyright (c) 2021, 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package netpol

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	vzconst "github.com/verrazzano/verrazzano/pkg/constants"
	"github.com/verrazzano/verrazzano/pkg/k8sutil"
	"github.com/verrazzano/verrazzano/pkg/test/framework"
	"github.com/verrazzano/verrazzano/pkg/test/framework/metrics"
	"github.com/verrazzano/verrazzano/platform-operator/constants"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/istio"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
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
	mysqlPort                    = 3306
	istiodMetricsPort            = 15014
	nodeExporterMetricsPort      = 9100

	kubernetesAppLabel  = "app.kubernetes.io/name"
	kubernetesCompLabel = "app.kubernetes.io/component"
	nodeExporter        = "prometheus-node-exporter"
	defaultBackend      = "default-backend"
	vzConsole           = "verrazzano-console"
	grafanaSys          = "system-grafana"
	kibanaSys           = "system-kibana"
	weblogicOperator    = "weblogic-operator"
)

// accessCheckConfig is the configuration used for the NetworkPolicy access check
type accessCheckConfig struct {
	// pod label selector for pods sending network traffic
	fromSelector metav1.LabelSelector
	// namespace of pod sending network traffic
	fromNamespace string
	// pod label selector for pods receiving network traffic
	toSelector metav1.LabelSelector
	// namespace of pod receiving network traffic
	toNamespace string
	// port that on the to pod that is tested for access
	port int
	// indicates if network traffic should be allowed
	expectAccess bool
	// ignore if pods not found
	ignorePodsNotFound bool
}

var (
	expectedPods             = []string{"netpol-test"}
	expectedPodsHelloHelidon = []string{"hello-helidon-deployment"}
	waitTimeout              = 15 * time.Minute
	pollingInterval          = 30 * time.Second
	shortWaitTimeout         = 30 * time.Second
	shortPollingInterval     = 10 * time.Second
	generatedNamespace       = pkg.GenerateNamespace("hello-helidon")
)

var t = framework.NewTestFramework("netpol")
var clusterDump = pkg.NewClusterDumpWrapper(generatedNamespace)

var _ = clusterDump.BeforeSuite(func() {
	start := time.Now()
	Eventually(func() (*corev1.Namespace, error) {
		nsLabels := map[string]string{}
		return pkg.CreateNamespace(testNamespace, nsLabels)
	}, waitTimeout, pollingInterval).ShouldNot(BeNil())
	Eventually(func() error {
		return pkg.CreateOrUpdateResourceFromFile("testdata/security/network-policies/netpol-test.yaml")
	}, waitTimeout, pollingInterval).ShouldNot(HaveOccurred())
	metrics.Emit(t.Metrics.With("deployment_elapsed_time", time.Since(start).Milliseconds()))

	start = time.Now()
	pkg.DeployHelloHelidonApplication(namespace, "", istioInjection, "")

	t.Logs.Info("Verify test pod is running")
	Eventually(func() bool {
		result, err := pkg.PodsRunning(testNamespace, expectedPods)
		if err != nil {
			AbortSuite(fmt.Sprintf("One or more pods are not running in the namespace: %v, error: %v", testNamespace, err))
		}
		return result
	}, waitTimeout, pollingInterval).Should(BeTrue())

	t.Logs.Info("hello-helidon pod")
	Eventually(func() bool {
		result, err := pkg.PodsRunning(namespace, expectedPodsHelloHelidon)
		if err != nil {
			AbortSuite(fmt.Sprintf("One or more pods are not running in the namespace: %v, error: %v", testNamespace, err))
		}
		return result
	}, waitTimeout, pollingInterval).Should(BeTrue())
	metrics.Emit(t.Metrics.With("deployment_elapsed_time", time.Since(start).Milliseconds()))
})

var _ = clusterDump.AfterEach(func() {}) // Dump cluster if spec fails
var _ = clusterDump.AfterSuite(func() {  // Dump cluster if aftersuite fails
	// undeploy the applications here
	start := time.Now()
	Eventually(func() error {
		return pkg.DeleteResourceFromFile("testdata/security/network-policies/netpol-test.yaml")
	}, waitTimeout, pollingInterval).ShouldNot(HaveOccurred())
	Eventually(func() error {
		return pkg.DeleteNamespace(testNamespace)
	}, waitTimeout, pollingInterval).ShouldNot(HaveOccurred())
	metrics.Emit(t.Metrics.With("undeployment_elapsed_time", time.Since(start).Milliseconds()))

	start = time.Now()
	pkg.UndeployHelloHelidonApplication(namespace, "")
	metrics.Emit(t.Metrics.With("undeployment_elapsed_time", time.Since(start).Milliseconds()))
})

var _ = t.Describe("Test Network Policies", Label("f:security.netpol"), func() {

	// GIVEN a Verrazzano deployment
	// WHEN access is attempted between pods within the ingress rules of the Verrazzano network policies
	// THEN the attempted access should succeed
	t.It("Test NetworkPolicy Rules", func() {
		pkg.Concurrently(
			func() {
				t.Logs.Info("Test rancher ingress rules")
				err := testAccess(metav1.LabelSelector{MatchLabels: map[string]string{"app": "rancher"}}, "cattle-system", metav1.LabelSelector{MatchLabels: map[string]string{"app": "rancher"}}, "cattle-system", 80, true)
				Expect(err).To(BeNil(), fmt.Sprintf("FAIL: Test rancher ingress failed: reason = %s", err))
				err = testAccess(metav1.LabelSelector{MatchLabels: map[string]string{"app.kubernetes.io/instance": "ingress-controller"}}, "ingress-nginx", metav1.LabelSelector{MatchLabels: map[string]string{"app": "rancher"}}, "cattle-system", 80, true)
				Expect(err).To(BeNil(), fmt.Sprintf("FAIL: Test rancher ingress failed: reason = %s", err))
			},
			func() {
				t.Logs.Info("Test rancher-webhook ingress rules")
				err := testAccess(metav1.LabelSelector{MatchLabels: map[string]string{"app": "rancher"}}, "cattle-system", metav1.LabelSelector{MatchLabels: map[string]string{"app": "rancher-webhook"}}, "cattle-system", 9443, true)
				Expect(err).To(BeNil(), fmt.Sprintf("FAIL: Test rancher-webhook ingress failed: reason = %s", err))
			},
			func() {
				t.Logs.Info("Test cert-manager ingress rules")
				err := testAccess(metav1.LabelSelector{MatchLabels: map[string]string{kubernetesAppLabel: "prometheus"}}, vzconst.PrometheusOperatorNamespace, metav1.LabelSelector{MatchLabels: map[string]string{"app": vzconst.CertManagerNamespace}}, vzconst.CertManagerNamespace, 9402, true)
				Expect(err).To(BeNil(), fmt.Sprintf("FAIL: Test cert-manager ingress failed: reason = %s", err))
			},
			func() {
				t.Logs.Info("Test ingress-nginx-controller ingress rules")
				err := testAccess(metav1.LabelSelector{MatchLabels: map[string]string{kubernetesAppLabel: "prometheus"}}, vzconst.PrometheusOperatorNamespace, metav1.LabelSelector{MatchLabels: map[string]string{kubernetesCompLabel: "controller"}}, "ingress-nginx", 443, true)
				Expect(err).To(BeNil(), fmt.Sprintf("FAIL: Test ingress-nginx-controller ingress failed: reason = %s", err))
				err = testAccess(metav1.LabelSelector{MatchLabels: map[string]string{kubernetesAppLabel: "prometheus"}}, vzconst.PrometheusOperatorNamespace, metav1.LabelSelector{MatchLabels: map[string]string{kubernetesCompLabel: "controller"}}, "ingress-nginx", ingressControllerMetricsPort, true)
				Expect(err).To(BeNil(), fmt.Sprintf("FAIL: Test ingress-nginx-controller ingress failed: reason = %s", err))
				err = testAccess(metav1.LabelSelector{MatchLabels: map[string]string{kubernetesAppLabel: "prometheus"}}, vzconst.PrometheusOperatorNamespace, metav1.LabelSelector{MatchLabels: map[string]string{kubernetesCompLabel: "controller"}}, "ingress-nginx", envoyStatsMetricsPort, true)
				Expect(err).To(BeNil(), fmt.Sprintf("FAIL: Test ingress-nginx-controller ingress failed: reason = %s", err))
			},
			func() {
				t.Logs.Info("Test ingress-nginx-default-backend ingress rules")
				err := testAccess(metav1.LabelSelector{MatchLabels: map[string]string{kubernetesCompLabel: "controller"}}, "ingress-nginx", metav1.LabelSelector{MatchLabels: map[string]string{kubernetesCompLabel: defaultBackend}}, "ingress-nginx", 8080, true)
				Expect(err).To(BeNil(), fmt.Sprintf("FAIL: Test ingress-nginx-default-backend ingress failed: reason = %s", err))
				err = testAccess(metav1.LabelSelector{MatchLabels: map[string]string{kubernetesAppLabel: "prometheus"}}, vzconst.PrometheusOperatorNamespace, metav1.LabelSelector{MatchLabels: map[string]string{kubernetesCompLabel: defaultBackend}}, "ingress-nginx", envoyStatsMetricsPort, true)
				Expect(err).To(BeNil(), fmt.Sprintf("FAIL: Test ingress-nginx-default-backend ingress failed: reason = %s", err))
			},
			func() {
				t.Logs.Info("Test istiod-verrazzano-system ingress rules")
				err := testAccess(metav1.LabelSelector{MatchLabels: map[string]string{kubernetesAppLabel: "prometheus"}}, vzconst.PrometheusOperatorNamespace, metav1.LabelSelector{MatchLabels: map[string]string{"app": "istiod"}}, vzconst.IstioSystemNamespace, 15012, true)
				Expect(err).To(BeNil(), fmt.Sprintf("FAIL: Test istiod-verrazzano-system ingress failed: reason = %s", err))
				err = testAccess(metav1.LabelSelector{MatchLabels: map[string]string{kubernetesAppLabel: "prometheus"}}, vzconst.PrometheusOperatorNamespace, metav1.LabelSelector{MatchLabels: map[string]string{"app": "istiod"}}, vzconst.IstioSystemNamespace, istiodMetricsPort, true)
				Expect(err).To(BeNil(), fmt.Sprintf("FAIL: Test istiod-verrazzano-system ingress failed: reason = %s", err))
			},
			func() {
				t.Logs.Info("Test istiod application namespace ingress rules")
				err := testAccess(metav1.LabelSelector{MatchLabels: map[string]string{"app": "hello-helidon"}}, generatedNamespace, metav1.LabelSelector{MatchLabels: map[string]string{"app": "istiod"}}, vzconst.IstioSystemNamespace, 15012, true)
				Expect(err).To(BeNil(), fmt.Sprintf("FAIL: Test istiod application namespace ingress failed: reason = %s", err))
			},
			func() {
				t.Logs.Info("Test keycloak ingress rules")
				err := testAccess(metav1.LabelSelector{MatchLabels: map[string]string{"app.kubernetes.io/instance": "ingress-controller"}}, "ingress-nginx", metav1.LabelSelector{MatchLabels: map[string]string{kubernetesAppLabel: "keycloak"}}, "keycloak", 8080, true)
				Expect(err).To(BeNil(), fmt.Sprintf("FAIL: Test keycloak ingress rules failed: reason = %s", err))
				err = testAccess(metav1.LabelSelector{MatchLabels: map[string]string{kubernetesAppLabel: "prometheus"}}, vzconst.PrometheusOperatorNamespace, metav1.LabelSelector{MatchLabels: map[string]string{kubernetesAppLabel: "keycloak"}}, "keycloak", envoyStatsMetricsPort, true)
				Expect(err).To(BeNil(), fmt.Sprintf("FAIL: Test keycloak ingress rules failed: reason = %s", err))
			},
			func() {
				t.Logs.Info("Test mysql ingress rules")
				kubeconfigPath, _ := k8sutil.GetKubeConfigLocation()
				label := "app"
				if ok, _ := pkg.IsVerrazzanoMinVersion("1.4.0", kubeconfigPath); ok {
					label = "tier"
				}
				err := testAccess(metav1.LabelSelector{MatchLabels: map[string]string{kubernetesAppLabel: "prometheus"}}, vzconst.PrometheusOperatorNamespace, metav1.LabelSelector{MatchLabels: map[string]string{label: "mysql"}}, "keycloak", envoyStatsMetricsPort, true)
				Expect(err).To(BeNil(), fmt.Sprintf("FAIL: Test mysql ingress rules failed: reason = %s", err))
				err = testAccess(metav1.LabelSelector{MatchLabels: map[string]string{"name": "mysql-operator"}}, vzconst.MySQLOperatorNamespace, metav1.LabelSelector{MatchLabels: map[string]string{label: "mysql"}}, "keycloak", mysqlPort, true)
				Expect(err).To(BeNil(), fmt.Sprintf("FAIL: Test mysql ingress rules failed: reason = %s", err))
			},
			func() {
				t.Logs.Info("Test verrazzano-platform-operator ingress rules")
			},
			func() {
				t.Logs.Info("Test verrazzano-platform-operator ingress rules")
				err := testAccess(metav1.LabelSelector{MatchLabels: map[string]string{"app": nodeExporter}}, vzconst.PrometheusOperatorNamespace, metav1.LabelSelector{MatchLabels: map[string]string{"app": "verrazzano-platform-operator"}}, "verrazzano-install", 9443, true)
				Expect(err).To(BeNil(), fmt.Sprintf("FAIL: Test verrazzano-platform-operator ingress rules failed: reason = %s", err))
			},
			func() {
				t.Logs.Info("Test coherence-operator ingress rules")
				// Allowing pods to be optional because some contexts in which this test is run disables the Coherence operator.
				err := testAccessPodsOptional(metav1.LabelSelector{MatchLabels: map[string]string{"app": "netpol-test"}}, "netpol-test", metav1.LabelSelector{MatchLabels: map[string]string{"control-plane": "coherence"}}, vzconst.VerrazzanoSystemNamespace, 9443, true)
				Expect(err).To(BeNil(), fmt.Sprintf("FAIL: Test coherence-operator ingress rules failed: reason = %s", err))
				// Allowing pods to be optional because some contexts in which this test is run disables the Coherence operator.
				err = testAccessPodsOptional(metav1.LabelSelector{MatchLabels: map[string]string{"app": "netpol-test"}}, "netpol-test", metav1.LabelSelector{MatchLabels: map[string]string{"control-plane": "coherence"}}, vzconst.VerrazzanoSystemNamespace, 8000, true)
				Expect(err).To(BeNil(), fmt.Sprintf("FAIL: Test coherence-operator ingress rules failed: reason = %s", err))
			},
			func() {
				t.Logs.Info("Test verrazzano-application-operator ingress rules")
				err := testAccess(metav1.LabelSelector{MatchLabels: map[string]string{"app": "netpol-test"}}, "netpol-test", metav1.LabelSelector{MatchLabels: map[string]string{"app": "verrazzano-application-operator"}}, vzconst.VerrazzanoSystemNamespace, 9443, true)
				Expect(err).To(BeNil(), fmt.Sprintf("FAIL: Test verrazzano-application-operator ingress rules failed: reason = %s", err))
				err = testAccess(metav1.LabelSelector{MatchLabels: map[string]string{"app": nodeExporter}}, vzconst.PrometheusOperatorNamespace, metav1.LabelSelector{MatchLabels: map[string]string{"app": "verrazzano-application-operator"}}, vzconst.VerrazzanoSystemNamespace, 9443, true)
				Expect(err).To(BeNil(), fmt.Sprintf("FAIL: Test verrazzano-application-operator ingress rules failed: reason = %s", err))
			},
			func() {
				t.Logs.Info("Test verrazzano-authproxy ingress rules")
				err := testAccess(metav1.LabelSelector{MatchLabels: map[string]string{"app.kubernetes.io/instance": "ingress-controller"}}, "ingress-nginx", metav1.LabelSelector{MatchLabels: map[string]string{"app": constants.VerrazzanoAuthProxyServiceName}}, vzconst.VerrazzanoSystemNamespace, 8775, true)
				Expect(err).To(BeNil(), fmt.Sprintf("FAIL: Test verrazzano-authproxy ingress rules failed: reason = %s", err))
				err = testAccess(metav1.LabelSelector{MatchLabels: map[string]string{"app": "fluentd"}}, vzconst.VerrazzanoSystemNamespace, metav1.LabelSelector{MatchLabels: map[string]string{"app": constants.VerrazzanoAuthProxyServiceName}}, vzconst.VerrazzanoSystemNamespace, 8775, true)
				Expect(err).To(BeNil(), fmt.Sprintf("FAIL: Test verrazzano-authproxy ingress rules failed: reason = %s", err))
				err = testAccess(metav1.LabelSelector{MatchLabels: map[string]string{kubernetesAppLabel: "prometheus"}}, vzconst.PrometheusOperatorNamespace, metav1.LabelSelector{MatchLabels: map[string]string{"app": constants.VerrazzanoAuthProxyServiceName}}, vzconst.VerrazzanoSystemNamespace, envoyStatsMetricsPort, true)
				Expect(err).To(BeNil(), fmt.Sprintf("FAIL: Test verrazzano-authproxy ingress rules failed: reason = %s", err))
			},
			func() {
				t.Logs.Info("Test verrazzano-console ingress rules")
				err := testAccess(metav1.LabelSelector{MatchLabels: map[string]string{"app": constants.VerrazzanoAuthProxyServiceName}}, vzconst.VerrazzanoSystemNamespace, metav1.LabelSelector{MatchLabels: map[string]string{"app": vzConsole}}, vzconst.VerrazzanoSystemNamespace, 8000, true)
				Expect(err).To(BeNil(), fmt.Sprintf("FAIL: Test verrazzano-console ingress rules failed: reason = %s", err))
				err = testAccess(metav1.LabelSelector{MatchLabels: map[string]string{kubernetesAppLabel: "prometheus"}}, vzconst.PrometheusOperatorNamespace, metav1.LabelSelector{MatchLabels: map[string]string{"app": vzConsole}}, vzconst.VerrazzanoSystemNamespace, envoyStatsMetricsPort, true)
				Expect(err).To(BeNil(), fmt.Sprintf("FAIL: Test verrazzano-console ingress rules failed: reason = %s", err))
			},
			func() {
				t.Logs.Info("Test prometheus-node-exporter ingress rules")
				err := testAccess(metav1.LabelSelector{MatchLabels: map[string]string{kubernetesAppLabel: "prometheus"}}, vzconst.PrometheusOperatorNamespace, metav1.LabelSelector{MatchLabels: map[string]string{"app": nodeExporter}}, vzconst.PrometheusOperatorNamespace, nodeExporterMetricsPort, true)
				Expect(err).To(BeNil(), fmt.Sprintf("FAIL: Test prometheus-node-exporter ingress rules failed: reason = %s", err))
			},
			func() {
				t.Logs.Info("Test istio-ingressgateway ingress rules")
				err := testAccess(metav1.LabelSelector{MatchLabels: map[string]string{kubernetesAppLabel: "prometheus"}}, vzconst.PrometheusOperatorNamespace, metav1.LabelSelector{MatchLabels: map[string]string{"app": istio.IstioIngressgatewayDeployment}}, vzconst.IstioSystemNamespace, envoyStatsMetricsPort, true)
				Expect(err).To(BeNil(), fmt.Sprintf("FAIL: Test istio-ingressgateway ingress rules failed: reason = %s", err))
			},
			func() {
				t.Logs.Info("Test istio-egressgateway ingress rules")
				err := testAccess(metav1.LabelSelector{MatchLabels: map[string]string{kubernetesAppLabel: "prometheus"}}, vzconst.PrometheusOperatorNamespace, metav1.LabelSelector{MatchLabels: map[string]string{"app": istio.IstioEgressgatewayDeployment}}, vzconst.IstioSystemNamespace, envoyStatsMetricsPort, true)
				Expect(err).To(BeNil(), fmt.Sprintf("FAIL: Test istio-egressgateway ingress rules failed: reason = %s", err))
			},
			func() {
				t.Logs.Info("Test vmi-system-es-master ingress rules")
				err := testAccess(metav1.LabelSelector{MatchLabels: map[string]string{kubernetesAppLabel: "prometheus"}}, vzconst.PrometheusOperatorNamespace, metav1.LabelSelector{MatchLabels: map[string]string{"app": "system-es-master"}}, vzconst.VerrazzanoSystemNamespace, envoyStatsMetricsPort, true)
				Expect(err).To(BeNil(), fmt.Sprintf("FAIL: Test vmi-system-es-master ingress rules failed: reason = %s", err))
				/* TODO:
				The following tests only work in Verrazzano prod profile. There is a differnce in network policies used in prod and
				dev profile. Once that is resolved, the following lines can be uncommented. They have been tested to work in prod profile.
				*/
				// err = testAccess(metav1.LabelSelector{MatchLabels: map[string]string{"app": "system-es-data"}}, vzconst.VerrazzanoSystemNamespace, metav1.LabelSelector{MatchLabels: map[string]string{"app": "system-es-master"}}, vzconst.VerrazzanoSystemNamespace, 9300, true)
				// Expect(err).To(BeNil(), fmt.Sprintf("FAIL: Test vmi-system-es-master ingress rules failed: reason = %s", err))
				// err = testAccess(metav1.LabelSelector{MatchLabels: map[string]string{"app": "system-es-ingest"}}, vzconst.VerrazzanoSystemNamespace, metav1.LabelSelector{MatchLabels: map[string]string{"app": "system-es-master"}}, vzconst.VerrazzanoSystemNamespace, 9300, true)
				// Expect(err).To(BeNil(), fmt.Sprintf("FAIL: Test vmi-system-es-master ingress rules failed: reason = %s", err))
			},
			/* TODO:
			The following tests only work in Verrazzano prod profile. There is a differnce in network policies used in prod and
			dev profile. Once that is resolved, the following lines can be uncommented. They have been tested to work in prod profile.
			*/
			// func() {
			// 	pkg.Log(pkg.Info, "Test vmi-system-es-data ingress rules")
			// 	err := testAccess(metav1.LabelSelector{MatchLabels: map[string]string{"app": "system-es-master"}}, vzconst.VerrazzanoSystemNamespace, metav1.LabelSelector{MatchLabels: map[string]string{"app": "system-es-data"}}, vzconst.VerrazzanoSystemNamespace, 9300, true)
			// 	Expect(err).To(BeNil(), fmt.Sprintf("FAIL: Test vmi-system-es-data ingress rules failed: reason = %s", err))
			// 	err = testAccess(metav1.LabelSelector{MatchLabels: map[string]string{"app": "system-es-ingest"}}, vzconst.VerrazzanoSystemNamespace, metav1.LabelSelector{MatchLabels: map[string]string{"app": "system-es-data"}}, vzconst.VerrazzanoSystemNamespace, 9200, true)
			// 	Expect(err).To(BeNil(), fmt.Sprintf("FAIL: Test vmi-system-es-data ingress rules failed: reason = %s", err))
			// 	err = testAccess(metav1.LabelSelector{MatchLabels: map[string]string{"app": "system-es-ingest"}}, vzconst.VerrazzanoSystemNamespace, metav1.LabelSelector{MatchLabels: map[string]string{"app": "system-es-data"}}, vzconst.VerrazzanoSystemNamespace, 9300, true)
			// 	Expect(err).To(BeNil(), fmt.Sprintf("FAIL: Test vmi-system-es-data ingress rules failed: reason = %s", err))
			// 	err = testAccess(metav1.LabelSelector{MatchLabels: map[string]string{"app": kibanaSys}}, vzconst.VerrazzanoSystemNamespace, metav1.LabelSelector{MatchLabels: map[string]string{"app": "system-es-data"}}, vzconst.VerrazzanoSystemNamespace, 9200, true)
			// 	Expect(err).To(BeNil(), fmt.Sprintf("FAIL: Test vmi-system-es-data ingress rules failed: reason = %s", err))
			// 	err = testAccess(metav1.LabelSelector{MatchLabels: map[string]string{kubernetesAppLabel: "prometheus"}}, vzconst.PrometheusOperatorNamespace, metav1.LabelSelector{MatchLabels: map[string]string{"app": "system-es-data"}}, vzconst.VerrazzanoSystemNamespace, envoyStatsMetricsPort, true)
			// 	Expect(err).To(BeNil(), fmt.Sprintf("FAIL: Test vmi-system-es-data ingress rules failed: reason = %s", err))
			// },
			// func() {
			// 	pkg.Log(pkg.Info, "Test vmi-system-es-ingest ingress rules")
			// 	err := testAccess(metav1.LabelSelector{MatchLabels: map[string]string{"app": "system-es-master"}}, vzconst.VerrazzanoSystemNamespace, metav1.LabelSelector{MatchLabels: map[string]string{"app": "system-es-ingest"}}, vzconst.VerrazzanoSystemNamespace, 9300, true)
			// 	Expect(err).To(BeNil(), fmt.Sprintf("FAIL: Test vmi-system-es-ingest ingress rules failed: reason = %s", err))
			// 	err = testAccess(metav1.LabelSelector{MatchLabels: map[string]string{"app": "system-es-data"}}, vzconst.VerrazzanoSystemNamespace, metav1.LabelSelector{MatchLabels: map[string]string{"app": "system-es-ingest"}}, vzconst.VerrazzanoSystemNamespace, 9300, true)
			// 	Expect(err).To(BeNil(), fmt.Sprintf("FAIL: Test vmi-system-es-ingest ingress rules failed: reason = %s", err))
			// 	err = testAccess(metav1.LabelSelector{MatchLabels: map[string]string{"app": kibanaSys}}, vzconst.VerrazzanoSystemNamespace, metav1.LabelSelector{MatchLabels: map[string]string{"app": "system-es-ingest"}}, vzconst.VerrazzanoSystemNamespace, 9200, true)
			// 	Expect(err).To(BeNil(), fmt.Sprintf("FAIL: Test vmi-system-es-ingest ingress rules failed: reason = %s", err))
			// 	err = testAccess(metav1.LabelSelector{MatchLabels: map[string]string{kubernetesAppLabel: "prometheus"}}, vzconst.PrometheusOperatorNamespace, metav1.LabelSelector{MatchLabels: map[string]string{"app": "system-es-ingest"}}, vzconst.VerrazzanoSystemNamespace, envoyStatsMetricsPort, true)
			// 	Expect(err).To(BeNil(), fmt.Sprintf("FAIL: Test vmi-system-es-ingest ingress rules failed: reason = %s", err))
			// },
			func() {
				t.Logs.Info("Test vmi-system-grafana ingress rules")
				err := testAccess(metav1.LabelSelector{MatchLabels: map[string]string{"app": constants.VerrazzanoAuthProxyServiceName}}, vzconst.VerrazzanoSystemNamespace, metav1.LabelSelector{MatchLabels: map[string]string{"app": grafanaSys}}, vzconst.VerrazzanoSystemNamespace, 3000, true)
				Expect(err).To(BeNil(), fmt.Sprintf("FAIL: Test vmi-system-grafana ingress rules failed: reason = %s", err))
				err = testAccess(metav1.LabelSelector{MatchLabels: map[string]string{kubernetesAppLabel: "prometheus"}}, vzconst.PrometheusOperatorNamespace, metav1.LabelSelector{MatchLabels: map[string]string{"app": grafanaSys}}, vzconst.VerrazzanoSystemNamespace, envoyStatsMetricsPort, true)
				Expect(err).To(BeNil(), fmt.Sprintf("FAIL: Test vmi-system-grafana ingress rules failed: reason = %s", err))

			},
			func() {
				t.Logs.Info("Test vmi-system-kibana ingress rules")
				err := testAccess(metav1.LabelSelector{MatchLabels: map[string]string{"app": constants.VerrazzanoAuthProxyServiceName}}, vzconst.VerrazzanoSystemNamespace, metav1.LabelSelector{MatchLabels: map[string]string{"app": kibanaSys}}, vzconst.VerrazzanoSystemNamespace, 5601, true)
				Expect(err).To(BeNil(), fmt.Sprintf("FAIL: Test vmi-system-kibana ingress rules failed: reason = %s", err))
				err = testAccess(metav1.LabelSelector{MatchLabels: map[string]string{kubernetesAppLabel: "prometheus"}}, vzconst.PrometheusOperatorNamespace, metav1.LabelSelector{MatchLabels: map[string]string{"app": kibanaSys}}, vzconst.VerrazzanoSystemNamespace, envoyStatsMetricsPort, true)
				Expect(err).To(BeNil(), fmt.Sprintf("FAIL: Test vmi-system-kibana ingress rules failed: reason = %s", err))
			},
			func() {
				t.Logs.Info("Test prometheus ingress rules")
				err := testAccess(metav1.LabelSelector{MatchLabels: map[string]string{"app": constants.VerrazzanoAuthProxyServiceName}}, vzconst.VerrazzanoSystemNamespace, metav1.LabelSelector{MatchLabels: map[string]string{kubernetesAppLabel: "prometheus"}}, vzconst.PrometheusOperatorNamespace, 9090, true)
				Expect(err).To(BeNil(), fmt.Sprintf("FAIL: Test prometheus ingress rules for the authproxy failed: reason = %s", err))
				err = testAccess(metav1.LabelSelector{MatchLabels: map[string]string{"app": grafanaSys}}, vzconst.VerrazzanoSystemNamespace, metav1.LabelSelector{MatchLabels: map[string]string{kubernetesAppLabel: "prometheus"}}, vzconst.PrometheusOperatorNamespace, 9090, true)
				Expect(err).To(BeNil(), fmt.Sprintf("FAIL: Test prometheus ingress rules for Grafana failed: reason = %s", err))
				err = testAccess(metav1.LabelSelector{MatchLabels: map[string]string{"app": "kiali"}}, vzconst.VerrazzanoSystemNamespace, metav1.LabelSelector{MatchLabels: map[string]string{kubernetesAppLabel: "prometheus"}}, vzconst.PrometheusOperatorNamespace, 9090, true)
				Expect(err).To(BeNil(), fmt.Sprintf("FAIL: Test prometheus ingress rules for Kiali failed: reason = %s", err))
			},
			func() {
				t.Logs.Info("Test prometheus-node-exporter ingress rules")
				err := testAccess(metav1.LabelSelector{MatchLabels: map[string]string{kubernetesAppLabel: "prometheus"}}, vzconst.PrometheusOperatorNamespace, metav1.LabelSelector{MatchLabels: map[string]string{"app": nodeExporter}}, vzconst.PrometheusOperatorNamespace, 9100, true)
				Expect(err).To(BeNil(), fmt.Sprintf("FAIL: Test prometheus-node-exporter ingress rules failed: reason = %s", err))
			},
			func() {
				t.Logs.Info("Test weblogic-operator ingress rules")
				// Allowing pods to be optional because some contexts in which this test is run disables the WebLogic operator.
				err := testAccessPodsOptional(metav1.LabelSelector{MatchLabels: map[string]string{"app": istio.IstioIngressgatewayDeployment}}, vzconst.IstioSystemNamespace, metav1.LabelSelector{MatchLabels: map[string]string{"app": weblogicOperator}}, vzconst.VerrazzanoSystemNamespace, envoyStatsMetricsPort, true)
				Expect(err).To(BeNil(), fmt.Sprintf("FAIL: Test weblogic-operator ingress rules failed: reason = %s", err))
				// Allowing pods to be optional because some contexts in which this test is run disables the WebLogic operator.
				err = testAccessPodsOptional(metav1.LabelSelector{MatchLabels: map[string]string{"app": istio.IstioEgressgatewayDeployment}}, vzconst.IstioSystemNamespace, metav1.LabelSelector{MatchLabels: map[string]string{"app": weblogicOperator}}, vzconst.VerrazzanoSystemNamespace, envoyStatsMetricsPort, true)
				Expect(err).To(BeNil(), fmt.Sprintf("FAIL: Test weblogic-operator ingress rules failed: reason = %s", err))
				// Allowing pods to be optional because some contexts in which this test is run disables the WebLogic operator.
				err = testAccessPodsOptional(metav1.LabelSelector{MatchLabels: map[string]string{"app": "istiod"}}, vzconst.IstioSystemNamespace, metav1.LabelSelector{MatchLabels: map[string]string{"app": weblogicOperator}}, vzconst.VerrazzanoSystemNamespace, envoyStatsMetricsPort, true)
				Expect(err).To(BeNil(), fmt.Sprintf("FAIL: Test weblogic-operator ingress rules failed: reason = %s", err))
				// Allowing pods to be optional because some contexts in which this test is run disables the WebLogic operator.
				err = testAccessPodsOptional(metav1.LabelSelector{MatchLabels: map[string]string{kubernetesAppLabel: "prometheus"}}, vzconst.PrometheusOperatorNamespace, metav1.LabelSelector{MatchLabels: map[string]string{"app": weblogicOperator}}, vzconst.VerrazzanoSystemNamespace, envoyStatsMetricsPort, true)
				Expect(err).To(BeNil(), fmt.Sprintf("FAIL: Test weblogic-operator ingress rules failed: reason = %s", err))
			},
			func() {
				t.Logs.Info("Test kiali ingress rules")
				err := testAccessPodsOptional(metav1.LabelSelector{MatchLabels: map[string]string{"app": constants.VerrazzanoAuthProxyServiceName}}, vzconst.VerrazzanoSystemNamespace, metav1.LabelSelector{MatchLabels: map[string]string{"app": "kiali"}}, vzconst.VerrazzanoSystemNamespace, 20001, true)
				Expect(err).To(BeNil(), fmt.Sprintf("FAIL: Test Kiali network ingress from verrazzano-authproxy failed: reason = %s", err))
				err = testAccessPodsOptional(metav1.LabelSelector{MatchLabels: map[string]string{kubernetesAppLabel: "prometheus"}}, vzconst.PrometheusOperatorNamespace, metav1.LabelSelector{MatchLabels: map[string]string{"app": "kiali"}}, vzconst.VerrazzanoSystemNamespace, envoyStatsMetricsPort, true)
				Expect(err).To(BeNil(), fmt.Sprintf("FAIL: Test Kiali network ingress from prometheus failed: reason = %s", err))
			},
		)
	})

	// GIVEN a Verrazzano deployment
	// WHEN access is attempted between pods that violate the rules of the Verrazzano network policies
	// THEN the attempted access should fail
	t.It("Negative Test NetworkPolicy Rules", func() {
		assertions := []func(){
			func() {
				t.Logs.Info("Negative test rancher ingress rules")
				err := testAccess(metav1.LabelSelector{MatchLabels: map[string]string{"app": "netpol-test"}}, "netpol-test", metav1.LabelSelector{MatchLabels: map[string]string{"app": "rancher"}}, "cattle-system", 80, false)
				Expect(err).To(BeNil(), fmt.Sprintf("FAIL: Negative test  rancher ingress failed: reason = %s", err))
			},
			func() {
				t.Logs.Info("Negative test cert-manager ingress rules")
				err := testAccess(metav1.LabelSelector{MatchLabels: map[string]string{"app": "netpol-test"}}, "netpol-test", metav1.LabelSelector{MatchLabels: map[string]string{"app": vzconst.CertManagerNamespace}}, vzconst.CertManagerNamespace, 9402, false)
				Expect(err).To(BeNil(), fmt.Sprintf("FAIL: Negative test cert-manager ingress failed: reason = %s", err))
			},
			func() {
				t.Logs.Info("Negative test ingress-nginx-controller ingress rules")
				err := testAccess(metav1.LabelSelector{MatchLabels: map[string]string{"app": "netpol-test"}}, "netpol-test", metav1.LabelSelector{MatchLabels: map[string]string{kubernetesCompLabel: "controller"}}, "ingress-nginx", 80, false)
				Expect(err).To(BeNil(), fmt.Sprintf("FAIL: Negative test ingress-nginx-controller ingress failed: reason = %s", err))
			},
			func() {
				t.Logs.Info("Negative test ingress-nginx-default-backend ingress rules")
				err := testAccess(metav1.LabelSelector{MatchLabels: map[string]string{"app": "netpol-test"}}, "netpol-test", metav1.LabelSelector{MatchLabels: map[string]string{kubernetesCompLabel: defaultBackend}}, "ingress-nginx", 8080, false)
				Expect(err).To(BeNil(), fmt.Sprintf("FAIL: Negative test ingress-nginx-default-backend ingress failed: reason = %s", err))
			},
			func() {
				t.Logs.Info("Negative test istio-egressgateway ingress rules")
				err := testAccess(metav1.LabelSelector{MatchLabels: map[string]string{"app": "netpol-test"}}, "netpol-test", metav1.LabelSelector{MatchLabels: map[string]string{"app": istio.IstioEgressgatewayDeployment}}, vzconst.IstioSystemNamespace, 6443, false)
				Expect(err).To(BeNil(), fmt.Sprintf("FAIL: Negative testistio-egressgateway ingress failed: reason = %s", err))
			},
			func() {
				t.Logs.Info("Negative test istio-ingressgateway ingress rules")
				err := testAccess(metav1.LabelSelector{MatchLabels: map[string]string{"app": "netpol-test"}}, "netpol-test", metav1.LabelSelector{MatchLabels: map[string]string{"app": istio.IstioIngressgatewayDeployment}}, vzconst.IstioSystemNamespace, 6443, false)
				Expect(err).To(BeNil(), fmt.Sprintf("FAIL: Negative testistio-ingressgateway ingress failed: reason = %s", err))
			},
			func() {
				t.Logs.Info("Negative test istiod-verrazzano-system ingress rules")
				err := testAccess(metav1.LabelSelector{MatchLabels: map[string]string{"app": "netpol-test"}}, "netpol-test", metav1.LabelSelector{MatchLabels: map[string]string{"app": "istiod"}}, vzconst.IstioSystemNamespace, 15012, false)
				Expect(err).To(BeNil(), fmt.Sprintf("FAIL: Negative test istiod-verrazzano-system ingress failed: reason = %s", err))
			},
			func() {
				t.Logs.Info("Negative test keycloak ingress rules")
				err := testAccess(metav1.LabelSelector{MatchLabels: map[string]string{"app": "netpol-test"}}, "netpol-test", metav1.LabelSelector{MatchLabels: map[string]string{kubernetesAppLabel: "keycloak"}}, "keycloak", 8080, false)
				Expect(err).To(BeNil(), fmt.Sprintf("FAIL: Negative test keycloak ingress rules failed: reason = %s", err))
			},
			func() {
				t.Logs.Info("Negative test oam-kubernetes-runtime ingress rules")
				err := testAccess(metav1.LabelSelector{MatchLabels: map[string]string{"app": "netpol-test"}}, "netpol-test", metav1.LabelSelector{MatchLabels: map[string]string{kubernetesAppLabel: "oam-kubernetes-runtime"}}, vzconst.VerrazzanoSystemNamespace, 8775, false)
				Expect(err).To(BeNil(), fmt.Sprintf("FAIL: Negative test oam-kubernetes-runtime ingress rules failed: reason = %s", err))
			},
			func() {
				t.Logs.Info("Negative test verrazzano-authproxy ingress rules")
				err := testAccess(metav1.LabelSelector{MatchLabels: map[string]string{"app": "netpol-test"}}, "netpol-test", metav1.LabelSelector{MatchLabels: map[string]string{"app": constants.VerrazzanoAuthProxyServiceName}}, vzconst.VerrazzanoSystemNamespace, 8775, false)
				Expect(err).To(BeNil(), fmt.Sprintf("FAIL: Negative test verrazzano-authproxy ingress rules failed: reason = %s", err))
			},
			func() {
				t.Logs.Info("Negative test verrazzano-console ingress rules")
				err := testAccess(metav1.LabelSelector{MatchLabels: map[string]string{"app": "netpol-test"}}, "netpol-test", metav1.LabelSelector{MatchLabels: map[string]string{"app": vzConsole}}, vzconst.VerrazzanoSystemNamespace, 8000, false)
				Expect(err).To(BeNil(), fmt.Sprintf("FAIL: Negative test verrazzano-console ingress rules failed: reason = %s", err))
			},
			func() {
				t.Logs.Info("Negative test verrazzano-monitoring-operator ingress rules")
				err := testAccess(metav1.LabelSelector{MatchLabels: map[string]string{"app": "netpol-test"}}, "netpol-test", metav1.LabelSelector{MatchLabels: map[string]string{"k8s-app": "verrazzano-monitoring-operator"}}, vzconst.VerrazzanoSystemNamespace, 8000, false)
				Expect(err).To(BeNil(), fmt.Sprintf("FAIL: Negative test verrazzano-monitoring-operator ingress rules failed: reason = %s", err))
			},
			func() {
				t.Logs.Info("Negative test vmi-system-es-master ingress rules")
				err := testAccess(metav1.LabelSelector{MatchLabels: map[string]string{"app": "netpol-test"}}, "netpol-test", metav1.LabelSelector{MatchLabels: map[string]string{"app": "system-es-master"}}, vzconst.VerrazzanoSystemNamespace, 9200, false)
				Expect(err).To(BeNil(), fmt.Sprintf("FAIL: Negative test vmi-system-es-master ingress rules failed: reason = %s", err))
			},
			/* TODO:
			The following tests only work in Verrazzano prod profile. There is a differnce in network policies used in prod and
			dev profile. Once that is resolved, the following lines can be uncommented. They have been tested to work in prod profile.
			*/
			// func() {
			// 	pkg.Log(pkg.Info, "Negative test vmi-system-es-data ingress rules")
			// 	err := testAccess(metav1.LabelSelector{MatchLabels: map[string]string{"app": "netpol-test"}}, "netpol-test", metav1.LabelSelector{MatchLabels: map[string]string{"app": "system-es-data"}}, vzconst.VerrazzanoSystemNamespace, 9200, false)
			// 	Expect(err).To(BeNil(), fmt.Sprintf("FAIL: Negative test vmi-system-es-data ingress rules failed: reason = %s", err))
			// },
			// func() {
			// 	pkg.Log(pkg.Info, "Negative test vmi-system-es-ingest ingress rules")
			// 	err := testAccess(metav1.LabelSelector{MatchLabels: map[string]string{"app": "netpol-test"}}, "netpol-test", metav1.LabelSelector{MatchLabels: map[string]string{"app": "system-es-ingest"}}, vzconst.VerrazzanoSystemNamespace, 9200, false)
			// 	Expect(err).To(BeNil(), fmt.Sprintf("FAIL: Negative test vmi-system-es-ingest ingress rules failed: reason = %s", err))
			// },
			func() {
				t.Logs.Info("Negative test vmi-system-grafana ingress rules")
				err := testAccess(metav1.LabelSelector{MatchLabels: map[string]string{"app": "netpol-test"}}, "netpol-test", metav1.LabelSelector{MatchLabels: map[string]string{"app": grafanaSys}}, vzconst.VerrazzanoSystemNamespace, 3000, false)
				Expect(err).To(BeNil(), fmt.Sprintf("FAIL: Negative test vmi-system-grafana ingress rules failed: reason = %s", err))
			},
			func() {
				t.Logs.Info("Negative test vmi-system-kibana ingress rules")
				err := testAccess(metav1.LabelSelector{MatchLabels: map[string]string{"app": "netpol-test"}}, "netpol-test", metav1.LabelSelector{MatchLabels: map[string]string{"app": kibanaSys}}, vzconst.VerrazzanoSystemNamespace, 5601, false)
				Expect(err).To(BeNil(), fmt.Sprintf("FAIL: Negative test vmi-system-kibana ingress rules failed: reason = %s", err))
			},
			func() {
				t.Logs.Info("Negative test prometheus ingress rules")
				err := testAccess(metav1.LabelSelector{MatchLabels: map[string]string{"app": "netpol-test"}}, "netpol-test", metav1.LabelSelector{MatchLabels: map[string]string{kubernetesAppLabel: "prometheus"}}, vzconst.PrometheusOperatorNamespace, 9090, false)
				Expect(err).To(BeNil(), fmt.Sprintf("FAIL: Negative test prometheus ingress rules failed: reason = %s", err))
			},
			func() {
				t.Logs.Info("Negative test weblogic-operator ingress rules")
				// Allowing pods to be optional because some contexts in which this test is run disables the WebLogic operator.
				err := testAccessPodsOptional(metav1.LabelSelector{MatchLabels: map[string]string{"app": "netpol-test"}}, "netpol-test", metav1.LabelSelector{MatchLabels: map[string]string{"app": weblogicOperator}}, vzconst.VerrazzanoSystemNamespace, 8000, false)
				Expect(err).To(BeNil(), fmt.Sprintf("FAIL: Negative test weblogic-operator ingress rules failed: reason = %s", err))
			},
			func() {
				t.Logs.Info("Negative test kiali ingress rules")
				err := testAccessPodsOptional(metav1.LabelSelector{MatchLabels: map[string]string{"app": "netpol-test"}}, "netpol-test", metav1.LabelSelector{MatchLabels: map[string]string{"app": "kiali"}}, vzconst.VerrazzanoSystemNamespace, envoyStatsMetricsPort, false)
				Expect(err).To(BeNil(), fmt.Sprintf("FAIL: Negative test kiali ingress rules failed: reason = %s", err))
			},
		}

		pkg.Concurrently(
			assertions...,
		)
	})
})

// testAccess attempts to access a given pod from another pod on a given port and tests for the expected result
func testAccess(fromSelector metav1.LabelSelector, fromNamespace string, toSelector metav1.LabelSelector, toNamespace string, port int, expectAccess bool) error {
	return doAccessCheck(accessCheckConfig{
		fromSelector:       fromSelector,
		fromNamespace:      fromNamespace,
		toSelector:         toSelector,
		toNamespace:        toNamespace,
		port:               port,
		expectAccess:       expectAccess,
		ignorePodsNotFound: false,
	})
}

// testAccessPodsOptional attempts to access a given pod from another pod on a given port and tests for the expected result
// Ignore pods not found
func testAccessPodsOptional(fromSelector metav1.LabelSelector, fromNamespace string, toSelector metav1.LabelSelector, toNamespace string, port int, expectAccess bool) error {
	return doAccessCheck(accessCheckConfig{
		fromSelector:       fromSelector,
		fromNamespace:      fromNamespace,
		toSelector:         toSelector,
		toNamespace:        toNamespace,
		port:               port,
		expectAccess:       expectAccess,
		ignorePodsNotFound: true,
	})
}

// doAccessCheck attempts to access a given pod from another pod on a given port and tests for the expected result.
func doAccessCheck(c accessCheckConfig) error {
	// get the FROM pod
	var pods []corev1.Pod
	Eventually(func() error {
		var err error
		pods, err = pkg.GetPodsFromSelector(&c.fromSelector, c.fromNamespace)
		if err != nil && errors.IsNotFound(err) && c.ignorePodsNotFound {
			// Ignore pods not found
			return nil
		}
		return err
	}, waitTimeout, pollingInterval).ShouldNot(HaveOccurred())
	if len(pods) == 0 && c.ignorePodsNotFound {
		return nil
	}

	mapFromSelector, _ := metav1.LabelSelectorAsMap(&c.fromSelector)
	jsonFromSelector, _ := json.Marshal(mapFromSelector)
	Expect(len(pods) > 0).To(BeTrue(), fmt.Sprintf("FAIL: Pod not found with label: %s in namespace: %s", jsonFromSelector, c.fromNamespace))
	fromPod := pods[0]

	// get the TO pod
	Eventually(func() error {
		var err error
		pods, err = pkg.GetPodsFromSelector(&c.toSelector, c.toNamespace)
		if err != nil && errors.IsNotFound(err) && c.ignorePodsNotFound {
			// Ignore pods not found
			return nil
		}
		return err
	}, waitTimeout, pollingInterval).ShouldNot(HaveOccurred())
	if len(pods) == 0 && c.ignorePodsNotFound {
		return nil
	}
	mapToSelector, _ := metav1.LabelSelectorAsMap(&c.toSelector)
	jsonToSelector, _ := json.Marshal(mapToSelector)
	Expect(len(pods) > 0).To(BeTrue(), fmt.Sprintf("FAIL: Pod not found with label: %s in namespace: %s", jsonToSelector, c.toNamespace))
	toPod := pods[0]

	if c.expectAccess {
		Eventually(func() bool {
			return attemptConnection(&fromPod, &toPod, c.port, 10)
		}, waitTimeout, pollingInterval).Should(BeTrue(), fmt.Sprintf("Should be able to access pod %s from pod %s on port %d", toPod.Name, fromPod.Name, c.port))
	} else {
		Consistently(func() bool {
			return attemptConnection(&fromPod, &toPod, c.port, 10)
		}, shortWaitTimeout, shortPollingInterval).Should(BeFalse(), fmt.Sprintf("Should NOT be able to access pod %s from pod %s on port %d", toPod.Name, fromPod.Name, c.port))
	}
	return nil
}

// attemptConnection attempts to access a given pod from another pod on a given port
func attemptConnection(fromPod, toPod *corev1.Pod, port int, duration time.Duration) bool {
	command := fmt.Sprintf(connectTestCmdFmt, duration, toPod.Status.PodIP, port)
	t.Logs.Infof("Executing command on pod %s.%s (%s)", fromPod.Namespace, fromPod.Name, command)
	stdout, _, err := pkg.Execute(fromPod.Name, fromPod.Spec.Containers[0].Name, fromPod.Namespace, []string{"sh", "-c", command})
	// check response for 'Connected' message; fail on error except for 'curl: (52) Empty reply from server'
	connected := strings.Contains(stdout, fmt.Sprintf(connectedFmt, toPod.Status.PodIP, toPod.Status.PodIP, port)) &&
		(err == nil || strings.Contains(fmt.Sprintf("%q", err), curlCode52))

	if connected {
		t.Logs.Infof("Connected from pod %s.%s to %s.%s on port %d", fromPod.Namespace, fromPod.Name, toPod.Namespace, toPod.Name, port)
	} else {
		t.Logs.Infof("Can NOT connect from pod %s.%s to %s.%s on port %d", fromPod.Namespace, fromPod.Name, toPod.Namespace, toPod.Name, port)
	}
	return connected
}
