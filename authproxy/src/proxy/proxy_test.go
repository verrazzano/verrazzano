// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package proxy

import (
	"fmt"
	"io"
	"k8s.io/apimachinery/pkg/runtime"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/hashicorp/go-retryablehttp"
	"github.com/stretchr/testify/assert"
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

// TestConfigureKubernetesAPIProxy tests the configuration of the API proxy
// GIVEN an Auth proxy object
// WHEN  the Kubernetes API proxy is configured
// THEN  the handler exists and there is no error
func TestConfigureKubernetesAPIProxy(t *testing.T) {
	c := fake.NewClientBuilder().Build()
	authproxy := InitializeProxy(8777, c)
	log := zap.S()

	getConfigFunc = testConfig
	defer func() { getConfigFunc = k8sutil.GetConfigFromController }()

	err := ConfigureKubernetesAPIProxy(authproxy, log)
	assert.NoError(t, err)
	assert.NotNil(t, authproxy.Handler)
}

// TestServeHTTP tests the proxy server forwarding requests
// GIVEN a request to the Auth proxy server
// WHEN  the request is formatted correctly
// THEN  the request is properly forwarded to the API server
func TestServeHTTP(t *testing.T) {
	ingressHost := "inghost.example.com"
	originVal := fmt.Sprintf("https://%s", ingressHost)
	tests := []struct {
		name             string
		reqMethod        string
		headers          map[string]string
		expectedStatus   int
		expectedRespHdrs map[string]string
	}{
		{"POST request with no added headers", http.MethodPost, map[string]string{}, http.StatusOK, map[string]string{contentTypeHeader: runtime.ContentTypeJSON}},
		{"GET request with Host header", http.MethodPost, map[string]string{"Host": ingressHost}, http.StatusOK, map[string]string{}},
		{"GET request with valid Origin and Host headers", http.MethodGet, map[string]string{"Host": ingressHost, "Origin": originVal}, http.StatusOK, map[string]string{"Access-Control-Allow-Origin": originVal, contentTypeHeader: runtime.ContentTypeJSON}},
		{"OPTIONS request with valid Origin and Host headers", http.MethodOptions, map[string]string{"Host": ingressHost, "Origin": originVal}, http.StatusOK, map[string]string{"Content-Length": "0", "Access-Control-Allow-Origin": originVal}},
		{"POST request with Host and invalid Origin header", http.MethodPost, map[string]string{"Host": ingressHost, "Origin": "https://notvalid"}, http.StatusForbidden, map[string]string{}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testBody := "test-body"

			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				assert.NotNil(t, r)
				assert.Equal(t, apiPath, r.URL.Path)
				body, err := io.ReadAll(r.Body)
				assert.NoError(t, err)
				assert.Equal(t, testBody, string(body))
				w.Header().Add(contentTypeHeader, runtime.ContentTypeJSON)
			}))
			defer server.Close()

			handler := Handler{
				URL:    server.URL,
				Client: retryablehttp.NewClient(),
				Log:    zap.S(),
			}

			url := fmt.Sprintf("%s/clusters/local%s", testAPIServerURL, apiPath)
			r := httptest.NewRequest(tt.reqMethod, url, strings.NewReader(testBody))

			for name, val := range tt.headers {
				r.Header.Set(name, val)
			}

			w := httptest.NewRecorder()
			handler.ServeHTTP(w, r)
			assert.Equal(t, tt.expectedStatus, w.Code)

			for name, val := range tt.expectedRespHdrs {
				assert.Equal(t, val, w.Header().Get(name))
			}

		})
	}
}

// TestReformatAPIRequest tests the reformatting of the request to be sent to the API server

func TestReformatAPIRequest(t *testing.T) {
	handler := Handler{
		URL:    testAPIServerURL,
		Client: retryablehttp.NewClient(),
		Log:    zap.S(),
	}

	tests := []struct {
		name        string
		url         string
		expectedURL string
	}{
		// GIVEN a request to the Auth proxy server
		// WHEN  the request is formatted correctly
		// THEN  the request is properly formatted to be sent to the API server
		{
			name:        "test cluster path",
			url:         fmt.Sprintf("https://authproxy.io/clusters/local%s", apiPath),
			expectedURL: fmt.Sprintf("%s%s", handler.URL, apiPath),
		},
		// GIVEN a request to the Auth proxy server
		// WHEN  the request is malformed
		// THEN  a malformed request is returned
		{
			name:        "test malformed request",
			url:         "malformed-request1234",
			expectedURL: fmt.Sprintf("%s/%s", handler.URL, "malformed-request1234"),
		},
		// GIVEN a request to the Auth proxy server
		// WHEN  the request has a query param
		// THEN  the query param is added to the outgoing request
		{
			name:        "test query param",
			url:         fmt.Sprintf("https://authproxy.io/clusters/local%s?watch=1", apiPath),
			expectedURL: fmt.Sprintf("%s%s?watch=1", handler.URL, apiPath),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, err := http.NewRequest(http.MethodGet, tt.url, strings.NewReader(""))
			assert.NoError(t, err)

			formattedReq, err := handler.reformatAPIRequest(req)
			assert.NoError(t, err)
			assert.Equal(t, tt.expectedURL, formattedReq.URL.String())
		})
	}
}

// TestValidateRequest tests the request validation for the Auth Proxy
func TestValidateRequest(t *testing.T) {
	// GIVEN a request without the cluster path
	// WHEN  the request is validated
	// THEN  an error is returned
	url := fmt.Sprintf("%s/%s", testAPIServerURL, apiPath)
	req, err := http.NewRequest(http.MethodGet, url, strings.NewReader(""))
	assert.NoError(t, err)
	err = validateRequest(req)
	assert.Error(t, err)

	// GIVEN a request with the cluster path
	// WHEN  the request is validated
	// THEN  no error is returned
	url = fmt.Sprintf("%s/clusters/local%s", testAPIServerURL, apiPath)
	req, err = http.NewRequest(http.MethodGet, url, strings.NewReader(""))
	assert.NoError(t, err)
	err = validateRequest(req)
	assert.NoError(t, err)
}

// TestObfuscateTestData tests that request authorization headers get scrubbed
// GIVEN a request with an authorization header
// WHEN  the request is scrubbed
// THEN  the header contains a different value
func TestObfuscateTestData(t *testing.T) {
	req, err := http.NewRequest(http.MethodGet, testAPIServerURL, strings.NewReader(""))
	assert.NoError(t, err)

	authKey := "Authorization"
	basicAuth := "Basic username:pass"
	tokenAuth := "Bearer test-token"
	req.Header[authKey] = []string{basicAuth, tokenAuth}

	obfReq := obfuscateRequestData(req)
	assert.NotEqual(t, basicAuth, obfReq.Header[authKey][0])
	assert.NotEqual(t, tokenAuth, obfReq.Header[authKey][1])
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

func testConfig() (*rest.Config, error) {
	return &rest.Config{
		Host: "test-host",
		TLSClientConfig: rest.TLSClientConfig{
			CAFile: caCertFile,
		},
	}, nil
}
