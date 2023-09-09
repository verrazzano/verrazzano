// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package client

import (
	"crypto/tls"
	"crypto/x509"
	"net/http"

	"github.com/hashicorp/go-retryablehttp"
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
