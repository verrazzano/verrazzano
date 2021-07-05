// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package restapi

import (
	"fmt"
	"net/http"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg"
)

var _ = Describe("rancher url test", func() {
	const (
		waitTimeout     = 5 * time.Minute
		pollingInterval = 5 * time.Second
	)

	Context("Fetching the rancher url using api and test ", func() {
		It("Fetches rancher url", func() {
			if !pkg.IsManagedClusterProfile() {
				kubeconfigPath := pkg.GetKubeConfigPathFromEnv()
				var rancherURL string

				Eventually(func() error {
					api, err := pkg.GetAPIEndpoint(kubeconfigPath)
					if err != nil {
						return err
					}
					ingress, err := api.GetIngress("cattle-system", "rancher")
					if err != nil {
						return err
					}
					rancherURL = fmt.Sprintf("https://%s", ingress.Spec.TLS[0].Hosts[0])
					pkg.Log(pkg.Info, fmt.Sprintf("Found ingress URL: %s", rancherURL))
					return nil
				}, waitTimeout, pollingInterval).Should(BeNil())

				Expect(rancherURL).NotTo(BeEmpty())
				var httpResponse *pkg.HTTPResponse

				Eventually(func() bool {
					httpClient, err := pkg.GetRancherHTTPClient(kubeconfigPath)
					if err != nil {
						pkg.Log(pkg.Error, fmt.Sprintf("Error getting HTTP client: %v", err))
						return false
					}
					httpResponse, err = pkg.GetWebPageWithClient(httpClient, rancherURL, "")
					if err != nil {
						pkg.Log(pkg.Error, fmt.Sprintf("Error making get request to url: %s, error: %v", rancherURL, err))
						return false
					}
					if httpResponse.StatusCode != http.StatusOK {
						pkg.Log(pkg.Error, fmt.Sprintf("Error making get request to url: %s, response: %v", rancherURL, httpResponse))
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
