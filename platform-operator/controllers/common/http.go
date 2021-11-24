// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package common

import "net/http"

type (
	// HTTPpDoSig provides a HTTP Client wrapper function for unit testing
	HTTPpDoSig func(hc *http.Client, req *http.Request) (*http.Response, error)
)

// HTTPDo is the default HTTP Client wrapper implementation
var HTTPDo HTTPpDoSig = func(hc *http.Client, req *http.Request) (*http.Response, error) {
	return hc.Do(req)
}
