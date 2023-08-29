// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package cors

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAddCORSHeaders(t *testing.T) {
	testURL := "https://some-url.example.com"
	optionsReqAllowHeaders := "authorization, content-type"
	optionsReqAllowMethods := "GET, HEAD, POST, PUT, DELETE, OPTIONS, PATCH"
	ingressHostVal := "someorigin.example.com"
	validOrigin := fmt.Sprintf("https://%s", ingressHostVal)
	tests := []struct {
		name              string
		reqMethod         string
		ingressHost       string
		originHeader      []string // use array to test invalid case of multiple origin headers
		expectCORSHeaders bool
		expectOptHeaders  bool
		want              int
		wantErr           bool
	}{
		{"No origin header", http.MethodGet, ingressHostVal, []string{}, false, false, http.StatusOK, false},
		{"Multiple origin headers", http.MethodGet, ingressHostVal, []string{"origin1", "origin2"}, false, false, http.StatusBadRequest, true},
		{"Disallowed origin header GET request", http.MethodGet, ingressHostVal, []string{"https://notallowed"}, false, false, http.StatusOK, false},
		{"Disallowed origin header POST request", http.MethodPost, ingressHostVal, []string{"https://notallowed"}, false, false, http.StatusForbidden, true},
		{"Valid origin header GET request", http.MethodGet, ingressHostVal, []string{validOrigin}, true, false, http.StatusOK, false},
		{"Valid origin header OPTIONS request", http.MethodOptions, ingressHostVal, []string{validOrigin}, true, true, http.StatusOK, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, err := http.NewRequest(tt.reqMethod, testURL, nil)
			assert.Nil(t, err)
			for _, org := range tt.originHeader {
				req.Header.Add("Origin", org)
			}
			rw := httptest.NewRecorder()
			got, err := AddCORSHeaders(req, rw, tt.ingressHost)
			if (err != nil) != tt.wantErr {
				t.Errorf("AddCORSHeaders() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("AddCORSHeaders() got = %v, want %v", got, tt.want)
			}

			expectedAllowCreds := ""
			expectedAllowOrigin := ""
			if tt.expectCORSHeaders {
				expectedAllowCreds = "true"
				expectedAllowOrigin = tt.originHeader[0]
			}
			assert.Equal(t, rw.Header().Get("Access-Control-Allow-Origin"), expectedAllowOrigin)
			assert.Equal(t, rw.Header().Get("Access-Control-Allow-Credentials"), expectedAllowCreds)

			expectedAllowHeaders := ""
			expectedAllowMethods := ""
			if tt.expectOptHeaders {
				expectedAllowHeaders = optionsReqAllowHeaders
				expectedAllowMethods = optionsReqAllowMethods
			}
			assert.Equal(t, rw.Header().Get("Access-Control-Allow-Headers"), expectedAllowHeaders)
			assert.Equal(t, rw.Header().Get("Access-Control-Allow-Methods"), expectedAllowMethods)
		})
	}
}

func TestOriginAllowed(t *testing.T) {
	ingressHostVal := "someorigin.example.com"
	oneAllowedOrigin := "https://allowedorigin.example.com"
	defaultAllowedOriginFunc := func() string { return "" }
	oneAllowedOriginFunc := func() string { return oneAllowedOrigin }
	multiAllowedOriginsFunc := func() string { return fmt.Sprintf("https://alsoallowed.example.com,%s", oneAllowedOrigin) }

	tests := []struct {
		name              string
		ingressHost       string
		origin            string
		allowedOriginFunc func() string
		want              bool
	}{
		{"origin equals ingress host", ingressHostVal, fmt.Sprintf("https://%s", ingressHostVal), defaultAllowedOriginFunc, true},
		{"origin has value 'null'", ingressHostVal, "null", defaultAllowedOriginFunc, false},
		{"origin not equal to ingress host, no whitelist", ingressHostVal, "https://otherorigin.example.com", defaultAllowedOriginFunc, false},
		{"origin not equal to ingress host, in whitelist with one entry", ingressHostVal, oneAllowedOrigin, oneAllowedOriginFunc, true},
		{"origin not equal to ingress host, in whitelist with multiple entries", ingressHostVal, oneAllowedOrigin, multiAllowedOriginsFunc, true},
		{"origin not equal to ingress host, not in whitelist", ingressHostVal, "someotheroriginentirely", multiAllowedOriginsFunc, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			allowedOriginsWhitelistFunc = tt.allowedOriginFunc
			defer func() {
				allowedOriginsWhitelistFunc = defaultAllowedOriginFunc
			}()

			if got := originAllowed(tt.origin, tt.ingressHost); got != tt.want {
				t.Errorf("originAllowed() = %v, want %v", got, tt.want)
			}
		})
	}
}
