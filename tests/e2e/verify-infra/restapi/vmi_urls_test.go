// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package restapi_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/hashicorp/go-retryablehttp"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg"
)

const (
	waitTimeout     = 5 * time.Minute
	pollingInterval = 5 * time.Second
)

var profile v1alpha1.ProfileType

var _ = BeforeSuite(func() {
	Eventually(func() (v1alpha1.ProfileType, error) {
		var err error
		profile, err = pkg.GetVerrazzanoProfile()
		return profile, err
	}, waitTimeout, pollingInterval).ShouldNot(BeNil())
})

var _ = Describe("VMI urls test", func() {
	Context("Fetching the system vmi using api and test urls", func() {
		var isEsEnabled = false
		var isKibanaEnabled = false
		var isPrometheusEnabled = false
		var isGrafanaEnabled = false

		It("Fetches VMI", func() {
			if profile != v1alpha1.ManagedCluster {
				Eventually(func() bool {
					api, err := pkg.GetAPIEndpoint(pkg.GetKubeConfigPathFromEnv())
					if err != nil {
						return false
					}
					response, err := api.Get("apis/verrazzano.io/v1/namespaces/verrazzano-system/verrazzanomonitoringinstances/system")
					if err != nil {
						pkg.Log(pkg.Error, fmt.Sprintf("Error fetching system VMI from api, error: %v", err))
						return false
					}
					if response.StatusCode != http.StatusOK {
						pkg.Log(pkg.Error, fmt.Sprintf("Error fetching system VMI from api, response: %v", response))
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
				}, waitTimeout, pollingInterval).Should(BeTrue())
			}
		})

		It("Accesses VMI endpoints", func() {
			if profile != v1alpha1.ManagedCluster {
				var api *pkg.APIEndpoint
				Eventually(func() (*pkg.APIEndpoint, error) {
					var err error
					api, err = pkg.GetAPIEndpoint(pkg.GetKubeConfigPathFromEnv())
					return api, err
				}, waitTimeout, pollingInterval).ShouldNot(BeNil())

				var vmiCredentials *pkg.UsernamePassword
				Eventually(func() (*pkg.UsernamePassword, error) {
					var err error
					vmiCredentials, err = pkg.GetSystemVMICredentials()
					return vmiCredentials, err
				}, waitTimeout, pollingInterval).ShouldNot(BeNil())

				// Test VMI endpoints
				var sysVmiHTTPClient *retryablehttp.Client
				Eventually(func() (*retryablehttp.Client, error) {
					var err error
					sysVmiHTTPClient, err = pkg.GetSystemVmiHTTPClient()
					return sysVmiHTTPClient, err
				}, waitTimeout, pollingInterval).ShouldNot(BeNil(), "Unable to get system VMI HTTP client")

				if isEsEnabled {
					Eventually(func() bool {
						return verifySystemVMIComponent(api, sysVmiHTTPClient, vmiCredentials, "vmi-system-es-ingest", "https://elasticsearch.vmi.system")
					}, waitTimeout, pollingInterval).Should(BeTrue(), "Unable to access ElasticSearch VMI url")
				}

				if isKibanaEnabled {
					Eventually(func() bool {
						return verifySystemVMIComponent(api, sysVmiHTTPClient, vmiCredentials, "vmi-system-kibana", "https://kibana.vmi.system")
					}, waitTimeout, pollingInterval).Should(BeTrue(), "Unable to access Kibana VMI url")
				}

				if isPrometheusEnabled {
					Eventually(func() bool {
						return verifySystemVMIComponent(api, sysVmiHTTPClient, vmiCredentials, "vmi-system-prometheus", "https://prometheus.vmi.system")
					}, waitTimeout, pollingInterval).Should(BeTrue(), "Unable to access Prometheus VMI url")
				}

				if isGrafanaEnabled {
					Eventually(func() bool {
						return verifySystemVMIComponent(api, sysVmiHTTPClient, vmiCredentials, "vmi-system-grafana", "https://grafana.vmi.system")
					}, waitTimeout, pollingInterval).Should(BeTrue(), "Unable to access Garafana VMI url")
				}
			}
		})
	})
})

func verifySystemVMIComponent(api *pkg.APIEndpoint, sysVmiHTTPClient *retryablehttp.Client, vmiCredentials *pkg.UsernamePassword, ingressName, expectedURLPrefix string) bool {
	ingress, err := api.GetIngress("verrazzano-system", ingressName)
	if err != nil {
		pkg.Log(pkg.Error, fmt.Sprintf("Error getting ingress from API: %v", err))
		return false
	}
	vmiComponentURL := fmt.Sprintf("https://%s", ingress.Spec.TLS[0].Hosts[0])
	if !strings.HasPrefix(vmiComponentURL, expectedURLPrefix) {
		pkg.Log(pkg.Error, fmt.Sprintf("URL '%s' does not have expected prefix: %s", vmiComponentURL, expectedURLPrefix))
		return false
	}
	return pkg.AssertURLAccessibleAndAuthorized(sysVmiHTTPClient, vmiComponentURL, vmiCredentials)
}
