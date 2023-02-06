// Copyright (c) 2022, 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package rancherutil

import (
	"bytes"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	pkgconst "github.com/verrazzano/verrazzano/pkg/constants"
	"github.com/verrazzano/verrazzano/pkg/log/vzlog"
	"github.com/verrazzano/verrazzano/pkg/test/mockmatchers"
	"github.com/verrazzano/verrazzano/platform-operator/mocks"
	corev1 "k8s.io/api/core/v1"
	networkv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

var (
	loginURLParts = strings.Split(loginPath, "?")
	loginURIPath  = loginURLParts[0]
	testToken     = "test"
)

// TestCreateRancherRequest tests the creation of a Rancher request sender to make sure that
// HTTP requests are properly constructed and sent to Rancher
func TestCreateRancherRequest(t *testing.T) {
	cli := createTestObjects()
	log := vzlog.DefaultLogger()

	testPath := "test/path"
	testBody := "test-body"

	savedRancherHTTPClient := RancherHTTPClient
	defer func() {
		RancherHTTPClient = savedRancherHTTPClient
	}()

	savedRetry := DefaultRetry
	defer func() {
		DefaultRetry = savedRetry
	}()
	DefaultRetry = wait.Backoff{
		Steps:    1,
		Duration: 1 * time.Millisecond,
		Factor:   1.0,
		Jitter:   0.1,
	}

	mocker := gomock.NewController(t)
	httpMock := mocks.NewMockRequestSender(mocker)
	httpMock = expectHTTPRequests(httpMock, testPath, testBody)
	RancherHTTPClient = httpMock

	// Test with the Verrazzano cluster user
	rc, err := NewVerrazzanoClusterRancherConfig(cli, log)
	assert.NoError(t, err)

	response, body, err := SendRequest(http.MethodGet, testPath, map[string]string{}, "", rc, log)
	assert.NoError(t, err)
	assert.Equal(t, testBody, body)
	assert.Equal(t, http.StatusOK, response.StatusCode)

	// Test with the admin user
	rc, err = NewAdminRancherConfig(cli, log)
	assert.NoError(t, err)

	response, body, err = SendRequest(http.MethodGet, testPath, map[string]string{}, "", rc, log)
	assert.NoError(t, err)
	assert.Equal(t, testBody, body)
	assert.Equal(t, http.StatusOK, response.StatusCode)

	response, _, err = SendRequest(http.MethodPost, tokensPath, map[string]string{}, "", rc, log)
	assert.NoError(t, err)

	response, _, err = SendRequest(http.MethodGet, tokensPath+testToken, map[string]string{}, "", rc, log)
	assert.NoError(t, err)
}

func createTestObjects() client.WithWatch {
	return fake.NewClientBuilder().WithRuntimeObjects(
		&networkv1.Ingress{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: rancherNamespace,
				Name:      rancherIngressName,
			},
			Spec: networkv1.IngressSpec{
				Rules: []networkv1.IngressRule{
					{
						Host: "test-rancher.com",
					},
				},
			},
		},
		&corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: pkgconst.VerrazzanoMultiClusterNamespace,
				Name:      pkgconst.VerrazzanoClusterRancherName,
			},
			Data: map[string][]byte{
				"password": []byte(""),
			},
		},
		&corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: rancherNamespace,
				Name:      rancherAdminSecret,
			},
			Data: map[string][]byte{
				"password": []byte(""),
			},
		}).Build()
}

func expectHTTPRequests(httpMock *mocks.MockRequestSender, testPath, testBody string) *mocks.MockRequestSender {
	httpMock.EXPECT().
		Do(gomock.Not(gomock.Nil()), mockmatchers.MatchesURI(tokensPath+testToken)).
		DoAndReturn(func(httpClient *http.Client, req *http.Request) (*http.Response, error) {
			var resp *http.Response
			r := io.NopCloser(bytes.NewReader([]byte(testBody)))
			resp = &http.Response{
				StatusCode: http.StatusOK,
				Body:       r,
			}
			return resp, nil
		})
	httpMock.EXPECT().
		Do(gomock.Not(gomock.Nil()), mockmatchers.MatchesURI(tokensPath)).
		DoAndReturn(func(httpClient *http.Client, req *http.Request) (*http.Response, error) {
			var resp *http.Response
			r := io.NopCloser(bytes.NewReader([]byte(testBody)))
			resp = &http.Response{
				StatusCode: http.StatusCreated,
				Body:       r,
			}
			return resp, nil
		}).Times(1)
	httpMock.EXPECT().
		Do(gomock.Not(gomock.Nil()), mockmatchers.MatchesURI(testPath)).
		DoAndReturn(func(httpClient *http.Client, req *http.Request) (*http.Response, error) {
			var resp *http.Response
			r := io.NopCloser(bytes.NewReader([]byte(testBody)))
			resp = &http.Response{
				StatusCode: http.StatusOK,
				Body:       r,
			}
			return resp, nil
		}).Times(2)
	httpMock.EXPECT().
		Do(gomock.Not(gomock.Nil()), mockmatchers.MatchesURI(loginURIPath)).
		DoAndReturn(func(httpClient *http.Client, req *http.Request) (*http.Response, error) {
			r := io.NopCloser(bytes.NewReader([]byte(`{"token":"unit-test-token"}`)))
			resp := &http.Response{
				StatusCode: http.StatusCreated,
				Body:       r,
				Request:    &http.Request{Method: http.MethodPost},
			}
			return resp, nil
		}).Times(2)
	return httpMock
}
