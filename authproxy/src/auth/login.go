// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"fmt"
	"net/http"

	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/verrazzano/verrazzano/authproxy/internal/httputil"
	"github.com/verrazzano/verrazzano/authproxy/src/cookie"
	"github.com/verrazzano/verrazzano/pkg/certs"
	"golang.org/x/oauth2"
	"k8s.io/client-go/util/cert"
)

// initExternalOIDCProvider initializes the external URL based OIDC Provider in the given Authenticator
func (a *OIDCAuthenticator) initExternalOIDCProvider() error {
	ctx, err := a.createContextWithHTTPClient()
	if err != nil {
		return err
	}
	provider, err := oidc.NewProvider(ctx, a.oidcConfig.ExternalURL)
	if err != nil {
		return err
	}
	a.ExternalProvider = provider
	return nil
}

// createContextWithHTTPClient creates a context with the correct certificates and
// client to redirect to the OIDC provider
func (a *OIDCAuthenticator) createContextWithHTTPClient() (context.Context, error) {
	caBundleData, err := certs.GetLocalClusterCABundleData(a.Log, a.k8sClient, context.TODO())
	if err != nil {
		return nil, err
	}
	var certPool *x509.CertPool
	if caBundleData != nil {
		if certPool, err = cert.NewPoolFromBytes(caBundleData); err != nil {
			return nil, err
		}
	}

	httpClient := httputil.GetHTTPClientWithCABundle(certPool)
	ctx := context.Background()
	return context.WithValue(ctx, oauth2.HTTPClient, httpClient.HTTPClient), nil
}

// performLoginRedirect redirects the incoming request to the OIDC provider
func (a *OIDCAuthenticator) performLoginRedirect(req *http.Request, rw http.ResponseWriter) error {
	var state string
	var nonce string
	var err error
	if state, err = randomBase64(32); err != nil {
		return fmt.Errorf("could not redirect for login - failed to generate random base64: %v", err)
	}
	if nonce, err = randomBase64(32); err != nil {
		return fmt.Errorf("could not redirect for login - failed to generate random base64: %v", err)
	}

	oauthConfig := oauth2.Config{
		ClientID:    a.oidcConfig.ClientID,
		Endpoint:    a.ExternalProvider.Endpoint(),
		RedirectURL: a.oidcConfig.CallbackURL,
		Scopes:      []string{oidc.ScopeOpenID, "profile", "email"},
	}

	// Create the PKCE challenge
	code, err := randomBase64(56)
	if err != nil {
		return fmt.Errorf("Could not redirect for login - failed to generate random base64: %v", err)
	}
	sha := sha256.New()
	sha.Write([]byte(code))
	challenge := oauth2.SetAuthURLParam("code_challenge", base64.RawURLEncoding.EncodeToString(sha.Sum(nil)))
	challengeMethod := oauth2.SetAuthURLParam("code_challenge_method", "S256")

	vzState := &cookie.VZState{
		State:        state,
		Nonce:        nonce,
		CodeVerifier: code,
		RedirectURI:  req.RequestURI,
	}
	cookie.SetStateCookie(rw, vzState)
	http.Redirect(rw, req, oauthConfig.AuthCodeURL(state, oidc.Nonce(nonce), challengeMethod, challenge), http.StatusFound)
	return nil
}

// randomBase64 returns a random base64-encoded string of the given size
func randomBase64(size int) (string, error) {
	b := make([]byte, size)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}
