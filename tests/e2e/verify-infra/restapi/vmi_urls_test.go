// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package restapi_test

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/hashicorp/go-retryablehttp"
	"github.com/onsi/ginkgo"
	"github.com/onsi/gomega"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg"
)

var _ = ginkgo.Describe("VMI urls test", func() {
	const (
		waitTimeout     = 5 * time.Minute
		pollingInterval = 5 * time.Second
	)

	ginkgo.Context("Fetching the system vmi using api and test urls", func() {
		isManagedClusterProfile := pkg.IsManagedClusterProfile()
		var isEsEnabled = false
		var isKibanaEnabled = false
		var isPrometheusEnabled = false
		var isGrafanaEnabled = false

		ginkgo.It("Fetches VMI", func() {
			if !isManagedClusterProfile {
				gomega.Eventually(func() bool {
					api := pkg.GetAPIEndpoint(pkg.GetKubeConfigPathFromEnv())
					response, err := api.Get("apis/verrazzano.io/v1/namespaces/verrazzano-system/verrazzanomonitoringinstances/system")
					if !pkg.IsHTTPStatusOk(response, err, fmt.Sprintf("Error fetching system VMI from api, error: %v, response: %v", err, response)) {
						return false
					}

					var vmi map[string]interface{}
					err = json.Unmarshal(response.Body, &vmi)
					if err != nil {
						pkg.Log(pkg.Error, fmt.Sprintf("Invalid response for system VMI from api, error: %v", err))
						return false
					}

					isEsEnabled = vmi["spec"].(map[string]interface{})["elasticsearch"].(map[string]interface{})["enabled"].(bool)
					isKibanaEnabled = vmi["spec"].(map[string]interface{})["kibana"].(map[string]interface{})["enabled"].(bool)
					isPrometheusEnabled = vmi["spec"].(map[string]interface{})["prometheus"].(map[string]interface{})["enabled"].(bool)
					isGrafanaEnabled = vmi["spec"].(map[string]interface{})["grafana"].(map[string]interface{})["enabled"].(bool)
					return true
				}, waitTimeout, pollingInterval).Should(gomega.BeTrue())
			}
		})

		ginkgo.It("Accesses VMI endpoints", func() {
			if !isManagedClusterProfile {
				api := pkg.GetAPIEndpoint(pkg.GetKubeConfigPathFromEnv())
				vmiCredentials, err := pkg.GetSystemVMICredentials()
				gomega.Expect(err).To(gomega.BeNil(), fmt.Sprintf("Error retrieving system VMI credentials: %v", err))

				// Test VMI endpoints
				sysVmiHTTPClient := pkg.GetSystemVmiHTTPClient()

				if isEsEnabled {
					gomega.Eventually(func() bool {
						return verifySystemVMIComponent(api, sysVmiHTTPClient, vmiCredentials, "vmi-system-es-ingest", "https://elasticsearch.vmi.system")
					}, waitTimeout, pollingInterval).Should(gomega.BeTrue(), "Unable to access ElasticSearch VMI url")
				}

				if isKibanaEnabled {
					gomega.Eventually(func() bool {
						return verifySystemVMIComponent(api, sysVmiHTTPClient, vmiCredentials, "vmi-system-kibana", "https://kibana.vmi.system")
					}, waitTimeout, pollingInterval).Should(gomega.BeTrue(), "Unable to access Kibana VMI url")
				}

				if isPrometheusEnabled {
					gomega.Eventually(func() bool {
						return verifySystemVMIComponent(api, sysVmiHTTPClient, vmiCredentials, "vmi-system-prometheus", "https://prometheus.vmi.system")
					}, waitTimeout, pollingInterval).Should(gomega.BeTrue(), "Unable to access Prometheus VMI url")
				}

				if isGrafanaEnabled {
					gomega.Eventually(func() bool {
						return verifySystemVMIComponent(api, sysVmiHTTPClient, vmiCredentials, "vmi-system-grafana", "https://grafana.vmi.system")
					}, waitTimeout, pollingInterval).Should(gomega.BeTrue(), "Unable to access Garafana VMI url")
				}
			}
		})
	})
})

func verifySystemVMIComponent(api *pkg.APIEndpoint, sysVmiHTTPClient *retryablehttp.Client, vmiCredentials *pkg.UsernamePassword, ingressName, expectedURLPrefix string) bool {
	ingress, err := api.GetIngress("verrazzano-system", ingressName)
	if err != nil {
		return false
	}
	vmiComponentURL := fmt.Sprintf("https://%s", ingress.Spec.TLS[0].Hosts[0])
	if !strings.HasPrefix(vmiComponentURL, expectedURLPrefix) {
		pkg.Log(pkg.Error, fmt.Sprintf("URL '%s' does not have expected prefix: %s", vmiComponentURL, expectedURLPrefix))
		return false
	}
	return pkg.AssertURLAccessibleAndAuthorized(sysVmiHTTPClient, vmiComponentURL, vmiCredentials)
}
