// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package restapi_test

import (
	"fmt"

	"github.com/onsi/ginkgo"
	"github.com/onsi/gomega"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg"
)

var api *pkg.APIEndpoint

var _ = ginkgo.Describe("keycloak url test", func() {

	var _ = ginkgo.BeforeEach(func() {
		api = pkg.GetAPIEndpoint()
	})

	ginkgo.Context("Fetching the keycloak url using api and test ", func() {
		ginkgo.It("Fetches keycloak url", func() {
			ingress := api.GetIngress("keycloak", "keycloak")
			keycloakUrl := fmt.Sprintf("https://%s", ingress.Spec.TLS[0].Hosts[0])
			gomega.Expect(keycloakUrl).NotTo(gomega.BeEmpty())

			httpClient := pkg.GetVerrazzanoHTTPClient()
			pkg.ExpectHTTPGetOk(httpClient, keycloakUrl)
		})
	})
})
