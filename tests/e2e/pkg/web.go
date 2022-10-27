// Copyright (c) 2020, 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package pkg

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/hashicorp/go-retryablehttp"
	"github.com/onsi/gomega"
	"github.com/onsi/gomega/types"
	"github.com/verrazzano/verrazzano/pkg/k8sutil"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	// defaultEnvName - default environment name
	defaultEnvName = "default"
)

// HTTPResponse represents an HTTP response including the read body
type HTTPResponse struct {
	StatusCode int
	Header     http.Header
	Body       []byte
}

// GetWebPage makes an HTTP GET request using a retryable client configured with the Verrazzano cert bundle
func GetWebPage(url string, hostHeader string) (*HTTPResponse, error) {
	kubeconfigPath, err := k8sutil.GetKubeConfigLocation()
	if err != nil {
		Log(Error, fmt.Sprintf("Error getting kubeconfig, error: %v", err))
		return nil, err
	}

	client, err := GetVerrazzanoHTTPClient(kubeconfigPath)
	if err != nil {
		return nil, err
	}
	return GetWebPageWithClient(client, url, hostHeader)
}

// GetWebPageInCluster makes an HTTP GET request using a retryable client configured with the Verrazzano cert bundle
func GetWebPageInCluster(url string, hostHeader string, kubeconfigPath string) (*HTTPResponse, error) {
	client, err := GetVerrazzanoHTTPClient(kubeconfigPath)
	if err != nil {
		return nil, err
	}
	return GetWebPageWithClient(client, url, hostHeader)
}

// GetWebPageWithClient submits a GET request using the specified client.
func GetWebPageWithClient(httpClient *retryablehttp.Client, url string, hostHeader string) (*HTTPResponse, error) {
	return doReq(url, "GET", "", hostHeader, "", "", nil, httpClient)
}

// GetWebPageWithBasicAuth gets a web page using basic auth, using a given kubeconfig
func GetWebPageWithBasicAuth(url string, hostHeader string, username string, password string, kubeconfigPath string) (*HTTPResponse, error) {
	client, err := GetVerrazzanoHTTPClient(kubeconfigPath)
	if err != nil {
		return nil, err
	}
	return doReq(url, "GET", "", hostHeader, username, password, nil, client)
}

// GetCertificates will return the server SSL certificates for the given URL.
func GetCertificates(url string) ([]*x509.Certificate, error) {
	kubeconfigPath, err := k8sutil.GetKubeConfigLocation()
	if err != nil {
		Log(Error, fmt.Sprintf("Error getting kubeconfig, error: %v", err))
		return nil, err
	}

	client, err := GetVerrazzanoHTTPClient(kubeconfigPath)
	if err != nil {
		return nil, err
	}
	resp, err := client.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	return resp.TLS.PeerCertificates, nil
}

// PostWithHostHeader posts a request with a specified Host header
func PostWithHostHeader(url, contentType string, hostHeader string, body io.Reader) (*HTTPResponse, error) {
	kubeconfigPath, err := k8sutil.GetKubeConfigLocation()
	if err != nil {
		Log(Error, fmt.Sprintf("Error getting kubeconfig, error: %v", err))
		return nil, err
	}

	client, err := GetVerrazzanoHTTPClient(kubeconfigPath)
	if err != nil {
		return nil, err
	}
	return doReq(url, "POST", contentType, hostHeader, "", "", body, client)
}

// Delete executes an HTTP DELETE
func Delete(url string, hostHeader string) (*HTTPResponse, error) {
	kubeconfigPath, err := k8sutil.GetKubeConfigLocation()
	if err != nil {
		Log(Error, fmt.Sprintf("Error getting kubeconfig, error: %v", err))
		return nil, err
	}

	client, err := GetVerrazzanoHTTPClient(kubeconfigPath)
	if err != nil {
		return nil, err
	}
	return doReq(url, "DELETE", "", hostHeader, "", "", nil, client)
}

// GetVerrazzanoNoRetryHTTPClient returns an Http client configured with the Verrazzano CA cert
func GetVerrazzanoNoRetryHTTPClient(kubeconfigPath string) (*http.Client, error) {
	caCert, err := getVerrazzanoCACert(kubeconfigPath)
	if err != nil {
		return nil, err
	}
	client, err := getHTTPClientWithCABundle(caCert, kubeconfigPath)
	if err != nil {
		return nil, err
	}
	return client, nil
}

// GetVerrazzanoHTTPClient returns a retryable Http client configured with the Verrazzano CA cert
func GetVerrazzanoHTTPClient(kubeconfigPath string) (*retryablehttp.Client, error) {
	client, err := GetVerrazzanoNoRetryHTTPClient(kubeconfigPath)
	if err != nil {
		return nil, err
	}
	retryableClient := newRetryableHTTPClient(client)
	return retryableClient, nil
}

// CheckNoServerHeader validates that the response does not include a Server header.
func CheckNoServerHeader(resp *HTTPResponse) bool {
	// HTTP Server headers should never be returned.
	for headerName, headerValues := range resp.Header {
		if strings.EqualFold(headerName, "Server") {
			Log(Error, fmt.Sprintf("Unexpected Server header %v", headerValues))
			return false
		}
	}

	return true
}

// CheckStatusAndResponseHeaderAbsent checks that the given header name is not present in the http response, and that the
// response status code is as expected. If the statusCode is <= 0, the status code check is skipped. If
// the badRespHeader is "", the response headers are not checked.
func CheckStatusAndResponseHeaderAbsent(httpClient *retryablehttp.Client, req *retryablehttp.Request, badRespHeader string, statusCode int) error {
	resp, err := httpClient.Do(req)
	if err != nil {
		return err
	}
	io.ReadAll(resp.Body)
	resp.Body.Close()
	if statusCode > 0 {
		if resp.StatusCode != statusCode {
			return fmt.Errorf("Expected status code %d but got %d", statusCode, resp.StatusCode)
		}
	}
	if badRespHeader != "" {
		// Check that the HTTP header we don't want is not present in the response.
		badHeaderLower := strings.ToLower(badRespHeader)
		for headerName, headerValues := range resp.Header {
			if strings.ToLower(headerName) == badHeaderLower {
				errMsg := fmt.Sprintf("Unexpected %s header %v", headerName, headerValues)
				Log(Error, errMsg)
				return fmt.Errorf(errMsg)
			}
		}
	}
	return nil
}

func EventuallyVerrazzanoRetryableHTTPClient() *retryablehttp.Client {
	var client *retryablehttp.Client
	gomega.Eventually(func() (*retryablehttp.Client, error) {
		var err error
		client, err = GetVerrazzanoRetryableHTTPClient()
		return client, err
	}, waitTimeout, pollingInterval).ShouldNot(gomega.BeNil(), "Unable to get Verrazzano HTTP client")
	return client
}

// GetVerrazzanoRetryableHTTPClient returns a retryable HTTP client configured with the CA cert
func GetVerrazzanoRetryableHTTPClient() (*retryablehttp.Client, error) {
	kubeconfigPath, err := k8sutil.GetKubeConfigLocation()
	if err != nil {
		Log(Error, fmt.Sprintf("Error getting kubeconfig, error: %v", err))
		return nil, err
	}

	caCert, err := getVerrazzanoCACert(kubeconfigPath)
	if err != nil {
		return nil, err
	}
	vmiRawClient, err := getHTTPClientWithCABundle(caCert, kubeconfigPath)
	if err != nil {
		return nil, err
	}
	return newRetryableHTTPClient(vmiRawClient), nil
}

func GetEnvName(kubeconfigPath string) (string, error) {
	vz, err := GetVerrazzanoInstallResourceInCluster(kubeconfigPath)
	if err != nil {
		return "", err
	}
	if len(vz.Spec.EnvironmentName) == 0 {
		return defaultEnvName, nil
	}
	return vz.Spec.EnvironmentName, nil
}

func AssertOauthURLAccessibleAndUnauthorized(httpClient *retryablehttp.Client, url string) bool {
	httpClient.HTTPClient.CheckRedirect = func(req *http.Request, via []*http.Request) error {
		Log(Info, fmt.Sprintf("oidcUnauthorized req: %v \nvia: %v\n", req, via))
		return http.ErrUseLastResponse
	}
	resp, err := httpClient.Get(url)
	if err != nil || resp == nil {
		Log(Error, fmt.Sprintf("Failed making request: %v", err))
		return false
	}
	location, err := resp.Location()
	if err != nil {
		Log(Error, fmt.Sprintf("Error getting location from response: %v, error: %v", resp, err))
		return false
	}

	if location == nil {
		Log(Error, fmt.Sprintf("Response location not found for %v", resp))
		return false
	}
	Log(Info, fmt.Sprintf("oidcUnauthorized %v StatusCode:%v host:%v", url, resp.StatusCode, location.Host))
	return resp.StatusCode == 302 && strings.Contains(location.Host, "keycloak")
}

func AssertBearerAuthorized(httpClient *retryablehttp.Client, url string) bool {
	kubeconfigPath, err := k8sutil.GetKubeConfigLocation()
	if err != nil {
		Log(Error, fmt.Sprintf("Error getting kubeconfig location: %v", err))
		return false
	}

	api := EventuallyGetAPIEndpoint(kubeconfigPath)
	req, _ := retryablehttp.NewRequest("GET", url, nil)
	if api.AccessToken != "" {
		bearer := fmt.Sprintf("Bearer %v", api.AccessToken)
		req.Header.Set("Authorization", bearer)
	}
	resp, err := httpClient.Do(req)
	if err != nil {
		Log(Error, fmt.Sprintf("Failed making request: %v", err))
		return false
	}
	resp.Body.Close()
	Log(Info, fmt.Sprintf("assertBearerAuthorized %v Response:%v Error:%v", url, resp.StatusCode, err))
	return resp.StatusCode == http.StatusOK
}

// PutWithHostHeader PUTs a request with a specified Host header
func PutWithHostHeader(url, contentType string, hostHeader string, body io.Reader) (*HTTPResponse, error) {
	kubeconfigPath, err := k8sutil.GetKubeConfigLocation()
	if err != nil {
		Log(Error, fmt.Sprintf("Error getting kubeconfig, error: %v", err))
		return nil, err
	}

	client, err := GetVerrazzanoHTTPClient(kubeconfigPath)
	if err != nil {
		return nil, err
	}
	return doReq(url, "PUT", contentType, hostHeader, "", "", body, client)
}

// PutWithHostHeaderInCluster PUTs a request with a specified Host header
func PutWithHostHeaderInCluster(url, contentType string, hostHeader string, body io.Reader, kubeconfigPath string) (*HTTPResponse, error) {
	client, err := GetVerrazzanoHTTPClient(kubeconfigPath)
	if err != nil {
		return nil, err
	}
	return doReq(url, "PUT", contentType, hostHeader, "", "", body, client)
}

// doReq executes an HTTP request with the specified method (GET, POST, DELETE, etc)
func doReq(url, method string, contentType string, hostHeader string, username string, password string,
	body io.Reader, httpClient *retryablehttp.Client, additionalHeaders ...string) (*HTTPResponse, error) {
	req, err := retryablehttp.NewRequest(method, url, body)
	if err != nil {
		return nil, err
	}
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}
	if hostHeader != "" {
		req.Host = hostHeader
	}

	for _, header := range additionalHeaders {
		splitArray := strings.Split(header, ":")
		if len(splitArray) != 2 {
			return nil, fmt.Errorf("Invalid additional header '%s'. Not in the format key:value", header)
		}
		req.Header.Set(splitArray[0], splitArray[1])
	}

	if username != "" && password != "" {
		req.SetBasicAuth(username, password)
	}
	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	return ProcessHTTPResponse(resp)
}

// getHTTPClientWithCABundle returns an HTTP client configured with the provided CA cert
func getHTTPClientWithCABundle(caData []byte, kubeconfigPath string) (*http.Client, error) {
	ca, err := rootCertPoolInCluster(caData, kubeconfigPath)
	if err != nil {
		return nil, err
	}
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{
			RootCAs:    ca,
			MinVersion: tls.VersionTLS12},
		Proxy: http.ProxyFromEnvironment,
	}

	// disable the custom DNS resolver
	// setupCustomDNSResolver(tr, kubeconfigPath)

	return &http.Client{Transport: tr}, nil
}

// getVerrazzanoCACert returns the Verrazzano CA cert in the specified cluster
func getVerrazzanoCACert(kubeconfigPath string) ([]byte, error) {
	cacert, err := GetCACertFromSecret("verrazzano-tls", "verrazzano-system", "ca.crt", kubeconfigPath)
	if err != nil {
		envName, err := GetEnvName(kubeconfigPath)
		if err != nil {
			return nil, err
		}
		return GetCACertFromSecret(envName+"-secret", "verrazzano-system", "ca.crt", kubeconfigPath)
	}
	return cacert, nil
}

// GetCACertFromSecret returns the CA cert from the specified kubernetes secret in the given cluster
func GetCACertFromSecret(secretName string, namespace string, caKey string, kubeconfigPath string) ([]byte, error) {
	clientset, err := GetKubernetesClientsetForCluster(kubeconfigPath)
	if err != nil {
		return nil, err
	}
	certSecret, err := clientset.CoreV1().Secrets(namespace).Get(context.TODO(), secretName, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}
	return certSecret.Data[caKey], nil
}

// newRetryableHTTPClient returns a new instance of a retryable HTTP client
func newRetryableHTTPClient(client *http.Client) *retryablehttp.Client {
	retryableClient := retryablehttp.NewClient()
	retryableClient.RetryMax = NumRetries
	retryableClient.RetryWaitMin = RetryWaitMinimum
	retryableClient.RetryWaitMax = RetryWaitMaximum
	retryableClient.HTTPClient = client
	retryableClient.CheckRetry = GetRetryPolicy()
	return retryableClient
}

// rootCertPoolInCluster returns the root cert pool
func rootCertPoolInCluster(caData []byte, kubeconfigPath string) (*x509.CertPool, error) {
	var certPool *x509.CertPool

	if len(caData) != 0 {
		// if we have caData, use it
		certPool = x509.NewCertPool()
		certPool.AppendCertsFromPEM(caData)
	}

	env, err := GetACMEEnvironment(kubeconfigPath)
	if err != nil {
		return nil, err
	}
	if env == "staging" {
		// Add the ACME staging CAs if necessary
		if certPool == nil {
			certPool = x509.NewCertPool()
		}
		for _, stagingCA := range getACMEStagingCAs() {
			if len(stagingCA) > 0 {
				certPool.AppendCertsFromPEM(stagingCA)
			}
		}
	}
	return certPool, nil
}

// HasStatus asserts that an HTTPResponse has a given status.
func HasStatus(expected int) types.GomegaMatcher {
	return gomega.WithTransform(func(response *HTTPResponse) int {
		if response == nil {
			return 0
		}
		return response.StatusCode
	}, gomega.Equal(expected))
}

// BodyContains asserts that an HTTPResponse body contains a given substring.
func BodyContains(expected string) types.GomegaMatcher {
	return gomega.WithTransform(func(response *HTTPResponse) string {
		if response == nil {
			return ""
		}
		return string(response.Body)
	}, gomega.ContainSubstring(expected))
}

// BodyDoesNotContain asserts that an HTTPResponse body does not contain a given substring.
func BodyDoesNotContain(unexpected string) types.GomegaMatcher {
	return gomega.WithTransform(func(response *HTTPResponse) string { return string(response.Body) }, gomega.Not(gomega.ContainSubstring(unexpected)))
}

// BodyEquals asserts that an HTTPResponse body equals a given string.
func BodyEquals(expected string) types.GomegaMatcher {
	return gomega.WithTransform(func(response *HTTPResponse) string {
		if response == nil {
			return ""
		}
		return string(response.Body)
	}, gomega.Equal(expected))
}

// BodyNotEmpty asserts that an HTTPResponse body is not empty.
func BodyNotEmpty() types.GomegaMatcher {
	return gomega.WithTransform(func(response *HTTPResponse) []byte {
		if response == nil {
			return nil
		}
		return response.Body
	}, gomega.Not(gomega.BeEmpty()))
}
