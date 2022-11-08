// Copyright (c) 2021, 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package verify

import (
	"fmt"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	vzconst "github.com/verrazzano/verrazzano/pkg/constants"
	"github.com/verrazzano/verrazzano/pkg/helm"
	"github.com/verrazzano/verrazzano/pkg/k8sutil"
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/constants"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/appoper"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/authproxy"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/certmanager"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/coherence"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/externaldns"
	compistio "github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/istio"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/keycloak"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/kiali"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/mysql"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/nginx"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/oam"
	promoperator "github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/prometheus/operator"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/rancher"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/verrazzano"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/weblogic"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg/test/framework"
	"k8s.io/apimachinery/pkg/api/errors"
)

const (
	shortWait  = 1 * time.Minute
	mediumWait = 2 * time.Minute
	longWait   = 5 * time.Minute

	pollingInterval = 10 * time.Second
	envoyImage      = "proxyv2:1.14.3"
	minimumVersion  = "1.1.0"
)

var t = framework.NewTestFramework("verify")

var vzcr *vzapi.Verrazzano
var kubeconfigPath string

var _ = t.BeforeSuite(func() {
	var err error
	kubeconfigPath, err = k8sutil.GetKubeConfigLocation()
	if err != nil {
		Fail(fmt.Sprintf("Failed to get default kubeconfig path: %s", err.Error()))
	}

	// Get the Verrazzano CR
	Eventually(func() error {
		var err error
		vzcr, err = pkg.GetVerrazzano()
		if err != nil {
			return err
		}
		return err
	}, shortWait, pollingInterval).Should(BeNil(), "Expected to get Verrazzano CR")
})
var _ = t.AfterSuite(func() {})
var _ = t.AfterEach(func() {})

var _ = t.Describe("Post upgrade", Label("f:platform-lcm.upgrade"), func() {

	// GIVEN the verrazzano-system namespace
	// WHEN the container images are retrieved
	// THEN verify that each pod that uses istio has the correct istio proxy image
	t.ItMinimumVersion("pods in verrazzano-system have correct istio proxy image", minimumVersion, kubeconfigPath, func() {
		Eventually(func() bool {
			return pkg.CheckPodsForEnvoySidecar(constants.VerrazzanoSystemNamespace, envoyImage)
		}, longWait, pollingInterval).Should(BeTrue(), "Expected to find istio proxy image in verrazzano-system")
	})

	// GIVEN the ingress-nginx namespace
	// WHEN the container images are retrieved
	// THEN verify that each pod that uses istio has the correct istio proxy image
	t.ItMinimumVersion("pods in ingress-nginx have correct istio proxy image", minimumVersion, kubeconfigPath, func() {
		Eventually(func() bool {
			return pkg.CheckPodsForEnvoySidecar(constants.IngressNginxNamespace, envoyImage)
		}, longWait, pollingInterval).Should(BeTrue(), "Expected to find istio proxy image in ingress-nginx")
	})

	// GIVEN the keycloak namespace
	// WHEN the container images are retrieved
	// THEN verify that each pod that uses istio has the correct istio proxy image
	t.ItMinimumVersion("pods in keycloak have correct istio proxy image", minimumVersion, kubeconfigPath, func() {
		Eventually(func() bool {
			return pkg.CheckPodsForEnvoySidecar(constants.KeycloakNamespace, envoyImage)
		}, longWait, pollingInterval).Should(BeTrue(), "Expected to find istio proxy image in keycloak")
	})
})

var _ = t.Describe("Application pods post-upgrade", Label("f:platform-lcm.upgrade"), func() {
	const (
		bobsBooksNamespace    = "bobs-books"
		helloHelidonNamespace = "hello-helidon"
		springbootNamespace   = "springboot"
		todoListNamespace     = "todo-list"
	)
	t.DescribeTable("should contain Envoy sidecar 1.14.3",
		func(namespace string, timeout time.Duration) {
			exists, err := pkg.DoesNamespaceExist(namespace)
			if err != nil {
				Fail(err.Error())
			}
			if exists {
				Eventually(func() bool {
					return pkg.CheckPodsForEnvoySidecar(namespace, envoyImage)
				}, timeout, pollingInterval).Should(BeTrue(), fmt.Sprintf("Expected to find envoy sidecar %s in %s namespace", envoyImage, namespace))
			} else {
				t.Logs.Infof("Skipping test since namespace %s doesn't exist", namespace)
			}
		},
		t.Entry(fmt.Sprintf("pods in namespace %s have Envoy sidecar", helloHelidonNamespace), helloHelidonNamespace, mediumWait),
		t.Entry(fmt.Sprintf("pods in namespace %s have Envoy sidecar", springbootNamespace), springbootNamespace, mediumWait),
		t.Entry(fmt.Sprintf("pods in namespace %s have Envoy sidecar", todoListNamespace), todoListNamespace, longWait),
		t.Entry(fmt.Sprintf("pods in namespace %s have Envoy sidecar", bobsBooksNamespace), bobsBooksNamespace, longWait),
	)
})

// Istio no longer uses Helm for 1.10, so make sure all the Helm releases have been deleted
var _ = t.Describe("Istio helm releases", Label("f:platform-lcm.upgrade"), func() {
	const (
		istiod       = "istiod"
		istioBase    = "istio"
		istioIngress = "istio-ingress"
		istioEgress  = "istio-egress"
		istioCoreDNS = "istiocoredns"
	)
	t.DescribeTable("should be removed from the istio-system namespace post upgrade",
		func(release string) {
			Eventually(func() bool {
				installed, _ := helm.IsReleaseInstalled(release, constants.IstioSystemNamespace)
				return installed
			}, mediumWait, pollingInterval).Should(BeFalse(), fmt.Sprintf("Expected to not find release %s in istio-system", release))
		},
		t.Entry(fmt.Sprintf("istio-system doesn't contain release %s", istiod), istiod),
		t.Entry(fmt.Sprintf("istio-system doesn't contain release %s", istioBase), istioBase),
		t.Entry(fmt.Sprintf("istio-system doesn't contain release %s", istioIngress), istioIngress),
		t.Entry(fmt.Sprintf("istio-system doesn't contain release %s", istioEgress), istioEgress),
		t.Entry(fmt.Sprintf("istio-system doesn't contain release %s", istioCoreDNS), istioCoreDNS),
	)
})

var _ = t.Describe("Checking if Verrazzano system components are ready, post-upgrade", Label("f:platform-lcm.upgrade"), func() {
	Context("Checking Deployments for post-upgrade", func() {
		t.DescribeTable("Deployment should be ready post-upgrade",
			func(namespace string, componentName string, deploymentName string) {
				// Currently we have no way of determining if some components are installed by looking at the status (grafana)
				// Because of this, make this test non-managed cluster only.
				if vzcr.Spec.Profile == vzapi.ManagedCluster {
					return
				}
				Eventually(func() bool {
					if isDisabled(componentName) {
						t.Logs.Infof("Skipping disabled component %s", componentName)
						return true
					}
					isVersionAbove1_4_0, err := pkg.IsVerrazzanoMinVersionEventually("1.4.0", kubeconfigPath)
					if err != nil {
						pkg.Log(pkg.Error, fmt.Sprintf("failed to find the verrazzano version: %v", err))
						return false
					}
					if deploymentName == "mysql" && isVersionAbove1_4_0 {
						// skip mysql for version greater than 1.4.0
						return true
					}
					deployment, err := pkg.GetDeployment(namespace, deploymentName)
					if err != nil {
						return false
					}
					return deployment.Status.ReadyReplicas > 0
				}, mediumWait, pollingInterval).Should(BeTrue(), fmt.Sprintf("Deployment %s for component %s is not ready", deploymentName, componentName))
			},
			t.Entry("Checking Deployment coherence-operator", constants.VerrazzanoSystemNamespace, coherence.ComponentName, "coherence-operator"),
			t.Entry("Checking Deployment oam-kubernetes-runtime", constants.VerrazzanoSystemNamespace, oam.ComponentName, "oam-kubernetes-runtime"),
			t.Entry("Checking Deployment verrazzano-application-operator", constants.VerrazzanoSystemNamespace, appoper.ComponentName, "verrazzano-application-operator"),
			t.Entry("Checking Deployment verrazzano-authproxy", constants.VerrazzanoSystemNamespace, authproxy.ComponentName, "verrazzano-authproxy"),
			t.Entry("Checking Deployment verrazzano-console", constants.VerrazzanoSystemNamespace, verrazzano.ComponentName, "verrazzano-console"),
			t.Entry("Checking Deployment verrazzano-monitoring-operator", constants.VerrazzanoSystemNamespace, verrazzano.ComponentName, "verrazzano-monitoring-operator"),
			t.Entry("Checking Deployment vmi-system-grafana", constants.VerrazzanoSystemNamespace, verrazzano.ComponentName, "vmi-system-grafana"),
			t.Entry("Checking Deployment vmi-system-kibana", constants.VerrazzanoSystemNamespace, verrazzano.ComponentName, "vmi-system-kibana"),
			t.Entry("Checking Deployment prometheus-operator-kube-p-operator", vzconst.PrometheusOperatorNamespace, promoperator.ComponentName, "prometheus-operator-kube-p-operator"),
			t.Entry("Checking Deployment weblogic-operator", constants.VerrazzanoSystemNamespace, weblogic.ComponentName, "weblogic-operator"),

			t.Entry("Checking Deployment cert-manager", certmanager.ComponentNamespace, certmanager.ComponentName, "cert-manager"),
			t.Entry("Checking Deployment cert-manager-cainjector", certmanager.ComponentNamespace, certmanager.ComponentName, "cert-manager-cainjector"),
			t.Entry("Checking Deployment cert-manager-webhook", certmanager.ComponentNamespace, certmanager.ComponentName, "cert-manager-webhook"),

			t.Entry("Checking Deployment external-dns", externaldns.ComponentNamespace, externaldns.ComponentName, "external-dns"),

			t.Entry("Checking Deployment istiod", compistio.IstioNamespace, compistio.ComponentName, "istiod"),
			t.Entry("Checking Deployment istio-ingressgateway", compistio.IstioNamespace, compistio.ComponentName, "istio-ingressgateway"),
			t.Entry("Checking Deployment istio-egressgateway", compistio.IstioNamespace, compistio.ComponentName, "istio-egressgateway"),

			t.Entry("Checking Deployment vmi-system-kiali", constants.VerrazzanoSystemNamespace, kiali.ComponentName, "vmi-system-kiali"),

			t.Entry("Checking Deployment mysql", mysql.ComponentNamespace, mysql.ComponentName, "mysql"),

			t.Entry("Checking Deployment ingress-controller-ingress-nginx-controller", nginx.ComponentNamespace, nginx.ComponentName, "ingress-controller-ingress-nginx-controller"),
			t.Entry("Checking Deployment ingress-controller-ingress-nginx-defaultbackend", nginx.ComponentNamespace, nginx.ComponentName, "ingress-controller-ingress-nginx-defaultbackend"),

			t.Entry("Checking Deployment rancher", rancher.ComponentNamespace, rancher.ComponentName, "rancher"),
			t.Entry("Checking Deployment rancher", rancher.ComponentNamespace, rancher.ComponentName, "rancher-webhook"),
			t.Entry("Checking Deployment fleet-agent", rancher.FleetLocalSystemNamespace, rancher.ComponentName, "fleet-agent"),
			t.Entry("Checking Deployment fleet-controller", rancher.FleetSystemNamespace, rancher.ComponentName, "fleet-controller"),
			t.Entry("Checking Deployment gitjob", rancher.FleetSystemNamespace, rancher.ComponentName, "gitjob"),
		)
	})

	Context("Checking optional Deployments for post-upgrade", func() {
		t.DescribeTable("Deployment should be ready post-upgrade",
			func(namespace string, componentName string, deploymentName string) {
				// Currently we have no way of determining if some components are installed by looking at the status (grafana)
				// Because of this, make this test non-managed cluster only.
				if vzcr.Spec.Profile == vzapi.ManagedCluster {
					return
				}
				Eventually(func() bool {
					if isDisabled(componentName) {
						t.Logs.Infof("Skipping disabled component %s", componentName)
						return true
					}
					deployment, err := pkg.GetDeployment(namespace, deploymentName)
					if err != nil {
						// Deployment is optional, ignore if not found
						// For example es-data and es-ingest won't be there for dev profile
						if errors.IsNotFound(err) {
							t.Logs.Infof("Skipping optional deployment %s since it is not found", deploymentName)
							return true
						}
						return false
					}
					return deployment.Status.ReadyReplicas > 0
				}, mediumWait, pollingInterval).Should(BeTrue(), fmt.Sprintf("Deployment %s for component %s is not ready", deploymentName, componentName))
			},
			t.Entry("Checking Deployment vmi-system-es-data-0", constants.VerrazzanoSystemNamespace, verrazzano.ComponentName, "vmi-system-es-data-0"),
			t.Entry("Checking Deployment vmi-system-es-data-1", constants.VerrazzanoSystemNamespace, verrazzano.ComponentName, "vmi-system-es-data-1"),
			t.Entry("Checking Deployment vmi-system-es-data-2", constants.VerrazzanoSystemNamespace, verrazzano.ComponentName, "vmi-system-es-data-2"),
			t.Entry("Checking Deployment vmi-system-os-ingest", constants.VerrazzanoSystemNamespace, verrazzano.ComponentName, "vmi-system-os-ingest"),
		)
	})

	Context("Checking StatefulSets for post-upgrade", func() {
		t.DescribeTable("StatefulSet should be ready post-upgrade",
			func(namespace string, componentName string, stsName string) {
				// Currently we have no way of determining if some components are installed by looking at the status (grafana)
				// Because of this, make this test non-managed cluster only.
				if vzcr.Spec.Profile == vzapi.ManagedCluster {
					return
				}
				Eventually(func() bool {
					if isDisabled(componentName) {
						t.Logs.Infof("Skipping disabled component %s", componentName)
						return true
					}
					isVersionAbove1_4_0, err := pkg.IsVerrazzanoMinVersionEventually("1.4.0", kubeconfigPath)
					if err != nil {
						pkg.Log(pkg.Error, fmt.Sprintf("failed to find the verrazzano version: %v", err))
						return false
					}
					if stsName == "mysql" && !isVersionAbove1_4_0 {
						// skip mysql for version less than 1.4.0
						return true
					}
					sts, err := pkg.GetStatefulSet(namespace, stsName)
					if err != nil {
						return false
					}
					return sts.Status.ReadyReplicas > 0
				}, mediumWait, pollingInterval).Should(BeTrue(), fmt.Sprintf("Statefulset %s for component %s is not ready", stsName, componentName))
			},
			t.Entry("Checking StatefulSet vmi-system-es-master", constants.VerrazzanoSystemNamespace, appoper.ComponentName, "vmi-system-es-master"),
			t.Entry("Checking StatefulSet keycloak", keycloak.ComponentNamespace, keycloak.ComponentName, "keycloak"),
			t.Entry("Checking StatefulSet mysql", mysql.ComponentNamespace, mysql.ComponentName, "mysql"),
		)
	})

	Context("Checking DaemonSets for post-upgrade", func() {
		t.DescribeTable("DaemonSet should be ready post-upgrade",
			func(namespace string, componentName string, dsName string) {
				// Currently we have no way of determining if some components are installed by looking at the status (grafana)
				// Because of this, make this test non-managed cluster only.
				if vzcr.Spec.Profile == vzapi.ManagedCluster {
					return
				}
				Eventually(func() bool {
					if isDisabled(componentName) {
						t.Logs.Infof("skipping disabled component %s", componentName)
						return true
					}
					ds, err := pkg.GetDaemonSet(namespace, dsName)
					if err != nil {
						return false
					}
					return ds.Status.NumberReady > 0
				}, mediumWait, pollingInterval).Should(BeTrue(), fmt.Sprintf("DaemonSet %s for component %s is not ready", dsName, componentName))
			},
			t.Entry("Checking StatefulSet fluentd", constants.VerrazzanoSystemNamespace, verrazzano.ComponentName, "fluentd"),
		)
	})
})

var _ = t.Describe("Verify prometheus configmap reconciliation,", Label("f:platform-lcm.upgrade", "f:observability.monitoring.prom"), func() {
	// Verify prometheus configmap is reconciled correctly
	// GIVEN upgrade has completed
	// WHEN the vmo pod is restarted
	// THEN expect the vmi system prometheus config to get deleted
	t.It("Verify prometheus configmap is deleted on vmo restart.", func() {
		Eventually(func() bool {
			_, _, _, err := pkg.GetPrometheusConfig()
			return err != nil
		}, mediumWait, pollingInterval).Should(BeTrue(), "Prometheus VMI config should be removed after upgrade.")
	})
})

func isDisabled(componentName string) bool {
	comp, ok := vzcr.Status.Components[componentName]
	if ok {
		return comp.State == vzapi.CompStateDisabled
	}
	return true
}
