// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package client

import (
	"crypto/tls"
	"crypto/x509"
	"github.com/hashicorp/go-retryablehttp"
	"net/http"
)

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
