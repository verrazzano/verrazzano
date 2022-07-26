// Copyright (c) 2021, 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package restapi_test

import (
	"encoding/json"
	"net/http"
	"time"

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

		t.It("Access VMI endpoints", FlakeAttempts(5), Label("f:ui.api"), func() {
			isManagedClusterProfile := pkg.IsManagedClusterProfile()
			var isEsEnabled = false
			var isKibanaEnabled = false
			var isPrometheusEnabled = false
			var isGrafanaEnabled = false

			if !isManagedClusterProfile {
				kubeconfigPath, err := k8sutil.GetKubeConfigLocation()
				if err != nil {
					t.Logs.Errorf("Error getting kubeconfig: %v", err)
					Fail(err.Error())
				}

				Eventually(func() bool {
					api := pkg.EventuallyGetAPIEndpoint(kubeconfigPath)
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

				kubeconfigPath, err = k8sutil.GetKubeConfigLocation()
				Expect(err).ShouldNot(HaveOccurred())
				api := pkg.EventuallyGetAPIEndpoint(kubeconfigPath)
				vmiCredentials := pkg.EventuallyGetSystemVMICredentials()
				// Test VMI endpoints
				sysVmiHTTPClient := pkg.EventuallyVerrazzanoRetryableHTTPClient()
				if isEsEnabled {
					Eventually(func() bool {
						return pkg.VerifyOpenSearchComponent(t.Logs, api, sysVmiHTTPClient, vmiCredentials)
					}, waitTimeout, pollingInterval).Should(BeTrue(), "Unable to access OpenSearch VMI URL")
				}
				if isKibanaEnabled {
					Eventually(func() bool {
						return pkg.VerifyOpenSearchDashboardsComponent(t.Logs, api, sysVmiHTTPClient, vmiCredentials)
					}, waitTimeout, pollingInterval).Should(BeTrue(), "Unable to access OpenSearch Dashboards VMI URL")
				}
				if isPrometheusEnabled {
					Eventually(func() bool {
						return pkg.VerifyPrometheusComponent(t.Logs, api, sysVmiHTTPClient, vmiCredentials)
					}, waitTimeout, pollingInterval).Should(BeTrue(), "Unable to access Prometheus VMI URL")
				}
				if isGrafanaEnabled {
					Eventually(func() bool {
						return pkg.VerifyGrafanaComponent(t.Logs, api, sysVmiHTTPClient, vmiCredentials)
					}, waitTimeout, pollingInterval).Should(BeTrue(), "Unable to access Grafana VMI URL")

				}
			}
		})
	})
})

var _ = t.AfterEach(func() {})
