// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package auth

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/coreos/go-oidc/v3/oidc"
)

// AuthenticateToken verifies a given bearer token against the OIDC key and verifies the issuer is correct
func (a *OIDCAuthenticator) AuthenticateToken(ctx context.Context, token string) (bool, error) {
	verifier, err := a.loadVerifier()
	if err != nil {
		return false, err
	}

	idToken, err := verifier.Verify(ctx, token)
	if err != nil {
		a.Log.Errorf("Failed to verify JWT token: %v", err)
		return false, err
	}

	// Do issuer check for external URL
	// This is skipped in the go-oidc package because it could be the service or the ingress
	if idToken.Issuer != a.oidcConfig.ExternalURL && idToken.Issuer != a.oidcConfig.ServiceURL {
		err := fmt.Errorf("failed to verify issuer, got %s, expected %s or %s", idToken.Issuer, a.oidcConfig.ServiceURL, a.oidcConfig.ExternalURL)
		a.Log.Errorf("Failed to validate JWT issuer: %v", err)
		return false, err
	}

	return true, nil
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

	provider, err := oidc.NewProvider(context.TODO(), a.oidcConfig.ServiceURL)
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
