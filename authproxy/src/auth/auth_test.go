// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package auth

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
	"k8s.io/apimachinery/pkg/runtime"
)

type mockVerifier struct {
	issuer string
	token  string
}

type mockProvider struct{}

var _ verifier = &mockVerifier{}

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

func TestAuthenticateRequest(t *testing.T) {
	const configURI = "/.well-known/openid-configuration"
	const authURI = "/auth"
	validToken := "token-valid"
	issuer := "test-issuer"
	testIssuerURL := fmt.Sprintf("http://%s", issuer)
	testClientID := "test-client"

	mockIdpServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.NotNil(t, r)
		// body, err := io.ReadAll(r.Body)
		// assert.NoError(t, err)
		serverURL := fmt.Sprintf("http://%s", r.Host)
		if r.RequestURI == configURI {
			respMap := map[string]string{
				"issuer":                 serverURL,
				"authorization_endpoint": fmt.Sprintf("%s%s", serverURL, authURI),
			}
			resp, err := json.Marshal(respMap)
			assert.Nil(t, err)
			w.Write(resp)
			w.Header().Add("Content-Type", runtime.ContentTypeJSON)
		}
	}))
	provider, err := oidc.NewProvider(context.Background(), mockIdpServer.URL)
	assert.Nil(t, err)
	authenticator := OIDCAuthenticator{
		Log: zap.S(),
		oidcConfig: &OIDCConfiguration{
			ServiceURL:  issuer,
			ExternalURL: mockIdpServer.URL,
			ClientID:    testClientID,
		},
		ExternalProvider: provider,
	}
	verifier := newMockVerifier(issuer, validToken)
	authenticator.verifier.Store(verifier)
	req, err := http.NewRequest(http.MethodGet, testIssuerURL, nil)
	rw := httptest.NewRecorder()
	assert.Nil(t, err)

	// no auth header, expect redirect
	continueProcessing, err := authenticator.AuthenticateRequest(req, rw)
	assert.Nil(t, err)
	assert.False(t, continueProcessing)
	assert.Equal(t, http.StatusFound, rw.Code)

	redirLocation := rw.Header().Get("Location")
	assert.NotEmpty(t, redirLocation)
	redirURL, err := url.Parse(redirLocation)
	assert.Nil(t, err)
	expectedRedirURL := fmt.Sprintf("%s%s", mockIdpServer.URL, authURI)
	actualRedirURL := fmt.Sprintf("http://%s%s", redirURL.Host, redirURL.Path)
	assert.Equal(t, expectedRedirURL, actualRedirURL)
	assert.Equal(t, testClientID, redirURL.Query().Get("client_id"))
	assert.Equal(t, "code", redirURL.Query().Get("response_type"))
	assert.NotEmpty(t, redirURL.Query().Get("nonce"))
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
