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

	// The intermediate certificate in PEM format
	IntermediateCertPEM []byte

	// The intermediate private key in PEM format
	IntermediatePrivateKeyPEM []byte

	// The root certificate in PEM format
	RootCertPEM []byte
}

// keyAndCert contains a private key and cert in PEM format
type keyAndCert struct {
	PrivateKeyPEM []byte
	CertPEM       []byte
}

// CreateSelfSignedCert creates a self signed cert and returns the generated PEM data
func CreateSelfSignedCert(rootConfig CertConfig, intermConfig CertConfig,) (*CertPemData, error) {
	// Create the object that will be loaded with the PEM data
	pems := CertPemData{}

	serialNumber, err := newSerialNumber()
	if err != nil {
		return nil, err
	}
	// Create the root cert
	rootKeyAndCert, rootCertInfo, err := createRootCert(rootConfig, serialNumber)
	if err != nil {
		return nil, err
	}
	pems.RootCertPEM = rootKeyAndCert.CertPEM

	// Create the intermediate cert
	intermediateKeyAndCert, err := createIntermediateCert(intermConfig, rootCertInfo, rootKeyAndCert)
	if err != nil {
		return nil, err
	}
	pems.IntermediateCertPEM = intermediateKeyAndCert.CertPEM
	pems.IntermediatePrivateKeyPEM = intermediateKeyAndCert.PrivateKeyPEM

	// Write the chain
	b := bytes.Buffer{}
	b.Write(pems.IntermediateCertPEM)
	b.Write(pems.RootCertPEM)
	pems.CertChainPEM = b.Bytes()

	return &pems, nil
}

// Create the root certificate
func createRootCert(config CertConfig, serialNumber *big.Int) (*keyAndCert, *x509.Certificate, error) {
	// create the root certificate info needed to create the certificate
	rootCertInfo := &x509.Certificate{
		DNSNames:     []string{config.CommonName},
		SerialNumber: serialNumber,
		Subject: pkix.Name{
			CommonName: config.CommonName,
		},
		NotBefore:             config.NotBefore,
		NotAfter:              config.NotAfter,
		IsCA:                  true,
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageDigitalSignature | x509.KeyUsageCRLSign,
		BasicConstraintsValid: true,
	}

	// create root private key
	rootPrivKey, err := rsa.GenerateKey(cryptorand.Reader, 4096)
	if err != nil {
		return nil, nil, err
	}

	// PEM encode root private key
	rootPrivKeyPEM := new(bytes.Buffer)
	_ = pem.Encode(rootPrivKeyPEM, &pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(rootPrivKey),
	})

	// create root self-signed CA certificate
	rootCertBytes, err := x509.CreateCertificate(cryptorand.Reader, rootCertInfo, rootCertInfo, &rootPrivKey.PublicKey, rootPrivKey)
	if err != nil {
		return nil, nil, err
	}

	// PEM encode root cert
	rootCertPEM := new(bytes.Buffer)
	_ = pem.Encode(rootCertPEM, &pem.Block{
		Type:  "CERTIFICATE",
		Bytes: rootCertBytes,
	})

	return &keyAndCert{
		PrivateKeyPEM: rootPrivKeyPEM.Bytes(),
		CertPEM:       rootCertPEM.Bytes(),
	}, rootCertInfo, nil
}

// Create the intermediate certificate
func createIntermediateCert(config CertConfig, rootCertInfo *x509.Certificate, rootKeyAndCert *keyAndCert) (*keyAndCert, error) {
	// create the intermediate certificate info needed to create the certificate
	intermediateCertInfo := &x509.Certificate{
		DNSNames:     []string{config.CommonName},
		SerialNumber: rootCertInfo.SerialNumber,
		Subject: pkix.Name{
			CommonName: config.CommonName,
		},
		NotBefore:    config.NotBefore,
		NotAfter:     config.NotAfter,
		IsCA:         true,
		AuthorityKeyId: rootCertInfo.SubjectKeyId,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth, x509.ExtKeyUsageServerAuth},
		KeyUsage:     x509.KeyUsageDigitalSignature,
	}

	// generate intermediate cert private key
	privKey, err := rsa.GenerateKey(cryptorand.Reader, 4096)
	if err != nil {
		return nil, err
	}

	// PEM encode the intermediate private key
	privKeyPEM := new(bytes.Buffer)
	_ = pem.Encode(privKeyPEM, &pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(privKey),
	})

	// sign the intermediate cert with the root cert
	certBytes, err := x509.CreateCertificate(cryptorand.Reader, intermediateCertInfo, rootCertInfo, &privKey.PublicKey, rootKeyAndCert.PrivateKeyPEM)
	if err != nil {
		return nil, err
	}

	// PEM encode the intermediate cert
	certPEM := new(bytes.Buffer)
	_ = pem.Encode(certPEM, &pem.Block{
		Type:  "CERTIFICATE",
		Bytes: certBytes,
	})

	return &keyAndCert{
		PrivateKeyPEM: privKeyPEM.Bytes(),
		CertPEM:       certPEM.Bytes(),
	}, nil
}
