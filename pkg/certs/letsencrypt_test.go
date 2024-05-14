// Copyright (c) 2021, 2024, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package certs

import (
	"bytes"
	"errors"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/common"
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
		httpDo   common.HTTPDoSig
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
			common.HTTPDo = tt.httpDo
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
	common.HTTPDo = func(hc *http.Client, req *http.Request) (*http.Response, error) {
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

// TestCreateLetsEncryptStagingBundle tests CreateLetsEncryptStagingBundle
// GIVEN a call to CreateLetsEncryptStagingBundle
//
//	WHEN CreateLetsEncryptStagingBundle is called
//	THEN CreateLetsEncryptStagingBundle should download the LE staging bundles if there is no error
func TestCreateLetsEncryptStagingBundle(t *testing.T) {
	var tests = []struct {
		testName string
		httpDo   common.HTTPDoSig
		bundle   []byte
		isErr    bool
	}{
		{
			"should be able to download staging bundles",
			func(hc *http.Client, req *http.Request) (*http.Response, error) {
				bundleData := ""
				switch req.URL.Path[len(req.URL.Path)-15:] {
				case "-stg-int-r3.pem":
					bundleData = "intR3PEM\n"
				case "-stg-int-e1.pem":
					bundleData = "intE1PEM\n"
				case "stg-root-x1.pem":
					bundleData = "rootX1PEM\n"
				}
				return &http.Response{
					Body:       io.NopCloser(strings.NewReader(bundleData)),
					StatusCode: http.StatusOK,
				}, nil
			},
			[]byte("intR3PEM\nintE1PEM\nrootX1PEM"),
			false,
		},
		{
			"should fail to download a cert when the request fails",
			func(hc *http.Client, req *http.Request) (*http.Response, error) {
				return nil, errors.New("boom")
			},
			[]byte{},
			true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.testName, func(t *testing.T) {
			common.HTTPDo = tt.httpDo
			bundleData, err := CreateLetsEncryptStagingBundle()
			byteSlicesEqualTrimmedWhitespace(t, tt.bundle, bundleData)
			if tt.isErr {
				assert.NotNil(t, err)
			} else {
				assert.Nil(t, err)
			}
		})
	}
}

func byteSlicesEqualTrimmedWhitespace(t *testing.T, byteSlice1, byteSlice2 []byte) bool {
	a := bytes.Trim(byteSlice1, " \t\n\r")
	b := bytes.Trim(byteSlice2, " \t\n\r")
	return assert.Equal(t, a, b)
}
