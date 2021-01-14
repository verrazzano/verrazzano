// Copyright (c) 2020, 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package util

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/hashicorp/go-retryablehttp"
	"github.com/onsi/ginkgo"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	// NumRetries - maximum number of retries
	NumRetries = 7
	// RetryWaitMin - minimum retry wait
	RetryWaitMin = 1 * time.Second
	// RetryWaitMax - maximum retry wait
	RetryWaitMax = 30 * time.Second
)

// GetWebPageWithCABundle - same as GetWebPage, but with additional caData
func GetWebPageWithCABundle(url string, hostHeader string) (int, string) {
	return doGetWebPage(url, hostHeader, GetVerrazzanoHTTPClient())
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

func doGetWebPage(url string, hostHeader string, httpClient *retryablehttp.Client) (int, string) {
	req, _ := retryablehttp.NewRequest("GET", url, nil)
	if hostHeader != "" {
		//_have_ to set req.Host, not use req.Header.Add - latter does not work by design in Go
		req.Host = hostHeader
	}
	resp, err := httpClient.Do(req)
	if err != nil {
		Log(Error, err.Error())
		ginkgo.Fail("Could not get web page " + url)
	}
	defer resp.Body.Close()
	html, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		Log(Error, err.Error())
		ginkgo.Fail("Could not read content of response body")
	}
	return resp.StatusCode, string(html)
}

// GetVerrazzanoHTTPClient returns the Http client
func GetVerrazzanoHTTPClient() *retryablehttp.Client {
	rawClient := getHTTPClientWIthCABundle(getVerrazzanoCACert())
	return newRetryableHTTPClient(rawClient)
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
	return doGetCACertFromSecret("default-secret", "verrazzano-system")
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
