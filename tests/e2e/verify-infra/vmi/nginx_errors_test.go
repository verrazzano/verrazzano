// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package vmi_test

import (
	"crypto/tls"
	"fmt"
	"net/http"
	"strings"

	"github.com/verrazzano/verrazzano/pkg/k8sutil"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

const (
	minimumVersion = "1.3.0"
	expected404    = "<html>\n<head><title>404 Not Found</title></head>\n<body>\n<center><h1>404 Not Found</h1></center>\n</body>\n</html>"
	expected302    = "<html>\n<head><title>302 Found</title></head>\n<body>\n<center><h1>302 Found</h1></center>\n</body>\n</html>"
	expected401    = "<html>\n<head><title>401 Unauthorized</title></head>\n<body>\n<center><h1>401 Unauthorized</h1></center>\n</body>\n</html>"
)

var _ = t.Describe("nginx", Label("f:infra-lcm"), func() {
	t.Context("custom error pages test to", func() {
		t.ItMinimumVersion("Return a 404", minimumVersion, func() {
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
				customTransport := http.DefaultTransport.(*http.Transport).Clone()
				customTransport.TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
				client := &http.Client{Transport: customTransport}
				req, err := http.NewRequest("GET", fmt.Sprintf("%s/invalid-url", esURL), nil)
				if err != nil {
					return "", err
				}
				password, err := pkg.GetVerrazzanoPasswordInCluster(kubeConfigPath)
				if err != nil {
					pkg.Log(pkg.Error, fmt.Sprintf("Error getting Verrazzano Password: %v", err))
					return "", err
				}
				req.SetBasicAuth(pkg.Username, password)
				response, err := client.Do(req)
				if err != nil {
					pkg.Log(pkg.Error, fmt.Sprintf("Error getting response: %v", err))
					return "", err
				}
				httpResp, err := pkg.ProcessHTTPResponse(response)
				if err != nil {
					pkg.Log(pkg.Error, fmt.Sprintf("Error reading response from GET %v error: %v", esURL, err))
					return "", err
				}
				return strings.TrimSpace(string(httpResp.Body)), err
			}, waitTimeout, pollingInterval).Should(Equal(strings.TrimSpace(expected404)),
				"Expected response to include custom 404 error page")
		})
		t.ItMinimumVersion("Return a 302", minimumVersion, func() {
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
				customTransport := http.DefaultTransport.(*http.Transport).Clone()
				customTransport.TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
				client := &http.Client{Transport: customTransport}
				req, err := http.NewRequest("GET", esURL, nil)
				if err != nil {
					pkg.Log(pkg.Error, fmt.Sprintf("Error creating request: %v", err))
					return "", err
				}
				response, err := client.Do(req)
				if err != nil {
					pkg.Log(pkg.Error, fmt.Sprintf("Error getting response: %v", err))
					return "", err
				}
				httpResp, err := pkg.ProcessHTTPResponse(response)
				if err != nil {
					pkg.Log(pkg.Error, fmt.Sprintf("Error reading response from GET %v error: %v", esURL, err))
					return "", err
				}
				return string(httpResp.Body), err
			}, waitTimeout, pollingInterval).Should(Equal(expected302),
				"Expected response to include custom 302 response page")
		})

		t.ItMinimumVersion("Return a 401", minimumVersion, func() {
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
				customTransport := http.DefaultTransport.(*http.Transport).Clone()
				customTransport.TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
				client := &http.Client{Transport: customTransport}
				req, err := http.NewRequest("GET", esURL, nil)
				if err != nil {
					pkg.Log(pkg.Error, fmt.Sprintf("Error creating request: %v", err))
					return "", err
				}
				req.SetBasicAuth(pkg.Username, "fake-password")
				if err != nil {
					pkg.Log(pkg.Error, fmt.Sprintf("Error getting response: %v", err))
					return "", err
				}
				response, err := client.Do(req)
				httpResp, err := pkg.ProcessHTTPResponse(response)
				if err != nil {
					pkg.Log(pkg.Error, fmt.Sprintf("Error reading response from GET %v error: %v", esURL, err))
					return "", err
				}
				return string(httpResp.Body), err
			}, waitTimeout, pollingInterval).Should(Equal(expected401),
				"Expected response to include custom 401 error page")
		})
	})
})
