// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package httputil

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"net/http"

	"github.com/hashicorp/go-retryablehttp"
	vzpassword "github.com/verrazzano/verrazzano/pkg/security/password"
)

// GetHTTPClientWithCABundle returns a retryable HTTP client with the given cert pool
func GetHTTPClientWithCABundle(rootCA *x509.CertPool) *retryablehttp.Client {
	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.TLSClientConfig = &tls.Config{
		RootCAs:    rootCA,
		MinVersion: tls.VersionTLS12,
	}

	client := retryablehttp.NewClient()
	client.HTTPClient.Transport = transport
	return client
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
