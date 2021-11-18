// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package verify

import (
	"fmt"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/verrazzano/verrazzano/platform-operator/constants"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg"
	"time"
)

const (
	waitTimeout     = 3 * time.Minute
	pollingInterval = 10 * time.Second
)

var _ = Describe("verify platform pods post upgrade", func() {

	// It Wrapper to only run spec if Kiali is supported on the current Verrazzano installation
	MinimumVerrazzanoIt := func(description string, f interface{}) {
		supported, err := pkg.IsVerrazzanoMinVersion("1.1.0")
		if err != nil {
			Fail(err.Error())
		}
		// Only run tests if Verrazzano is not at least version 1.1.0
		if supported {
			It(description, f)
		} else {
			pkg.Log(pkg.Info, fmt.Sprintf("Skipping check '%v', Verrazzano is not at version 1.1.0", description))
		}
	}

	// GIVEN the verrazzano-system namespace
	// WHEN the annotations from the pods are retrieved
	// THEN verify that the have the verrazzano.io/restartedAt annotations
	MinimumVerrazzanoIt("Verify pods in verrazzano-system restarted post upgrade", func() {
		Eventually(func() bool {
			return pkg.PodsHaveAnnotation(constants.VerrazzanoSystemNamespace, constants.VerrazzanoRestartAnnotation)
		}, waitTimeout, pollingInterval).Should(BeTrue(), "Expected to find restart annotation in verrazzano-system")
	})

	// GIVEN the ingress-nginx namespace
	// WHEN the annotations from the pods are retrieved
	// THEN verify that the have the verrazzano.io/restartedAt annotations
	MinimumVerrazzanoIt("Verify pods in ingress-nginx restarted post upgrade", func() {
		Eventually(func() bool {
			return pkg.PodsHaveAnnotation(constants.IngressNginxNamespace, constants.VerrazzanoRestartAnnotation)
		}, waitTimeout, pollingInterval).Should(BeTrue(), "Expected to find restart annotation in ingress-nginx")
	})

	// GIVEN the keycloak namespace
	// WHEN the annotations from the pods are retrieved
	// THEN verify that the have the verrazzano.io/restartedAt annotations
	MinimumVerrazzanoIt("Verify pods in keycloak restarted post upgrade", func() {
		Eventually(func() bool {
			return pkg.PodsHaveAnnotation(constants.KeycloakNamespace, constants.VerrazzanoRestartAnnotation)
		}, waitTimeout, pollingInterval).Should(BeTrue(), "Expected to find restart annotation in keycloak")
	})

	// GIVEN the verrazzano-system namespace
	// WHEN the container images are retrieved
	// THEN verify that each pod that uses istio has the correct istio proxy image
	MinimumVerrazzanoIt("Verify pods in verrazzano-system have correct istio proxy image", func() {
		Eventually(func() bool {
			return pkg.CheckPodsForIstioImage(constants.VerrazzanoSystemNamespace)
		}, waitTimeout, pollingInterval).Should(BeTrue(), "Expected to find istio proxy image in verrazzano-system")
	})

	// GIVEN the ingress-nginx namespace
	// WHEN the container images are retrieved
	// THEN verify that each pod that uses istio has the correct istio proxy image
	MinimumVerrazzanoIt("Verify pods in ingress-nginx have correct istio proxy image", func() {
		Eventually(func() bool {
			return pkg.CheckPodsForIstioImage(constants.IngressNginxNamespace)
		}, waitTimeout, pollingInterval).Should(BeTrue(), "Expected to find istio proxy image in ingress-nginx")
	})

	// GIVEN the keycloak namespace
	// WHEN the container images are retrieved
	// THEN verify that each pod that uses istio has the correct istio proxy image
	MinimumVerrazzanoIt("Verify pods in keycloak have correct istio proxy image", func() {
		Eventually(func() bool {
			return pkg.CheckPodsForIstioImage(constants.KeycloakNamespace)
		}, waitTimeout, pollingInterval).Should(BeTrue(), "Expected to find istio proxy image in keycloak")
	})
})
