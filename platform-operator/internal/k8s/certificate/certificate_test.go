// Copyright (c) 2020, 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package certificate

import (
	"context"
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
	v1 "k8s.io/api/core/v1"
	errors2 "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/fake"
	fakecorev1 "k8s.io/client-go/kubernetes/typed/core/v1/fake"
	testing2 "k8s.io/client-go/testing"
)

// TestCreateWebhookCertificates tests that the certificates needed for webhooks are created
// GIVEN an output directory for certificates
//
//	WHEN I call CreateWebhookCertificates
//	THEN all the needed certificate artifacts are created
func TestCreateWebhookCertificates(t *testing.T) {
	asserts := assert.New(t)
	client := fake.NewSimpleClientset()

	// create temp dir for certs
	tempDir, err := os.MkdirTemp("", "certs")
	t.Logf("Using temp dir %s", tempDir)
	defer func() {
		err := os.RemoveAll(tempDir)
		if err != nil {
			t.Logf("Error removing tempdir %s", tempDir)
		}
	}()
	asserts.Nil(err)

	err = CreateWebhookCertificates(zap.S(), client, tempDir)
	asserts.Nil(err)

	// Verify generated certs
	cert1 := validateFile(asserts, tempDir+"/tls.crt", "-----BEGIN CERTIFICATE-----")
	key1 := validateFile(asserts, tempDir+"/tls.key", "-----BEGIN RSA PRIVATE KEY-----")

	// Verify generated secrets
	var secret *v1.Secret
	secret, err = client.CoreV1().Secrets(OperatorNamespace).Get(context.TODO(), OperatorCA, metav1.GetOptions{})
	_, err2 := client.CoreV1().Secrets(OperatorNamespace).Get(context.TODO(), OperatorTLS, metav1.GetOptions{})
	asserts.Nil(err, "error should not be returned setting up certificates")
	asserts.Nil(err2, "error should not be returned setting up certificates")
	asserts.NotEmpty(string(secret.Data[CertKey]))

	// create temp dir for certs
	tempDir2, err := os.MkdirTemp("", "certs")
	t.Logf("Using temp dir %s", tempDir2)
	defer func() {
		err := os.RemoveAll(tempDir2)
		if err != nil {
			t.Logf("Error removing tempdir %s", tempDir2)
		}
	}()
	asserts.Nil(err)

	// Call it again, should create new certs in location with identical contents
	err = CreateWebhookCertificates(zap.S(), client, tempDir2)
	asserts.Nil(err)
	asserts.Nil(err)

	// Verify generated certs
	cert2 := validateFile(asserts, tempDir+"/tls.crt", "-----BEGIN CERTIFICATE-----")
	key2 := validateFile(asserts, tempDir+"/tls.key", "-----BEGIN RSA PRIVATE KEY-----")

	asserts.Equalf(cert1, cert2, "Certs did not match")
	asserts.Equalf(key1, key2, "Keys did not match")
}

// TestCreateWebhookCertificatesRaceCondition the handling of the race condition for the CA and TLS
// certificates creation
// GIVEN a call to CreateWebhookCertificates
//
//	WHEN the secrets initially do not exist at the beginning of the method invocation, but later do on the Create call
//	THEN the previously stored secret data is used for the generated certificate/key files
func TestCreateWebhookCertificatesRaceCondition(t *testing.T) {
	asserts := assert.New(t)
	client := fake.NewSimpleClientset()

	commonName := fmt.Sprintf("%s.%s.svc", OperatorName, OperatorNamespace)

	log := zap.S()

	// Call the internal routines to create the cert data and secrets within the fake client;
	// later we will simulate the race condition using reactors with the fake

	// Create the CA cert and key, and verify the secret is tracked in the fake client
	ca, caKey, err := createCACert(log, client, commonName)
	asserts.Nil(err)
	_, err = client.CoreV1().Secrets(OperatorNamespace).Get(context.TODO(), OperatorCA, metav1.GetOptions{})
	asserts.Nil(err)

	// Create the TLS cert and key, and verify the secret is tracked in the fake client
	serverPEM, serverKeyPEM, err := createTLSCert(log, client, commonName, ca, caKey)
	asserts.Nil(err)
	_, err = client.CoreV1().Secrets(OperatorNamespace).Get(context.TODO(), OperatorTLS, metav1.GetOptions{})
	asserts.Nil(err)

	// create temp dir for certs
	tempDir, err := os.MkdirTemp("", "certs")
	t.Logf("Using temp dir %s", tempDir)
	defer func() {
		err := os.RemoveAll(tempDir)
		if err != nil {
			t.Logf("Error removing tempdir %s", tempDir)
		}
	}()
	asserts.Nil(err)

	// Simulate the race condition where the secrets do not exist when CreateWebhookCertificates is called
	// but do exist when it attempts to create them later in the routine.
	//
	// We do this by having the initial secret Get() operations return a NotExists error,
	// but later the Create() calls return an Already exists error (the code before this causes the
	// secrets to exist in the fake client).

	getCAInvokes := 0
	getTLSInvokes := 0
	client.CoreV1().(*fakecorev1.FakeCoreV1).
		PrependReactor("get", "secrets",
			func(action testing2.Action) (handled bool, obj runtime.Object, err error) {
				getActionImpl := action.(testing2.GetActionImpl)
				resName := getActionImpl.GetName()
				if resName == OperatorCA {
					getCAInvokes++
					if getCAInvokes > 1 {
						// Return not handled so the test secret is returned
						return false, nil, nil
					}
				} else if resName == OperatorTLS {
					getTLSInvokes++
					if getTLSInvokes > 1 {
						// Return not handled so the test secret is returned
						return false, nil, nil
					}
				}
				// Return not found on the initial Get call for each secret
				return true, nil, errors2.NewNotFound(action.GetResource().GroupResource(), getActionImpl.GetName())
			})

	err = CreateWebhookCertificates(log, client, tempDir)
	asserts.Nil(err)

	// Verify generated certs
	cert1 := validateFile(asserts, tempDir+"/tls.crt", "-----BEGIN CERTIFICATE-----")
	key1 := validateFile(asserts, tempDir+"/tls.key", "-----BEGIN RSA PRIVATE KEY-----")

	asserts.Equalf(serverPEM, cert1, "Certs did not match")
	asserts.Equalf(serverKeyPEM, key1, "Keys did not match")
}

func validateFile(asserts *assert.Assertions, certFile string, certPrefix string) []byte {
	file, err := os.ReadFile(certFile)
	asserts.Nilf(err, "Error reading file", certFile)
	asserts.True(strings.Contains(string(file), certPrefix))
	return file
}
