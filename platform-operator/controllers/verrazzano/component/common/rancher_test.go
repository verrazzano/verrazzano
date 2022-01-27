// Copyright (c) 2021, 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package common

import (
	"errors"
	"github.com/stretchr/testify/assert"
	"io"
	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	networking "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"net/http"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"strings"
	"testing"
)

const (
	dummyToken = "token"
)

func getScheme() *runtime.Scheme {
	scheme := runtime.NewScheme()
	_ = networking.AddToScheme(scheme)
	_ = appsv1.AddToScheme(scheme)
	_ = v1.AddToScheme(scheme)
	return scheme
}

func createRootCASecret() v1.Secret {
	return v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: CattleSystem,
			Name:      RancherIngressCAName,
		},
		Data: map[string][]byte{
			RancherCACert: []byte("blahblah"),
		},
	}
}

func createAdminSecret() v1.Secret {
	return v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: CattleSystem,
			Name:      RancherAdminSecret,
		},
		Data: map[string][]byte{
			"password": []byte("foobar"),
		},
	}
}

func tokenResponse(h *http.Client, request *http.Request) (*http.Response, error) {
	return &http.Response{
		StatusCode: 201,
		Body:       io.NopCloser(strings.NewReader(`{"token":"token"}`)),
	}, nil
}

func unauthorizedResponse(h *http.Client, request *http.Request) (*http.Response, error) {
	return &http.Response{
		StatusCode: 401,
		Body:       io.NopCloser(strings.NewReader(`{"error":"unauthorized"}`)),
	}, nil
}

func errorResponse(h *http.Client, request *http.Request) (*http.Response, error) {
	return nil, errors.New("boom")
}

func okResponse(h *http.Client, request *http.Request) (*http.Response, error) {
	return &http.Response{
		StatusCode: 200,
		Body:       io.NopCloser(strings.NewReader("blahblah")),
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

// TestNewClient tests creating a new Rancher REST client
// GIVEN the root CA secret exists on the cluster
//  WHEN NewClient is called
//  THEN NewClient should return a new client
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
//  THEN GetAdminSecret should return the value of the Rancher admin secret
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
//  THEN GetRootCA should return the value of the Rancher CA secret
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
			hc, err := HTTPClient(tt.c, "foobar.com")
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

// TestSetLoginToken tests setting the Rancher client login token
func TestSetLoginToken(t *testing.T) {
	var tests = []struct {
		testName string
		rest     *RESTClient
		isError  bool
	}{
		{
			// GIVEN Rancher is up and running and credentials are correct
			//  WHEN SetLoginToken is called
			//  THEN SetLoginToken should set the login token in the Rancher client
			"should be able to set the login token",
			testClient(tokenResponse),
			false,
		},
		{
			// GIVEN Rancher credentials are incorrect
			//  WHEN SetLoginToken is called
			//  THEN SetLoginToken should fail with an error
			"should fail to set the login token when not present in the response",
			testClient(unauthorizedResponse),
			true,
		},
		{
			// GIVEN Rancher is not ready
			//  WHEN SetLoginToken is called
			//  THEN SetLoginToken should return an error
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

// TestSetAccessToken tests setting the Rancher client access token
func TestSetAccessToken(t *testing.T) {
	var tests = []struct {
		testName   string
		loginToken string
		rest       *RESTClient
		isError    bool
	}{
		{
			// GIVEN Rancher is running and the credentials are correct
			//  WHEN SetAccessToken is called
			//  THEN SetAccessToken should set the Rancher client access token
			"should be able to set the access token",
			dummyToken,
			testClient(tokenResponse),
			false,
		},
		{
			// GIVEN Rancher is running and the credentials are correct, and hte login token is not present
			//  WHEN SetAccessToken is called
			//  THEN SetAccessToken should set both the Rancher client access token and login token
			"should be able to set the access token when no login token is present",
			"",
			testClient(tokenResponse),
			false,
		},
		{
			// GIVEN Rancher is running and the credentials are invalid
			//  WHEN SetAccessToken is called
			//  THEN SetAccessToken should fail to set the access token
			"should fail to set the access token when not present in the response",
			dummyToken,
			testClient(unauthorizedResponse),
			true,
		},
		{
			// GIVEN Rancher is not running
			//  WHEN SetAccessToken is called
			//  THEN SetAccessToken should fail to set the access token
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

// TestPutServerURL verifies how the Rancher client updates the server URL
func TestPutServerURL(t *testing.T) {
	var tests = []struct {
		testName string
		rest     *RESTClient
		isErr    bool
	}{
		{
			// GIVEN Rancher is running
			//  WHEN PutServerURL is called
			//  THEN PutServerURL should set the Rancher server URL
			"should be able to put the server url",
			testClient(okResponse),
			false,
		},
		{
			// GIVEN Rancher is not running
			//  WHEN PutServerURL is called
			//  THEN PutServerURL should fail to set the Rancher server URL
			"should fail to put the server url if the request fails",
			testClient(errorResponse),
			true,
		},
		{
			// GIVEN The access token is invalid
			//  WHEN PutServerURL is called
			//  THEN PutServerURL should fail to set the Rancher server URL
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
