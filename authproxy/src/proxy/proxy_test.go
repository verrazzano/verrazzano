// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package proxy

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/verrazzano/verrazzano/pkg/k8sutil"
	"go.uber.org/zap"
	"k8s.io/client-go/rest"
)

const apiPath = "/api/v1/pods"

// TestConfigureKubernetesAPIProxy tests the configuration of the API proxy
// GIVEN an Auth proxy object
// WHEN  the Kubernetes API proxy is configured
// THEN  the handler exists and there is no error
func TestConfigureKubernetesAPIProxy(t *testing.T) {
	authproxy := InitializeProxy()
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
	testBody := "test-body"

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.NotNil(t, r)
		assert.Equal(t, apiPath, r.URL.Path)
		body, err := io.ReadAll(r.Body)
		assert.NoError(t, err)
		assert.Equal(t, testBody, string(body))
	}))
	defer server.Close()

	handler := Handler{
		URL:    server.URL,
		Client: &http.Client{},
		Log:    zap.S(),
	}

	url := fmt.Sprintf("https://authproxy.io/clusters/local%s", apiPath)
	r := httptest.NewRequest(http.MethodPost, url, strings.NewReader(testBody))
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, r)
}

// TestReformatAPIRequest tests the reformatting of the request to be sent to the API server
// GIVEN a request to the Auth proxy server
// WHEN  the request is formatted correctly
// THEN  the request is properly formatted to be sent to the API server
func TestReformatAPIRequest(t *testing.T) {
	handler := Handler{
		URL:    "https://api-server.io",
		Client: &http.Client{},
		Log:    zap.S(),
	}
	url := fmt.Sprintf("https://authproxy.io/clusters/local%s", apiPath)
	req, err := http.NewRequest(http.MethodGet, url, strings.NewReader(""))
	assert.NoError(t, err)

	formattedReq, err := handler.reformatAPIRequest(req)
	assert.NoError(t, err)
	expectedURL := fmt.Sprintf("%s%s", handler.URL, apiPath)
	assert.Equal(t, expectedURL, formattedReq.URL.String())
}

// TestValidateRequest tests the request validation for the Auth Proxy
func TestValidateRequest(t *testing.T) {
	// GIVEN a request without the cluster path
	// WHEN  the request is validated
	// THEN  an error is returned
	url := fmt.Sprintf("https://authproxy.io/%s", apiPath)
	req, err := http.NewRequest(http.MethodGet, url, strings.NewReader(""))
	assert.NoError(t, err)
	err = validateRequest(req)
	assert.Error(t, err)

	// GIVEN a request with the cluster path
	// WHEN  the request is validated
	// THEN  no error is returned
	url = fmt.Sprintf("https://authproxy.io/clusters/local%s", apiPath)
	req, err = http.NewRequest(http.MethodGet, url, strings.NewReader(""))
	assert.NoError(t, err)
	err = validateRequest(req)
	assert.NoError(t, err)
}

func testConfig() (*rest.Config, error) {
	return &rest.Config{
		Host: "test-host",
	}, nil
}
