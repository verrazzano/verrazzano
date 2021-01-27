// Copyright (c) 2020, 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package util

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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	// EnvName - default environment name
	EnvName = "default"

	// DnsZone - default DNS zone
	DnsZone = "127.0.0.1.xip.io"

	// NumRetries - maximum number of retries
	NumRetries = 7

	// RetryWaitMin - minimum retry wait
	RetryWaitMin = 1 * time.Second

	// RetryWaitMax - maximum retry wait
	RetryWaitMax = 30 * time.Second

	username               = "verrazzano"
	clientId               = "admin-cli"
	realm                  = "verrazzano-system"
	verrazzanoApiUriPrefix = "20210501"
)

// The HTTP response
type HttpResponse struct {
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

// GetWebPageWithBasicAuth - get web page using basic auth
func GetWebPageWithBasicAuth(url string, hostHeader string, username string, password string) (int, string) {
	return doGetWebPage(url, hostHeader, GetVerrazzanoHTTPClient(), username, password)
}

func doGetWebPage(url string, hostHeader string, httpClient *retryablehttp.Client, username string, password string) (int, string) {
	return doReq(url, "GET", "", hostHeader, username, password, nil, httpClient)
}

func Delete(url string, hostHeader string) (int, string) {
	return doReq(url, "DELETE", "", hostHeader, "", "", nil, GetVerrazzanoHTTPClient())
}

// GetVerrazzanoHTTPClient returns the Http client
func GetVerrazzanoHTTPClient() *retryablehttp.Client {
	rawClient := getHTTPClientWIthCABundle(getVerrazzanoCACert())
	return newRetryableHTTPClient(rawClient)
}

// GetKeycloakHTTPClient returns the Keycloak Http client
func GetKeycloakHTTPClient() *retryablehttp.Client {
	keycloakRawClient := getHTTPClientWIthCABundle(getKeycloakCACert())
	return newRetryableHTTPClient(keycloakRawClient)
}

// ExpectHttpStatusOk validates that this is no error and a that the status is 200
func ExpectHttpOk(resp *HttpResponse, err error, msg string) {
	ExpectHttpStatus(http.StatusOK, resp, err, msg)
}

// ExpectHttpStatus validates that this is no error and a that the status matchs
func ExpectHttpStatus(status int, resp *HttpResponse, err error, msg string) {
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
	httpResp := ProcHttpResponse(resp, err)
	ExpectHttpOk(httpResp, err, "Error doing http(s) get from "+url)
}

// GetSystemVmiHttpClient
func GetSystemVmiHttpClient() *retryablehttp.Client {
	vmiRawClient := getHTTPClientWIthCABundle(getSystemVMICACert())
	return newRetryableHTTPClient(vmiRawClient)
}

func Post(url, contentType string, body io.Reader) (int, string) {
	return PostWithHostHeader(url, contentType, "", body)
}

func PostWithHostHeader(url, contentType string, hostHeader string, body io.Reader) (int, string) {
	return doPost(url, contentType, hostHeader, body, GetVerrazzanoHTTPClient())
}

// postWithClient
func postWithClient(url, contentType string, body io.Reader, httpClient *retryablehttp.Client) (int, string) {
	return doPost(url, contentType, "", body, httpClient)
}

// doPost
func doPost(url, contentType string, hostHeader string, body io.Reader, httpClient *retryablehttp.Client) (int, string) {
	return doReq(url, "POST", contentType, hostHeader, "", "", body, httpClient)
}

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
		ginkgo.Fail("Could not POST " + url)
	}
	defer resp.Body.Close()
	html, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		Log(Error, err.Error())
		ginkgo.Fail("Could not read content of response body")
	}
	return resp.StatusCode, string(html)
}

func getHTTPClientWIthCABundle(caData []byte) *http.Client {
	tr := &http.Transport{TLSClientConfig: &tls.Config{RootCAs: rootCertPool(caData)}}

	proxyURL := getProxyURL()
	if proxyURL != "" {
		tURL := url.URL{}
		tURLProxy, _ := tURL.Parse(proxyURL)
		tr.Proxy = http.ProxyURL(tURLProxy)
	}

	ipResolve := getNodeIP()
	if ipResolve != "" {
		dialer := &net.Dialer{
			Timeout:   30 * time.Second,
			KeepAlive: 30 * time.Second,
		}
		tr.DialContext = func(ctx context.Context, network, addr string) (net.Conn, error) {
			Log(Debug, fmt.Sprintf("original address %s", addr))
			if strings.Contains(addr, "127.0.0.1") && strings.Contains(addr, ":443") {
				addr = ipResolve + ":443"
				Log(Debug, fmt.Sprintf("modified address %s", addr))
			}
			return dialer.DialContext(ctx, network, addr)
		}
	}

	return &http.Client{Transport: tr}
}

func getVerrazzanoCACert() []byte {
	return doGetCACertFromSecret(EnvName+"-secret", "verrazzano-system")
}

func getKeycloakCACert() []byte {
	return doGetCACertFromSecret(EnvName+"-secret", "keycloak")
}

func getSystemVMICACert() []byte {
	return doGetCACertFromSecret("system-tls", "verrazzano-system")
}

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

func doGetCACertFromSecret(secretName string, namespace string) []byte {
	clientset := GetKubernetesClientset()
	certSecret, _ := clientset.CoreV1().Secrets(namespace).Get(context.TODO(), secretName, metav1.GetOptions{})
	return certSecret.Data["ca.crt"]
}

// Returns the control-plane node ip
func getNodeIP() string {
	clientset := GetKubernetesClientset()
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

func newRetryableHTTPClient(client *http.Client) *retryablehttp.Client {
	retryableClient := retryablehttp.NewClient() //default of 4 retries is sufficient for us
	retryableClient.RetryMax = NumRetries
	retryableClient.RetryWaitMin = RetryWaitMin
	retryableClient.RetryWaitMax = RetryWaitMax
	retryableClient.HTTPClient = client
	return retryableClient
}

func rootCertPool(caData []byte) *x509.CertPool {
	if len(caData) == 0 {
		return nil
	}

	// if we have caData, use it
	certPool := x509.NewCertPool()
	certPool.AppendCertsFromPEM(caData)
	return certPool
}
