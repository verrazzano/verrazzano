// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package certrotation

import (
	"bytes"
	cryptorand "crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"github.com/stretchr/testify/assert"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	k8scheme "k8s.io/client-go/kubernetes/scheme"
	"math/big"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"testing"
	"time"
)

var (
	testScheme          = runtime.NewScheme()
	checkPeriodDuration = time.Duration(1) * time.Second
)

func init() {
	_ = k8scheme.AddToScheme(testScheme)
}

const (
	VZNamespace         = "verrazzano-install"
	VZWebhookDeployment = "verrazzano-platform-operator-webhook"
	VZCertificate       = "verrazzano-platform-operator-tls"
)

// TestExpiredValidateCertificate validate certificate
// if certificate is expired
// THEN restart the target deployment
// (restart of operator will re-generate the certificate)
func TestExpiredValidateCertificate(t *testing.T) {
	deployment := getDeployment()
	secret := getSecret(5)
	// Set up the initial context
	cli := fake.NewClientBuilder().WithScheme(testScheme).WithObjects(&deployment, &secret).Build()
	fakeCtx := spi.NewFakeContext(cli, nil, nil, false)
	certificateCheck, err := NewCertificateRotationManager(fakeCtx.Client(), checkPeriodDuration, VZNamespace, VZNamespace, VZWebhookDeployment)
	assert.NoError(t, err)
	err = certificateCheck.CheckCertificateExpiration()
	assert.NoError(t, err)
}

// TestValidateCertificate checks certificate
// if certificate is expired
// THEN restart the target deployment
func TestValidateCertificate(t *testing.T) {
	deployment := getDeployment()
	secret := getSecret(15)
	// Set up the initial context
	cli := fake.NewClientBuilder().WithScheme(testScheme).WithObjects(&deployment, &secret).Build()
	fakeCtx := spi.NewFakeContext(cli, nil, nil, false)
	certificateCheck, err := NewCertificateRotationManager(fakeCtx.Client(), checkPeriodDuration, VZNamespace, VZNamespace, VZWebhookDeployment)
	assert.NoError(t, err)
	err = certificateCheck.CheckCertificateExpiration()
	assert.NoError(t, err)
}

// TestCertificate1 checks certificate
// if target deployment doesn't exist
// THEN error should be thrown
func TestCertificate1(t *testing.T) {
	deployment := getDeployment()
	secret := getSecret(5)
	// Set up the initial context
	cli := fake.NewClientBuilder().WithScheme(testScheme).WithObjects(&deployment, &secret).Build()
	fakeCtx := spi.NewFakeContext(cli, nil, nil, false)
	certificateCheck, err := NewCertificateRotationManager(fakeCtx.Client(), checkPeriodDuration, VZNamespace, VZNamespace, "operator")
	assert.NoError(t, err)
	err = certificateCheck.CheckCertificateExpiration()
	assert.Equal(t, "an error occurred restarting the deployment operator in namespace verrazzano-install", err.Error())
}

// TestCertificate2 checks certificate
// if target Namespace doesn't exist
// THEN error should be thrown
func TestCertificate2(t *testing.T) {
	deployment := getDeployment()
	secret := getSecret(5)
	// Set up the initial context
	cli := fake.NewClientBuilder().WithScheme(testScheme).WithObjects(&deployment, &secret).Build()
	fakeCtx := spi.NewFakeContext(cli, nil, nil, false)
	certificateCheck, err := NewCertificateRotationManager(fakeCtx.Client(), checkPeriodDuration, VZNamespace, "verrazzano", "verrazzano-operator")
	assert.NoError(t, err)
	err = certificateCheck.CheckCertificateExpiration()
	assert.Equal(t, "an error occurred restarting the deployment verrazzano-operator in namespace verrazzano", err.Error())
}

// TestCertificate3 checks certificate
// if secret Namespace doesn't exist
// THEN error should be thrown
func TestCertificate3(t *testing.T) {
	deployment := getDeployment()
	secret := getSecret(5)
	// Set up the initial context
	cli := fake.NewClientBuilder().WithScheme(testScheme).WithObjects(&deployment, &secret).Build()
	fakeCtx := spi.NewFakeContext(cli, nil, nil, false)
	certificateCheck, err := NewCertificateRotationManager(fakeCtx.Client(), checkPeriodDuration, "verrazzano", VZNamespace, VZWebhookDeployment)
	assert.NoError(t, err)
	err = certificateCheck.CheckCertificateExpiration()
	assert.Equal(t, "no certificate found in namespace verrazzano", err.Error())
}

// TestCertificate4 checks certificate
// if secret/certificate doesn't exist
// THEN error should be thrown
func TestCertificate4(t *testing.T) {
	deployment := getDeployment()
	// Set up the initial context
	cli := fake.NewClientBuilder().WithScheme(testScheme).WithObjects(&deployment).Build()
	fakeCtx := spi.NewFakeContext(cli, nil, nil, false)
	certificateCheck, err := NewCertificateRotationManager(fakeCtx.Client(), checkPeriodDuration, VZNamespace, VZNamespace, VZWebhookDeployment)
	assert.NoError(t, err)
	err = certificateCheck.CheckCertificateExpiration()
	assert.Equal(t, "no certificate found in namespace verrazzano-install", err.Error())
}
func getInt32Ptr(i int32) *int32 {
	return &i
}

func getSecret(days int) v1.Secret {
	crt, key := getSecretValidData(days)
	secret := &v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      VZCertificate,
			Namespace: VZNamespace,
		},
		Immutable: nil,
		Data: map[string][]byte{
			"tls.crt": crt,
			"tls.key": key,
		},
		StringData: nil,
		Type:       "kubernetes.io/tls",
	}
	return *secret
}

func getDeployment() appsv1.Deployment {
	deployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      VZWebhookDeployment,
			Namespace: VZNamespace,
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: getInt32Ptr(1),
		},
	}
	return *deployment
}

func getSecretValidData(days int) ([]byte, []byte) {
	value, _ := newSerialNumber()
	caValue, caPrivateKey := getCaCert()
	cert := &x509.Certificate{
		DNSNames:     []string{VZCertificate},
		SerialNumber: value,
		Subject: pkix.Name{
			CommonName: VZCertificate,
		},
		NotBefore:    time.Now(),
		NotAfter:     time.Now().AddDate(0, 0, days),
		IsCA:         false,
		SubjectKeyId: []byte{1, 2, 3, 4, 6},
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth, x509.ExtKeyUsageServerAuth},
		KeyUsage:     x509.KeyUsageDigitalSignature,
	}
	// server private key
	serverPrivKey, _ := rsa.GenerateKey(cryptorand.Reader, 4096)
	// sign the server cert
	//certBytes, err := x509.CreateCertificate(rand.Reader, cert, ca, &certPrivKey.PublicKey, caPrivKey)
	serverCertBytes, _ := x509.CreateCertificate(cryptorand.Reader, cert, caValue, &serverPrivKey.PublicKey, caPrivateKey)
	certPEM := new(bytes.Buffer)
	pem.Encode(certPEM, &pem.Block{
		Type:  "CERTIFICATE",
		Bytes: serverCertBytes,
	})

	certPrivKeyPEM := new(bytes.Buffer)
	pem.Encode(certPrivKeyPEM, &pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(serverPrivKey),
	})
	return certPEM.Bytes(), certPrivKeyPEM.Bytes()
}

// newSerialNumber returns a new random serial number suitable for use in a certificate.
func newSerialNumber() (*big.Int, error) {
	// A serial number can be up to 20 octets in size.
	return cryptorand.Int(cryptorand.Reader, new(big.Int).Lsh(big.NewInt(1), 8*20))
}

func getCaCert() (*x509.Certificate, *rsa.PrivateKey) {
	value, _ := newSerialNumber()
	// CA config
	ca := &x509.Certificate{
		DNSNames:     []string{VZCertificate},
		SerialNumber: value,
		Subject: pkix.Name{
			CommonName: VZCertificate,
		},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().AddDate(2, 0, 0),
		IsCA:                  true,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth, x509.ExtKeyUsageServerAuth},
		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageCertSign,
		BasicConstraintsValid: true,
	}

	// CA private key
	caPrivKey, _ := rsa.GenerateKey(cryptorand.Reader, 4096)
	// Self signed CA certificate
	caBytes, _ := x509.CreateCertificate(cryptorand.Reader, ca, ca, &caPrivKey.PublicKey, caPrivKey)
	caPEM := new(bytes.Buffer)
	pem.Encode(caPEM, &pem.Block{
		Type:  "CERTIFICATE",
		Bytes: caBytes,
	})

	caPrivKeyPEM := new(bytes.Buffer)
	pem.Encode(caPrivKeyPEM, &pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(caPrivKey),
	})
	return ca, caPrivKey
}

// TestStart Validates the controller
func TestStart(t *testing.T) {
	p := newTestCertificateCheck()
	assert.Nil(t, p.shutdown)
	p.Start()
	assert.NotNil(t, p.shutdown)
	p.Start()
	assert.NotNil(t, p.shutdown)
}

func newTestCertificateCheck() *CertificateRotationManager {
	c := fake.NewClientBuilder().WithScheme(testScheme).Build()
	certificateWatcher, _ := NewCertificateRotationManager(c, 2*time.Second, VZNamespace, VZNamespace, VZWebhookDeployment)
	return certificateWatcher
}
