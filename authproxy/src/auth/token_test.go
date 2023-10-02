// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package auth

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"testing"

	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
)

// TestAuthenticateToken tests that tokens are properly processed and validated
// GIVEN a request to authenticate a token
// WHEN  the request is processed
// THEN  the proper validation result is returned
func TestAuthenticateToken(t *testing.T) {
	validToken := "token-valid"
	issuer := "test-issuer"
	authenticator := OIDCAuthenticator{
		Log: zap.S(),
		oidcConfig: &OIDCConfiguration{
			ServiceURL: issuer,
		},
	}
	verifier := newMockVerifier(issuer, validToken)
	authenticator.verifier.Store(verifier)

	tests := []struct {
		name             string
		token            string
		expectValidation bool
	}{
		{
			name:             "valid token provided",
			token:            validToken,
			expectValidation: true,
		},
		{
			name:             "invalid token provided",
			token:            "token-invalid",
			expectValidation: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			idToken, err := authenticator.AuthenticateToken(context.TODO(), tt.token)
			if tt.expectValidation {
				assert.NoError(t, err)
				assert.NotNil(t, idToken)
				return
			}
			assert.Error(t, err)
			assert.Nil(t, idToken)
		})
	}
}

// TestGetTokenFromAuthHeader tests that the token can be extracted from an auth header
// GIVEN an auth header
// WHEN  the bearer token is properly formatted
// THEN  the token value is returned
func TestGetTokenFromAuthHeader(t *testing.T) {
	tests := []struct {
		name          string
		authHeader    string
		expectedToken string
		expectError   bool
	}{
		{
			name:        "empty auth header",
			expectError: true,
		},
		{
			name:        "no token",
			authHeader:  "Bearer",
			expectError: true,
		},
		{
			name:          "valid token",
			authHeader:    "Bearer token",
			expectedToken: "token",
		},
		{
			name:          "token with params",
			authHeader:    "Bearer token param1 param2",
			expectedToken: "token",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			token, err := getTokenFromAuthHeader(tt.authHeader)
			if tt.expectError {
				assert.Error(t, err)
				return
			}

			assert.NoError(t, err)
			assert.Equal(t, tt.expectedToken, token)
		})
	}
}

// TestInitServiceOIDCVerifier tests that the OIDC verifier can be properly initialized
func TestInitServiceOIDCVerifier(t *testing.T) {
	authenticator := OIDCAuthenticator{
		Log: zap.S(),
	}

	// GIVEN a valid configuration
	// WHEN  the service URL is not set
	// THEN  an error is returned
	err := authenticator.initServiceOIDCVerifier()
	assert.Error(t, err)

	issuer := "test-issuer"
	authenticator.oidcConfig = &OIDCConfiguration{
		ServiceURL: issuer,
	}
	// GIVEN a valid configuration
	// WHEN  the service URL is set
	// THEN  no error is returned
	err = authenticator.initServiceOIDCVerifier()
	assert.Error(t, err)

}

// TestLoadVerifier tests loading the verifier object from the atomic source
func TestLoadVerifier(t *testing.T) {
	authenticator := OIDCAuthenticator{
		Log: zap.S(),
	}

	// GIVEN an Authenticator object
	// WHEN  the verifier is not set
	// THEN  an error is returned
	_, err := authenticator.loadVerifier()
	assert.Error(t, err)

	// GIVEN an Authenticator object
	// WHEN  the verifier is not the correct value
	// THEN  an error is returned
	authenticator.verifier.Store("incorrect value")
	_, err = authenticator.loadVerifier()
	assert.Error(t, err)

	// GIVEN an Authenticator object
	// WHEN  the verifier is correctly set
	// THEN  no error is returned
	authenticator = OIDCAuthenticator{
		Log: zap.S(),
	}
	authenticator.verifier.Store(newMockVerifier("", ""))
	v, err := authenticator.loadVerifier()
	assert.NoError(t, err)
	assert.NotNil(t, v)
	assert.Implements(t, (*verifier)(nil), v)
}

// TestGetImpersonationHeadersFromRequest tests that the impersonation user and groups can be collected from a request
func TestGetImpersonationHeadersFromRequest(t *testing.T) {
	testUser := "user"
	testGroups := []string{
		"group1",
		"group2",
	}

	testImp := ImpersonationHeaders{
		User:   testUser,
		Groups: testGroups,
	}

	impJSON, err := json.Marshal(testImp)
	assert.NoError(t, err)
	validToken := fmt.Sprintf("info.%s.info", base64.RawURLEncoding.EncodeToString(impJSON))

	tests := []struct {
		name           string
		token          string
		expectedUser   string
		expectedGroups []string
		expectError    bool
	}{
		// GIVEN a request with a valid token
		// WHEN  the request is evaluated
		// THEN  the expected users and groups are populated
		{
			name:           "valid token provided",
			token:          validToken,
			expectedUser:   testUser,
			expectedGroups: testGroups,
		},
		// GIVEN a request with a bad JWT token
		// WHEN  the request is evaluated
		// THEN  an error is returned
		{
			name:        "malformed token provided",
			token:       "token-invalid",
			expectError: true,
		},
		// GIVEN a request with an empty token body
		// WHEN  the request is evaluated
		// THEN  no error is returned
		{
			name:  "empty token provided",
			token: fmt.Sprintf("info.%s.info", base64.RawURLEncoding.EncodeToString([]byte("{}"))),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := http.Request{
				Header: map[string][]string{
					authHeaderKey: {"Bearer " + tt.token},
				},
			}
			imp, err := GetImpersonationHeadersFromRequest(&req)
			if tt.expectError {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
			assert.Equal(t, tt.expectedUser, imp.User)
			assert.ElementsMatch(t, tt.expectedGroups, imp.Groups)
		})
	}
}

func (m mockVerifier) Verify(_ context.Context, rawIDToken string) (*oidc.IDToken, error) {
	if rawIDToken != m.token {
		return nil, fmt.Errorf("provided token does not match the mocked token")
	}
	return &oidc.IDToken{Issuer: m.issuer}, nil
}

func newMockVerifier(issuer, token string) *mockVerifier {
	return &mockVerifier{
		issuer: issuer,
		token:  token,
	}
}
