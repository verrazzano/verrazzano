// Copyright (c) 2021, 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package restapi_test

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"time"

	"github.com/verrazzano/verrazzano/pkg/test/framework/metrics"

	"github.com/hashicorp/go-retryablehttp"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/verrazzano/verrazzano/pkg/httputil"
	"github.com/verrazzano/verrazzano/pkg/k8sutil"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg"
)

var _ = t.Describe("rancher", Label("f:infra-lcm",
	"f:ui.console"), func() {
	const (
		waitTimeout     = 5 * time.Minute
		pollingInterval = 5 * time.Second
	)

	t.Context("url test to", func() {
		t.It("Fetch rancher url", func() {
			if !pkg.IsManagedClusterProfile() {
				kubeconfigPath, err := k8sutil.GetKubeConfigLocation()
				if err != nil {
					t.Logs.Error(fmt.Sprintf("Error getting kubeconfig: %v", err))
					t.Fail(err.Error())
				}

				api := pkg.EventuallyGetAPIEndpoint(kubeconfigPath)
				rancherURL := pkg.EventuallyGetRancherURL(t.Logs, api)
				httpClient := pkg.EventuallyVerrazzanoRetryableHTTPClient()
				var httpResponse *pkg.HTTPResponse

				Eventually(func() (*pkg.HTTPResponse, error) {
					httpResponse, err = pkg.GetWebPageWithClient(httpClient, rancherURL, "")
					return httpResponse, err
				}, waitTimeout, pollingInterval).Should(pkg.HasStatus(http.StatusOK))

				Expect(pkg.CheckNoServerHeader(httpResponse)).To(BeTrue(), "Found unexpected server header in response")
				var token string
				start := time.Now()
				Eventually(func() string {
					token = pkg.GetRancherAdminToken(t.Logs, httpClient, rancherURL)
					return token
				}, waitTimeout, pollingInterval).ShouldNot(BeEmpty())
				metrics.Emit(t.Metrics.With("get_token_elapsed_time", time.Since(start).Milliseconds()))
				Expect(token).NotTo(BeEmpty(), "Invalid token returned by rancher")
				start = time.Now()
				Eventually(func() (string, error) {
					return getFieldOrErrorFromRancherAPIResponse(rancherURL, "v3/clusters/local", token, httpClient, "state")
				}, waitTimeout, pollingInterval).Should(Equal("active"), "rancher local cluster not in active state")
				metrics.Emit(t.Metrics.With("get_cluster_state_elapsed_time", time.Since(start).Milliseconds()))

				minVer14, err := pkg.IsVerrazzanoMinVersion("1.4.0", kubeconfigPath)
				Expect(err).ToNot(HaveOccurred())
				if minVer14 {
					start = time.Now()
					Eventually(func() (string, error) {
						return getFieldOrErrorFromRancherAPIResponse(rancherURL, "v3/nodeDrivers/oci", token, httpClient, "state")
					}, waitTimeout, pollingInterval).Should(Equal("active"), "rancher oci driver not activated")
					metrics.Emit(t.Metrics.With("get_oci_driver_state_elapsed_time", time.Since(start).Milliseconds()))

					start = time.Now()
					Eventually(func() (string, error) {
						return getFieldOrErrorFromRancherAPIResponse(rancherURL, "v3/kontainerDrivers/oraclecontainerengine", token, httpClient, "state")
					}, waitTimeout, pollingInterval).Should(Equal("active"), "rancher oke driver not activated")
					metrics.Emit(t.Metrics.With("get_oke_driver_state_elapsed_time", time.Since(start).Milliseconds()))
				}
			}
		})
	})
})

func getFieldOrErrorFromRancherAPIResponse(rancherURL string, apiPath string, token string, httpClient *retryablehttp.Client, field string) (string, error) {
	req, err := retryablehttp.NewRequest("GET", fmt.Sprintf("%s/%s", rancherURL, apiPath), nil)
	if err != nil {
		t.Logs.Error(fmt.Sprintf("error creating rancher api request for %s: %v", apiPath, err))
		return "", err
	}

	req.Header.Set("Authorization", fmt.Sprintf("Bearer %v", token))
	req.Header.Set("Accept", "application/json")
	response, err := httpClient.Do(req)
	if err != nil {
		t.Logs.Error(fmt.Sprintf("error invoking rancher api request %s: %v", apiPath, err))
		return "", err
	}

	err = httputil.ValidateResponseCode(response, http.StatusOK)
	if err != nil {
		return "", err
	}

	defer response.Body.Close()

	// extract the response body
	body, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return "", err
	}

	return httputil.ExtractFieldFromResponseBodyOrReturnError(string(body), field, fmt.Sprintf("unable to find %s in rancher api response for %s", field, apiPath))
}

var _ = t.AfterEach(func() {})
