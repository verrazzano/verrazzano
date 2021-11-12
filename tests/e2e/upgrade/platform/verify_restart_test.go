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

	It("Verify pods in verrazzano-system restarted post upgrade", func() {
		Eventually(func() bool {
			return pkg.PodsHasAnnotation(constants.VerrazzanoSystemNamespace, constants.VerrazzanoRestartAnnotation)
		}, waitTimeout, pollingInterval).Should(BeTrue(), "Expected to find restart annotation in verrazzano-system")
	})

	It("Verify pods in ingress-nginx restarted post upgrade", func() {
		Eventually(func() bool {
			return pkg.PodsHasAnnotation(constants.IngressNginxNamespace, constants.VerrazzanoRestartAnnotation)
		}, waitTimeout, pollingInterval).Should(BeTrue(), "Expected to find restart annotation in ingress-nginx")
	})

	It("Verify pods in keycloak restarted post upgrade", func() {
		Eventually(func() bool {
			return pkg.PodsHasAnnotation(constants.KeycloakNamespace, constants.VerrazzanoRestartAnnotation)
		}, waitTimeout, pollingInterval).Should(BeTrue(), "Expected to find restart annotation in keycloak")
	})
})
