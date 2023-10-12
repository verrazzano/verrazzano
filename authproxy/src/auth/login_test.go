// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package auth

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/stretchr/testify/assert"
	"github.com/verrazzano/verrazzano/authproxy/internal/testutil/testserver"
	globalconst "github.com/verrazzano/verrazzano/pkg/constants"
	"go.uber.org/zap"
	"golang.org/x/oauth2"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

// TestPerformLoginRedirect tests that the redirect occurs without an error
// GIVEN a login request
// WHEN  the redirect is processed
// THEN  no error occurs
func TestPerformLoginRedirect(t *testing.T) {
	authenticator := OIDCAuthenticator{
		oidcConfig: &OIDCConfiguration{
			ClientID: "",
		},
		ExternalProvider: &oidc.Provider{},
	}

	req := httptest.NewRequest(http.MethodGet, "https://authproxy.io", strings.NewReader(""))

	w := httptest.NewRecorder()
	err := authenticator.performLoginRedirect(req, w)
	assert.NoError(t, err)
}

// TestCreateContextWithHTTPClient tests that the context client can be created
func TestCreateContextWithHTTPClient(t *testing.T) {
	tests := []struct {
		name    string
		objects []k8sclient.Object
	}{
		// GIVEN a request to create a context client
		// WHEN  the CA cert does not exist
		// THEN  a client with no CA certificates is created
		{
			name: "no CA cert",
		},
		// GIVEN a request to create a context client
		// WHEN  the CA exists
		// THEN  a client with the CA certificate is created
		{
			name: "CA cert exists",
			objects: []k8sclient.Object{
				&v1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      globalconst.VerrazzanoSystemNamespace,
						Namespace: globalconst.PrivateCABundle,
					},
					Data: map[string][]byte{
						"cacert.pem": []byte("cert"),
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := fake.NewClientBuilder().WithObjects(tt.objects...).Build()

			authenticator := OIDCAuthenticator{
				Log:       zap.S(),
				k8sClient: client,
			}

			context, err := authenticator.createContextWithHTTPClient()
			assert.NoError(t, err)
			assert.NotNil(t, context)
			httpClientAny := context.Value(oauth2.HTTPClient)
			assert.NotNil(t, httpClientAny)

			httpClient, ok := httpClientAny.(*http.Client)
			assert.True(t, ok)
			assert.NotNil(t, httpClient)
			assert.NotNil(t, httpClient.Transport)
		})
	}
}

// TestInitExternalOIDCProvider tests that the OIDC provider can be initialized for the login flow
// GIVEN a request to initialize the OIDC provider
// WHEN  the OIDC server responds with correct initialization information
// THEN  no error is returned

func TestInitExternalOIDCProvider(t *testing.T) {
	client := fake.NewClientBuilder().Build()

	server := testserver.FakeOIDCProviderServer(t)

	authenticator := OIDCAuthenticator{
		Log:       zap.S(),
		k8sClient: client,
		oidcConfig: &OIDCConfiguration{
			ExternalURL: server.URL,
		},
		ctx: context.WithValue(context.Background(), oauth2.HTTPClient, server.Client()),
	}

	err := authenticator.initExternalOIDCProvider()
	assert.NoError(t, err)
}
