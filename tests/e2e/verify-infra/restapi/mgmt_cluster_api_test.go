// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package restapi_test

import (
	"encoding/json"
	"fmt"

	"github.com/onsi/ginkgo"
	"github.com/onsi/gomega"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg"
)

var _ = ginkgo.Describe("managed cluster api test", func() {

	var _ = ginkgo.BeforeEach(func() {
		api = pkg.GetApiEndpoint()
	})

	ginkgo.Context("Fetching the managed clusters", func() {
		ginkgo.It("Fetches managed clusters from api", func() {
			response, err := api.Get("apis/verrazzano.io/v1beta1/verrazzanomanagedclusters")
			pkg.ExpectHttpOk(response, err, fmt.Sprintf("Error fetching managed clusters from api, error: %v, response: %v", err, response))

			var managedClustersResponse map[string]interface{}
			err = json.Unmarshal(response.Body, &managedClustersResponse)
			gomega.Expect(err).To(gomega.BeNil(), fmt.Sprintf("Invalid response for managed clusters from api, error: %v", err))

			var managedClusters []interface{}
			managedClusters = managedClustersResponse["items"].([]interface{})
			gomega.Expect(managedClusters != nil && len(managedClusters) > 0).To(gomega.BeTrue(), fmt.Sprintf("No managed clusters returned from api"))
		})
	})
})
