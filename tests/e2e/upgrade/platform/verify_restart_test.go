// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package platform

import (
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

var _ = Describe("verify platform pods restarted", func() {

	// GIVEN the verrazzano-system namespace
	// WHEN the annotations from the pods are retrieved
	// THEN verify that the have the verrazzano.io/restartedAt annotations
	It("Verify pods in verrazzano-system restarted post upgrade", func() {
		Eventually(func() bool {
			return pkg.PodsHaveAnnotation(constants.VerrazzanoSystemNamespace, constants.VerrazzanoRestartAnnotation)
		}, waitTimeout, pollingInterval).Should(BeTrue(), "Expected to find restart annotation in verrazzano-system")
	})

	// GIVEN the ingress-nginx namespace
	// WHEN the annotations from the pods are retrieved
	// THEN verify that the have the verrazzano.io/restartedAt annotations
	It("Verify pods in ingress-nginx restarted post upgrade", func() {
		Eventually(func() bool {
			return pkg.PodsHaveAnnotation(constants.IngressNginxNamespace, constants.VerrazzanoRestartAnnotation)
		}, waitTimeout, pollingInterval).Should(BeTrue(), "Expected to find restart annotation in ingress-nginx")
	})

	// GIVEN the keycloak namespace
	// WHEN the annotations from the pods are retrieved
	// THEN verify that the have the verrazzano.io/restartedAt annotations
	It("Verify pods in keycloak restarted post upgrade", func() {
		Eventually(func() bool {
			return pkg.PodsHaveAnnotation(constants.KeycloakNamespace, constants.VerrazzanoRestartAnnotation)
		}, waitTimeout, pollingInterval).Should(BeTrue(), "Expected to find restart annotation in keycloak")
	})
})

var _ = Describe("verify platform pods have correct istio proxy image", func() {

	// GIVEN the verrazzano-system namespace
	// WHEN the container images are retrieved
	// THEN verify that each pod that uses istio has the correct istio proxy image
	It("Verify pods in verrazzano-system have correct istio proxy image", func() {
		Eventually(func() bool {
			return pkg.PodsHaveIstioSidecar(constants.VerrazzanoSystemNamespace)
		}, waitTimeout, pollingInterval).Should(BeTrue(), "Expected to find istio proxy image in verrazzano-system")
	})

	// GIVEN the ingress-nginx namespace
	// WHEN the container images are retrieved
	// THEN verify that each pod that uses istio has the correct istio proxy image
	It("Verify pods in ingress-nginx have correct istio proxy image", func() {
		Eventually(func() bool {
			return pkg.PodsHaveIstioSidecar(constants.IngressNginxNamespace)
		}, waitTimeout, pollingInterval).Should(BeTrue(), "Expected to find istio proxy image in ingress-nginx")
	})

	// GIVEN the keycloak namespace
	// WHEN the container images are retrieved
	// THEN verify that each pod that uses istio has the correct istio proxy image
	It("Verify pods in keycloak have correct istio proxy image", func() {
		Eventually(func() bool {
			return pkg.PodsHaveIstioSidecar(constants.KeycloakNamespace)
		}, waitTimeout, pollingInterval).Should(BeTrue(), "Expected to find istio proxy image in keycloak")
	})
})
