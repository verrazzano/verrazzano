// Copyright (c) 2021, 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package common

import (
	"errors"
	"github.com/golang/mock/gomock"
	"github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	"github.com/verrazzano/verrazzano/platform-operator/mocks"
	"io"
	v12 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestCertBuilder verifies downloading certs from the web
// GIVEN a cert URI
//
//	WHEN appendCertWithHTTP is called
//	THEN appendCertWithHTTP should download the cert if it exists
func TestCertBuilder(t *testing.T) {
	var tests = []struct {
		testName string
		httpDo   HTTPDoSig
		isErr    bool
	}{
		{
			"should be able to download a cert",
			func(hc *http.Client, req *http.Request) (*http.Response, error) {
				return &http.Response{
					Body:       io.NopCloser(strings.NewReader("cert")),
					StatusCode: http.StatusOK,
				}, nil
			},
			false,
		},
		{
			"should fail to download a cert when the upstream server is down",
			func(hc *http.Client, req *http.Request) (*http.Response, error) {
				return &http.Response{
					Body:       io.NopCloser(strings.NewReader("cert")),
					StatusCode: http.StatusBadGateway,
				}, nil
			},
			true,
		},
		{
			"should fail to download a cert when the request fails",
			func(hc *http.Client, req *http.Request) (*http.Response, error) {
				return nil, errors.New("boom")
			},
			true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.testName, func(t *testing.T) {
			c := certBuilder{hc: &http.Client{}}
			HTTPDo = tt.httpDo
			err := c.appendCertWithHTTP(rootX1PEM)
			if tt.isErr {
				assert.NotNil(t, err)
			} else {
				assert.Nil(t, err)
			}
		})
	}
}

// TestBuildLetsEncryptChain verifies building the LetsEncrypt staging certificate chain
// GIVEN a certBuilder
//
//	WHEN buildLetsEncryptStagingChain is called
//	THEN buildLetsEncryptStagingChain should build the cert chain for LetsEncrypt
func TestBuildLetsEncryptChain(t *testing.T) {
	HTTPDo = func(hc *http.Client, req *http.Request) (*http.Response, error) {
		return &http.Response{
			Body:       io.NopCloser(strings.NewReader("cert")),
			StatusCode: http.StatusOK,
		}, nil
	}
	builder := &certBuilder{hc: &http.Client{}}
	err := builder.buildLetsEncryptStagingChain()
	assert.Nil(t, err)
	assert.Equal(t, "certcertcert", string(builder.cert))
}

// TestProcessAdditionalCertificates verifies building the LetsEncrypt staging certificate chain
// GIVEN a logger, client and Verrazzano CR with valid Certmanager spec
//
//	WHEN ProcessAdditionalCertificates is called
//	THEN ProcessAdditionalCertificates should process the additional certs successfully with no error
func TestProcessAdditionalCertificates(t *testing.T) {
	mock := gomock.NewController(t)
	client := mocks.NewMockClient(mock)
	a := true
	ctx := spi.NewFakeContext(client, &v1alpha1.Verrazzano{ObjectMeta: v12.ObjectMeta{Namespace: "foo"}}, nil, false)
	vz := v1alpha1.Verrazzano{
		Spec: v1alpha1.VerrazzanoSpec{
			Components: v1alpha1.ComponentSpec{
				CertManager: &v1alpha1.CertManagerComponent{
					Enabled:     &a,
					Certificate: v1alpha1.Certificate{Acme: v1alpha1.Acme{Environment: "dev"}},
				},
			},
		},
	}
	client.EXPECT().Get(gomock.Any(), gomock.Any(), gomock.Not(gomock.Nil())).Return(nil).AnyTimes()
	client.EXPECT().Update(gomock.Any(), gomock.Any()).Return(nil).AnyTimes()

	err := ProcessAdditionalCertificates(ctx.Log(), client, &vz)
	assert.Nil(t, err)

}

// TestProcessAdditionalCertificatesFailure verifies building the LetsEncrypt staging certificate chain
// GIVEN a logger, client and vz instance with no Certmanager spec defined
//
//	WHEN ProcessAdditionalCertificates is called
//	THEN ProcessAdditionalCertificates function call fails and returns error
func TestProcessAdditionalCertificatesFailure(t *testing.T) {
	mock := gomock.NewController(t)
	client := mocks.NewMockClient(mock)
	ctx := spi.NewFakeContext(client, &v1alpha1.Verrazzano{ObjectMeta: v12.ObjectMeta{Namespace: "foo"}}, nil, false)
	vz := v1alpha1.Verrazzano{}
	client.EXPECT().Get(gomock.Any(), gomock.Any(), gomock.Not(gomock.Nil())).Return(nil).AnyTimes()
	client.EXPECT().Delete(gomock.Any(), gomock.Any(), gomock.Not(gomock.Nil())).Return(nil).AnyTimes()
	client.EXPECT().Update(gomock.Any(), gomock.Any()).Return(nil).AnyTimes()

	err := ProcessAdditionalCertificates(ctx.Log(), client, &vz)
	assert.Nil(t, err)

}
