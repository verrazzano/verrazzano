// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package ocnedriver

import (
	"github.com/verrazzano/verrazzano/pkg/k8sutil"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg/test/framework"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var t = framework.NewTestFramework("install")

var beforeSuite = t.BeforeSuiteFunc(func() {
	kubeconfigPath, err := k8sutil.GetKubeConfigLocation()
	Expect(err).ShouldNot(HaveOccurred())
	httpClient := pkg.EventuallyVerrazzanoRetryableHTTPClient()
	api := pkg.EventuallyGetAPIEndpoint(kubeconfigPath)
	rancherURL := pkg.EventuallyGetURLForIngress(t.Logs, api, "cattle-system", "rancher", "https")
	adminToken := pkg.GetRancherAdminToken(t.Logs, httpClient, rancherURL)
	_ = adminToken
	// TODO: authenticate using Bearer Token
})
var _ = BeforeSuite(beforeSuite)

var _ = t.Describe("OCNE Cluster Driver", Label("TODO: appropriate label"), Serial, func() {
	t.Context("Cluster Creation", func() {
		t.It("creates an active cluster", func() {
			Expect(1+1).Should(Equal(2))
			// TODO: create a cluster with an HTTP POST request, then verify that the cluster is eventually active
		})
	})
})