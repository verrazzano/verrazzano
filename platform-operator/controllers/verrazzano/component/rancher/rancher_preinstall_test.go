// Copyright (c) 2021, 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package rancher

import (
	"errors"
	"github.com/stretchr/testify/assert"
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/common"
	"io"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"net/http"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"strings"
	"testing"
)

// TestCreateOperatorNamespace verifies creation of the Rancher operator namespace
// GIVEN a CertManager component
//  WHEN createRancherOperatorNamespace is called
//  THEN createRancherOperatorNamespace the Rancher operator namespace should be created
func TestCreateOperatorNamespace(t *testing.T) {
	log := getTestLogger(t)

	var tests = []struct {
		testName string
		c        client.Client
	}{
		{
			"should create the Rancher operator namespace",
			fake.NewFakeClientWithScheme(getScheme()),
		},
		{
			"should not fail if the Rancher operator namespace already exists",
			fake.NewFakeClientWithScheme(getScheme(), &v1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name: OperatorNamespace,
				},
			}),
		},
	}

	for _, tt := range tests {
		t.Run(tt.testName, func(t *testing.T) {
			assert.Nil(t, createRancherOperatorNamespace(log, tt.c))
		})
	}
}

// TestCreateCattleNamespace verifies creation of the cattle-system namespace
// GIVEN a CertManager component
//  WHEN createCattleSystemNamespace is called
//  THEN createCattleSystemNamespace the cattle-system namespace should be created
func TestCreateCattleNamespace(t *testing.T) {
	log := getTestLogger(t)

	var tests = []struct {
		testName string
		c        client.Client
	}{
		{
			"should create the cattle namespace",
			fake.NewFakeClientWithScheme(getScheme()),
		},
		{
			"should edit the cattle namespace if already exists",
			fake.NewFakeClientWithScheme(getScheme(), &v1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name: common.CattleSystem,
				},
			}),
		},
	}

	for _, tt := range tests {
		t.Run(tt.testName, func(t *testing.T) {
			assert.Nil(t, createCattleSystemNamespace(log, tt.c))
		})
	}
}

// TestIsUsingDefaultCACertificate verifies whether the CerManager component specifies to use the default CA certificate or not
// GIVEN a CertManager component
//  WHEN isUsingDefaultCACertificate is called
//  THEN isUsingDefaultCACertificate should return true or false if the default CA certificate is required
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
			fake.NewFakeClientWithScheme(getScheme()),
			&vzAcmeDev,
			false,
		},
		{
			"should fail to copy the CA secret when it does not exist",
			fake.NewFakeClientWithScheme(getScheme()),
			&vzDefaultCA,
			true,
		},
		{
			"should copy the CA secret when using the CA secret",
			fake.NewFakeClientWithScheme(getScheme(), &secret),
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
//  WHEN isUsingDefaultCACertificate is called
//  THEN isUsingDefaultCACertificate should return true or false if the default CA certificate is required
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
//  WHEN createAdditionalCertificates is called
//  THEN createAdditionalCertificates should create additional certificates as necessary from the Verrazzano CR information
func TestCreateAdditionalCertificates(t *testing.T) {
	log := getTestLogger(t)
	c := fake.NewFakeClientWithScheme(getScheme())
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
			err := createAdditionalCertificates(log, c, tt.vz)
			if tt.isErr {
				assert.NotNil(t, err)
			} else {
				assert.Nil(t, err)
			}
		})
	}
}
