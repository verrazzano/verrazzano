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

// rootResult contains the root cert generated results
type rootResult struct {
	PrivateKey *rsa.PrivateKey
	CertPEM    []byte
	CertInfo   *x509.Certificate
}

// intermResult contains the intermediate cert private key and cert, both in PEM format
type intermResult struct {
	PrivateKeyPEM []byte
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
	pem.RootCertPEM = rResult.CertPEM

	// Create the intermediate cert
	iResult, err := createIntermediateCert(intermConfig, rResult)
	if err != nil {
		return nil, err
	}
	pem.IntermediateCertPEM = iResult.CertPEM
	pem.IntermediatePrivateKeyPEM = iResult.PrivateKeyPEM

	// Write the chain
	b := bytes.Buffer{}
	b.Write(pem.IntermediateCertPEM)
	b.Write(pem.RootCertPEM)
	pem.CertChainPEM = b.Bytes()

	return &pem, nil
}

// Create the root certificate
func createRootCert(config CertConfig, serialNumber *big.Int) (*rootResult, error) {
	// create the root certificate info needed to create the certificate
	rootCertInfo := &x509.Certificate{
		DNSNames:     []string{config.CommonName},
		SerialNumber: serialNumber,
		Subject: pkix.Name{
			CommonName: config.CommonName,
			Country: []string{config.CountryName},
			Province: []string{config.StateOrProvinceName},
			Organization: []string{config.OrgName},
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
		return nil, err
	}

	// create root self-signed CA certificate
	rootCertBytes, err := x509.CreateCertificate(cryptorand.Reader, rootCertInfo, rootCertInfo, &rootPrivKey.PublicKey, rootPrivKey)
	if err != nil {
		return nil, err
	}

	// PEM encode root cert
	rootCertPEM := new(bytes.Buffer)
	_ = pem.Encode(rootCertPEM, &pem.Block{
		Type:  "CERTIFICATE",
		Bytes: rootCertBytes,
	})

	return &rootResult{
		PrivateKey: rootPrivKey,
		CertPEM:    rootCertPEM.Bytes(),
		CertInfo:   rootCertInfo,
	}, nil
}

// Create the intermediate certificate
func createIntermediateCert(config CertConfig, rootResult *rootResult) (*intermResult, error) {
	// create the intermediate certificate info needed to create the certificate
	intermediateCertInfo := &x509.Certificate{
		DNSNames:     []string{config.CommonName},
		SerialNumber: rootResult.CertInfo.SerialNumber,
		Subject: pkix.Name{
			CommonName: config.CommonName,
			Country: []string{config.CountryName},
			Province: []string{config.StateOrProvinceName},
			Organization: []string{config.OrgName},
		},
		NotBefore:      config.NotBefore,
		NotAfter:       config.NotAfter,
		IsCA:           true,
		AuthorityKeyId: rootResult.CertInfo.SubjectKeyId,
		KeyUsage:       x509.KeyUsageCertSign | x509.KeyUsageDigitalSignature | x509.KeyUsageCRLSign,
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
	certBytes, err := x509.CreateCertificate(cryptorand.Reader, intermediateCertInfo, rootResult.CertInfo, &privKey.PublicKey, rootResult.PrivateKey)
	if err != nil {
		return nil, err
	}

	// PEM encode the intermediate cert
	certPEM := new(bytes.Buffer)
	_ = pem.Encode(certPEM, &pem.Block{
		Type:  "CERTIFICATE",
		Bytes: certBytes,
	})

	return &intermResult{
		PrivateKeyPEM: privKeyPEM.Bytes(),
		CertPEM:       certPEM.Bytes(),
	}, nil
}
