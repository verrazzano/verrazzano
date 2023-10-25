// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package httputil

import (
	"crypto/x509"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestGetHTTPClientWithCABundle tests that a retryable http client can be created with a cert pool
// GIVEN a request to create a http client
// WHEN the cert pool is given
// THEN the client returned is not nil
func TestGetHTTPClientWithCABundle(t *testing.T) {
	cli, err := GetHTTPClientWithCABundle(&x509.CertPool{})
	assert.NoError(t, err)
	assert.NotNil(t, cli)
}

// TestObfuscateTestData tests that request authorization headers get scrubbed
// GIVEN a request with an authorization header
// WHEN  the request is scrubbed
// THEN  the header contains a different value
func TestObfuscateTestData(t *testing.T) {
	req, err := http.NewRequest(http.MethodGet, "", strings.NewReader(""))
	assert.NoError(t, err)

	authKey := "Authorization"
	basicAuth := "Basic username:pass"
	tokenAuth := "Bearer test-token"
	req.Header[authKey] = []string{basicAuth, tokenAuth}

	obfReq := ObfuscateRequestData(req)
	assert.NotEqual(t, basicAuth, obfReq.Header[authKey][0])
	assert.NotEqual(t, tokenAuth, obfReq.Header[authKey][1])
}
