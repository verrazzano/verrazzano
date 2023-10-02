// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package auth

import (
	"context"
	"net/http"
	"sync/atomic"

	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/hashicorp/go-retryablehttp"
	"go.uber.org/zap"
	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"
)

// Authenticator is the interface implemented by OIDCAuthenticator
type Authenticator interface {
	AuthenticateToken(ctx context.Context, token string) (*oidc.IDToken, error)
	AuthenticateRequest(req *http.Request, rw http.ResponseWriter) (bool, error)
	SetCallbackURL(url string)
	ExchangeCodeForToken(req *http.Request, rw http.ResponseWriter, codeVerifier string) (string, error)
}

// OIDCAuthenticator authenticates incoming requests against the Identity Provider
type OIDCAuthenticator struct {
	k8sClient        k8sclient.Client
	oidcConfig       *OIDCConfiguration
	client           *retryablehttp.Client
	ExternalProvider *oidc.Provider
	verifier         atomic.Value
	Log              *zap.SugaredLogger
}

var _ Authenticator = &OIDCAuthenticator{}

// OIDCConfiguration holds the data necessary to configure the OIDC interface
type OIDCConfiguration struct {
	ExternalURL string
	ServiceURL  string
	ClientID    string
	CallbackURL string
}

// ImpersonationHeaders returns the user and group impersonation headers from JWT tokens
type ImpersonationHeaders struct {
	User   string   `json:"preferred_username"`
	Groups []string `json:"groups"`
}
