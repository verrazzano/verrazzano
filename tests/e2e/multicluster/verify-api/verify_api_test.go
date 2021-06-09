// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package api_test

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/onsi/gomega"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg"

	"github.com/onsi/ginkgo"
	"github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
)

var managedClusterName = os.Getenv("MANAGED_CLUSTER_NAME")
var adminKubeconfig = os.Getenv("ADMIN_KUBECONFIG")

var _ = ginkgo.Describe("Multi Cluster Verify API", func() {
	ginkgo.Context("Admin Cluster", func() {
		ginkgo.It("Get and Validate Verrazzano resource for admin cluster", func() {
			// create a project
			api := pkg.GetAPIEndpoint(adminKubeconfig)
			response, err := api.Get("apis/install.verrazzano.io/v1alpha1/verrazzanos")
			validateVerrazzanosResponse(response, err)
		})

		ginkgo.It("Get and Validate Verrazzano resource for managed cluster", func() {
			// create a project
			api := pkg.GetAPIEndpoint(adminKubeconfig)
			response, err := api.Get("apis/install.verrazzano.io/v1alpha1/verrazzanos?cluster=" + managedClusterName)
			validateVerrazzanosResponse(response, err)
		})
	})
})

func validateVerrazzanosResponse(response *pkg.HTTPResponse, err error) {
	pkg.ExpectHTTPOk(response, err, fmt.Sprintf("Error fetching verrazzanos from api, error: %v, response: %v", err, response))
	verrazzanos := v1alpha1.VerrazzanoList{}
	err = json.Unmarshal(response.Body, &verrazzanos)
	gomega.Expect(err).To(gomega.BeNil(), fmt.Sprintf("Invalid response for verrazzanos from api, error: %v", err))
	gomega.Expect(len(verrazzanos.Items)).To(gomega.Equal(1), fmt.Sprintf("Invalid number of verrazzanos from api, error: %v", err))
	gomega.Expect(verrazzanos.Items[0].Spec.Version).To(gomega.Not(gomega.BeNil()), fmt.Sprintf("Invalid version of  verrazzano resource from api, error: %v", err))
}
