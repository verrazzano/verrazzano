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
	"os"
	"testing"

	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/stretchr/testify/assert"
	"github.com/verrazzano/verrazzano/authproxy/internal/testutil/testserver"
	"go.uber.org/zap"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

type mockVerifier struct {
	issuer string
	token  string
}

var _ verifier = &mockVerifier{}

func TestNewAuthenticator(t *testing.T) {
	// unset these otherwise the test fails due to not being able to connect to the proxy
	os.Unsetenv("HTTPS_PROXY")
	os.Unsetenv("https_proxy")

	server := testserver.FakeOIDCProviderServer(t)

	config := &OIDCConfiguration{
		ExternalURL: server.URL,
		ServiceURL:  server.URL,
	}
	client := fake.NewClientBuilder().Build()

	authenticator, err := NewAuthenticator(config, zap.S(), client)
	assert.NoError(t, err)
	assert.NotNil(t, authenticator)
	assert.Equal(t, config, authenticator.oidcConfig)
	assert.Equal(t, client, authenticator.k8sClient)
	assert.NotNil(t, authenticator.ctx)
	assert.Implements(t, (*verifier)(nil), authenticator.verifier.Load())
}

// TestAuthenticaterRequest tests that the login redirect can be performed for a given request
// GIVEN a request without an authorization header
// WHEN the request is processed
// THEN the redirect should occur
func TestAuthenticateRequest(t *testing.T) {
	const configURI = "/.well-known/openid-configuration"
	const authURI = "/auth"
	validToken := "token-valid"
	issuer := "test-issuer"
	testIssuerURL := fmt.Sprintf("http://%s", issuer)
	testClientID := "test-client"

	mockIdpServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.NotNil(t, r)
		serverURL := fmt.Sprintf("http://%s", r.Host)
		if r.RequestURI == configURI {
			respMap := map[string]string{
				"issuer":                 serverURL,
				"authorization_endpoint": fmt.Sprintf("%s%s", serverURL, authURI),
			}
			resp, err := json.Marshal(respMap)
			assert.Nil(t, err)
			_, err = w.Write(resp)
			assert.NoError(t, err)
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
