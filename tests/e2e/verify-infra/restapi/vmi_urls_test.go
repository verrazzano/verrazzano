// Copyright (c) 2021, 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package restapi_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/hashicorp/go-retryablehttp"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/verrazzano/verrazzano/pkg/k8sutil"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg"
)

var _ = t.Describe("VMI", Label("f:infra-lcm",
	"f:ui.console"), func() {
	const (
		waitTimeout     = 5 * time.Minute
		pollingInterval = 5 * time.Second
	)

	t.Context("urls test to", func() {
		isManagedClusterProfile := pkg.IsManagedClusterProfile()
		var isEsEnabled = false
		var isKibanaEnabled = false
		var isPrometheusEnabled = false
		var isGrafanaEnabled = false

		t.It("Fetch VMI", func() {
			if !isManagedClusterProfile {
				kubeconfigPath, err := k8sutil.GetKubeConfigLocation()
				if err != nil {
					t.Logs.Errorf("Error getting kubeconfig: %v", err)
					Fail(err.Error())
				}

				Eventually(func() bool {
					api, err := pkg.GetAPIEndpoint(kubeconfigPath)
					if err != nil {
						return false
					}
					response, err := api.Get("apis/verrazzano.io/v1/namespaces/verrazzano-system/verrazzanomonitoringinstances/system")
					if err != nil {
						t.Logs.Errorf("Error fetching system VMI from api, error: %v", err)
						return false
					}
					if response.StatusCode != http.StatusOK {
						t.Logs.Errorf("Error fetching system VMI from api, response: %v", response)
						return false
					}

					var vmi map[string]interface{}
					err = json.Unmarshal(response.Body, &vmi)
					if err != nil {
						t.Logs.Errorf("Invalid response for system VMI from api, error: %v", err)
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

		t.It("Access VMI endpoints", FlakeAttempts(5), Label("f:ui.api"), func() {
			if !isManagedClusterProfile {
				var api *pkg.APIEndpoint
				kubeconfigPath, err := k8sutil.GetKubeConfigLocation()
				Expect(err).ShouldNot(HaveOccurred())
				Eventually(func() (*pkg.APIEndpoint, error) {
					var err error
					api, err = pkg.GetAPIEndpoint(kubeconfigPath)
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
					sysVmiHTTPClient, err = pkg.GetVerrazzanoRetryableHTTPClient()
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

var _ = t.AfterEach(func() {})

func verifySystemVMIComponent(api *pkg.APIEndpoint, sysVmiHTTPClient *retryablehttp.Client, vmiCredentials *pkg.UsernamePassword, ingressName, expectedURLPrefix string) bool {
	ingress, err := api.GetIngress("verrazzano-system", ingressName)
	if err != nil {
		t.Logs.Errorf("Error getting ingress from API: %v", err)
		return false
	}
	vmiComponentURL := fmt.Sprintf("https://%s", ingress.Spec.Rules[0].Host)
	if !strings.HasPrefix(vmiComponentURL, expectedURLPrefix) {
		t.Logs.Errorf("URL '%s' does not have expected prefix: %s", vmiComponentURL, expectedURLPrefix)
		return false
	}
	return pkg.AssertURLAccessibleAndAuthorized(sysVmiHTTPClient, vmiComponentURL, vmiCredentials)
}
