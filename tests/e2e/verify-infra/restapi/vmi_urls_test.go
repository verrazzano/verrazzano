// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package restapi_test

import (
	"encoding/json"
	"fmt"

	"github.com/hashicorp/go-retryablehttp"
	"github.com/onsi/ginkgo"
	"github.com/onsi/gomega"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg"
)

var _ = ginkgo.Describe("vmi urls test", func() {

	var _ = ginkgo.BeforeEach(func() {
		api = pkg.GetApiEndpoint()
	})

	ginkgo.Context("Fetching the system vmi using api and test urls", func() {
		ginkgo.It("Fetches vmi", func() {
			response, err := api.Get("apis/verrazzano.io/v1/namespaces/verrazzano-system/verrazzanomonitoringinstances/system")
			pkg.ExpectHttpOk(response, err, fmt.Sprintf("Error fetching system vmi from api, error: %v, response: %v", err, response))

			var vmi map[string]interface{}
			err = json.Unmarshal(response.Body, &vmi)
			gomega.Expect(err).To(gomega.BeNil(), fmt.Sprintf("Invalid response for system vmi from api, error: %v", err))

			isEsEnabled := vmi["spec"].(map[string]interface{})["elasticsearch"].(map[string]interface{})["enabled"].(bool)
			isKibanaEnabled := vmi["spec"].(map[string]interface{})["kibana"].(map[string]interface{})["enabled"].(bool)
			isPrometheusEnabled := vmi["spec"].(map[string]interface{})["prometheus"].(map[string]interface{})["enabled"].(bool)
			isGrafanaEnabled := vmi["spec"].(map[string]interface{})["grafana"].(map[string]interface{})["enabled"].(bool)

			vmiCredentials, err := pkg.GetSystemVMICredentials()
			gomega.Expect(err).To(gomega.BeNil(), fmt.Sprintf("Error retrieving system VMI credentials: %v", err))

			// Test VMI endpoints
			sysVmiHttpClient := pkg.GetSystemVmiHttpClient()

			if isEsEnabled {
				gomega.Expect(verifySystemVMIComponent(sysVmiHttpClient, vmiCredentials, "vmi-system-es-ingest", "https://elasticsearch.vmi.system")).To(gomega.BeTrue(), fmt.Sprintf("Unable to access ElasticSearch VMI url"))
			}

			if isKibanaEnabled {
				gomega.Expect(verifySystemVMIComponent(sysVmiHttpClient, vmiCredentials, "vmi-system-kibana", "https://kibana.vmi.system")).To(gomega.BeTrue(), fmt.Sprintf("Unable to access Kibana VMI url"))
			}

			if isPrometheusEnabled {
				gomega.Expect(verifySystemVMIComponent(sysVmiHttpClient, vmiCredentials, "vmi-system-prometheus", "https://prometheus.vmi.system")).To(gomega.BeTrue(), fmt.Sprintf("Unable to access Prometheus VMI url"))
			}

			if isGrafanaEnabled {
				gomega.Expect(verifySystemVMIComponent(sysVmiHttpClient, vmiCredentials, "vmi-system-grafana", "https://grafana.vmi.system")).To(gomega.BeTrue(), fmt.Sprintf("Unable to access Garafana VMI url"))
			}

		})
	})
})

func verifySystemVMIComponent(sysVmiHttpClient *retryablehttp.Client, vmiCredentials *pkg.UsernamePassword, ingressName, expectedUrlPrefix string) bool {
	ingress := api.GetIngress("verrazzano-system", ingressName)
	vmiComponentURL := fmt.Sprintf("https://%s", ingress.Spec.TLS[0].Hosts[0])
	gomega.Expect(vmiComponentURL).Should(gomega.HavePrefix(expectedUrlPrefix))
	pkg.AssertURLAccessibleAndAuthorized(sysVmiHttpClient, vmiComponentURL, vmiCredentials)
	return true
}
