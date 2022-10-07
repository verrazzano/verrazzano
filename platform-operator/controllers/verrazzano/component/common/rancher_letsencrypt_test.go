// Copyright (c) 2021, 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package common

import (
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestCertBuilder verifies downloading certs from the web
// GIVEN a cert URI
//
//	WHEN appendCertWithHTTP is called
//	THEN appendCertWithHTTP should download the cert if it exists
func TestCertBuilder(t *testing.T) {
	var tests = []struct {
		testName string
		httpDo   HTTPDoSig
		isErr    bool
	}{
		{
			"should be able to download a cert",
			func(hc *http.Client, req *http.Request) (*http.Response, error) {
				return &http.Response{
					Body:       io.NopCloser(strings.NewReader("cert")),
					StatusCode: http.StatusOK,
				}, nil
			},
			false,
		},
		{
			"should fail to download a cert when the upstream server is down",
			func(hc *http.Client, req *http.Request) (*http.Response, error) {
				return &http.Response{
					Body:       io.NopCloser(strings.NewReader("cert")),
					StatusCode: http.StatusBadGateway,
				}, nil
			},
			true,
		},
		{
			"should fail to download a cert when the request fails",
			func(hc *http.Client, req *http.Request) (*http.Response, error) {
				return nil, errors.New("boom")
			},
			true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.testName, func(t *testing.T) {
			c := certBuilder{hc: &http.Client{}}
			HTTPDo = tt.httpDo
			err := c.appendCertWithHTTP(rootX1PEM)
			if tt.isErr {
				assert.NotNil(t, err)
			} else {
				assert.Nil(t, err)
			}
		})
	}
}

// TestBuildLetsEncryptChain verifies building the LetsEncrypt staging certificate chain
// GIVEN a certBuilder
//
//	WHEN buildLetsEncryptStagingChain is called
//	THEN buildLetsEncryptStagingChain should build the cert chain for LetsEncrypt
func TestBuildLetsEncryptChain(t *testing.T) {
	HTTPDo = func(hc *http.Client, req *http.Request) (*http.Response, error) {
		return &http.Response{
			Body:       io.NopCloser(strings.NewReader("cert")),
			StatusCode: http.StatusOK,
		}, nil
	}
	builder := &certBuilder{hc: &http.Client{}}
	err := builder.buildLetsEncryptStagingChain()
	assert.Nil(t, err)
	assert.Equal(t, "certcertcert", string(builder.cert))
}
