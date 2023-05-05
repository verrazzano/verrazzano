// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package common

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"net"
	"time"
)

var testRSAKey *rsa.PrivateKey

func getRSAKey() (*rsa.PrivateKey, error) {
	if testRSAKey == nil {
		var err error
		if testRSAKey, err = rsa.GenerateKey(rand.Reader, 2048); err != nil {
			return nil, err
		}
	}
	return testRSAKey, nil
}

func CreateFakeCertBytes(cn string, parent *x509.Certificate) ([]byte, error) {
	rsaKey, err := getRSAKey()
	if err != nil {
		return []byte{}, err
	}

	cert := CreateFakeCertificate(cn)
	if parent == nil {
		parent = cert
	}
	pubKey := &rsaKey.PublicKey
	certBytes, err := x509.CreateCertificate(rand.Reader, cert, parent, pubKey, rsaKey)
	if err != nil {
		return []byte{}, err
	}
	certPem := pem.EncodeToMemory(&pem.Block{
		Type:  "CERTIFICATE",
		Bytes: certBytes,
	})
	return certPem, nil
}

func CreateFakeCertificate(cn string) *x509.Certificate {
	duration30, _ := time.ParseDuration("-30h")
	notBefore := time.Now().Add(duration30) // valid 30 hours ago
	duration1Year, _ := time.ParseDuration("90h")
	notAfter := notBefore.Add(duration1Year) // for 90 hours
	serialNo := big.NewInt(int64(123123413123))
	cert := &x509.Certificate{
		Subject: pkix.Name{
			Country:      []string{"US"},
			Organization: []string{"BarOrg"},
			SerialNumber: "2234",
			CommonName:   cn,
		},
		SerialNumber:          serialNo,
		NotBefore:             notBefore,
		NotAfter:              notAfter,
		KeyUsage:              x509.KeyUsageKeyEncipherment,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
		IsCA:                  false,
		SignatureAlgorithm:    x509.SHA256WithRSA,
	}

	cert.IPAddresses = append(cert.IPAddresses, net.ParseIP("127.0.0.1"))
	cert.IPAddresses = append(cert.IPAddresses, net.ParseIP("::"))
	cert.DNSNames = append(cert.DNSNames, "localhost")
	return cert
}

// GetBoolPtr Create a bool pointer
func GetBoolPtr(b bool) *bool {
	return &b
}
