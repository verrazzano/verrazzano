// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package proxy

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"reflect"
	"runtime"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/verrazzano/verrazzano/authproxy/internal/testutil/file"
	"github.com/verrazzano/verrazzano/authproxy/internal/testutil/testauth"
	"github.com/verrazzano/verrazzano/authproxy/internal/testutil/testserver"
	"github.com/verrazzano/verrazzano/authproxy/src/auth"
	"github.com/verrazzano/verrazzano/authproxy/src/cookie"
	"github.com/verrazzano/verrazzano/pkg/k8sutil"
	"go.uber.org/zap"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

const (
	apiPath          = "/api/v1/pods"
	testAPIServerURL = "https://api-server.io"
	caCertFile       = "./testdata/test-ca.crt"
)

var serverURL string

// TestConfigureKubernetesAPIProxy tests the configuration of the API proxy
// GIVEN an Auth proxy object
// WHEN  the Kubernetes API proxy is configured
// THEN  the handler exists and there is no error
func TestConfigureKubernetesAPIProxy(t *testing.T) {
	c := fake.NewClientBuilder().Build()
	authproxy := InitializeProxy(8777)
	log := zap.S()

	getConfigFunc = testConfig
	defer func() { getConfigFunc = k8sutil.GetConfigFromController }()

	err := ConfigureKubernetesAPIProxy(authproxy, c, log)
	assert.NoError(t, err)
	assert.NotNil(t, authproxy.Handler)
}

// TestLoadCAData tests that the CA data is properly loaded from sources
func TestLoadCAData(t *testing.T) {
	// GIVEN a config with the CA Data populated
	// WHEN  the cert pool is generated
	// THEN  no error is returned
	caData, err := os.ReadFile(caCertFile)
	assert.NoError(t, err)
	config := &rest.Config{
		TLSClientConfig: rest.TLSClientConfig{
			CAData: caData,
		},
	}
	log := zap.S()
	pool, err := loadCAData(config, log)
	assert.NoError(t, err)
	assert.NotEmpty(t, pool)

	// GIVEN a config with the CA File populated
	// WHEN  the cert pool is generated
	// THEN  no error is returned
	config = &rest.Config{
		TLSClientConfig: rest.TLSClientConfig{
			CAFile: caCertFile,
		},
	}
	pool, err = loadCAData(config, log)
	assert.NoError(t, err)
	assert.NotEmpty(t, pool)
}

// TestLoadBearerToken tests that the bearer token is properly loaded from the config
func TestLoadBearerToken(t *testing.T) {
	log := zap.S()

	// GIVEN a config with Bearer Token populated
	// WHEN  the bearer token is loaded
	// THEN  the handler gets the bearer token data
	testToken := "test-token"
	config := &rest.Config{
		BearerToken: testToken,
	}
	bearerToken, err := loadBearerToken(config, log)
	assert.NoError(t, err)
	assert.Equal(t, testToken, bearerToken)

	// GIVEN a config with the Bearer Token file populated
	// WHEN  the bearer token is loaded
	// THEN  the handler gets the bearer token data
	testTokenFile, err := file.MakeTempFile(testToken)
	if testTokenFile != nil {
		defer os.Remove(testTokenFile.Name())
	}
	assert.NoError(t, err)
	config = &rest.Config{
		BearerTokenFile: testTokenFile.Name(),
	}
	bearerToken, err = loadBearerToken(config, log)
	assert.NoError(t, err)
	assert.Equal(t, testToken, bearerToken)

	// GIVEN a config with no bearer information
	// WHEN  the bearer token is loaded
	// THEN  the handler gets not bearer token data
	config = &rest.Config{}
	bearerToken, err = loadBearerToken(config, log)
	assert.NoError(t, err)
	assert.Empty(t, bearerToken)
}

// TestInitializeAuthenticator tests that the authenticator gets initialized if it has not previously
func TestInitializeAuthenticator(t *testing.T) {
	// unset these otherwise the test fails due to not being able to connect to the proxy
	os.Unsetenv("HTTPS_PROXY")
	os.Unsetenv("https_proxy")

	handler := Handler{
		URL:       testAPIServerURL,
		K8sClient: fake.NewClientBuilder().Build(),
		Log:       zap.S(),
	}

	server := testserver.FakeOIDCProviderServer(t)
	serverURL = server.URL

	getOIDCConfigFunc = fakeOIDCConfig
	defer func() { getOIDCConfigFunc = getOIDCConfiguration }()

	// GIVEN a request to initialize the authenticator
	// WHEN the authenticator has already been initialized
	// THEN no error is returned
	handler.AuthInited.Store(true)
	err := handler.initializeAuthenticator()
	assert.NoError(t, err)

	// GIVEN a request to initialize the authenticator
	// WHEN the authenticator has not been initialized
	// THEN no error is returned
	handler.AuthInited.Store(false)
	err = handler.initializeAuthenticator()
	assert.NoError(t, err)
}

// TestFindPathHandler tests that the correct handler is returned for a given request
func TestFindPathHandler(t *testing.T) {
	handler := Handler{
		URL:       testAPIServerURL,
		K8sClient: fake.NewClientBuilder().Build(),
		Log:       zap.S(),
	}

	// GIVEN a request
	// WHEN the url has the callback path
	// THEN the callback function is returned
	callbackURL, err := url.Parse(fmt.Sprintf("%s%s", testAPIServerURL, callbackPath))
	assert.NoError(t, err)
	req := &http.Request{URL: callbackURL}
	handlerfunc := handler.findPathHandler(req)
	handlerName := runtime.FuncForPC(reflect.ValueOf(handlerfunc).Pointer()).Name()
	authCallbackName := runtime.FuncForPC(reflect.ValueOf(handler.handleAuthCallback).Pointer()).Name()
	assert.Equal(t, handlerName, authCallbackName)

	// GIVEN a request
	// WHEN the url has the logout path
	// THEN the logout function is returned
	logoutURL, err := url.Parse(fmt.Sprintf("%s%s", testAPIServerURL, logoutPath))
	assert.NoError(t, err)
	req = &http.Request{URL: logoutURL}
	handlerfunc = handler.findPathHandler(req)
	handlerName = runtime.FuncForPC(reflect.ValueOf(handlerfunc).Pointer()).Name()
	logoutName := runtime.FuncForPC(reflect.ValueOf(handler.handleLogout).Pointer()).Name()
	assert.Equal(t, handlerName, logoutName)

	// GIVEN a request
	// WHEN the url has any path
	// THEN the api server function is returned
	apiReqURL, err := url.Parse(testAPIServerURL)
	assert.NoError(t, err)
	req = &http.Request{URL: apiReqURL}
	handlerfunc = handler.findPathHandler(req)
	handlerName = runtime.FuncForPC(reflect.ValueOf(handlerfunc).Pointer()).Name()
	apiReqName := runtime.FuncForPC(reflect.ValueOf(handler.handleAPIRequest).Pointer()).Name()
	assert.Equal(t, handlerName, apiReqName)

}

// TestServeHTTP tests that the incoming HTTP requests can be properly handled and forwarded
// GIVEN an HTTP request
// WHEN the request is processed
// THEN no error is returned
func TestServeHTTP(t *testing.T) {
	handler := Handler{
		URL:       testAPIServerURL,
		K8sClient: fake.NewClientBuilder().Build(),
		Log:       zap.S(),
	}

	server := testserver.FakeOIDCProviderServer(t)
	serverURL = server.URL

	getOIDCConfigFunc = fakeOIDCConfig
	defer func() { getOIDCConfigFunc = getOIDCConfiguration }()

	// Sending an option request so the API Server request terminates early
	req := httptest.NewRequest(http.MethodOptions, serverURL, strings.NewReader(""))
	rw := httptest.NewRecorder()
	handler.ServeHTTP(rw, req)
}

func testConfig() (*rest.Config, error) {
	return &rest.Config{
		Host: "test-host",
		TLSClientConfig: rest.TLSClientConfig{
			CAFile: caCertFile,
		},
	}, nil
}

func fakeOIDCConfig() auth.OIDCConfiguration {
	return auth.OIDCConfiguration{
		ExternalURL: serverURL,
		ServiceURL:  serverURL,
	}
}

// TestHandleAuthCallback tests the handleAuthCallback handler
func TestHandleAuthCallback(t *testing.T) {
	// create a temporary file with a generated cookie encryption key
	filename, err := writeEncryptionKeyFile()
	assert.NoError(t, err)
	defer os.Remove(filename)
	prevEncryptionKeyFile := cookie.GetEncryptionKeyFile()
	defer cookie.SetEncryptionKeyFile(prevEncryptionKeyFile)
	cookie.SetEncryptionKeyFile(filename)

	handler := Handler{
		Authenticator: testauth.NewFakeAuthenticator(),
		URL:           testAPIServerURL,
		K8sClient:     fake.NewClientBuilder().Build(),
		Log:           zap.S(),
	}

	const stateValue = "test-state"
	const redirectURI = "/someplace-great"
	vzState := &cookie.VZState{State: stateValue, RedirectURI: redirectURI}

	tests := []struct {
		name                       string
		req                        *http.Request
		expectedResponseStatusCode int
		expectRedirect             bool
	}{
		// GIVEN the state query param value matches the state in the VZ cookie
		// WHEN the auth callback handler is called
		// THEN all validation passes and the HTTP response is a redirect
		{
			name:                       "state matches",
			req:                        createHTTPRequest(vzState, stateValue),
			expectedResponseStatusCode: http.StatusFound,
			expectRedirect:             true,
		},
		// GIVEN the state query param value does not match the state in the VZ cookie
		// WHEN the auth callback handler is called
		// THEN an unauthorized response is returned
		{
			name:                       "state does not match",
			req:                        createHTTPRequest(vzState, "bad-state"),
			expectedResponseStatusCode: http.StatusUnauthorized,
			expectRedirect:             false,
		},
		// GIVEN there is no VZ cookie in the request
		// WHEN the auth callback handler is called
		// THEN an unauthorized response is returned
		{
			name:                       "no cookie",
			req:                        createHTTPRequest(nil, stateValue),
			expectedResponseStatusCode: http.StatusUnauthorized,
			expectRedirect:             false,
		},
		// GIVEN there is no state query param
		// WHEN the auth callback handler is called
		// THEN an unauthorized response is returned
		{
			name:                       "no query param",
			req:                        createHTTPRequest(vzState, ""),
			expectedResponseStatusCode: http.StatusUnauthorized,
			expectRedirect:             false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rw := httptest.NewRecorder()
			handler.handleAuthCallback(rw, tt.req)
			assert.Equal(t, tt.expectedResponseStatusCode, rw.Result().StatusCode)

			loc := rw.Header().Get("Location")
			if tt.expectRedirect {
				assert.Equal(t, redirectURI, loc)
			} else {
				assert.Equal(t, "", loc)
			}
		})
	}
}

// createHTTPRequest creates an HTTP request for testing
func createHTTPRequest(vzState *cookie.VZState, queryParam string) *http.Request {
	url := "https://example.com/"
	if queryParam != "" {
		url += "?state=" + queryParam
	}
	req := httptest.NewRequest("", url, nil)
	if vzState != nil {
		cookie, err := cookie.CreateStateCookie(vzState)
		if err != nil {
			panic(err)
		}
		req.AddCookie(cookie)
	}
	return req
}

// writeEncryptionKeyFile creates a temporary file and writes an encryption key. The function returns the file name.
func writeEncryptionKeyFile() (string, error) {
	f, err := os.CreateTemp("", "")
	if err != nil {
		return "", err
	}
	f.Write([]byte("abcdefghijklmnopqrstuvwxyz1234567890"))
	f.Close()
	return f.Name(), nil
}
