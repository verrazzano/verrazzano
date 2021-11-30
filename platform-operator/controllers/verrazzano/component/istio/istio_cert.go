// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package istio

import (
	"crypto/x509"
	"crypto/x509/pkix"
	"github.com/verrazzano/verrazzano/platform-operator/internal/k8s/certificate"
	"go.uber.org/zap"
	"math/big"
	"os"
	clipkg "sigs.k8s.io/controller-runtime/pkg/client"
	"time"
)

// createCert creates the cert used by Istio MTLS
func createCert(log *zap.SugaredLogger, client clipkg.Client, namespace string) error {
	certDir := os.TempDir()
	config := certificate.CreateIstioCertConfig(certDir)
	_, err := certificate.CreateSelfSignedCert(config)
	if err != nil {
		log.Errorf("Failed to create Certificate for Istio: %v", err)
		return err
	}

	return nil
}

func newRootCert() x509.Certificate {
	cert := x509.Certificate{
		Raw:                     nil,
		RawTBSCertificate:       nil,
		RawSubjectPublicKeyInfo: nil,
		RawSubject:              nil,
		RawIssuer:               nil,
		Signature:               nil,
		SignatureAlgorithm:      0,
		PublicKeyAlgorithm:      0,
		PublicKey:               nil,
		Version:                 0,
		SerialNumber:            &big.Int{},
		Issuer: pkix.Name{
			Country:            nil,
			Organization:       nil,
			OrganizationalUnit: nil,
			Locality:           nil,
			Province:           nil,
			StreetAddress:      nil,
			PostalCode:         nil,
			SerialNumber:       "",
			CommonName:         "",
			Names:              nil,
			ExtraNames:         nil,
		},
		Subject: pkix.Name{
			Country:            nil,
			Organization:       nil,
			OrganizationalUnit: nil,
			Locality:           nil,
			Province:           nil,
			StreetAddress:      nil,
			PostalCode:         nil,
			SerialNumber:       "",
			CommonName:         "",
			Names:              nil,
			ExtraNames:         nil,
		},
		NotBefore:                   time.Time{},
		NotAfter:                    time.Time{},
		KeyUsage:                    0,
		Extensions:                  nil,
		ExtraExtensions:             nil,
		UnhandledCriticalExtensions: nil,
		ExtKeyUsage:                 nil,
		UnknownExtKeyUsage:          nil,
		BasicConstraintsValid:       false,
		IsCA:                        false,
		MaxPathLen:                  0,
		MaxPathLenZero:              false,
		SubjectKeyId:                nil,
		AuthorityKeyId:              nil,
		OCSPServer:                  nil,
		IssuingCertificateURL:       nil,
		DNSNames:                    nil,
		EmailAddresses:              nil,
		IPAddresses:                 nil,
		URIs:                        nil,
		PermittedDNSDomainsCritical: false,
		PermittedDNSDomains:         nil,
		ExcludedDNSDomains:          nil,
		PermittedIPRanges:           nil,
		ExcludedIPRanges:            nil,
		PermittedEmailAddresses:     nil,
		ExcludedEmailAddresses:      nil,
		PermittedURIDomains:         nil,
		ExcludedURIDomains:          nil,
		CRLDistributionPoints:       nil,
		PolicyIdentifiers:           nil,
	}
	return cert
}


// CreateIstioCertConfig creates the config needed to create an Istio certificate
func CreateIstioCertConfig(certDir string) Config {
	return Config{
		CertDir:                     certDir,
		CommonName:                  fmt.Sprintf("%s.%s.svc", OperatorName, OperatorNamespace),
		IntermediatePrivKeyFilename: "ca-key.pem",
		IntermediateCertFilename:    "ca-cert.pem",
		RootCertFilename:            "root-cert.pem",
		CertChainFilename:           "cert-chain.pem",
		NotBefore:                   time.Now(),
		NotAfter:                    time.Now().AddDate(1, 0, 0),
	}
}
