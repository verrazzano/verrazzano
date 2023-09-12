// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package apiserver

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"github.com/verrazzano/verrazzano/cluster-operator/apis/clusters/v1alpha1"
	"github.com/verrazzano/verrazzano/pkg/constants"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8scheme "k8s.io/client-go/kubernetes/scheme"
	"net/http"
	"net/http/httptest"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"strings"
	"testing"

	"github.com/hashicorp/go-retryablehttp"
	"github.com/stretchr/testify/assert"
	"github.com/verrazzano/verrazzano/authproxy/internal/testutil/testauth"
	"github.com/verrazzano/verrazzano/authproxy/src/auth"
	"go.uber.org/zap"
)

const (
	apiPath          = "/api/v1/pods"
	testAPIServerURL = "https://api-server.io"
)

// TestForwardAPIRequest tests that API requests are properly formatted and sent to the API server
func TestForwardAPIRequest(t *testing.T) {
	tests := []struct {
		name             string
		reqMethod        string
		reqHeaders       map[string]string
		expectedStatus   int
		expectedRespHdrs map[string]string
		clusterPath      string
		unauthenticated  bool
	}{
		// GIVEN an options request
		// WHEN  the request is received
		// THEN  the content length header is set
		{
			name:           "options request",
			reqMethod:      http.MethodOptions,
			expectedStatus: http.StatusOK,
			expectedRespHdrs: map[string]string{
				"Content-Length": "0",
			},
			clusterPath: localClusterPrefix,
		},
		// GIVEN a processed request
		// WHEN  the request is received
		// THEN  an OK response is returned
		{
			name:            "processed request",
			reqMethod:       http.MethodGet,
			expectedStatus:  http.StatusOK,
			clusterPath:     localClusterPrefix,
			unauthenticated: true,
		},
		// GIVEN a get request
		// WHEN  the request is authorized
		// THEN  the status returned is okay
		{
			name:           "get request",
			reqMethod:      http.MethodGet,
			expectedStatus: http.StatusOK,
			clusterPath:    localClusterPrefix,
		},
		// GIVEN a post request with headers
		// WHEN  the request is forwarded
		// THEN  the headers are properly added to the request
		{
			name:      "post request with headers",
			reqMethod: http.MethodPost,
			reqHeaders: map[string]string{
				"test1": "header1",
				"test2": "header2",
			},
			expectedStatus: http.StatusOK,
			clusterPath:    localClusterPrefix,
		},
		// GIVEN a managed cluster request
		// WHEN  the request is forwarded
		// THEN  the managed cluster processing is done correctly
		{
			name:      "managed cluster request",
			reqMethod: http.MethodDelete,
			reqHeaders: map[string]string{
				"test1": "header1",
				"test2": "header2",
			},
			expectedStatus: http.StatusOK,
			clusterPath:    "/clusters/managed1",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, tt.reqMethod, r.Method)
				for k, v := range tt.reqHeaders {
					assert.Contains(t, r.Header.Get(k), v)
				}
			}))
			defer server.Close()

			scheme := k8scheme.Scheme
			err := v1alpha1.AddToScheme(scheme)
			assert.NoError(t, err)
			k8sCli := fake.NewClientBuilder().WithScheme(scheme).WithObjects(
				&v1alpha1.VerrazzanoManagedCluster{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "managed1",
						Namespace: constants.VerrazzanoMultiClusterNamespace,
					},
					Status: v1alpha1.VerrazzanoManagedClusterStatus{
						APIUrl: server.URL,
					},
				},
			).Build()

			url := fmt.Sprintf("%s%s%s", testAPIServerURL, localClusterPrefix, apiPath)
			w := httptest.NewRecorder()
			cli := retryablehttp.NewClient()
			request := httptest.NewRequest(tt.reqMethod, url, strings.NewReader(""))
			for k, v := range tt.reqHeaders {
				request.Header.Set(k, v)
			}
			setEmptyToken(request)
			authenticator := testauth.NewFakeAuthenticator()

			apiRequest := APIRequest{
				Request:       request,
				RW:            w,
				Client:        cli,
				Authenticator: authenticator,
				APIServerURL:  server.URL,
				K8sClient:     k8sCli,
				Log:           zap.S(),
			}

			if tt.unauthenticated {
				authenticator.SetRequestFunc(testauth.AuthenticateFalse)
				defer authenticator.SetRequestFunc(testauth.AuthenticateTrue)
			}

			apiRequest.ForwardAPIRequest()
			assert.Equal(t, tt.expectedStatus, w.Code)

			for k, v := range tt.expectedRespHdrs {
				assert.Equal(t, v, w.Header().Get(k))
			}

		})
	}
}

// TestReformatAPIRequest tests the reformatting of the request to be sent to the API server

func TestReformatAPIRequest(t *testing.T) {
	apiRequest := APIRequest{
		APIServerURL: testAPIServerURL,
		Client:       retryablehttp.NewClient(),
		Log:          zap.S(),
	}

	tests := []struct {
		name        string
		url         string
		expectedURL string
		expectError bool
	}{
		// GIVEN a request to the Auth proxy server
		// WHEN  the request is formatted correctly
		// THEN  the request is properly formatted to be sent to the API server
		{
			name:        "test cluster path",
			url:         fmt.Sprintf("https://authproxy.io/clusters/local%s", apiPath),
			expectedURL: fmt.Sprintf("%s%s", apiRequest.APIServerURL, apiPath),
		},
		// GIVEN a request to the Auth proxy server
		// WHEN  the request is malformed
		// THEN  a processing error is returned
		{
			name:        "test malformed request",
			url:         "https://authproxy.io/malformedrequest1234",
			expectError: true,
		},
		// GIVEN a request to the Auth proxy server
		// WHEN  the request has a query param
		// THEN  the query param is added to the outgoing request
		{
			name:        "test query param",
			url:         fmt.Sprintf("https://authproxy.io/clusters/local%s?watch=1", apiPath),
			expectedURL: fmt.Sprintf("%s%s?watch=1", apiRequest.APIServerURL, apiPath),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, tt.url, strings.NewReader(""))
			setEmptyToken(req)

			formattedReq, err := apiRequest.reformatAPIRequest(req)
			if tt.expectError {
				assert.Error(t, err)
				return
			}

			assert.NoError(t, err)
			assert.NotNil(t, formattedReq.URL)
			assert.Equal(t, tt.expectedURL, formattedReq.URL.String())
		})
	}
}

// TestSetImpersonationHeaders tests that the impersonation headers can be set for an API server request
func TestSetImpersonationHeaders(t *testing.T) {
	// GIVEN a request with a bad JWT token
	// WHEN  the request is evaluated
	// THEN  an error is returned
	req := &http.Request{
		Header: map[string][]string{
			"Authorization": {
				"bad-jwt-token",
			},
		},
	}
	err := setImpersonationHeaders(req)
	assert.Error(t, err)

	// GIVEN a request with a valid JWT token
	// WHEN  the request is evaluated
	// THEN  the request has the impersonation headers set
	testUser := "test-user"
	testGroups := []string{
		"group1",
		"group2",
	}
	headers := auth.ImpersonationHeaders{
		User:   testUser,
		Groups: testGroups,
	}
	tokenJSON, err := json.Marshal(headers)
	assert.NoError(t, err)

	tokenBase64 := base64.RawURLEncoding.EncodeToString(tokenJSON)
	jwtToken := fmt.Sprintf("test.%s.test", tokenBase64)

	req = &http.Request{
		Header: map[string][]string{
			"Authorization": {
				"Bearer " + jwtToken,
			},
		},
	}
	err = setImpersonationHeaders(req)
	assert.NoError(t, err)
	assert.Len(t, req.Header.Values(userImpersontaionHeader), 1)
	assert.Equal(t, testUser, req.Header.Get(userImpersontaionHeader))
	assert.ElementsMatch(t, testGroups, req.Header.Values(groupImpersonationHeader))
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

func setEmptyToken(req *http.Request) {
	testToken := fmt.Sprintf("info.%s.info", base64.RawURLEncoding.EncodeToString([]byte("{}")))
	if req.Header == nil {
		req.Header = make(map[string][]string)
	}

	req.Header.Set("Authorization", "Bearer "+testToken)
}
