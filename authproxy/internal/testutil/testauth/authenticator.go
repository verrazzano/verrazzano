// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package testauth

import (
	"context"
	"net/http"

	"github.com/verrazzano/verrazzano/authproxy/src/auth"
)

// FakeAuthenticator returns a fake implementation of the Authenticator interface
type FakeAuthenticator struct {
	authenticateTokenFunc   func() (bool, error)
	authenticateRequestFunc func() (bool, error)
}

// NewFakeAuthenticator returns a new FakeAuthenticator object with authentication set to true
func NewFakeAuthenticator() *FakeAuthenticator {
	return &FakeAuthenticator{
		authenticateTokenFunc:   AuthenticateTrue,
		authenticateRequestFunc: AuthenticateTrue,
	}
}

func (f *FakeAuthenticator) AuthenticateToken(_ context.Context, _ string) (bool, error) {
	return f.authenticateTokenFunc()
}
func (f *FakeAuthenticator) AuthenticateRequest(_ *http.Request, _ http.ResponseWriter) (bool, error) {
	return f.authenticateRequestFunc()
}
func (f *FakeAuthenticator) SetCallbackURL(_ string) {}

func (f *FakeAuthenticator) SetTokenFunc(fun func() (bool, error)) {
	f.authenticateTokenFunc = fun
}

func (f *FakeAuthenticator) SetRequestFunc(fun func() (bool, error)) {
	f.authenticateRequestFunc = fun
}

func AuthenticateTrue() (bool, error) {
	return true, nil
}

func AuthenticateFalse() (bool, error) {
	return false, nil
}

var _ auth.Authenticator = &FakeAuthenticator{}
