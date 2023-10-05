// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package auth

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/coreos/go-oidc/v3/oidc"
	"golang.org/x/oauth2"
)

// AuthenticateToken verifies a given bearer token against the OIDC key and verifies the issuer is correct
func (a *OIDCAuthenticator) AuthenticateToken(ctx context.Context, token string) (*oidc.IDToken, error) {
	verifier, err := a.loadVerifier()
	if err != nil {
		return nil, err
	}

	idToken, err := verifier.Verify(ctx, token)
	if err != nil {
		a.Log.Errorf("Failed to verify JWT token: %v", err)
		return nil, err
	}

	// Do issuer check for external URL
	// This is skipped in the go-oidc package because it could be the service or the ingress
	if idToken.Issuer != a.oidcConfig.ExternalURL && idToken.Issuer != a.oidcConfig.ServiceURL {
		err := fmt.Errorf("failed to verify issuer, got %s, expected %s or %s", idToken.Issuer, a.oidcConfig.ServiceURL, a.oidcConfig.ExternalURL)
		a.Log.Errorf("Failed to validate JWT issuer: %v", err)
		return nil, err
	}

	return idToken, nil
}

func (a *OIDCAuthenticator) ExchangeCodeForToken(req *http.Request, codeVerifier string) (string, error) {
	// TODO Create this once?
	oauthConfig := oauth2.Config{
		ClientID:    a.oidcConfig.ClientID,
		Endpoint:    a.ExternalProvider.Endpoint(),
		RedirectURL: a.oidcConfig.CallbackURL,
		Scopes:      []string{oidc.ScopeOpenID, "profile", "email"},
	}
	codeVerifierParam := oauth2.SetAuthURLParam("code_verifier", codeVerifier)

	oauth2Token, err := oauthConfig.Exchange(a.ctx, req.URL.Query().Get("code"), codeVerifierParam)
	if err != nil {
		a.Log.Errorf("Failed exchanging code for token: %v", err)
		return "", err
	}
	rawIDToken, ok := oauth2Token.Extra("id_token").(string)
	if !ok {
		errStr := "ID token not found in oauth token"
		a.Log.Error(errStr)
		return "", fmt.Errorf(errStr)
	}

	return rawIDToken, nil
}

// getTokenFromAuthHeader returns the bearer token from the authorization header
func getTokenFromAuthHeader(authHeader string) (string, error) {
	splitHeader := strings.SplitN(authHeader, " ", 3)

	if len(splitHeader) < 2 || !strings.EqualFold(splitHeader[0], authTypeBearer) {
		return "", fmt.Errorf("failed to verify authorization bearer header")
	}

	return splitHeader[1], nil
}

// initServiceOIDCVerifier creates an OIDC provider using the Service URL
// and populates the authenticator with a verifier
func (a *OIDCAuthenticator) initServiceOIDCVerifier() error {
	if a.oidcConfig == nil {
		err := fmt.Errorf("the OIDC config is not initialized")
		a.Log.Errorf("Failed to set up token verifier: %v", err)
		return err
	}

	provider, err := oidc.NewProvider(a.ctx, a.oidcConfig.ServiceURL)
	if err != nil {
		a.Log.Errorf("Failed to load OIDC provider: %v", err)
		return err
	}

	config := &oidc.Config{
		ClientID:             a.oidcConfig.ClientID,
		SkipIssuerCheck:      true,
		SupportedSigningAlgs: []string{oidc.RS256},
		Now:                  time.Now,
	}

	verifier := provider.Verifier(config)
	a.verifier.Store(verifier)
	return nil
}

// loadVerifier returns the stored value and casts it to a verifier object
func (a *OIDCAuthenticator) loadVerifier() (verifier, error) {
	vAny := a.verifier.Load()
	if vAny == nil {
		err := fmt.Errorf("nil verifier object")
		a.Log.Errorf("Failed to load verifier: %v", err)
	}

	if vTyped, ok := vAny.(verifier); ok {
		return vTyped, nil
	}

	err := fmt.Errorf("object does not implement the verifier interface")
	a.Log.Errorf("Failed to load verifier: %v", err)
	return nil, err
}

// GetImpersonationHeadersFromRequest returns the user and group fields from the bearer token to be used as
// impersonation headers for the API server request
func GetImpersonationHeadersFromRequest(req *http.Request) (ImpersonationHeaders, error) {
	var headers ImpersonationHeaders

	token, err := getTokenFromAuthHeader(req.Header.Get(authHeaderKey))
	if err != nil {
		return headers, err
	}

	jwtParts := strings.SplitN(token, ".", 3)
	if len(jwtParts) != 3 {
		return headers, fmt.Errorf("malformed jwt token, found %d sections", len(jwtParts))
	}

	payload, err := base64.RawURLEncoding.DecodeString(jwtParts[1])
	if err != nil {
		return headers, err
	}

	err = json.Unmarshal(payload, &headers)
	if err != nil {
		return headers, err
	}

	return headers, nil
}
