// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package restapi_test

import (
	"fmt"
	"time"

	"github.com/onsi/ginkgo"
	"github.com/onsi/gomega"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg"
)

var _ = ginkgo.Describe("keycloak url test", func() {
	const (
		waitTimeout     = 5 * time.Minute
		pollingInterval = 5 * time.Second
	)

	ginkgo.Context("Fetching the keycloak url using api and test ", func() {
		ginkgo.It("Fetches keycloak url", func() {
			if !pkg.IsManagedClusterProfile() {
				var keycloakURL string
				gomega.Eventually(func() error {
					api, err := pkg.GetAPIEndpoint(pkg.GetKubeConfigPathFromEnv())
					if err != nil {
						return err
					}
					ingress, err := api.GetIngress("keycloak", "keycloak")
					if err != nil {
						return err
					}
					keycloakURL = fmt.Sprintf("https://%s", ingress.Spec.TLS[0].Hosts[0])
					pkg.Log(pkg.Info, fmt.Sprintf("Found ingress URL: %s", keycloakURL))
					return nil
				}, waitTimeout, pollingInterval).Should(gomega.BeNil())

				gomega.Expect(keycloakURL).NotTo(gomega.BeEmpty())
				httpClient := pkg.GetVerrazzanoHTTPClient()

				gomega.Eventually(func() bool {
					return pkg.MakeHTTPGetRequest(httpClient, keycloakURL)
				}, waitTimeout, pollingInterval).Should(gomega.BeTrue())
			}
		})
	})
})
