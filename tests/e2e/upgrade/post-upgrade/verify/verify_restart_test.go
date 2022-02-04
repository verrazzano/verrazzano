// Copyright (c) 2021, 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package verify

import (
	"fmt"
	"time"

	"github.com/verrazzano/verrazzano/pkg/helm"
	"github.com/verrazzano/verrazzano/pkg/istio"
	"github.com/verrazzano/verrazzano/pkg/log/vzlog"
	"github.com/verrazzano/verrazzano/pkg/test/framework"
	"github.com/verrazzano/verrazzano/platform-operator/constants"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

const (
	twoMinutes   = 1 * time.Minute
	threeMinutes = 3 * time.Minute
	fiveMinutes  = 5 * time.Minute

	pollingInterval = 10 * time.Second
	envoyImage      = "proxyv2:1.10"
)

var t = framework.NewTestFramework("verify")

var _ = t.BeforeSuite(func() {})
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
		}, threeMinutes, pollingInterval).Should(BeTrue(), "Expected to find istio proxy image in verrazzano-system")
	})

	// GIVEN the ingress-nginx namespace
	// WHEN the container images are retrieved
	// THEN verify that each pod that uses istio has the correct istio proxy image
	MinimumVerrazzanoIt("pods in ingress-nginx have correct istio proxy image", func() {
		Eventually(func() bool {
			return pkg.CheckPodsForEnvoySidecar(constants.IngressNginxNamespace, envoyImage)
		}, threeMinutes, pollingInterval).Should(BeTrue(), "Expected to find istio proxy image in ingress-nginx")
	})

	// GIVEN the keycloak namespace
	// WHEN the container images are retrieved
	// THEN verify that each pod that uses istio has the correct istio proxy image
	MinimumVerrazzanoIt("pods in keycloak have correct istio proxy image", func() {
		Eventually(func() bool {
			return pkg.CheckPodsForEnvoySidecar(constants.KeycloakNamespace, envoyImage)
		}, threeMinutes, pollingInterval).Should(BeTrue(), "Expected to find istio proxy image in keycloak")
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

var _ = t.Describe("Istio helm releases", func() {
	const (
		istiod       = "istiod"
		istioBase    = "istio"
		istioIngress = "istio-ingress"
		istioEgress  = "istio-egress"
		istioCoreDNS = "istiocoredns"
	)
	t.DescribeTable("should be removed from the istio-system namepsace post upgrade",
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
