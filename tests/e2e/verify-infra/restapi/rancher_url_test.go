// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package restapi

import (
	"fmt"
	"time"

	"github.com/onsi/gomega"

	"github.com/onsi/ginkgo"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg"
)

var _ = ginkgo.Describe("rancher url test", func() {
	const (
		waitTimeout     = 5 * time.Minute
		pollingInterval = 5 * time.Second
	)

	ginkgo.Context("Fetching the rancher url using api and test ", func() {
		ginkgo.It("Fetches rancher url", func() {
			if !pkg.IsManagedClusterProfile() {
				kubeconfigPath := pkg.GetKubeConfigPathFromEnv()
				var rancherURL string

				gomega.Eventually(func() error {
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
				}, waitTimeout, pollingInterval).Should(gomega.BeNil())

				gomega.Expect(rancherURL).NotTo(gomega.BeEmpty())
				httpClient := pkg.GetRancherHTTPClient(kubeconfigPath)

				gomega.Eventually(func() bool {
					return pkg.MakeHTTPGetRequest(httpClient, rancherURL)
				}, waitTimeout, pollingInterval).Should(gomega.BeTrue())
			}
		})
	})
})
