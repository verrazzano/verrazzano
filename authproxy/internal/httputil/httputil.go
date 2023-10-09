// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package httputil

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"net/http"
	"net/url"
	"os"

	"github.com/hashicorp/go-retryablehttp"
	vzpassword "github.com/verrazzano/verrazzano/pkg/security/password"
)

// GetHTTPClientWithCABundle returns a retryable HTTP client with the given cert pool
func GetHTTPClientWithCABundle(rootCA *x509.CertPool) (*retryablehttp.Client, error) {
	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.TLSClientConfig = &tls.Config{
		RootCAs:    rootCA,
		MinVersion: tls.VersionTLS12,
	}

	proxyURL := getProxyURL()
	if proxyURL != "" {
		u := url.URL{}
		proxy, err := u.Parse(proxyURL)
		if err != nil {
			return nil, err
		}
		transport.Proxy = http.ProxyURL(proxy)
	}

	client := retryablehttp.NewClient()
	client.HTTPClient.Transport = transport
	return client, nil
}

// ObfuscateRequestData removes the Authorization header data from the request before logging
func ObfuscateRequestData(req *http.Request) *http.Request {
	hiddenReq := req.Clone(context.TODO())
	authKey := "Authorization"
	for i := range hiddenReq.Header[authKey] {
		hiddenReq.Header[authKey][i] = vzpassword.MaskFunction("")(hiddenReq.Header[authKey][i])
	}
	return hiddenReq
}

// getProxyURL returns an HTTP proxy from the environment if one is set, otherwise an empty string
func getProxyURL() string {
	if proxyURL := os.Getenv("https_proxy"); proxyURL != "" {
		return proxyURL
	}
	if proxyURL := os.Getenv("HTTPS_PROXY"); proxyURL != "" {
		return proxyURL
	}
	return ""
}
