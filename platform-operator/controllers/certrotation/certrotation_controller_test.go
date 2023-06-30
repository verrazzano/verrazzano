// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package certrotation

import (
	"bytes"
	"context"
	cryptorand "crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	vzstatus "github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/healthcheck"
	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	k8scheme "k8s.io/client-go/kubernetes/scheme"
	"math/big"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/verrazzano/verrazzano/pkg/log/vzlog"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

const (
	VZNamespace         = "verrazzano-install"
	VZWebhookDeployment = "verrazzano-platform-operator-webhook"
	VZCertificate       = "verrazzano-platform-operator-tls"
	CheckPeriodDuration = 1
)

var (
	testScheme = runtime.NewScheme()
	//checkPeriodDuration = time.Duration(1) * time.Second
)

func init() {
	_ = k8scheme.AddToScheme(testScheme)
}

// TestExpiredValidateCertificate validate certificate
// if certificate is expired
// THEN restart the target deployment
// (restart of operator will re-generate the certificate)
func TestExpiredValidateCertificate(t *testing.T) {
	asserts := assert.New(t)
	deployment := getDeployment()
	secret := getSecret(2)
	cli := fake.NewClientBuilder().WithScheme(testScheme).WithObjects(&deployment, &secret).Build()
	request0 := newRequest(deployment.Name, secret.Name)
	reconciler := newCertificateManagerReconciler(cli, VZNamespace, VZNamespace, VZWebhookDeployment, 10)
	res0, err0 := reconciler.Reconcile(context.TODO(), request0)
	asserts.NoError(err0)
	asserts.Equal(true, res0.Requeue)
}

// TestValidateCertificate checks certificate
// if certificate is expired
// THEN restart the target deployment
func TestValidateCertificate(t *testing.T) {
	asserts := assert.New(t)
	deployment := getDeployment()
	secret := getSecret(1)
	cli := fake.NewClientBuilder().WithScheme(testScheme).WithObjects(&deployment, &secret).Build()
	request0 := newRequest(deployment.Name, secret.Name)
	reconciler := newCertificateManagerReconciler(cli, VZNamespace, VZNamespace, VZWebhookDeployment, 25)
	res0, err0 := reconciler.Reconcile(context.TODO(), request0)
	asserts.NoError(err0)
	asserts.Equal(true, res0.Requeue)
}

// TestCertificate1 checks certificate
// if target deployment doesn't exist
// THEN Requeue should happen.
func TestCertificate1(t *testing.T) {
	asserts := assert.New(t)
	deployment := getDeployment()
	secret := getSecret(1)
	// Set up the initial context
	cli := fake.NewClientBuilder().WithScheme(testScheme).WithObjects(&deployment, &secret).Build()
	request0 := newRequest(deployment.Name, secret.Name)
	reconciler := newCertificateManagerReconciler(cli, VZNamespace, VZNamespace, "verrazzano", 25)
	res0, err := reconciler.Reconcile(context.TODO(), request0)
	asserts.Equal(true, res0.Requeue)
	asserts.Equal(err, nil)
}

// TestCertificate2 checks certificate
// if target Namespace doesn't exist
// THEN Requeue should happen.
func TestCertificate2(t *testing.T) {
	asserts := assert.New(t)
	deployment := getDeployment()
	secret := getSecret(1)
	// Set up the initial context
	cli := fake.NewClientBuilder().WithScheme(testScheme).WithObjects(&deployment, &secret).Build()
	request0 := newRequest(deployment.Name, secret.Name)
	reconciler := newCertificateManagerReconciler(cli, VZNamespace, "verrazzano", VZWebhookDeployment, 25)
	res0, err := reconciler.Reconcile(context.TODO(), request0)
	asserts.Equal(true, res0.Requeue)
	asserts.Equal(err, nil)
}

// TestCertificate3 checks certificate
// if secret Namespace doesn't exist
// THEN Requeue should happen.
func TestCertificate3(t *testing.T) {
	asserts := assert.New(t)
	deployment := getDeployment()
	secret := getSecret(1)
	// Set up the initial context
	cli := fake.NewClientBuilder().WithScheme(testScheme).WithObjects(&deployment, &secret).Build()
	request0 := newRequest(deployment.Name, secret.Name)
	reconciler := newCertificateManagerReconciler(cli, "verrazzano", VZNamespace, VZWebhookDeployment, 25)
	res0, err := reconciler.Reconcile(context.TODO(), request0)
	asserts.Equal(true, res0.Requeue)
	asserts.Equal(err, nil)
}

// TestCertificate4 checks certificate
// if secret/certificate doesn't exist
// THEN Requeue should happen.
func TestCertificate4(t *testing.T) {
	asserts := assert.New(t)
	deployment := getDeployment()
	// Set up the initial context
	cli := fake.NewClientBuilder().WithScheme(testScheme).WithObjects(&deployment).Build()
	request0 := newRequest(deployment.Name, deployment.Name)
	reconciler := newCertificateManagerReconciler(cli, VZNamespace, VZNamespace, VZWebhookDeployment, 10)
	res0, err := reconciler.Reconcile(context.TODO(), request0)
	asserts.Equal(true, res0.Requeue)
	asserts.Equal(err, nil)
}

// newScheme creates a new scheme that includes this package's object to use for testing
func newScheme() *runtime.Scheme {
	scheme := runtime.NewScheme()
	_ = v1.AddToScheme(scheme)
	return scheme
}

// newRequest creates a new reconciler request for testing
// namespace - The namespace to use in the request
// name - The name to use in the request
func newRequest(namespace string, name string) ctrl.Request {
	return ctrl.Request{
		NamespacedName: types.NamespacedName{
			Namespace: namespace,
			Name:      name}}
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

// newConfigMapReconciler creates a new reconciler for testing
func newCertificateManagerReconciler(c client.Client, watchNamespace, targetNamespace, targetDeployment string, compareWindow int64) CertificateRotationManagerReconciler {
	vzLog := vzlog.DefaultLogger()
	scheme := newScheme()
	reconciler := CertificateRotationManagerReconciler{
		Client:           c,
		Scheme:           scheme,
		StatusUpdater:    &vzstatus.FakeVerrazzanoStatusUpdater{Client: c},
		log:              vzLog,
		WatchNamespace:   watchNamespace,
		TargetNamespace:  targetNamespace,
		TargetDeployment: targetDeployment,
		CompareWindow:    time.Duration(compareWindow),
	}
	return reconciler
}
