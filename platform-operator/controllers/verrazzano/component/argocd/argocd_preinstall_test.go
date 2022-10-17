// Copyright (c) 2021, 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package argocd

import (
	"errors"
	"github.com/stretchr/testify/assert"
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/common"
	"io"
	"net/http"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"strings"
	"testing"
)

// TestIsUsingDefaultCACertificate verifies whether the CerManager component specifies to use the default CA certificate or not
// GIVEN a CertManager component
//
//	WHEN isUsingDefaultCACertificate is called
//	THEN isUsingDefaultCACertificate should return true or false if the default CA certificate is required
func TestCopyDefaultCACertificate(t *testing.T) {
	log := getTestLogger(t)
	secret := createCASecret()
	var tests = []struct {
		testName string
		c        client.Client
		vz       *vzapi.Verrazzano
		isErr    bool
	}{
		{
			"should not copy CA secret when not using the CA secret",
			fake.NewClientBuilder().WithScheme(getScheme()).Build(),
			&vzAcmeDev,
			false,
		},
		{
			"should fail to copy the CA secret when it does not exist",
			fake.NewClientBuilder().WithScheme(getScheme()).Build(),
			&vzDefaultCA,
			true,
		},
		{
			"should copy the CA secret when using the CA secret",
			fake.NewClientBuilder().WithScheme(getScheme()).WithObjects(&secret).Build(),
			&vzDefaultCA,
			false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.testName, func(t *testing.T) {
			err := copyDefaultCACertificate(log, tt.c, tt.vz)
			if tt.isErr {
				assert.NotNil(t, err)
			} else {
				assert.Nil(t, err)
			}
		})
	}
}

// TestIsUsingDefaultCACertificate verifies whether the CerManager component specifies to use the default CA certificate or not
// GIVEN a CertManager component
//
//	WHEN isUsingDefaultCACertificate is called
//	THEN isUsingDefaultCACertificate should return true or false if the default CA certificate is required
func TestIsUsingDefaultCACertificate(t *testing.T) {
	var tests = []struct {
		testName string
		*vzapi.CertManagerComponent
		out bool
	}{
		{
			"no CA",
			nil,
			false,
		},
		{
			"acme CA",
			vzAcmeDev.Spec.Components.CertManager,
			false,
		},
		{
			"private CA",
			vzDefaultCA.Spec.Components.CertManager,
			true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.testName, func(t *testing.T) {
			assert.Equal(t, tt.out, isUsingDefaultCACertificate(tt.CertManagerComponent))
		})
	}
}

// TestCreateAdditionalCertificates verifies creation of additional certificates when they are necessary
// GIVEN a Verrazzano CR
//
//	WHEN createAdditionalCertificates is called
//	THEN createAdditionalCertificates should create additional certificates as necessary from the Verrazzano CR information
func TestCreateAdditionalCertificates(t *testing.T) {
	log := getTestLogger(t)
	c := fake.NewClientBuilder().WithScheme(getScheme()).Build()
	doOK := func(hc *http.Client, req *http.Request) (*http.Response, error) {
		return &http.Response{
			Body:       io.NopCloser(strings.NewReader("cert")),
			StatusCode: http.StatusOK,
		}, nil
	}
	doFail := func(hc *http.Client, req *http.Request) (*http.Response, error) {
		return nil, errors.New("boom")
	}
	var tests = []struct {
		testName string
		httpDo   common.HTTPDoSig
		vz       *vzapi.Verrazzano
		isErr    bool
	}{
		{
			"should create additional CA Certificates when using Acme Dev",
			doOK,
			&vzAcmeDev,
			false,
		},
		{
			"should fail to download additional CA Certificates",
			doFail,
			&vzAcmeDev,
			true,
		},
		{
			"should not download additional CA Certificates when using a private CA",
			doOK,
			&vzDefaultCA,
			false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.testName, func(t *testing.T) {
			common.HTTPDo = tt.httpDo
			err := common.ProcessAdditionalCertificates(log, c, tt.vz)
			if tt.isErr {
				assert.NotNil(t, err)
			} else {
				assert.Nil(t, err)
			}
		})
	}
}
