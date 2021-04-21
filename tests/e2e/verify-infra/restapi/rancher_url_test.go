// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package restapi

import (
	"fmt"
	"github.com/onsi/gomega"
	"time"

	"github.com/onsi/ginkgo"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg"
)

const (
	waitTimeout     = 5 * time.Minute
	pollingInterval = 5 * time.Second
)

var _ = ginkgo.Describe("rancher url test", func() {
	ginkgo.Context("Fetching the rancher url using api and test ", func() {
		ginkgo.It("Fetches rancher url", func() {
			isManagedClusterProfile := pkg.IsManagedClusterProfile()
			if !isManagedClusterProfile {
				kubeconfigPath := pkg.GetKubeConfigPathFromEnv()
				api := pkg.GetAPIEndpoint(kubeconfigPath)
				rancherURL := func() string {
					ingress := api.GetIngress("cattle-system", "rancher")
					return fmt.Sprintf("https://%s", ingress.Spec.TLS[0].Hosts[0])
				}
				gomega.Eventually(rancherURL, waitTimeout, pollingInterval).ShouldNot(gomega.BeEmpty())

				httpClient := pkg.GetRancherHTTPClient(kubeconfigPath)
				pkg.ExpectHTTPGetOk(httpClient, rancherURL())
			}
		})
	})
})
