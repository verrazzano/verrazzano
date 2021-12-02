// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package certificate

import (
	"bytes"
	cryptorand "crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"time"
)

// CertConfig specifies the certificate configuration
type CertConfig struct {
	// CommonName is the certificate common name
	CommonName string

	// CountryName is the certificate country name
	CountryName string

	// OrgName is the organization name
	OrgName string

	// StateOrProvinceName is the certificate state or province
	StateOrProvinceName string

	// NotBefore time when certificate is valid
	NotBefore time.Time

	// NotAfter time when certificate is valid
	NotAfter time.Time
}

// CertPemData contains certificate data in PEM format
type CertPemData struct {
	// The certificate chain in PEM format.  This contains the intermediate cert followed
	// by the root cert
	CertChainPEM []byte

	// The intermediate cert results
	IntermediateCertResult *CertResult

	// The root cert results
	RootCertResult *CertResult
}

// CertResult contains the generated cert results
type CertResult struct {
	PrivateKey    *rsa.PrivateKey
	PrivateKeyPEM []byte
	Cert          *x509.Certificate
	CertPEM       []byte
}

// CreateSelfSignedCert creates a self signed cert and returns the generated PEM data
func CreateSelfSignedCert(rootConfig CertConfig, intermConfig CertConfig) (*CertPemData, error) {
	// Create the object that will be loaded with the PEM data
	pem := CertPemData{}

	serialNumber, err := newSerialNumber()
	if err != nil {
		return nil, err
	}

	// Create the root cert
	rResult, err := createRootCert(rootConfig, serialNumber)
	if err != nil {
		return nil, err
	}
	pem.RootCertResult = rResult

	// Create the intermediate cert
	iResult, err := createIntermediateCert(intermConfig, rResult)
	if err != nil {
		return nil, err
	}
	pem.IntermediateCertResult = iResult

	// Write the chain
	b := bytes.Buffer{}
	b.Write(pem.IntermediateCertResult.CertPEM)
	b.Write(pem.RootCertResult.CertPEM)
	pem.CertChainPEM = b.Bytes()

	return &pem, nil
}

// Create the root certificate
func createRootCert(config CertConfig, serialNumber *big.Int) (*CertResult, error) {
	// create the root certificate info needed to create the certificate
	rootCert := &x509.Certificate{
		DNSNames:     []string{config.CommonName},
		SerialNumber: serialNumber,
		Subject: pkix.Name{
			CommonName:   config.CommonName,
			Country:      []string{config.CountryName},
			Province:     []string{config.StateOrProvinceName},
			Organization: []string{config.OrgName},
		},
		NotBefore:             config.NotBefore,
		NotAfter:              config.NotAfter,
		IsCA:                  true,
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageDigitalSignature | x509.KeyUsageCRLSign,
		BasicConstraintsValid: true,
	}

	return createCert(rootCert, rootCert, nil)
}

// Create the intermediate certificate
func createIntermediateCert(config CertConfig, rootResult *CertResult) (*CertResult, error) {
	// create the intermediate certificate info needed to create the certificate
	intermediateCert := &x509.Certificate{
		DNSNames:     []string{config.CommonName},
		SerialNumber: rootResult.Cert.SerialNumber,
		Subject: pkix.Name{
			CommonName:   config.CommonName,
			Country:      []string{config.CountryName},
			Province:     []string{config.StateOrProvinceName},
			Organization: []string{config.OrgName},
		},
		NotBefore:             config.NotBefore,
		NotAfter:              config.NotAfter,
		IsCA:                  true,
		AuthorityKeyId:        rootResult.Cert.SubjectKeyId,
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageDigitalSignature | x509.KeyUsageCRLSign,
		BasicConstraintsValid: true,
	}
	return createCert(intermediateCert, rootResult.Cert, rootResult.PrivateKey)
}

// Create the certificate. The parent certificate will be used to sign the new cert
func createCert(partialCert *x509.Certificate, parentCert *x509.Certificate, parentPrivKey *rsa.PrivateKey) (*CertResult, error) {
	// create private key for this cert
	privKey, err := rsa.GenerateKey(cryptorand.Reader, 4096)
	if err != nil {
		return nil, err
	}

	// For root cert there is no parent key
	if parentPrivKey == nil {
		parentPrivKey = privKey
	}

	// PEM encode the private key
	privKeyPEM := new(bytes.Buffer)
	_ = pem.Encode(privKeyPEM, &pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(privKey),
	})

	// create the CA certificate using the partial cert as input
	certBytes, err := x509.CreateCertificate(cryptorand.Reader, partialCert, parentCert, &privKey.PublicKey, parentPrivKey)
	if err != nil {
		return nil, err
	}

	// PEM encode the new cert
	certPEM := new(bytes.Buffer)
	_ = pem.Encode(certPEM, &pem.Block{
		Type:  "CERTIFICATE",
		Bytes: certBytes,
	})

	// Reload the cert so that we get fields like the Subject Key ID that are generated
	newCert, err := x509.ParseCertificate(certBytes)
	if err != nil {
		return nil, err
	}

	return &CertResult{
		PrivateKey:    privKey,
		PrivateKeyPEM: privKeyPEM.Bytes(),
		CertPEM:       certPEM.Bytes(),
		Cert:          newCert,
	}, nil
}
