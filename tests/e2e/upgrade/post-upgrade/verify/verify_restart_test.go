// Copyright (c) 2021, 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package verify

import (
	"fmt"
	"time"

	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/keycloak"

	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/oam"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/weblogic"

	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/appoper"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/authproxy"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/coherence"

	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"

	vzconst "github.com/verrazzano/verrazzano/pkg/constants"

	"github.com/verrazzano/verrazzano/pkg/helm"
	"github.com/verrazzano/verrazzano/pkg/istio"
	"github.com/verrazzano/verrazzano/pkg/log/vzlog"
	"github.com/verrazzano/verrazzano/pkg/test/framework"
	"github.com/verrazzano/verrazzano/platform-operator/constants"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/certmanager"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/externaldns"
	compistio "github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/istio"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/kiali"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/mysql"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/nginx"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/rancher"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/verrazzano"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

const (
	oneMinute   = 1 * time.Minute
	twoMinutes  = 2 * time.Minute
	fiveMinutes = 5 * time.Minute

	pollingInterval = 10 * time.Second
	envoyImage      = "proxyv2:1.10"
)

var t = framework.NewTestFramework("verify")

var vzcr *vzapi.Verrazzano

var _ = t.BeforeSuite(func() {
	// Get the Verrazzano CR
	Eventually(func() error {
		var err error
		vzcr, err = pkg.GetVerrazzano()
		return err
	}, oneMinute, pollingInterval).Should(BeNil(), "Expected to get Verrazzano CR")
})
var _ = t.AfterSuite(func() {})
var _ = t.AfterEach(func() {})

var _ = t.Describe("Post upgrade", Label("f:platform-lcm.upgrade"), func() {

	// It Wrapper to only run spec if component is supported on the current Verrazzano installation
	MinimumVerrazzanoIt := func(description string, f interface{}) {
		supported, err := pkg.IsVerrazzanoMinVersion("1.1.0")
		if err != nil {
			Fail(err.Error())
		}
		// Only run tests if Verrazzano is not at least version 1.1.0
		if supported {
			t.It(description, f)
		} else {
			pkg.Log(pkg.Info, fmt.Sprintf("Skipping check '%v', Verrazzano is not at version 1.1.0", description))
		}
	}

	// GIVEN the verrazzano-system namespace
	// WHEN the container images are retrieved
	// THEN verify that each pod that uses istio has the correct istio proxy image
	MinimumVerrazzanoIt("pods in verrazzano-system have correct istio proxy image", func() {
		Eventually(func() bool {
			return pkg.CheckPodsForEnvoySidecar(constants.VerrazzanoSystemNamespace, envoyImage)
		}, fiveMinutes, pollingInterval).Should(BeTrue(), "Expected to find istio proxy image in verrazzano-system")
	})

	// GIVEN the ingress-nginx namespace
	// WHEN the container images are retrieved
	// THEN verify that each pod that uses istio has the correct istio proxy image
	MinimumVerrazzanoIt("pods in ingress-nginx have correct istio proxy image", func() {
		Eventually(func() bool {
			return pkg.CheckPodsForEnvoySidecar(constants.IngressNginxNamespace, envoyImage)
		}, fiveMinutes, pollingInterval).Should(BeTrue(), "Expected to find istio proxy image in ingress-nginx")
	})

	// GIVEN the keycloak namespace
	// WHEN the container images are retrieved
	// THEN verify that each pod that uses istio has the correct istio proxy image
	MinimumVerrazzanoIt("pods in keycloak have correct istio proxy image", func() {
		Eventually(func() bool {
			return pkg.CheckPodsForEnvoySidecar(constants.KeycloakNamespace, envoyImage)
		}, fiveMinutes, pollingInterval).Should(BeTrue(), "Expected to find istio proxy image in keycloak")
	})
})

var _ = t.Describe("Application pods post-upgrade", Label("f:platform-lcm.upgrade"), func() {
	const (
		bobsBooksNamespace    = "bobs-books"
		helloHelidonNamespace = "hello-helidon"
		springbootNamespace   = "springboot"
		todoListNamespace     = "todo-list"
	)
	t.DescribeTable("should contain Envoy sidecar 1.10.4",
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
				pkg.Log(pkg.Info, fmt.Sprintf("Skipping test since namespace %s doesn't exist", namespace))
			}
		},
		t.Entry(fmt.Sprintf("pods in namespace %s have Envoy sidecar", helloHelidonNamespace), helloHelidonNamespace, twoMinutes),
		t.Entry(fmt.Sprintf("pods in namespace %s have Envoy sidecar", springbootNamespace), springbootNamespace, twoMinutes),
		t.Entry(fmt.Sprintf("pods in namespace %s have Envoy sidecar", todoListNamespace), todoListNamespace, fiveMinutes),
		t.Entry(fmt.Sprintf("pods in namespace %s have Envoy sidecar", bobsBooksNamespace), bobsBooksNamespace, fiveMinutes),
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
			}, twoMinutes, pollingInterval).Should(BeFalse(), fmt.Sprintf("Expected to not find release %s in istio-system", release))
		},
		t.Entry(fmt.Sprintf("istio-system doesn't contain release %s", istiod), istiod),
		t.Entry(fmt.Sprintf("istio-system doesn't contain release %s", istioBase), istioBase),
		t.Entry(fmt.Sprintf("istio-system doesn't contain release %s", istioIngress), istioIngress),
		t.Entry(fmt.Sprintf("istio-system doesn't contain release %s", istioEgress), istioEgress),
		t.Entry(fmt.Sprintf("istio-system doesn't contain release %s", istioCoreDNS), istioCoreDNS),
	)
})

var _ = t.Describe("istioctl verify-install", func() {
	framework.VzIt("should not return an error", func() {
		Eventually(func() error {
			stdout, _, err := istio.VerifyInstall(vzlog.DefaultLogger())
			if err != nil {
				pkg.Log(pkg.Error, string(stdout))
			}
			return err
		}, twoMinutes, pollingInterval).Should(BeNil(), "istioctl verify-install return with stderr")
	})
})

var _ = t.Describe("Checking if Verrazzano system components are ready, post-upgrade", Label("f:platform-lcm.upgrade"), func() {
	Context("Checking Deployments for post-upgrade", func() {
		t.DescribeTable("Deployment should be ready post-upgrade",
			func(namespace string, componentName string, deploymentName string) {
				Eventually(func() bool {
					if isDisabled(deploymentName) {
						pkg.Log(pkg.Info, fmt.Sprintf("skipping disabled component %s", componentName))
						return true
					}
					pkg.Log(pkg.Info, fmt.Sprintf("checking Deployment %s for component %s", deploymentName, componentName))
					deployment, err := pkg.GetDeployment(namespace, deploymentName)
					if err != nil {
						return false
					}
					return deployment.Status.ReadyReplicas > 0
				}, twoMinutes, pollingInterval).Should(BeFalse(), fmt.Sprintf("Deployment %s for component %s is not ready", deploymentName, componentName))
			},
			t.Entry(constants.VerrazzanoSystemNamespace, appoper.ComponentName, "verrazzano-application-operator"),
			t.Entry(constants.VerrazzanoSystemNamespace, authproxy.ComponentName, "verrazzano-authproxy"),
			t.Entry(constants.VerrazzanoSystemNamespace, coherence.ComponentName, "coherence-operator"),
			t.Entry(constants.VerrazzanoSystemNamespace, oam.ComponentName, "oam-kubernetes-runtime"),
			t.Entry(constants.VerrazzanoSystemNamespace, verrazzano.ComponentName, "verrazzano-console"),
			t.Entry(constants.VerrazzanoSystemNamespace, verrazzano.ComponentName, "vmi-system-grafana"),
			t.Entry(constants.VerrazzanoSystemNamespace, verrazzano.ComponentName, "verrazzano-console"),
			t.Entry(constants.VerrazzanoSystemNamespace, verrazzano.ComponentName, "vmi-system-kiali"),
			t.Entry(constants.VerrazzanoSystemNamespace, verrazzano.ComponentName, "vmi-system-kibana"),
			t.Entry(constants.VerrazzanoSystemNamespace, verrazzano.ComponentName, "vmi-system-prometheus-0"),
			t.Entry(constants.VerrazzanoSystemNamespace, weblogic.ComponentName, "weblogic-operator"),

			t.Entry(certmanager.ComponentNamespace, certmanager.ComponentName, "cert-manager"),
			t.Entry(certmanager.ComponentNamespace, certmanager.ComponentName, "cert-manager-cainjector"),
			t.Entry(certmanager.ComponentNamespace, certmanager.ComponentName, "cert-manager-webhook"),

			t.Entry(externaldns.ComponentNamespace, externaldns.ComponentName, externaldns.ComponentName),

			t.Entry(compistio.IstioNamespace, compistio.ComponentName, compistio.ComponentName),

			t.Entry(kiali.ComponentNamespace, kiali.ComponentName, kiali.ComponentName),

			t.Entry(mysql.ComponentNamespace, mysql.ComponentName, mysql.ComponentName),

			t.Entry(nginx.ComponentNamespace, nginx.ComponentName, "ingress-controller-ingress-nginx-controller"),
			t.Entry(nginx.ComponentNamespace, nginx.ComponentName, "ingress-controller-ingress-nginx-defaultbackend"),

			t.Entry(rancher.ComponentNamespace, rancher.ComponentName, "rancher"),
			t.Entry(rancher.ComponentNamespace, rancher.ComponentName, "rancher-webhook"),
			t.Entry("fleet-system", rancher.ComponentName, "fleet-agent"),
			t.Entry("fleet-system", rancher.ComponentName, "fleet-controller"),
			t.Entry("fleet-system", rancher.ComponentName, "gitjob"),
			t.Entry("rancher-operator-system", rancher.ComponentName, "rancher-operator"),
		)
	})

	Context("Checking StatefulSets for post-upgrade", func() {
		t.DescribeTable("StatefulSet should be ready post-upgrade",
			func(namespace string, componentName string, stsName string) {
				Eventually(func() bool {
					if isDisabled(stsName) {
						pkg.Log(pkg.Info, fmt.Sprintf("skipping disabled component %s", componentName))
						return true
					}
					pkg.Log(pkg.Info, fmt.Sprintf("checking StatefulSet %s for component %s", stsName, componentName))
					sts, err := pkg.GetStatefulSet(namespace, stsName)
					if err != nil {
						return false
					}
					return sts.Status.ReadyReplicas > 0
				}, twoMinutes, pollingInterval).Should(BeFalse(), fmt.Sprintf("Statefulset %s for component %s is not ready", stsName, componentName))
			},
			t.Entry(constants.VerrazzanoSystemNamespace, appoper.ComponentName, "vmi-system-es-master"),
			t.Entry(keycloak.ComponentNamespace, keycloak.ComponentName, "keycloak"),
		)
	})

	Context("Checking DaemonSets for post-upgrade", func() {
		t.DescribeTable("DaemonSet should be ready post-upgrade",
			func(namespace string, componentName string, dsName string) {
				Eventually(func() bool {
					if isDisabled(dsName) {
						pkg.Log(pkg.Info, fmt.Sprintf("skipping disabled component %s", componentName))
						return true
					}
					pkg.Log(pkg.Info, fmt.Sprintf("checking DaemonSets %s for component %s", dsName, componentName))
					ds, err := pkg.GetDaemonSet(namespace, dsName)
					if err != nil {
						return false
					}
					return ds.Status.NumberReady > 0
				}, twoMinutes, pollingInterval).Should(BeFalse(), fmt.Sprintf("DaemonSet %s for component %s is not ready", dsName, componentName))
			},
			t.Entry(constants.VerrazzanoSystemNamespace, verrazzano.ComponentName, "fluentd"),
			t.Entry(vzconst.VerrazzanoMonitoringNamespace, verrazzano.ComponentName, "node-exporter"),
		)
	})
})

func isDisabled(componentName string) bool {
	comp, ok := vzcr.Status.Components[componentName]
	if ok {
		return comp.State == vzapi.CompStateDisabled
	}
	return true
}
