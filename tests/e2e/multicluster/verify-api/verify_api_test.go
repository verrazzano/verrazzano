// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package api_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/verrazzano/pkg/k8sutil"
	"github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg"
)

const (
	waitTimeout     = 3 * time.Minute
	pollingInterval = 10 * time.Second
)

var managedClusterName = os.Getenv("MANAGED_CLUSTER_NAME")
var adminKubeconfig = os.Getenv("ADMIN_KUBECONFIG")

var _ = Describe("Multi Cluster Verify API", func() {
	Context("Admin Cluster", func() {
		BeforeEach(func() {
			os.Setenv(k8sutil.ENV_VAR_TEST_KUBECONFIG, os.Getenv("ADMIN_KUBECONFIG"))
		})

		It("Get and Validate Verrazzano resource for admin cluster", func() {
			Eventually(func() bool {
				api, err := pkg.GetAPIEndpoint(adminKubeconfig)
				if err != nil {
					pkg.Log(pkg.Error, fmt.Sprintf("Error fetching api: %v", err))
					return false
				}
				response, err := api.Get("apis/install.verrazzano.io/v1alpha1/verrazzanos")
				return isValidVerrazzanosResponse(response, err)
			}, waitTimeout, pollingInterval).Should(BeTrue())
		})

		It("Get and Validate Verrazzano resource for managed cluster", func() {
			Eventually(func() bool {
				api, err := pkg.GetAPIEndpoint(adminKubeconfig)
				if err != nil {
					pkg.Log(pkg.Error, fmt.Sprintf("Error fetching api: %v", err))
					return false
				}
				response, err := api.Get("apis/install.verrazzano.io/v1alpha1/verrazzanos?cluster=" + managedClusterName)
				return isValidVerrazzanosResponse(response, err)
			}, waitTimeout, pollingInterval).Should(BeTrue())
		})
	})
})

func isValidVerrazzanosResponse(response *pkg.HTTPResponse, err error) bool {
	if err != nil {
		pkg.Log(pkg.Error, fmt.Sprintf("Error fetching verrazzanos from api, error: %v", err))
		return false
	}
	if response.StatusCode != http.StatusOK {
		pkg.Log(pkg.Error, fmt.Sprintf("Error fetching verrazzanos from api, response: %v", response))
		return false
	}

	verrazzanos := v1alpha1.VerrazzanoList{}
	err = json.Unmarshal(response.Body, &verrazzanos)
	if err != nil {
		pkg.Log(pkg.Error, fmt.Sprintf("Unable to unmarshal api response, error: %v", err))
		return false
	}

	if len(verrazzanos.Items) != 1 {
		pkg.Log(pkg.Error, fmt.Sprintf("Expected to find 1 verrazzanos but found: %d", len(verrazzanos.Items)))
		return false
	}

	return true
}
