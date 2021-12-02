// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package certificate

import (
	"crypto/x509"
	"crypto/x509/pkix"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// TestCreateSelfSignedCert tests that a intermediate cert can be created that can sign another cert
// GIVEN a cert config for root and intermediate certs
//  WHEN I call CreateSelfSignedCert
//  THEN the resulting intermediate cert can sign another cert
func TestCreateSelfSignedCert(t *testing.T) {
	assert := assert.New(t)

	pem, err := createTestCerts()
	assert.NoError(err, "Error creating self-signed certs")

	parent := pem.IntermediateCertResult
	certInfo := createPartialCert(parent, "testname")

	// sign the intermediate cert with the root cert
	certResult, err := createCert(certInfo, parent.Cert, parent.PrivateKey)
	assert.NoError(err, "Error parsing new certificate")
	assert.NotNil(certResult, "Nil new certificate")
}

// TestCreateNoCACert tests that a generated cannot sign another cert
// GIVEN a cert that cannot sign another cert
//  WHEN I call CreateSelfSignedCert
//  THEN the cert creation should fail
func TestCreateNoCACert(t *testing.T) {
	assert := assert.New(t)

	pem, err := createTestCerts()
	assert.NoError(err, "Error creating self-signed certs")

	parent := pem.IntermediateCertResult
	certInfo := createPartialCert(parent, "testname")

	// sign the intermediate cert with the root cert
	certResult, err := createCert(certInfo, parent.Cert, parent.PrivateKey)
	assert.NoError(err, "Error parsing new certificate")
	assert.NotNil(certResult, "Nil new certificate")

	// Now try creating a cert using the new cert which cannot sign.  It should fail
	// sign the intermediate cert with the root cert
	badCertInfo := createPartialCert(certResult, "badcert")
	badResult, err := createCert(badCertInfo, certResult.Cert, certResult.PrivateKey)
	assert.Error(err, "Expect error creating new certificate")
	assert.Nil(badResult, "Expected nil new certificate")
}

// createCert creates a test cert
func createTestCerts() (*CertPemData, error) {
	rootConfig := createConfig("Root CA")
	intermConfig := createConfig("Intermediate CA")
	pemData, err := CreateSelfSignedCert(rootConfig, intermConfig)
	if err != nil {
		return nil, err
	}
	return pemData, nil
}

// createConfig creates a cert config
func createConfig(cn string) CertConfig {
	const (
		country = "US"
		org     = "Fake Corporation"
		state   = "NH"
	)
	return CertConfig{
		CountryName:         country,
		OrgName:             org,
		StateOrProvinceName: state,
		CommonName:          cn,
		NotBefore:           time.Now(),
		NotAfter:            time.Now().AddDate(1, 0, 0),
	}
}

// Create a partial cert for testing
func createPartialCert(parent *CertResult, cn string) *x509.Certificate {

	config := createConfig(cn)

	// create the new certificate info needed to create the certificate
	return &x509.Certificate{
		DNSNames:     []string{config.CommonName},
		SerialNumber: parent.Cert.SerialNumber,
		Subject: pkix.Name{
			CommonName:   config.CommonName,
			Country:      []string{config.CountryName},
			Province:     []string{config.StateOrProvinceName},
			Organization: []string{config.OrgName},
		},
		NotBefore:             config.NotBefore,
		NotAfter:              config.NotAfter,
		IsCA:                  false,
		AuthorityKeyId:        parent.Cert.SubjectKeyId,
		BasicConstraintsValid: true,
	}
}
