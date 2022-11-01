// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package mockmatchers

import (
	"net/http"

	"github.com/golang/mock/gomock"
)

// URIMatcher for use with gomock for http requests
type URIMatcher struct {
	path string
}

func MatchesURI(path string) gomock.Matcher {
	return &URIMatcher{path}
}

func (u *URIMatcher) Matches(x interface{}) bool {
	if !gomock.AssignableToTypeOf(&http.Request{}).Matches(x) {
		return false
	}
	req := x.(*http.Request)
	return req.URL.Path == u.path
}

func (u *URIMatcher) String() string {
	return "is a request whose URI path matches " + u.path
}
