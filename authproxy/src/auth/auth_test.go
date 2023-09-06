// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package auth

import (
	"context"
	"fmt"
	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
	"net/http"
	"net/http/httptest"
	"testing"
)

type mockVerifier struct {
	issuer string
	token  string
}

var _ verifier = &mockVerifier{}

func TestAuthenticateRequest(t *testing.T) {
	validToken := "token-valid"
	issuer := "test-issuer"
	authenticator := OIDCAuthenticator{
		Log: zap.S(),
		oidcConfig: &OIDCConfiguration{
			IssuerURL: issuer,
		},
	}
	verifier := newMockVerifier(issuer, validToken)
	authenticator.verifier.Store(verifier)

	tests := []struct {
		name             string
		request          http.Request
		expectValidation bool
	}{
		{
			name: "valid token provided",
			request: http.Request{
				Header: map[string][]string{
					authHeaderKey: {
						"Bearer " + validToken,
					},
				},
			},
			expectValidation: true,
		},
		{
			name: "invalid token provided",
			request: http.Request{
				Header: map[string][]string{
					authHeaderKey: {
						"Bearer token-invalid",
					},
				},
			},
			expectValidation: false,
		},
		{
			name:             "no authorization header",
			request:          http.Request{},
			expectValidation: false,
		},
		{
			name: "malformed authorization header",
			request: http.Request{
				Header: map[string][]string{
					authHeaderKey: {
						"malformed-header",
					},
				},
			},
			expectValidation: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			validated, err := authenticator.AuthenticateRequest(&tt.request, httptest.NewRecorder())
			if tt.expectValidation {
				assert.NoError(t, err)
				assert.True(t, validated)
				return
			}
			assert.Error(t, err)
			assert.False(t, validated)
		})
	}
}

func TestAuthenticateToken(t *testing.T) {
	validToken := "token-valid"
	issuer := "test-issuer"
	authenticator := OIDCAuthenticator{
		Log: zap.S(),
		oidcConfig: &OIDCConfiguration{
			IssuerURL: issuer,
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
			validated, err := authenticator.AuthenticateToken(context.TODO(), tt.token)
			if tt.expectValidation {
				assert.NoError(t, err)
				assert.True(t, validated)
				return
			}
			assert.Error(t, err)
			assert.False(t, validated)
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
