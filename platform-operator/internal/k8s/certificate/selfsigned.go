// Copyright (c) 2020, 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package certificate

import (
	"bytes"
	cryptorand "crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"os"
	"time"
)

// Config specifies the certificate configuration
type Config struct {
	// CertDir is the directory where the certificate should be written
	CertDir string

	// CommonName is the certificate common name
	CommonName string

	// NotBefore time when certificate is valid
	NotBefore time.Time

	// NotAfter time when certificate is valid
	NotAfter time.Time

	// FilenameRootCert is the filename in CertDir where the root cert PEM will be written
	FilenameRootCert string

	// FilenameRootPrivKey is the filename in CertDir where the root private key PEM will be written
	FilenameRootPrivKey string

	// FilenameIntermediateCert is the filename in CertDir where the intermediate cert PEM will be written
	FilenameIntermediateCert string

	// FilenameIntermediatePrivKey is the filename in CertDir where the intermediate private key PEM will be written
	FilenameIntermediatePrivKey string

	// FilenameCertChain is the filename in CertDir where that cert chain PEM (ca cert + root cert) will be written
	FilenameCertChain string
}

// CreateSelfSignedCert creates the needed certificates for the validating webhook
func CreateSelfSignedCert(config Config) (*bytes.Buffer, error) {
	serialNumber, err := newSerialNumber()
	if err != nil {
		return nil, err
	}

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
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth, x509.ExtKeyUsageServerAuth},
		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageCertSign,
		BasicConstraintsValid: true,
	}

	// create root private key
	rootPrivKey, err := rsa.GenerateKey(cryptorand.Reader, 4096)
	if err != nil {
		return nil, err
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
		return nil, err
	}

	// PEM encode root cert
	rootCertPEM := new(bytes.Buffer)
	_ = pem.Encode(rootCertPEM, &pem.Block{
		Type:  "CERTIFICATE",
		Bytes: rootCertBytes,
	})

	serialNumber, err = newSerialNumber()
	if err != nil {
		return nil, err
	}

	// create the intermediate certificate info needed to create the certificate
	intermediateCertInfo := &x509.Certificate{
		DNSNames:     []string{config.CommonName},
		SerialNumber: serialNumber,
		Subject: pkix.Name{
			CommonName: config.CommonName,
		},
		NotBefore:    config.NotBefore,
		NotAfter:     config.NotAfter,
		IsCA:         true,
		SubjectKeyId: []byte{1, 2, 3, 4, 6},
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth, x509.ExtKeyUsageServerAuth},
		KeyUsage:     x509.KeyUsageDigitalSignature,
	}

	// generate intermediate cert private key
	intermediatePrivKey, err := rsa.GenerateKey(cryptorand.Reader, 4096)
	if err != nil {
		return nil, err
	}

	// PEM encode the intermediate private key
	intermediatePrivKeyPEM := new(bytes.Buffer)
	_ = pem.Encode(intermediatePrivKeyPEM, &pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(intermediatePrivKey),
	})

	// sign the intermediate cert with the root cert
	intermediateCertBytes, err := x509.CreateCertificate(cryptorand.Reader, intermediateCertInfo, rootCertInfo, &intermediatePrivKey.PublicKey, rootPrivKey)
	if err != nil {
		return nil, err
	}

	// PEM encode the intermediate cert and key
	intermediateCertPEM := new(bytes.Buffer)
	_ = pem.Encode(intermediateCertPEM, &pem.Block{
		Type:  "CERTIFICATE",
		Bytes: intermediateCertBytes,
	})

	// Write the files that are specified in the config
	certDir := config.CertDir
	err = os.MkdirAll(certDir, 0666)
	if err != nil {
		return nil, err
	}

	if len(config.FilenameRootPrivKey) > 0 {
		err = writeFile(fmt.Sprintf("%s/%s", certDir, config.FilenameRootPrivKey), rootPrivKeyPEM)
		if err != nil {
			return nil, err
		}
	}
	if len(config.FilenameRootCert) > 0 {
		err = writeFile(fmt.Sprintf("%s/%s", certDir, config.FilenameRootCert), rootCertPEM)
		if err != nil {
			return nil, err
		}
	}
	if len(config.FilenameIntermediateCert) > 0 {
		err = writeFile(fmt.Sprintf("%s/%s", certDir, config.FilenameIntermediateCert), intermediateCertPEM)
		if err != nil {
			return nil, err
		}
	}
	if len(config.FilenameIntermediatePrivKey) > 0 {
		err = writeFile(fmt.Sprintf("%s/%s", certDir, config.FilenameIntermediatePrivKey), intermediatePrivKeyPEM)
		if err != nil {
			return nil, err
		}
	}
	if len(config.FilenameCertChain) > 0 {
		err = writeFile(fmt.Sprintf("%s/%s", certDir, config.FilenameCertChain), intermediateCertPEM, rootCertPEM)
		if err != nil {
			return nil, err
		}
	}
	return rootCertPEM, nil
}

// CreateWebhookCertConfig creates the config needed to create platform operator webhook certificate
func CreateWebhookCertConfig(certDir string) Config {
	return Config{
		CertDir:                     certDir,
		CommonName:                  fmt.Sprintf("%s.%s.svc", OperatorName, OperatorNamespace),
		FilenameIntermediateCert:    "tls.crt",
		FilenameIntermediatePrivKey: "tls.key",
		NotBefore:                   time.Now(),
		NotAfter:                    time.Now().AddDate(1, 0, 0),
	}
}

// CreateIstioCertConfig creates the config needed to create an Istio certificate
func CreateIstioCertConfig(certDir string) Config {
	return Config{
		CertDir:                     certDir,
		CommonName:                  fmt.Sprintf("%s.%s.svc", OperatorName, OperatorNamespace),
		FilenameIntermediatePrivKey: "ca-key.pem",
		FilenameIntermediateCert:    "ca-cert.pem",
		FilenameRootCert:            "root-cert.pem",
		FilenameCertChain:           "cert-chain.pem",
		NotBefore:                   time.Now(),
		NotAfter:                    time.Now().AddDate(1, 0, 0),
	}
}
