// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package httputil

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/Jeffail/gabs/v2"
)

// Helper function to extract field from json response body or returns an error containing input error message
func ExtractFieldFromResponseBodyOrReturnError(responseBody string, field string, errMsg ...string) (string, error) {
	jsonString, err := gabs.ParseJSON([]byte(responseBody))
	if err != nil {
		return "", err
	}

	if token, ok := jsonString.Path(field).Data().(string); ok {
		return token, nil
	}

	if toString := jsonString.Path(field).String(); toString != "null" {
		return toString, nil
	}

	errorString := "unable to find token in response"
	if errMsg != nil {
		errorString = errMsg[0]
	}
	return "", errors.New(errorString)

}

// Helper function to validate response code for http response
func ValidateResponseCode(response *http.Response, validResponseCodes ...int) error {
	if response != nil && !integerSliceContains(validResponseCodes, response.StatusCode) {
		statusCodeMsg := `one of response codes ` + fmt.Sprintf("%v", validResponseCodes)
		if len(validResponseCodes) == 1 {
			statusCodeMsg = `response code ` + fmt.Sprintf("%v", validResponseCodes[0])
		}
		return fmt.Errorf("expected %s from %s but got %d: %v", statusCodeMsg, response.Request.Method, response.StatusCode, response)
	}

	return nil
}

func integerSliceContains(slice []int, i int) bool {
	for _, item := range slice {
		if item == i {
			return true
		}
	}
	return false
}
