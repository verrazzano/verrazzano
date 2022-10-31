// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package mockmatchers

import (
	"fmt"
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

// URIMethodMatcher for use with gomock for http requests
// matches both the path and the method for and http request
type URIMethodMatcher struct {
	method, path string
}

func MatchesURIMethod(method, path string) gomock.Matcher {
	return &URIMethodMatcher{method, path}
}

func (u *URIMethodMatcher) Matches(x interface{}) bool {
	if !gomock.AssignableToTypeOf(&http.Request{}).Matches(x) {
		return false
	}
	req := x.(*http.Request)
	return req.URL.Path == u.path && req.Method == u.method
}

func (u *URIMethodMatcher) String() string {
	return fmt.Sprintf("is a request whose method matches %s and URI path matches %s", u.method, u.path)
}
