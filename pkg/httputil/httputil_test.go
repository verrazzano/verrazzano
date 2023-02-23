// Copyright (c) 2021, 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
package httputil_test

import (
	"fmt"
	"io/ioutil"
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

	filename := "/Users/sdosapat/vz-7954/response_out_raw_formatted_2"
	data, err := ioutil.ReadFile(filename)
	if err != nil {
		fmt.Println("Error reading file:", err)
		return
	}
	// Convert the byte slice to a string
	str := string(data)
	data1, err := httputil.ExtractFieldFromResponseBodyOrReturnError(str, "data")
	token, err = httputil.ExtractFieldFromResponseBodyOrReturnError(data1, "token")
	fmt.Println("sdosapat " + token)
	//asserts.NoError(err)
	//asserts.Equal(`[{"abcd":"efgh"}]`, token)

	// Valid response
	body = `{"token": "abcd"}`
	token, err = httputil.ExtractFieldFromResponseBodyOrReturnError(body, "token")
	asserts.NoError(err)
	asserts.Equal("abcd", token)

	// Valid non-string response
	body = `{"token": [{"abcd": "efgh"}]}`
	token, err = httputil.ExtractFieldFromResponseBodyOrReturnError(body, "token")
	asserts.NoError(err)

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
