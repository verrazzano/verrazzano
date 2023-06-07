// Copyright (c) 2021, 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package rancher

import (
	"errors"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/common"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

// TestCreateCattleNamespace verifies creation of the cattle-system namespace
// GIVEN a CertManager component
//
//	WHEN createCattleSystemNamespace is called
//	THEN createCattleSystemNamespace the cattle-system namespace should be created
func TestCreateCattleNamespace(t *testing.T) {
	log := getTestLogger(t)

	var tests = []struct {
		testName string
		c        client.Client
	}{
		{
			"should create the cattle namespace",
			fake.NewClientBuilder().WithScheme(getScheme()).Build(),
		},
		{
			"should edit the cattle namespace if already exists",
			fake.NewClientBuilder().WithScheme(getScheme()).WithObjects(&v1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name: common.CattleSystem,
				},
			}).Build(),
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
			ctx := spi.NewFakeContext(tt.c, tt.vz, nil, false, profilesRelativePath)
			err := copyDefaultCACertificate(log, tt.c, ctx.EffectiveCR())
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
		*vzapi.ClusterIssuerComponent
		out bool
	}{
		{
			"no Issuer",
			nil,
			false,
		},
		{
			"LetsEncrypt Issuer",
			&vzapi.ClusterIssuerComponent{
				IssuerConfig: vzapi.IssuerConfig{
					LetsEncrypt: &vzapi.LetsEncryptACMEIssuer{
						EmailAddress: "myemail@fooo.com",
						Environment:  "staging",
					},
				},
			},
			false,
		},
		{
			"Default CA",
			vzapi.NewDefaultClusterIssuer(),
			true,
		},
		{
			"Custom CA",
			&vzapi.ClusterIssuerComponent{
				ClusterResourceNamespace: "customnamespace",
				IssuerConfig: vzapi.IssuerConfig{
					CA: &vzapi.CAIssuer{
						SecretName: "customSecret",
					},
				},
			},
			false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.testName, func(t *testing.T) {
			assert.Equal(t, tt.out, isUsingDefaultCACertificate(tt.ClusterIssuerComponent))
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
			ctx := spi.NewFakeContext(c, tt.vz, nil, false, profilesRelativePath)
			err := common.ProcessAdditionalCertificates(log, c, ctx.EffectiveCR())
			if tt.isErr {
				assert.NotNil(t, err)
				assert.True(t, strings.Contains(err.Error(), "boom"))
			} else {
				assert.Nil(t, err)
			}
		})
	}
}
