// Copyright (c) 2022, 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
package httputil_test

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/verrazzano/verrazzano/pkg/httputil"
)

func TestExtractTokenFromResponseBody(t *testing.T) {
	asserts := assert.New(t)

	// Empty response
	var body = ""
	token, err := httputil.ExtractFieldFromResponseBodyOrReturnError(body, "token")
	asserts.Error(err)
	asserts.Empty(token)

	// Valid response
	body = `{"token": "abcd"}`
	token, err = httputil.ExtractFieldFromResponseBodyOrReturnError(body, "token")
	asserts.NoError(err)
	asserts.Equal("abcd", token)

	// Valid non-string response
	body = `{"token": [{"abcd": "efgh"}]}`
	token, err = httputil.ExtractFieldFromResponseBodyOrReturnError(body, "token")
	asserts.NoError(err)
	asserts.Equal(`[{"abcd":"efgh"}]`, token)

	// Expected error message
	body = `{"notoken": "yes"}`
	errMsg := "unable to find auth code"
	token, err = httputil.ExtractFieldFromResponseBodyOrReturnError(body, "token", errMsg)
	asserts.Error(err)
	asserts.Empty(token)
	asserts.Equal(err.Error(), errMsg)
}

func TestValidateResponseCode(t *testing.T) {
	asserts := assert.New(t)

	// Unexpected response code
	var response = http.Response{StatusCode: http.StatusAccepted, Request: &http.Request{}}
	err := httputil.ValidateResponseCode(&response, http.StatusCreated)
	asserts.Error(err)
	asserts.Contains(err.Error(), "expected response code")

	// Unexpected response code with multiple accepted response codes
	response = http.Response{StatusCode: http.StatusAccepted, Request: &http.Request{}}
	err = httputil.ValidateResponseCode(&response, http.StatusCreated, http.StatusAlreadyReported)
	asserts.Error(err)
	asserts.Contains(err.Error(), "expected one of response codes")

	// Valid response code
	response = http.Response{StatusCode: http.StatusAccepted, Request: &http.Request{}}
	err = httputil.ValidateResponseCode(&response, http.StatusAccepted)
	asserts.NoError(err)

}
