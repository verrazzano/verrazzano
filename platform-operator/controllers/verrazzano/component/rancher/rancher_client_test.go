package rancher

import (
	"errors"
	"fmt"
	"github.com/stretchr/testify/assert"
	"io"
	"net/http"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"strings"
	"testing"
)

const (
	dummyToken = "token"
)

func tokenResponse(h *http.Client, request *http.Request) (*http.Response, error) {
	return &http.Response{
		StatusCode: 201,
		Body:       io.NopCloser(strings.NewReader(fmt.Sprintf(`{"token":"token"}`))),
	}, nil
}

func unauthorizedResponse(h *http.Client, request *http.Request) (*http.Response, error) {
	return &http.Response{
		StatusCode: 401,
		Body:       io.NopCloser(strings.NewReader(fmt.Sprintf(`{"error":"unauthorized"}`))),
	}, nil
}

func errorResponse(h *http.Client, request *http.Request) (*http.Response, error) {
	return nil, errors.New("boom")
}

func okResponse(h *http.Client, request *http.Request) (*http.Response, error) {
	return &http.Response{
		StatusCode: 200,
		Body:       io.NopCloser(strings.NewReader(fmt.Sprintf("blahblah"))),
	}, nil
}

func testClient(doer func(*http.Client, *http.Request) (*http.Response, error)) *RESTClient {
	return &RESTClient{
		client:   &http.Client{},
		do:       doer,
		hostname: "hostname",
		password: "password",
	}
}

func TestNewClient(t *testing.T) {
	s := createRootCASecret()
	c := fake.NewFakeClientWithScheme(getScheme(), &s)
	rest, err := NewClient(c, "hostname", "password")
	assert.Nil(t, err)
	assert.NotNil(t, rest)
}

// TestGetAdminSecret tests retrieving the Rancher admin secret value
// GIVEN a client
//  WHEN GetAdminSecret is called with a client that has an admin secret in cluster
//  THEN GetAdminSecret should return the value of the rancher admin secret
func TestGetAdminSecret(t *testing.T) {
	secret := createAdminSecret()
	var tests = []struct {
		testName string
		c        client.Client
		isError  bool
	}{
		{
			"should retrieve the secret when it exists",
			fake.NewFakeClientWithScheme(getScheme(), &secret),
			false,
		},
		{
			"should throw an error when the secret is not present",
			fake.NewFakeClientWithScheme(getScheme()),
			true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.testName, func(t *testing.T) {
			_, err := GetAdminSecret(tt.c)
			if tt.isError {
				assert.NotNil(t, err)
			} else {
				assert.Nil(t, err)
			}
		})
	}
}

// TestGetRancherTLSRootCA tests retrieving the Rancher CA certificate
// GIVEN a client
//  WHEN GetRootCA is called with a client that has an CA secret in cluster
//  THEN GetRootCA should return the value of the rancher CA secret
func TestGetRancherTLSRootCA(t *testing.T) {
	secret := createRootCASecret()
	var tests = []struct {
		testName string
		c        client.Client
		notFound bool
	}{
		{
			"should retrieve the secret when it exists",
			fake.NewFakeClientWithScheme(getScheme(), &secret),
			false,
		},
		{
			"should throw an error when the secret is not present",
			fake.NewFakeClientWithScheme(getScheme()),
			true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.testName, func(t *testing.T) {
			cert, err := GetRootCA(tt.c)
			assert.Nil(t, err)
			if tt.notFound {
				assert.Nil(t, cert)
			} else {
				assert.NotNil(t, cert)
			}
		})
	}
}

func TestHttpClient(t *testing.T) {
	secret := createRootCASecret()

	var tests = []struct {
		testName string
		c        client.Client
		isErr    bool
	}{
		{
			"should get an HTTP Client when CA Secret exists",
			fake.NewFakeClientWithScheme(getScheme(), &secret),
			false,
		},
		{
			"should fail to create an HTTP Client when CA Secret is not present",
			fake.NewFakeClientWithScheme(getScheme()),
			true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.testName, func(t *testing.T) {
			hc, err := HttpClient(tt.c)
			if tt.isErr {
				assert.NotNil(t, err)
				assert.Nil(t, hc)
			} else {
				assert.Nil(t, err)
				assert.NotNil(t, hc)
			}
		})
	}
}

func TestSetLoginToken(t *testing.T) {
	var tests = []struct {
		testName string
		rest     *RESTClient
		isError  bool
	}{
		{
			"should be able to set the login token",
			testClient(tokenResponse),
			false,
		},
		{
			"should fail to set the login token when not present in the response",
			testClient(unauthorizedResponse),
			true,
		},
		{
			"should fail to set the login token when the request fails",
			testClient(errorResponse),
			true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.testName, func(t *testing.T) {
			assert.Equal(t, "", tt.rest.GetLoginToken())
			err := tt.rest.SetLoginToken()
			if tt.isError {
				assert.NotNil(t, err)
			} else {
				assert.Nil(t, err)
				token := tt.rest.GetLoginToken()
				assert.Equal(t, dummyToken, token)
			}
		})
	}
}

func TestSetAccessToken(t *testing.T) {
	var tests = []struct {
		testName   string
		loginToken string
		rest       *RESTClient
		isError    bool
	}{
		{
			"should be able to set the access token",
			dummyToken,
			testClient(tokenResponse),
			false,
		},
		{
			"should be able to set the access token when no login token is present",
			"",
			testClient(tokenResponse),
			false,
		},
		{
			"should fail to set the access token when not present in the response",
			dummyToken,
			testClient(unauthorizedResponse),
			true,
		},
		{
			"should fail to set the access token when the request fails",
			dummyToken,
			testClient(errorResponse),
			true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.testName, func(t *testing.T) {
			assert.Equal(t, "", tt.rest.GetAccessToken())
			err := tt.rest.SetAccessToken()
			if tt.isError {
				assert.NotNil(t, err)
			} else {
				assert.Nil(t, err)
				token := tt.rest.GetAccessToken()
				assert.Equal(t, dummyToken, token)
			}
		})
	}
}

func TestPutServerURL(t *testing.T) {
	var tests = []struct {
		testName string
		rest     *RESTClient
		isErr    bool
	}{
		{
			"should be able to put the server url",
			testClient(okResponse),
			false,
		},
		{
			"should fail to put the server url if the request fails",
			testClient(errorResponse),
			true,
		},
		{
			"should fail to put the server URL if the status is not expected",
			testClient(unauthorizedResponse),
			true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.testName, func(t *testing.T) {
			tt.rest.accessToken = dummyToken
			err := tt.rest.PutServerURL()
			if tt.isErr {
				assert.NotNil(t, err)
			} else {
				assert.Nil(t, err)
			}
		})
	}
}
