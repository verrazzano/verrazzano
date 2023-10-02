// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package auth

import (
	"context"
	"crypto/x509"
	"fmt"
	"net/http"

	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/verrazzano/verrazzano/authproxy/internal/httputil"
	"go.uber.org/zap"
	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	authHeaderKey = "Authorization"

	authTypeBearer string = "Bearer"
)

// verifier interface is implemented by the OIDC token verifier
type verifier interface {
	Verify(ctx context.Context, rawIDToken string) (*oidc.IDToken, error)
}

// NewAuthenticator returns a new OIDC authenticator with an initialized verifier
func NewAuthenticator(oidcConfig *OIDCConfiguration, log *zap.SugaredLogger, client k8sclient.Client) (*OIDCAuthenticator, error) {
	authenticator := &OIDCAuthenticator{
		Log:        log,
		client:     httputil.GetHTTPClientWithCABundle(&x509.CertPool{}),
		oidcConfig: oidcConfig,
		k8sClient:  client,
	}

	if err := authenticator.initExternalOIDCProvider(); err != nil {
		log.Errorf("Failed to initialize OIDC provider for the authenticator: %v", err)
		return nil, err
	}

	if err := authenticator.initServiceOIDCVerifier(); err != nil {
		log.Errorf("Failed to store verifier for the authenticator: %v", err)
		return nil, err
	}

	return authenticator, nil
}

// AuthenticateRequest performs login redirect if the authorization header is not provided.
// If the header is provided, the bearer token is validated against the OIDC key
func (a *OIDCAuthenticator) AuthenticateRequest(req *http.Request, rw http.ResponseWriter) (bool, error) {
	authHeader := req.Header.Get(authHeaderKey)

	if a.ExternalProvider == nil {
		return false, fmt.Errorf("the OIDC provider for authentication is not initialized")
	}
	if authHeader == "" {
		err := a.performLoginRedirect(req, rw)
		if err != nil {
			return false, fmt.Errorf("could not redirect for login: %v", err)
		}
		// we performed a redirect, so request processing is done and
		// no further processing is needed
		return false, nil
	}

	token, err := getTokenFromAuthHeader(authHeader)
	if err != nil {
		return false, fmt.Errorf("failed to get token from authorization header: %v", err)
	}

	_, err = a.AuthenticateToken(req.Context(), token)
	return err == nil, err
}

// SetCallbackURL sets the OIDC Callback URL for redirects
func (a *OIDCAuthenticator) SetCallbackURL(url string) {
	a.oidcConfig.CallbackURL = url
}
