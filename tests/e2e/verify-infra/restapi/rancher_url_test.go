// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package restapi_test

import (
	"fmt"
	"github.com/verrazzano/verrazzano/pkg/test/framework"
	"github.com/verrazzano/verrazzano/pkg/test/framework/metrics"
	"io/ioutil"
	"net/http"
	"strings"
	"time"

	"github.com/hashicorp/go-retryablehttp"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/verrazzano/verrazzano/pkg/httputil"
	"github.com/verrazzano/verrazzano/pkg/k8sutil"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg"
)

var _ = framework.VzDescribe("rancher url test", func() {
	const (
		waitTimeout     = 5 * time.Minute
		pollingInterval = 5 * time.Second
	)

	Context("Fetching the rancher url using api and test ", func() {
		framework.ItM(metricsLogger,"Fetches rancher url", func() {
			if !pkg.IsManagedClusterProfile() {
				kubeconfigPath, err := k8sutil.GetKubeConfigLocation()
				if err != nil {
					pkg.Log(pkg.Error, fmt.Sprintf("Error getting kubeconfig: %v", err))
					Fail(err.Error())
				}

				var rancherURL string

				Eventually(func() error {
					api, err := pkg.GetAPIEndpoint(kubeconfigPath)
					if err != nil {
						return err
					}
					ingress, err := api.GetIngress("cattle-system", "rancher")
					if err != nil {
						return err
					}
					rancherURL = fmt.Sprintf("https://%s", ingress.Spec.TLS[0].Hosts[0])
					pkg.Log(pkg.Info, fmt.Sprintf("Found ingress URL: %s", rancherURL))
					return nil
				}, waitTimeout, pollingInterval).Should(BeNil())

				Expect(rancherURL).NotTo(BeEmpty())
				var httpClient *retryablehttp.Client
				Eventually(func() error {
					httpClient, err = pkg.GetRancherHTTPClient(kubeconfigPath)
					if err != nil {
						pkg.Log(pkg.Error, fmt.Sprintf("Error getting HTTP client: %v", err))
						return err
					}
					return nil
				}, waitTimeout, pollingInterval).Should(BeNil())
				var httpResponse *pkg.HTTPResponse

				Eventually(func() (*pkg.HTTPResponse, error) {
					httpResponse, err = pkg.GetWebPageWithClient(httpClient, rancherURL, "")
					return httpResponse, err
				}, waitTimeout, pollingInterval).Should(pkg.HasStatus(http.StatusOK))

				Expect(pkg.CheckNoServerHeader(httpResponse)).To(BeTrue(), "Found unexpected server header in response")

				var token string
				start := time.Now()
				Eventually(func() error {
					var err error
					secret, err := pkg.GetSecret("cattle-system", "rancher-admin-secret")
					if err != nil {
						pkg.Log(pkg.Error, fmt.Sprintf("Error getting rancher-admin-secret: %v", err))
						return err
					}

					var rancherAdminPassword []byte
					var ok bool
					if rancherAdminPassword, ok = secret.Data["password"]; !ok {
						pkg.Log(pkg.Error, fmt.Sprintf("Error getting rancher admin credentials: %v", err))
						return err
					}

					rancherLoginURL := fmt.Sprintf("%s/%s", rancherURL, "v3-public/localProviders/local?action=login")
					payload := `{"Username": "admin", "Password": "` + string(rancherAdminPassword) + `"}`
					response, err := httpClient.Post(rancherLoginURL, "application/json", strings.NewReader(payload))
					if err != nil {
						pkg.Log(pkg.Error, fmt.Sprintf("Error getting rancher admin token: %v", err))
						return err
					}

					err = httputil.ValidateResponseCode(response, http.StatusCreated)
					if err != nil {
						return err
					}

					defer response.Body.Close()

					// extract the response body
					body, err := ioutil.ReadAll(response.Body)
					if err != nil {
						return err
					}

					token, err = httputil.ExtractFieldFromResponseBodyOrReturnError(string(body), "token", "unable to find token in Rancher response")
					if err != nil {
						return err
					}

					return nil
				}, waitTimeout, pollingInterval).Should(BeNil())
				metrics.Emit(metricsLogger.With("get_token_elapsed_time", time.Since(start)))

				Expect(token).NotTo(BeEmpty(), "Invalid token returned by rancher")
				state := ""
				start = time.Now()
				Eventually(func() error {
					req, err := retryablehttp.NewRequest("GET", fmt.Sprintf("%s/%s", rancherURL, "v3/clusters/local"), nil)
					if err != nil {
						pkg.Log(pkg.Error, fmt.Sprintf("Error creating rancher clusters api request: %v", err))
						return err
					}

					req.Header.Set("Authorization", fmt.Sprintf("Bearer %v", token))
					req.Header.Set("Accept", "application/json")
					response, err := httpClient.Do(req)
					if err != nil {
						pkg.Log(pkg.Error, fmt.Sprintf("Error invoking rancher clusters api request: %v", err))
						return err
					}

					err = httputil.ValidateResponseCode(response, http.StatusOK)
					if err != nil {
						return err
					}

					defer response.Body.Close()

					// extract the response body
					body, err := ioutil.ReadAll(response.Body)
					if err != nil {
						return err
					}

					state, err = httputil.ExtractFieldFromResponseBodyOrReturnError(string(body), "state", "unable to find state in Rancher clusters response")
					if err != nil {
						return err
					}

					return nil
				}, waitTimeout, pollingInterval).Should(BeNil())
				metrics.Emit(metricsLogger.With("get_cluster_state_elapsed_time", time.Since(start)))

				Expect(state).To(Equal("active"), "rancher local cluster not in active state")
			}
		})
	})
})

var _ = framework.AfterEachM(metricsLogger, func() {})
