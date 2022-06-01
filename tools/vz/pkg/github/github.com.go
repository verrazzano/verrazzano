// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package github

import (
	"net/http"
)

// NewClient - create an HTTP client for accessing GitHub
func NewClient() (*http.Client, error) {
	tr := &http.Transport{}
	httpClient := &http.Client{Transport: tr}

	return httpClient, nil
}
