// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package auth

import (
	"context"
	"net/http"

	"go.uber.org/zap"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var _ Authenticator = FakeAuthenticator{nil, nil, nil}

type Authenticator interface {
	AuthenticateRequest(req *http.Request) (bool, error)
	AuthenticateToken(ctx context.Context, token string) (bool, error)
	SetCallbackURL(url string)
}

type OIDCConfiguration struct {
	ExternalURL string
	ServiceURL  string
	ClientID    string
	CallbackURL string
}

type FakeAuthenticator struct {
	oidcConfig *OIDCConfiguration
	Log        *zap.SugaredLogger
	K8sClient  client.Client
}

func NewFakeAuthenticator(oidcConfig *OIDCConfiguration, log *zap.SugaredLogger, client client.Client) *FakeAuthenticator {
	return &FakeAuthenticator{oidcConfig: oidcConfig, Log: log, K8sClient: client}
}

// AuthenticateRequest authenticates the given request. If a redirect or error has been processed, then
// return true to indicate the request has been fully processed. Otherwise return false to indicate
// that request processing should continue
func (a FakeAuthenticator) AuthenticateRequest(req *http.Request) (bool, error) {
	// request is not processed
	return false, nil
}

// AuthenticateToken authenticates the given token
func (a FakeAuthenticator) AuthenticateToken(ctx context.Context, token string) (bool, error) {
	return true, nil
}

func (a FakeAuthenticator) SetCallbackURL(url string) {
	a.oidcConfig.CallbackURL = url
}
