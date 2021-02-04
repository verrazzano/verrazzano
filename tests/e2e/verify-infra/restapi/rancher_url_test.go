// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package restapi

import (
	"fmt"

	"github.com/onsi/ginkgo"
	"github.com/onsi/gomega"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg"
)

var api *pkg.ApiEndpoint

var _ = ginkgo.Describe("rancher url test", func() {

	var _ = ginkgo.BeforeEach(func() {
		api = pkg.GetApiEndpoint()
	})

	ginkgo.Context("Fetching the rancher url using api and test ", func() {
		ginkgo.It("Fetches rancher url", func() {
			ingress := api.GetIngress("cattle-system", "rancher")
			keycloakURL := fmt.Sprintf("https://%s", ingress.Spec.TLS[0].Hosts[0])
			gomega.Expect(keycloakURL).NotTo(gomega.BeEmpty())

			httpClient := pkg.GetVerrazzanoHTTPClient()
			pkg.ExpectHTTPGetOk(httpClient, keycloakURL)
		})
	})
})
