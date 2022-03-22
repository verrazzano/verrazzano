// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package web_test

import (
	"context"
	"crypto/tls"
	"fmt"
	"net/http"
	"os"
	"strings"

	"github.com/verrazzano/verrazzano/pkg/k8sutil"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg"

	"github.com/hashicorp/go-retryablehttp"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

const (
	minimumVersion = "1.3.0"
	expected400    = "<html>\n<head><title>400 Bad Request</title></head>\n<body>\n<center><h1>400 Bad Request</h1></center>\n</body>\n</html>"
)

var _ = t.Describe("nginx error pages", Label("f:mesh.ingress", "f:mesh.traffic-mgmt"), func() {
	t.Context("test that an", func() {
		t.ItMinimumVersion("Incorrect path returns a 404", minimumVersion, func() {
			if !pkg.IsManagedClusterProfile() {
				Eventually(func() (string, error) {
					kubeConfigPath, err := k8sutil.GetKubeConfigLocation()
					if err != nil {
						pkg.Log(pkg.Error, fmt.Sprintf("Error getting kubeconfig: %v", err))
						return "", err
					}
					api, err := pkg.GetAPIEndpoint(kubeConfigPath)
					if err != nil {
						pkg.Log(pkg.Error, fmt.Sprintf("Error getting API endpoint: %v", err))
						return "", err
					}
					esURL, err := api.GetElasticURL()
					if err != nil {
						pkg.Log(pkg.Error, fmt.Sprintf("Error getting Elasticsearch URL: %v", err))
						return "", err
					}
					req, err := retryablehttp.NewRequest("GET", esURL+"/invalid-url", nil)
					if err != nil {
						pkg.Log(pkg.Error, fmt.Sprintf("Error creating Request: %v", err))
						return "", err
					}
					password, err := pkg.GetVerrazzanoPasswordInCluster(kubeConfigPath)
					if err != nil {
						pkg.Log(pkg.Error, fmt.Sprintf("Error getting Verrazzano Password: %v", err))
						return "", err
					}
					req.SetBasicAuth(pkg.Username, password)
					return checkNGINXErrorPage(req, 404)
				}, waitTimeout, pollingInterval).Should(Not(ContainSubstring("nginx")),
					"Expected response to not leak the name nginx")
			}
		})

		t.ItMinimumVersion("Incorrect password returns a 401", minimumVersion, func() {
			if !pkg.IsManagedClusterProfile() {
				Eventually(func() (string, error) {
					kubeConfigPath, err := k8sutil.GetKubeConfigLocation()
					if err != nil {
						pkg.Log(pkg.Error, fmt.Sprintf("Error getting kubeconfig: %v", err))
						return "", err
					}
					api, err := pkg.GetAPIEndpoint(kubeConfigPath)
					if err != nil {
						pkg.Log(pkg.Error, fmt.Sprintf("Error getting API endpoint: %v", err))
						return "", err
					}
					esURL, err := api.GetElasticURL()
					if err != nil {
						pkg.Log(pkg.Error, fmt.Sprintf("Error getting Elasticsearch URL: %v", err))
						return "", err
					}
					req, err := retryablehttp.NewRequest("GET", esURL, nil)
					if err != nil {
						pkg.Log(pkg.Error, fmt.Sprintf("Error creating Request: %v", err))
						return "", err
					}
					req.SetBasicAuth(pkg.Username, "fake-password")
					return checkNGINXErrorPage(req, 401)
				}, waitTimeout, pollingInterval).Should(Not(ContainSubstring("nginx")),
					"Expected response to not leak the name nginx")
			}
		})

		t.ItMinimumVersion("Incorrect host returns a 404", minimumVersion, func() {
			if !pkg.IsManagedClusterProfile() && os.Getenv("TEST_ENV") != "ocidns_oke" {
				Eventually(func() (string, error) {
					kubeConfigPath, err := k8sutil.GetKubeConfigLocation()
					if err != nil {
						pkg.Log(pkg.Error, fmt.Sprintf("Error getting kubeconfig: %v", err))
						return "", err
					}
					api, err := pkg.GetAPIEndpoint(kubeConfigPath)
					if err != nil {
						pkg.Log(pkg.Error, fmt.Sprintf("Error getting API endpoint: %v", err))
						return "", err
					}
					vzURL, err := api.GetVerrazzanoIngressURL()
					if err != nil {
						pkg.Log(pkg.Error, fmt.Sprintf("Error getting Verrazzano Ingress URL: %v", err))
						return "", err
					}
					badHost := strings.Replace(vzURL, "verrazzano", "badhost", 1)
					req, err := retryablehttp.NewRequest("GET", badHost, nil)
					if err != nil {
						pkg.Log(pkg.Error, fmt.Sprintf("Error creating Request: %v", err))
						return "", err
					}
					password, err := pkg.GetVerrazzanoPasswordInCluster(kubeConfigPath)
					if err != nil {
						pkg.Log(pkg.Error, fmt.Sprintf("Error getting Verrazzano Password: %v", err))
						return "", err
					}
					req.SetBasicAuth(pkg.Username, password)
					return checkNGINXErrorPage(req, 404)
				}, waitTimeout, pollingInterval).Should(Not(ContainSubstring("nginx")),
					"Expected response to not leak the name nginx")
			}
		})

		t.ItMinimumVersion("Directory traversal returns a 400", minimumVersion, func() {
			if !pkg.IsManagedClusterProfile() && os.Getenv("TEST_ENV") != "ocidns_oke" {
				Eventually(func() (string, error) {
					kubeConfigPath, err := k8sutil.GetKubeConfigLocation()
					if err != nil {
						pkg.Log(pkg.Error, fmt.Sprintf("Error getting kubeconfig: %v", err))
						return "", err
					}
					api, err := pkg.GetAPIEndpoint(kubeConfigPath)
					if err != nil {
						pkg.Log(pkg.Error, fmt.Sprintf("Error getting API endpoint: %v", err))
						return "", err
					}
					vzURL, err := api.GetVerrazzanoIngressURL()
					if err != nil {
						pkg.Log(pkg.Error, fmt.Sprintf("Error getting Verrazzano Ingress URL: %v", err))
						return "", err
					}
					clusterIP := strings.Replace(vzURL, "verrazzano.", "", 1)
					req, err := retryablehttp.NewRequest("GET", clusterIP+"/../../", nil)
					if err != nil {
						pkg.Log(pkg.Error, fmt.Sprintf("Error creating Request: %v", err))
						return "", err
					}
					return checkNGINXErrorPage(req, 400)
				}, waitTimeout, pollingInterval).Should(Equal(strings.TrimSpace(expected400)),
					"Expected response to be the custom 400 error page")
			}
		})
	})
})

func checkNGINXErrorPage(req *retryablehttp.Request, expectedStatus int) (string, error) {
	transport := &http.Transport{TLSClientConfig: &tls.Config{InsecureSkipVerify: true}} //nolint:gosec //#gosec G402
	c, err := pkg.GetVerrazzanoRetryableHTTPClient()
	if err != nil {
		pkg.Log(pkg.Info, fmt.Sprintf("Error getting HTTP client: %v", err))
		return "", err
	}
	c.HTTPClient.Transport = transport
	c.CheckRetry = func(ctx context.Context, resp *http.Response, err error) (bool, error) {
		if resp == nil {
			if err != nil {
				pkg.Log(pkg.Error, fmt.Sprintf("Request returned a nil response, error: %v", err))
			}
			return true, err
		}
		if resp.StatusCode == expectedStatus {
			return false, nil
		}
		pkg.Log(pkg.Info, fmt.Sprintf("Request returned response code: %d, error: %v", resp.StatusCode, err))
		return true, err
	}
	response, err := c.Do(req)
	if err != nil {
		pkg.Log(pkg.Error, fmt.Sprintf("Error getting response: %v", err))
		return "", err
	}
	httpResp, err := pkg.ProcessHTTPResponse(response)
	if err != nil {
		pkg.Log(pkg.Error, fmt.Sprintf("Error reading response: %v", err))
		return "", err
	}
	return strings.TrimSpace(string(httpResp.Body)), err
}
