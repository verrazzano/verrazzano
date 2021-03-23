// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package restapi

import (
	"fmt"
	"time"

	"github.com/onsi/ginkgo"
	"github.com/onsi/gomega"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg"
)

var api *pkg.APIEndpoint

const (
	waitTimeout     = 5 * time.Minute
	pollingInterval = 5 * time.Second
)

var _ = ginkgo.Describe("rancher url test", func() {

	var _ = ginkgo.BeforeEach(func() {
		api = pkg.GetAPIEndpoint()
	})

	ginkgo.Context("Fetching the rancher url using api and test ", func() {
		ginkgo.It("Fetches rancher url", func() {
			keycloakURL := func() string {
				ingress := api.GetIngress("cattle-system", "rancher")
				return fmt.Sprintf("https://%s", ingress.Spec.TLS[0].Hosts[0])
			}
			gomega.Eventually(keycloakURL, waitTimeout, pollingInterval).ShouldNot(gomega.BeEmpty())

			httpClient := pkg.GetVerrazzanoHTTPClient()
			pkg.ExpectHTTPGetOk(httpClient, keycloakURL())
		})
	})
})
