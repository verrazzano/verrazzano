// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package restapi_test

import (
	"fmt"
	"net/http"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg"
)

var _ = Describe("keycloak url test", func() {
	const (
		waitTimeout     = 5 * time.Minute
		pollingInterval = 5 * time.Second
	)

	Context("Fetching the keycloak url using api and test ", func() {
		It("Fetches keycloak url", func() {
			if !pkg.IsManagedClusterProfile() {
				var keycloakURL string
				Eventually(func() error {
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
				}, waitTimeout, pollingInterval).Should(BeNil())

				Expect(keycloakURL).NotTo(BeEmpty())
				var httpResponse *pkg.HTTPResponse

				Eventually(func() bool {
					var err error
					httpResponse, err = pkg.GetWebPage(keycloakURL, "")
					if err != nil {
						pkg.Log(pkg.Error, fmt.Sprintf("Error making get request to url: %s, error: %v", keycloakURL, err))
						return false
					}
					if httpResponse.StatusCode != http.StatusOK {
						pkg.Log(pkg.Error, fmt.Sprintf("Error making get request to url: %s, response: %v", keycloakURL, httpResponse))
						return false
					}
					return true
				}, waitTimeout, pollingInterval).Should(BeTrue())

				Expect(httpResponse).NotTo(BeNil())
				Expect(pkg.CheckNoServerHeader(httpResponse)).To(BeTrue(), "Found unexpected server header in response")
			}
		})
	})
})
