// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package auth

import (
	"context"
	"net/http"

	"go.uber.org/zap"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var _ Authenticator = fakeAuthenticator{nil, nil, nil}

type Authenticator interface {
	AuthenticateRequest(req *http.Request, rw http.ResponseWriter) (bool, error)
	AuthenticateToken(ctx context.Context, token string) (bool, error)
}

type OIDCConfiguration struct {
	IssuerURL   string
	ClientID    string
	CallbackURL string
}

type fakeAuthenticator struct {
	oidcConfig *OIDCConfiguration
	Log        *zap.SugaredLogger
	K8sClient  client.Client
}

func NewFakeAuthenticator(oidcConfig *OIDCConfiguration, log *zap.SugaredLogger, client client.Client) *fakeAuthenticator {
	return &fakeAuthenticator{oidcConfig: oidcConfig, Log: log, K8sClient: client}
}

// Authenticate authenticates the given request. If a redirect or error has been processed, then
// return true to indicate the request has been fully processed. Otherwise return false to indicate
// that request processing should continue
func (a fakeAuthenticator) AuthenticateRequest(req *http.Request, rw http.ResponseWriter) (bool, error) {
	// request is not processed
	return false, nil
}

// AuthenticateToken authenticates the given token
func (a fakeAuthenticator) AuthenticateToken(ctx context.Context, token string) (bool, error) {
	return true, nil
}
