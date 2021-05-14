// Copyright (c) 2020, 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package pkg

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/hashicorp/go-retryablehttp"
	"github.com/onsi/ginkgo"
	"github.com/onsi/gomega"
	"github.com/onsi/gomega/types"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	// DefaultEnvName - default environment name
	DefaultEnvName = "default"

	// Username - the username of the verrazzano admin user
	Username               = "verrazzano"
	clientID               = "verrazzano-oauth-client"
	realm                  = "verrazzano-system"
	verrazzanoAPIURLPrefix = "20210501"
	teapot                 = 418
)

// HTTPResponse represents an HTTP response
type HTTPResponse struct {
	StatusCode int
	Body       []byte
	BodyErr    error
}

// GetWebPageWithCABundle - same as GetWebPage, but with additional caData
func GetWebPageWithCABundle(url string, hostHeader string) (int, string) {
	return doGetWebPage(url, hostHeader, GetVerrazzanoHTTPClient(), "", "")
}

// GetCertificates will return the server SSL certificates for the given URL.
func GetCertificates(url string) ([]*x509.Certificate, error) {
	resp, err := GetVerrazzanoHTTPClient().Get(url)
	if err != nil {
		Log(Error, err.Error())
		ginkgo.Fail("Could not get web page " + url)
	}
	defer resp.Body.Close()
	return resp.TLS.PeerCertificates, nil
}

// GetWebPageWithBasicAuth gets a web page using basic auth
func GetWebPageWithBasicAuth(url string, hostHeader string, username string, password string) (int, string) {
	return doGetWebPage(url, hostHeader, GetVerrazzanoHTTPClient(), username, password)
}

// RetryGetWithBasicAuth retries getting a web page using basic auth
func RetryGetWithBasicAuth(url string, hostHeader string, username string, password string, kubeconfigPath string) (int, string) {
	client := GetVerrazzanoHTTPClientForCluster(kubeconfigPath)
	client.CheckRetry = GetRetryPolicy()
	return doGetWebPage(url, hostHeader, client, username, password)
}

// RetryPostWithBasicAuth retries POST using basic auth
func RetryPostWithBasicAuth(url, body, username, password, kubeconfigPath string) (int, string) {
	client := GetVerrazzanoHTTPClientForCluster(kubeconfigPath)
	client.CheckRetry = GetRetryPolicy()
	return doReq(url, "POST", "application/json", "", username, password, strings.NewReader(body), client)
	//return doGetWebPage(url, hostHeader, client, username, password)
}

// doGetWebPage retries a web page
func doGetWebPage(url string, hostHeader string, httpClient *retryablehttp.Client, username string, password string) (int, string) {
	return doReq(url, "GET", "", hostHeader, username, password, nil, httpClient)
}

// Delete executes an HTTP DELETE
func Delete(url string, hostHeader string) (int, string) {
	return doReq(url, "DELETE", "", hostHeader, "", "", nil, GetVerrazzanoHTTPClient())
}

// GetVerrazzanoNoRetryHTTPClient returns an Http client configured with the verrazzano CA cert
func GetVerrazzanoNoRetryHTTPClient() *http.Client {
	kubeconfigPath := GetKubeConfigPathFromEnv()
	return getHTTPClientWithCABundle(getVerrazzanoCACert(kubeconfigPath), kubeconfigPath)
}

// GetVerrazzanoHTTPClient returns an Http client configured with the verrazzano CA cert
func GetVerrazzanoHTTPClient() *retryablehttp.Client {
	kubeconfigPath := GetKubeConfigPathFromEnv()
	rawClient := getHTTPClientWithCABundle(getVerrazzanoCACert(kubeconfigPath), kubeconfigPath)
	return newRetryableHTTPClient(rawClient)
}

// GetVerrazzanoHTTPClient returns an Http client configured with the verrazzano CA cert
func GetVerrazzanoHTTPClientForCluster(kubeconfigPath string) *retryablehttp.Client {
	rawClient := getHTTPClientWithCABundle(getVerrazzanoCACert(kubeconfigPath), kubeconfigPath)
	return newRetryableHTTPClient(rawClient)
}

// GetRancherHTTPClient returns an Http client configured with the Rancher CA cert
func GetRancherHTTPClient(kubeconfigPath string) *retryablehttp.Client {
	rawClient := getHTTPClientWithCABundle(getRancherCACert(kubeconfigPath), kubeconfigPath)
	return newRetryableHTTPClient(rawClient)
}

// GetKeycloakHTTPClient returns the Keycloak Http client
func GetKeycloakHTTPClient(kubeconfigPath string) *retryablehttp.Client {
	keycloakRawClient := getHTTPClientWithCABundle(getKeycloakCACert(kubeconfigPath), kubeconfigPath)
	client := newRetryableHTTPClient(keycloakRawClient)
	client.CheckRetry = GetRetryPolicy()
	return client
}

// ExpectHTTPOk validates that this is no error and a that the status is 200
func ExpectHTTPOk(resp *HTTPResponse, err error, msg string) {
	ExpectHTTPStatus(http.StatusOK, resp, err, msg)
}

// ExpectHTTPStatus validates that this is no error and a that the status matchs
func ExpectHTTPStatus(status int, resp *HTTPResponse, err error, msg string) {
	gomega.Expect(err).To(gomega.BeNil(), msg)

	if resp.StatusCode != status {
		if len(resp.Body) > 0 {
			msg = msg + "\n" + string(resp.Body)
		}
		gomega.Expect(resp.StatusCode).To(gomega.Equal(status), msg)
	}
}

// ExpectHTTPGetOk submits a GET request and expect a status 200 response
func ExpectHTTPGetOk(httpClient *retryablehttp.Client, url string) {
	resp, err := httpClient.Get(url)
	httpResp := ProcHTTPResponse(resp, err)
	ExpectHTTPOk(httpResp, err, "Error doing http(s) get from "+url)
}

// GetSystemVmiHTTPClient returns an HTTP client configured with the system vmi CA cert
func GetSystemVmiHTTPClient() *retryablehttp.Client {
	kubeconfigPath := GetKubeConfigPathFromEnv()
	vmiRawClient := getHTTPClientWithCABundle(getSystemVMICACert(kubeconfigPath), kubeconfigPath)
	client := newRetryableHTTPClient(vmiRawClient)
	client.CheckRetry = GetRetryPolicy()
	return client
}

// PostWithHostHeader posts a request with a specified Host header
func PostWithHostHeader(url, contentType string, hostHeader string, body io.Reader) (int, string) {
	return doPost(url, contentType, hostHeader, body, GetVerrazzanoHTTPClient())
}

// postWithClient posts a request using the specified HTTP client
func postWithClient(url, contentType string, body io.Reader, httpClient *retryablehttp.Client) (int, string) {
	return doPost(url, contentType, "", body, httpClient)
}

// doPost executes a POST request
func doPost(url, contentType string, hostHeader string, body io.Reader, httpClient *retryablehttp.Client) (int, string) {
	return doReq(url, "POST", contentType, hostHeader, "", "", body, httpClient)
}

// PutWithHostHeader PUTs a request with a specified Host header
func PutWithHostHeader(url, contentType string, hostHeader string, body io.Reader) (int, string) {
	return doPut(url, contentType, hostHeader, body, GetVerrazzanoHTTPClient())
}

// doPut executes a PUT request
func doPut(url, contentType string, hostHeader string, body io.Reader, httpClient *retryablehttp.Client) (int, string) {
	return doReq(url, "PUT", contentType, hostHeader, "", "", body, httpClient)
}

// doReq executes an HTTP request with the specified method (GET, POST, DELETE, etc)
func doReq(url, method string, contentType string, hostHeader string, username string, password string,
	body io.Reader, httpClient *retryablehttp.Client) (int, string) {
	req, err := retryablehttp.NewRequest(method, url, body)
	if err != nil {
		Log(Error, err.Error())
		ginkgo.Fail("Could not create request")
	}
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}
	if hostHeader != "" {
		req.Host = hostHeader
	}
	if username != "" && password != "" {
		req.SetBasicAuth(username, password)
	}
	resp, err := httpClient.Do(req)
	if err != nil {
		Log(Error, err.Error())
		// Do not call Fail() here - this is not necessarily a permanent failure and
		// we should not call Fail() inside a func that is called from an Eventually()
		// a later retry may be successful - for example the endpoint may not be available
		// since the pod has not reached ready state yet.
		// We cannot return status code, because the resp is likely nil, so instead
		// return a valid HTTP status code which nonetheless communicates some kind of failure :)
		return teapot, ""
	}
	defer resp.Body.Close()
	html, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		Log(Error, err.Error())
		ginkgo.Fail("Could not read content of response body")
	}
	return resp.StatusCode, string(html)
}

// getHTTPClientWithCABundle returns an HTTP client configured with the provided CA cert
func getHTTPClientWithCABundle(caData []byte, kubeconfigPath string) *http.Client {
	tr := &http.Transport{TLSClientConfig: &tls.Config{RootCAs: rootCertPool(caData)}}

	proxyURL := getProxyURL()
	if proxyURL != "" {
		tURL := url.URL{}
		tURLProxy, _ := tURL.Parse(proxyURL)
		tr.Proxy = http.ProxyURL(tURLProxy)
	}

	// disable the custom DNS resolver
	// setupCustomDNSResolver(tr, kubeconfigPath)

	return &http.Client{Transport: tr}
}

func setupCustomDNSResolver(tr *http.Transport, kubeconfigPath string) {
	ipResolve := getNginxNodeIP(kubeconfigPath)
	if ipResolve != "" {
		dialer := &net.Dialer{
			Timeout:   30 * time.Second,
			KeepAlive: 30 * time.Second,
		}
		tr.DialContext = func(ctx context.Context, network, addr string) (net.Conn, error) {
			Log(Debug, fmt.Sprintf("original address %s", addr))
			wildcardSuffix := GetWildcardDNS(addr)
			if strings.Contains(addr, "127.0.0.1") && strings.Contains(addr, ":443") {
				// resolve to the nginx node ip if address contains 127.0.0.1, for node port installation
				addr = ipResolve + ":443"
				Log(Debug, fmt.Sprintf("modified address %s", addr))
			} else if wildcardSuffix != "" && strings.Contains(addr, ":443") {
				// resolve DNS wildcard ourselves
				resolving := strings.TrimSuffix(strings.TrimSuffix(addr, ":443"), "."+wildcardSuffix)
				four := resolving[strings.LastIndex(resolving, "."):]
				resolving = strings.TrimSuffix(resolving, four)
				three := resolving[strings.LastIndex(resolving, "."):]
				resolving = strings.TrimSuffix(resolving, three)
				two := resolving[strings.LastIndex(resolving, "."):]
				resolving = strings.TrimSuffix(resolving, two)
				one := resolving[strings.LastIndex(resolving, ".")+1:]
				addr = one + two + three + four + ":443"
				Log(Debug, fmt.Sprintf("modified address %s", addr))
			}
			return dialer.DialContext(ctx, network, addr)
		}
	}
}

func getEnvName(kubeconfigPath string) string {
	installedEnvName := GetVerrazzanoInstallResourceInCluster(kubeconfigPath).Spec.EnvironmentName
	if len(installedEnvName) == 0 {
		return DefaultEnvName
	}
	return installedEnvName
}

// getVerrazzanoCACert returns the verrazzano CA cert in the specified cluster
func getVerrazzanoCACert(kubeconfigPath string) []byte {
	return doGetCACertFromSecret(getEnvName(kubeconfigPath)+"-secret", "verrazzano-system", kubeconfigPath)
}

// getRancherCACert returns the Rancher CA cert
func getRancherCACert(kubeconfigPath string) []byte {
	return doGetCACertFromSecret("tls-rancher-ingress", "cattle-system", kubeconfigPath)
}

// getKeycloakCACert returns the keycloak CA cert
func getKeycloakCACert(kubeconfigPath string) []byte {
	return doGetCACertFromSecret(getEnvName(kubeconfigPath)+"-secret", "keycloak", kubeconfigPath)
}

// getSystemVMICACert returns the system vmi CA cert
func getSystemVMICACert(kubeconfigPath string) []byte {
	return doGetCACertFromSecret("system-tls", "verrazzano-system", kubeconfigPath)
}

// getProxyURL returns the proxy URL from the proxy env variables
func getProxyURL() string {
	if proxyURL := os.Getenv("https_proxy"); proxyURL != "" {
		return proxyURL
	}
	if proxyURL := os.Getenv("HTTPS_PROXY"); proxyURL != "" {
		return proxyURL
	}
	if proxyURL := os.Getenv("http_proxy"); proxyURL != "" {
		return proxyURL
	}
	if proxyURL := os.Getenv("HTTP_PROXY"); proxyURL != "" {
		return proxyURL
	}
	return ""
}

// doGetCACertFromSecret returns the CA cert from the specified kubernetes secret in the given cluster
func doGetCACertFromSecret(secretName string, namespace string, kubeconfigPath string) []byte {
	clientset := GetKubernetesClientsetForCluster(kubeconfigPath)
	certSecret, _ := clientset.CoreV1().Secrets(namespace).Get(context.TODO(), secretName, metav1.GetOptions{})
	return certSecret.Data["ca.crt"]
}

// Returns the nginx controller node ip
func getNginxNodeIP(kubeconfigPath string) string {
	clientset := GetKubernetesClientsetForCluster(kubeconfigPath)
	pods, err := clientset.CoreV1().Pods("ingress-nginx").List(context.TODO(), metav1.ListOptions{})
	if err == nil {
		for i := range pods.Items {
			pod := pods.Items[i]
			if strings.HasPrefix(pod.Name, "ingress-controller-ingress-nginx-controller-") {
				return pod.Status.HostIP
			}
		}
	}
	return ""
}

// newRetryableHTTPClient returns a new instance of a retryable HTTP client
func newRetryableHTTPClient(client *http.Client) *retryablehttp.Client {
	retryableClient := retryablehttp.NewClient() //default of 4 retries is sufficient for us
	retryableClient.RetryMax = NumRetries
	retryableClient.RetryWaitMin = RetryWaitMin
	retryableClient.RetryWaitMax = RetryWaitMax
	retryableClient.HTTPClient = client
	return retryableClient
}

// rootCertPool returns the root cert pool
func rootCertPool(caData []byte) *x509.CertPool {
	// if we have caData, use it
	certPool := x509.NewCertPool()
	for _, stagingCA := range getACMEStagingCAs() {
		if len(stagingCA) > 0 {
			certPool.AppendCertsFromPEM(stagingCA)
		}
	}
	if len(caData) != 0 {
		certPool.AppendCertsFromPEM(caData)
	}
	return certPool
}

// WebResponse contains the response from a web request
type WebResponse struct {
	Status  int
	Content string
}

// HaveStatus asserts that a WebResponse has a given status.
func HaveStatus(expected int) types.GomegaMatcher {
	return gomega.WithTransform(func(response WebResponse) int { return response.Status }, gomega.Equal(expected))
}

// ContainContent asserts that a WebResponse contains a given substring.
func ContainContent(expected string) types.GomegaMatcher {
	return gomega.WithTransform(func(response WebResponse) string { return response.Content }, gomega.ContainSubstring(expected))
}
