// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package api_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg"
)

var managedClusterName = os.Getenv("MANAGED_CLUSTER_NAME")
var adminKubeconfig = os.Getenv("ADMIN_KUBECONFIG")

var _ = Describe("Multi Cluster Verify API", func() {
	Context("Admin Cluster", func() {
		BeforeEach(func() {
			os.Setenv("TEST_KUBECONFIG", os.Getenv("ADMIN_KUBECONFIG"))
		})

		It("Get and Validate Verrazzano resource for admin cluster", func() {
			api, err := pkg.GetAPIEndpoint(adminKubeconfig)
			Expect(err).ShouldNot(HaveOccurred(), "Erroring getting API endpoint")
			response, err := api.Get("apis/install.verrazzano.io/v1alpha1/verrazzanos")
			validateVerrazzanosResponse(response, err)
		})

		It("Get and Validate Verrazzano resource for managed cluster", func() {
			api, err := pkg.GetAPIEndpoint(adminKubeconfig)
			Expect(err).ShouldNot(HaveOccurred(), "Erroring getting API endpoint")
			response, err := api.Get("apis/install.verrazzano.io/v1alpha1/verrazzanos?cluster=" + managedClusterName)
			validateVerrazzanosResponse(response, err)
		})
	})
})

func validateVerrazzanosResponse(response *pkg.HTTPResponse, err error) {
	Expect(err).ShouldNot(HaveOccurred(), fmt.Sprintf("Error fetching verrazzanos from api, error: %v", err))
	Expect(response.StatusCode).To(Equal(http.StatusOK), fmt.Sprintf("Error fetching verrazzanos from api, response: %v", response))

	verrazzanos := v1alpha1.VerrazzanoList{}
	err = json.Unmarshal(response.Body, &verrazzanos)
	Expect(err).To(BeNil(), fmt.Sprintf("Invalid response for verrazzanos from api, error: %v", err))
	Expect(len(verrazzanos.Items)).To(Equal(1), fmt.Sprintf("Invalid number of verrazzanos from api, error: %v", err))
	Expect(verrazzanos.Items[0].Spec.Version).To(Not(BeNil()), fmt.Sprintf("Invalid version of  verrazzano resource from api, error: %v", err))
}
