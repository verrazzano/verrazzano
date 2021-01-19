// Copyright (c) 2020, 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package restapi_test

import (
	"fmt"

	"github.com/onsi/ginkgo"
	"github.com/onsi/gomega"
	"github.com/verrazzano/verrazzano/tests/e2e/util"
)

var _ = ginkgo.Describe("instances", func() {

	var _ = ginkgo.BeforeEach(func() {
		api = util.GetApiEndpoint()
	})

	ginkgo.Context("Fetching the instance ", func() {
		ginkgo.It("", func() {
			httpClient := util.GetVerrazzanoHTTPClient()
			instance := util.GetVerrazzanoInstance()

			resp, err := api.Get()
			util.ExpectHttpOk(resp, err, "Error doing http(s) get from "+instance.VzAPIURI)

			gomega.Expect(instance.ID).To(gomega.Equal("0"), "Id is wrong")
			gomega.Expect(instance.Name).NotTo(gomega.BeEmpty(), "Name is empty string")
			gomega.Expect(instance.MgmtCluster).NotTo(gomega.BeEmpty(), "Cluster is empty string")
			/* This check has been commented out until the script installers set this field correctly
			   https://jira.oraclecorp.com/jira/browse/VZ-573
			Expect(instance.MgmtPlatform).NotTo(BeEmpty(), "Platform is empty string")  */
			gomega.Expect(instance.Status).To(gomega.Equal("OK"), "Status is not OK")
			gomega.Expect(instance.Version).Should(gomega.MatchRegexp(`\d+\.\d+\.\d+`))

			util.ExpectHTTPGetOk(httpClient, instance.KeyCloakURL)
			util.ExpectHTTPGetOk(httpClient, instance.RancherURL)

			// Get VMI Credentials
			vmiCredentials, err := util.GetSystemVMICredentials()
			if err != nil {
				ginkgo.Fail(fmt.Sprintf("Error retrieving system VMI credentials: %v", err))
			}

			// Test VMI endpoints
			sysVmiHttpClient := util.GetSystemVmiHttpClient()
			gomega.Expect(instance.ElasticURL).Should(gomega.HavePrefix("https://elasticsearch.vmi.system"))
			util.AssertURLAccessibleAndAuthorized(sysVmiHttpClient, instance.ElasticURL, vmiCredentials)
			gomega.Expect(instance.GrafanaURL).Should(gomega.HavePrefix("https://grafana.vmi.system"))
			util.AssertURLAccessibleAndAuthorized(sysVmiHttpClient, instance.GrafanaURL, vmiCredentials)
			gomega.Expect(instance.KibanaURL).Should(gomega.HavePrefix("https://kibana.vmi.system"))
			util.AssertURLAccessibleAndAuthorized(sysVmiHttpClient, instance.KibanaURL, vmiCredentials)
			gomega.Expect(instance.PrometheusURL).Should(gomega.HavePrefix("https://prometheus.vmi.system"))
			util.AssertURLAccessibleAndAuthorized(sysVmiHttpClient, instance.PrometheusURL, vmiCredentials)
		})
	})
})
