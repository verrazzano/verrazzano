// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package certmanager

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	cmutil "github.com/jetstack/cert-manager/pkg/api/util"
	certv1 "github.com/jetstack/cert-manager/pkg/apis/certmanager/v1"
	cmmeta "github.com/jetstack/cert-manager/pkg/apis/meta/v1"
	certv1fake "github.com/jetstack/cert-manager/pkg/client/clientset/versioned/fake"
	certv1client "github.com/jetstack/cert-manager/pkg/client/clientset/versioned/typed/certmanager/v1"
	"github.com/stretchr/testify/assert"
	"github.com/verrazzano/verrazzano/pkg/k8sutil"
	"github.com/verrazzano/verrazzano/pkg/log/vzlog"
	"github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1beta1"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/common"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	corev1 "k8s.io/api/core/v1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	k8sfake "k8s.io/client-go/kubernetes/fake"
	v1 "k8s.io/client-go/kubernetes/typed/core/v1"
	"math/big"
	"net"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"testing"
	"time"

	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
)

const (
	testDNSDomain  = "example.dns.io"
	testOCIDNSName = "ociDNS"
)

var (
	mockNamespaceCoreV1Client = common.MockGetCoreV1(&corev1.Namespace{ObjectMeta: metav1.ObjectMeta{
		Name:      ComponentName,
		Namespace: ComponentNamespace,
		Labels:    map[string]string{vzNsLabel: ComponentNamespace},
	}})
	mockNamespaceWithoutLabelClient = common.MockGetCoreV1(&corev1.Namespace{ObjectMeta: metav1.ObjectMeta{
		Name:      ComponentName,
		Namespace: ComponentNamespace,
	}})
)

// TestValidateUpdate tests the ValidateUpdate function
// GIVEN a call to ValidateUpdate
//
//	WHEN for various CM configurations
//	THEN an error is returned if anything is misconfigured
func TestValidateUpdate(t *testing.T) {
	validationTests(t, true)
}

// TestValidateInstall tests the ValidateInstall function
// GIVEN a call to ValidateInstall
//
//	WHEN for various CM configurations
//	THEN an error is returned if anything is misconfigured
func TestValidateInstall(t *testing.T) {
	validationTests(t, false)
}

// TestPostInstallCA tests the PostInstall function
// GIVEN a call to PostInstall
//
//	WHEN the cert type is CA
//	THEN no error is returned
func TestPostInstallCA(t *testing.T) {
	localvz := defaultVZConfig.DeepCopy()
	localvz.Spec.Components.CertManager.Certificate.CA = ca

	defer func() { getClientFunc = k8sutil.GetCoreV1Client }()
	getClientFunc = createClientFunc(localvz.Spec.Components.CertManager.Certificate.CA, "defaultVZConfig-cn")

	defer func() { getCMClientFunc = GetCertManagerClientset }()
	getCMClientFunc = func() (certv1client.CertmanagerV1Interface, error) {
		cmClient := certv1fake.NewSimpleClientset()
		return cmClient.CertmanagerV1(), nil
	}

	client := fake.NewClientBuilder().WithScheme(testScheme).Build()
	err := fakeComponent.PostInstall(spi.NewFakeContext(client, localvz, nil, false))
	assert.NoError(t, err)
}

// TestPostUpgradeUpdateCA tests the PostUpgrade function
// GIVEN a call to PostUpgrade
//
//	WHEN the type is CA and the CA is updated
//	THEN the ClusterIssuer is updated correctly and no error is returned
func TestPostUpgradeUpdateCA(t *testing.T) {
	runCAUpdateTest(t, true)
}

// TestPostInstallUpdateCA tests the PostInstall function
// GIVEN a call to PostInstall
//
//	WHEN the type is CA and the CA is updated
//	THEN the ClusterIssuer is updated correctly and no error is returned
func TestPostInstallUpdateCA(t *testing.T) {
	runCAUpdateTest(t, false)
}

func runCAUpdateTest(t *testing.T, upgrade bool) {
	localvz := defaultVZConfig.DeepCopy()
	localvz.Spec.Components.CertManager.Certificate.CA = ca

	updatedVZ := defaultVZConfig.DeepCopy()
	newCA := vzapi.CA{
		SecretName:               "newsecret",
		ClusterResourceNamespace: "newnamespace",
	}
	updatedVZ.Spec.Components.CertManager.Certificate.CA = newCA

	//vzCertSecret := createCertSecret("verrazzano-ca-certificate-secret", constants2.CertManagerNamespace, "defaultVZConfig-cn")
	//caSecret := createCertSecret("newsecret", "newnamespace", "defaultVZConfig-cn")
	defer func() { getClientFunc = k8sutil.GetCoreV1Client }()
	getClientFunc = createClientFunc(updatedVZ.Spec.Components.CertManager.Certificate.CA, "defaultVZConfig-cn")

	defer func() { getCMClientFunc = GetCertManagerClientset }()
	cmClient := certv1fake.NewSimpleClientset()
	getCMClientFunc = func() (certv1client.CertmanagerV1Interface, error) {
		return cmClient.CertmanagerV1(), nil
	}

	expectedIssuer := &certv1.ClusterIssuer{
		Spec: certv1.IssuerSpec{
			IssuerConfig: certv1.IssuerConfig{
				CA: &certv1.CAIssuer{
					SecretName: newCA.SecretName,
				},
			},
		},
	}

	client := fake.NewClientBuilder().WithScheme(testScheme).WithObjects(localvz).Build()
	ctx := spi.NewFakeContext(client, updatedVZ, nil, false)

	var err error
	if upgrade {
		err = fakeComponent.PostUpgrade(ctx)
	} else {
		err = fakeComponent.PostInstall(ctx)
	}
	assert.NoError(t, err)

	actualIssuer := &certv1.ClusterIssuer{}
	assert.NoError(t, client.Get(context.TODO(), types.NamespacedName{Name: verrazzanoClusterIssuerName}, actualIssuer))
	assert.Equal(t, expectedIssuer.Spec.CA, actualIssuer.Spec.CA)
}

// TestPostInstallAcme tests the PostInstall function
// GIVEN a call to PostInstall
//
//	WHEN the cert type is Acme
//	THEN no error is returned
func TestPostInstallAcme(t *testing.T) {
	localvz := defaultVZConfig.DeepCopy()
	localvz.Spec.Components.CertManager.Certificate.Acme = acme
	client := fake.NewClientBuilder().WithScheme(testScheme).Build()
	// set OCI DNS secret value and create secret
	localvz.Spec.Components.DNS = &vzapi.DNSComponent{
		OCI: &vzapi.OCI{
			OCIConfigSecret: testOCIDNSName,
			DNSZoneName:     testDNSDomain,
		},
	}
	_ = client.Create(context.TODO(), &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      testOCIDNSName,
			Namespace: ComponentNamespace,
		},
	})
	err := fakeComponent.PostInstall(spi.NewFakeContext(client, localvz, nil, false))
	assert.NoError(t, err)
}

// TestPostUpgradeAcmeUpdate tests the PostUpgrade function
// GIVEN a call to PostUpgrade
//
//	WHEN the cert type is Acme and the config has been updated
//	THEN the ClusterIssuer is updated as expected no error is returned
func TestPostUpgradeAcmeUpdate(t *testing.T) {
	runAcmeUpdateTest(t, true)
}

// TestPostInstallAcme tests the PostInstall function
// GIVEN a call to PostInstall
//
//	WHEN the cert type is Acme and the config has been updated
//	THEN the ClusterIssuer is updated as expected no error is returned
func TestPostInstallAcmeUpdate(t *testing.T) {
	runAcmeUpdateTest(t, false)
}

func runAcmeUpdateTest(t *testing.T, upgrade bool) {
	localvz := defaultVZConfig.DeepCopy()
	localvz.Spec.Components.CertManager.Certificate.Acme = acme
	// set OCI DNS secret value and create secret
	oci := &vzapi.OCI{
		OCIConfigSecret: testOCIDNSName,
		DNSZoneName:     testDNSDomain,
	}
	localvz.Spec.Components.DNS = &vzapi.DNSComponent{
		OCI: oci,
	}

	oldSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      testOCIDNSName,
			Namespace: ComponentNamespace,
		},
	}

	newSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "newociDNSSecret",
			Namespace: ComponentNamespace,
		},
	}

	existingIssuer, _ := createAcmeClusterIssuer(vzlog.DefaultLogger(), templateData{
		Email:          acme.EmailAddress,
		AcmeSecretName: caAcmeSecretName,
		Server:         acme.Environment,
		SecretName:     oci.OCIConfigSecret,
		OCIZoneName:    oci.DNSZoneName,
	})

	updatedVz := defaultVZConfig.DeepCopy()
	newAcme := vzapi.Acme{
		Provider:     "letsEncrypt",
		EmailAddress: "slbronkowitz@gmail.com",
		Environment:  "production",
	}
	newOCI := &vzapi.OCI{
		DNSZoneCompartmentOCID: "somenewocid",
		OCIConfigSecret:        newSecret.Name,
		DNSZoneName:            "newzone.dns.io",
	}
	updatedVz.Spec.Components.CertManager.Certificate.Acme = newAcme
	updatedVz.Spec.Components.DNS = &vzapi.DNSComponent{OCI: newOCI}

	expectedIssuer, _ := createAcmeClusterIssuer(vzlog.DefaultLogger(), templateData{
		Email:          newAcme.EmailAddress,
		AcmeSecretName: caAcmeSecretName,
		Server:         letsEncryptProdEndpoint,
		SecretName:     newOCI.OCIConfigSecret,
		OCIZoneName:    newOCI.DNSZoneName,
	})

	client := fake.NewClientBuilder().WithScheme(testScheme).WithObjects(localvz, oldSecret, newSecret, existingIssuer).Build()
	ctx := spi.NewFakeContext(client, updatedVz, nil, false)

	var err error
	if upgrade {
		err = fakeComponent.PostUpgrade(ctx)
	} else {
		err = fakeComponent.PostInstall(ctx)
	}
	assert.NoError(t, err)

	actualIssuer, _ := createACMEIssuerObject(ctx)
	assert.Equal(t, expectedIssuer.Object["spec"], actualIssuer.Object["spec"])
	assert.NoError(t, client.Get(context.TODO(), types.NamespacedName{Name: verrazzanoClusterIssuerName}, actualIssuer))

}

// TestClusterIssuerUpdated tests the createOrUpdateClusterIssuer function
// GIVEN a call to createOrUpdateClusterIssuer
// WHEN the ClusterIssuer is updated and there are existing certificates with failed and successful CertificateRequests
// THEN the Cert status is updated to request a renewal, and any failed CertificateRequests are cleaned up beforehand
func TestClusterIssuerUpdated(t *testing.T) {
	asserts := assert.New(t)

	localvz := defaultVZConfig.DeepCopy()
	localvz.Spec.Components.CertManager.Certificate.Acme = acme
	// set OCI DNS secret value and create secret
	oci := &vzapi.OCI{
		OCIConfigSecret: testOCIDNSName,
		DNSZoneName:     testDNSDomain,
	}
	localvz.Spec.Components.DNS = &vzapi.DNSComponent{
		OCI: oci,
	}

	// The existing cluster issuer that will be updated
	existingClusterIssuer := &certv1.ClusterIssuer{
		ObjectMeta: metav1.ObjectMeta{
			Name: verrazzanoClusterIssuerName,
		},
		Spec: certv1.IssuerSpec{
			IssuerConfig: certv1.IssuerConfig{
				CA: &certv1.CAIssuer{
					SecretName: ca.SecretName,
				},
			},
		},
	}

	// The a certificate that we expect to be renewed
	certName := "mycert"
	certNamespace := "certns"
	certificate := &certv1.Certificate{
		ObjectMeta: metav1.ObjectMeta{Name: certName, Namespace: certNamespace},
		Spec: certv1.CertificateSpec{
			IssuerRef: cmmeta.ObjectReference{
				Name: verrazzanoClusterIssuerName,
			},
			SecretName: certName,
		},
		Status: certv1.CertificateStatus{},
	}

	// A certificate request for the above cert that was successful
	certificateRequest1 := &certv1.CertificateRequest{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "foorequest1",
			Namespace: certificate.Namespace,
			Annotations: map[string]string{
				certRequestNameAnnotation: certificate.Name,
			},
		},
		Status: certv1.CertificateRequestStatus{
			Conditions: []certv1.CertificateRequestCondition{
				{Type: certv1.CertificateRequestConditionReady, Status: cmmeta.ConditionTrue, Reason: certv1.CertificateRequestReasonIssued},
			},
		},
	}

	// A certificate request for the above cert that is in a failed state; this should be deleted (or it will block an Issuing request)
	certificateRequest2 := &certv1.CertificateRequest{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "foorequest2",
			Namespace: certificate.Namespace,
			Annotations: map[string]string{
				certRequestNameAnnotation: certificate.Name,
			},
		},
		Status: certv1.CertificateRequestStatus{
			Conditions: []certv1.CertificateRequestCondition{
				{Type: certv1.CertificateRequestConditionReady, Status: cmmeta.ConditionFalse, Reason: certv1.CertificateRequestReasonFailed},
			},
		},
	}

	// An unrelated certificate request, for different certificate; this should be untouched
	otherCertRequest := &certv1.CertificateRequest{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "barrequest",
			Namespace: certificate.Namespace,
			Annotations: map[string]string{
				certRequestNameAnnotation: "someothercert",
			},
		},
		Status: certv1.CertificateRequestStatus{
			Conditions: []certv1.CertificateRequestCondition{
				{Type: certv1.CertificateRequestConditionReady, Status: cmmeta.ConditionFalse, Reason: certv1.CertificateRequestReasonFailed},
			},
		},
	}

	// The OCI DNS secret is expected to be present for this configuration
	ociSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      testOCIDNSName,
			Namespace: ComponentNamespace,
		},
	}

	// Fake controllerruntime client and ComponentContext for the call
	client := fake.NewClientBuilder().WithScheme(testScheme).WithObjects(existingClusterIssuer, certificate, ociSecret).Build()
	ctx := spi.NewFakeContext(client, localvz, nil, false)

	// Fake Go client for the CertManager clientset
	cmClient := certv1fake.NewSimpleClientset(certificate, certificateRequest1, certificateRequest2, otherCertRequest)

	defer func() { getCMClientFunc = GetCertManagerClientset }()
	getCMClientFunc = func() (certv1client.CertmanagerV1Interface, error) {
		return cmClient.CertmanagerV1(), nil
	}

	// Create an issuer
	issuerName := caCertCommonName + "-a23asdfa"
	fakeIssuerCert := createFakeCertificate(issuerName)
	fakeIssuerCertBytes, err := createFakeCertBytes(issuerName, nil)
	if err != nil {
		return
	}
	issuerSecret, err := createCertSecret(caCertificateName, ComponentNamespace, fakeIssuerCertBytes)
	if err != nil {
		return
	}
	// Create a leaf cert signed by the above issuer
	fakeCertBytes, err := createFakeCertBytes("common-name", fakeIssuerCert)
	if err != nil {
		return
	}
	certSecret, err := createCertSecret(certName, certNamespace, fakeCertBytes)
	if !asserts.NoError(err, "Error creating test cert secret") {
		return
	}
	defer func() { getClientFunc = k8sutil.GetCoreV1Client }()
	getClientFunc = func(log ...vzlog.VerrazzanoLogger) (v1.CoreV1Interface, error) {
		return k8sfake.NewSimpleClientset(certSecret, issuerSecret).CoreV1(), nil
	}

	// create the component and issue the call
	component := NewComponent().(certManagerComponent)
	asserts.NoError(component.createOrUpdateClusterIssuer(ctx))

	// Verify the certificate status has an Issuing condition; this informs CertManager to renew the certificate
	updatedCert, err := cmClient.CertmanagerV1().Certificates(certificate.Namespace).Get(context.TODO(), certificate.Name, metav1.GetOptions{})
	asserts.NoError(err)
	asserts.True(cmutil.CertificateHasCondition(updatedCert, certv1.CertificateCondition{
		Type:   certv1.CertificateConditionIssuing,
		Status: cmmeta.ConditionTrue,
	}))

	// Verify the successful CertificateRequest was NOT deleted
	certReq1, err := cmClient.CertmanagerV1().CertificateRequests(certificate.Namespace).Get(context.TODO(), certificateRequest1.Name, metav1.GetOptions{})
	asserts.NoError(err)
	asserts.NotNil(certReq1)

	// Verify the failed CertificateRequest for the target certificate WAS deleted
	certReq2, err := cmClient.CertmanagerV1().CertificateRequests(certificate.Namespace).Get(context.TODO(), certificateRequest2.Name, metav1.GetOptions{})
	asserts.Error(err)
	asserts.True(errors.IsNotFound(err))
	asserts.Nil(certReq2)

	// Verify the unrelated CertificateRequest was NOT deleted
	otherReq, err := cmClient.CertmanagerV1().CertificateRequests(certificate.Namespace).Get(context.TODO(), otherCertRequest.Name, metav1.GetOptions{})
	asserts.NoError(err)
	asserts.NotNil(otherReq)
}

// TestDryRun tests the behavior when DryRun is enabled, mainly for code coverage
// GIVEN a call to PostInstall/PostUpgrade/PreInstall
//
//	WHEN the ComponentContext has DryRun set to true
//	THEN no error is returned
func TestDryRun(t *testing.T) {
	client := fake.NewClientBuilder().WithScheme(testScheme).Build()
	ctx := spi.NewFakeContext(client, defaultVZConfig, nil, true)

	assert.NoError(t, fakeComponent.PreInstall(ctx))
	assert.NoError(t, fakeComponent.PostInstall(ctx))
	assert.NoError(t, fakeComponent.PostUpgrade(ctx))
}

func createFakeClient(objs ...runtime.Object) *k8sfake.Clientset {
	return k8sfake.NewSimpleClientset(objs...)
}

func createCertSecret(name string, namespace string, fakeCertBytes []byte) (*corev1.Secret, error) {
	//fakeCertBytes, err := createFakeCertBytes(cn)
	//if err != nil {
	//	return nil, err
	//}
	secret := &corev1.Secret{
		TypeMeta: metav1.TypeMeta{},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Data: map[string][]byte{
			corev1.TLSCertKey: fakeCertBytes,
		},
		Type: corev1.SecretTypeTLS,
	}
	return secret, nil
}

func createCertSecretNoParent(name string, namespace string, cn string) (*corev1.Secret, error) {
	fakeCertBytes, err := createFakeCertBytes(cn, nil)
	if err != nil {
		return nil, err
	}
	secret := &corev1.Secret{
		TypeMeta: metav1.TypeMeta{},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Data: map[string][]byte{
			corev1.TLSCertKey: fakeCertBytes,
		},
		Type: corev1.SecretTypeTLS,
	}
	return secret, nil
}

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

func createFakeCertBytes(cn string, parent *x509.Certificate) ([]byte, error) {
	rsaKey, err := getRSAKey()
	if err != nil {
		return []byte{}, err
	}

	cert := createFakeCertificate(cn)
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

func createFakeCertificate(cn string) *x509.Certificate {
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

// All of this below is to make Sonar happy
type validationTestStruct struct {
	name      string
	old       *vzapi.Verrazzano
	new       *vzapi.Verrazzano
	coreV1Cli func(_ ...vzlog.VerrazzanoLogger) (v1.CoreV1Interface, error)
	caSecret  *corev1.Secret
	wantErr   bool
}

var disabled = false

const emailAddress = "joeblow@foo.com"
const secretName = "newsecret"
const secretNamespace = "ns"

var tests = []validationTestStruct{
	{
		name: "enable",
		old: &vzapi.Verrazzano{
			Spec: vzapi.VerrazzanoSpec{
				Components: vzapi.ComponentSpec{
					CertManager: &vzapi.CertManagerComponent{
						Enabled: &disabled,
					},
				},
			},
		},
		new:       &vzapi.Verrazzano{},
		coreV1Cli: mockNamespaceCoreV1Client,
		wantErr:   false,
	},
	{
		name: "Cert Manager Namespace already exists",
		old: &vzapi.Verrazzano{
			Spec: vzapi.VerrazzanoSpec{
				Components: vzapi.ComponentSpec{
					CertManager: &vzapi.CertManagerComponent{
						Enabled: &disabled,
					},
				},
			},
		},
		new:       &vzapi.Verrazzano{},
		coreV1Cli: mockNamespaceWithoutLabelClient,
		wantErr:   true,
	},
	{
		name: "disable",
		old:  &vzapi.Verrazzano{},
		new: &vzapi.Verrazzano{
			Spec: vzapi.VerrazzanoSpec{
				Components: vzapi.ComponentSpec{
					CertManager: &vzapi.CertManagerComponent{
						Enabled: &disabled,
					},
				},
			},
		},
		coreV1Cli: mockNamespaceCoreV1Client,
		wantErr:   true,
	},
	{
		name:    "no change",
		old:     &vzapi.Verrazzano{},
		new:     &vzapi.Verrazzano{},
		wantErr: false,
	},
	{
		name: "updateCustomCA",
		old:  &vzapi.Verrazzano{},
		new:  getCaSecretCR(),
		caSecret: &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{Name: secretName, Namespace: secretNamespace},
		},
		wantErr: false,
	},
	{
		name:    "updateCustomCASecretNotFound",
		old:     &vzapi.Verrazzano{},
		new:     getCaSecretCR(),
		wantErr: true,
	},
	{
		name:    "no change",
		old:     &vzapi.Verrazzano{},
		new:     &vzapi.Verrazzano{},
		wantErr: false,
	},
	{
		name: "updateInvalidBothConfigured",
		old:  &vzapi.Verrazzano{},
		new: &vzapi.Verrazzano{
			Spec: vzapi.VerrazzanoSpec{
				Components: vzapi.ComponentSpec{
					CertManager: &vzapi.CertManagerComponent{
						Certificate: vzapi.Certificate{
							CA: vzapi.CA{
								SecretName:               secretName,
								ClusterResourceNamespace: secretNamespace,
							},
							Acme: vzapi.Acme{
								Provider:     vzapi.LetsEncrypt,
								EmailAddress: emailAddress,
								Environment:  "staging",
							},
						},
					},
				},
			},
		},
		wantErr: true,
	},
	{
		name:    "validLetsEncryptStaging",
		old:     &vzapi.Verrazzano{},
		new:     getAcmeCR(vzapi.LetsEncrypt, emailAddress, "staging"),
		wantErr: false,
	},
	{
		name: "validLetsEncryptProviderCaseInsensitivity",
		old:  &vzapi.Verrazzano{},
		new: &vzapi.Verrazzano{
			Spec: vzapi.VerrazzanoSpec{
				Components: vzapi.ComponentSpec{
					CertManager: &vzapi.CertManagerComponent{
						Certificate: vzapi.Certificate{
							Acme: vzapi.Acme{
								Provider:     "LETSENCRYPT",
								EmailAddress: emailAddress,
								Environment:  letsEncryptStaging,
							},
						},
					},
				},
			},
		},
		wantErr: false,
	},
	{
		name:    "validLetsEncryptStagingCaseInsensitivity",
		old:     &vzapi.Verrazzano{},
		new:     getAcmeCR(vzapi.LetsEncrypt, emailAddress, "STAGING"),
		wantErr: false,
	},
	{
		name:    "validLetsEncryptProdCaseInsensitivity",
		old:     &vzapi.Verrazzano{},
		new:     getAcmeCR(vzapi.LetsEncrypt, emailAddress, "PRODUCTION"),
		wantErr: false,
	},
	{
		name:    "validLetsEncryptDefaultStagingEnv",
		old:     &vzapi.Verrazzano{},
		new:     getAcmeCR(vzapi.LetsEncrypt, emailAddress, ""),
		wantErr: false,
	},
	{
		name:    "validLetsEncryptProd",
		old:     &vzapi.Verrazzano{},
		new:     getAcmeCR(vzapi.LetsEncrypt, emailAddress, letsencryptProduction),
		wantErr: false,
	},
	{
		name: "invalidACMEProvider",
		old:  &vzapi.Verrazzano{},
		new: &vzapi.Verrazzano{
			Spec: vzapi.VerrazzanoSpec{
				Components: vzapi.ComponentSpec{
					CertManager: &vzapi.CertManagerComponent{
						Certificate: vzapi.Certificate{
							Acme: vzapi.Acme{
								Provider:     "blah",
								EmailAddress: emailAddress,
								Environment:  letsencryptProduction,
							},
						},
					},
				},
			},
		},
		wantErr: true,
	},
	{
		name:    "invalidLetsEncryptEnv",
		old:     &vzapi.Verrazzano{},
		new:     getAcmeCR(vzapi.LetsEncrypt, emailAddress, "myenv"),
		wantErr: true,
	},
	{
		name:    "invalidACMEEmail",
		old:     &vzapi.Verrazzano{},
		new:     getAcmeCR(vzapi.LetsEncrypt, "joeblow", letsEncryptStaging),
		wantErr: true,
	},
	{
		name: "singleOverride",
		new:  getSingleOverrideCR(),
		caSecret: &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{Name: secretName, Namespace: secretNamespace},
		},
		wantErr: false,
	},
	{
		name: "multipleOverridesInOneListValue",
		new:  getMultipleOverrideCR(),
		caSecret: &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{Name: secretName, Namespace: secretNamespace},
		},
		wantErr: true,
	},
}

func validationTests(t *testing.T, isUpdate bool) {
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.name == "Cert Manager Namespace already exists" && isUpdate { // will throw error only during installation
				tt.wantErr = false
			}
			c := NewComponent()
			getClientFunc = getTestClient(tt)
			runValidationTest(t, tt, isUpdate, c)
		})
	}
}

func runValidationTest(t *testing.T, tt validationTestStruct, isUpdate bool, c spi.Component) {
	defer func() { k8sutil.GetCoreV1Func = k8sutil.GetCoreV1Client }()
	if isUpdate {
		if err := c.ValidateUpdate(tt.old, tt.new); (err != nil) != tt.wantErr {
			t.Errorf("ValidateUpdate() error = %v, wantErr %v", err, tt.wantErr)
		}
		v1beta1New := &v1beta1.Verrazzano{}
		v1beta1Old := &v1beta1.Verrazzano{}
		err := tt.new.ConvertTo(v1beta1New)
		assert.NoError(t, err)
		err = tt.old.ConvertTo(v1beta1Old)
		assert.NoError(t, err)
		if err := c.ValidateUpdateV1Beta1(v1beta1Old, v1beta1New); (err != nil) != tt.wantErr {
			t.Errorf("ValidateUpdateV1Beta1() error = %v, wantErr %v", err, tt.wantErr)
		}

	} else {
		wantErr := tt.name != "disable" && tt.wantErr // hack for disable validation, allowed on initial install but not on update
		if tt.coreV1Cli != nil {
			k8sutil.GetCoreV1Func = tt.coreV1Cli
		} else {
			k8sutil.GetCoreV1Func = common.MockGetCoreV1()
		}
		if err := c.ValidateInstall(tt.new); (err != nil) != wantErr {
			t.Errorf("ValidateInstall() error = %v, wantErr %v", err, tt.wantErr)
		}
		v1beta1Vz := &v1beta1.Verrazzano{}
		err := tt.new.ConvertTo(v1beta1Vz)
		assert.NoError(t, err)
		if err := c.ValidateInstallV1Beta1(v1beta1Vz); (err != nil) != wantErr {
			t.Errorf("ValidateInstallV1Beta1() error = %v, wantErr %v", err, tt.wantErr)
		}
	}
}

func getTestClient(tt validationTestStruct) func(log ...vzlog.VerrazzanoLogger) (v1.CoreV1Interface, error) {
	return func(log ...vzlog.VerrazzanoLogger) (v1.CoreV1Interface, error) {
		if tt.caSecret != nil {
			return createFakeClient(tt.caSecret).CoreV1(), nil
		}
		return createFakeClient().CoreV1(), nil
	}
}

func getAcmeCR(provider vzapi.ProviderType, emailAddr string, env string) *vzapi.Verrazzano {
	return &vzapi.Verrazzano{
		Spec: vzapi.VerrazzanoSpec{
			Components: vzapi.ComponentSpec{
				CertManager: &vzapi.CertManagerComponent{
					Certificate: vzapi.Certificate{
						Acme: vzapi.Acme{
							Provider:     provider,
							EmailAddress: emailAddr,
							Environment:  env,
						},
					},
				},
			},
		},
	}
}

func getCaSecretCR() *vzapi.Verrazzano {
	return &vzapi.Verrazzano{
		Spec: vzapi.VerrazzanoSpec{
			Components: vzapi.ComponentSpec{
				CertManager: &vzapi.CertManagerComponent{
					Certificate: vzapi.Certificate{
						CA: vzapi.CA{
							SecretName:               secretName,
							ClusterResourceNamespace: secretNamespace,
						},
					},
				},
			},
		},
	}
}

func getMultipleOverrideCR() *vzapi.Verrazzano {
	return &vzapi.Verrazzano{
		Spec: vzapi.VerrazzanoSpec{
			Components: vzapi.ComponentSpec{
				CertManager: &vzapi.CertManagerComponent{
					Certificate: vzapi.Certificate{
						CA: vzapi.CA{
							SecretName:               secretName,
							ClusterResourceNamespace: secretNamespace,
						},
					},
					InstallOverrides: vzapi.InstallOverrides{
						ValueOverrides: []vzapi.Overrides{
							{
								Values: &apiextensionsv1.JSON{
									Raw: []byte("certManagerCROverride"),
								},
								ConfigMapRef: &corev1.ConfigMapKeySelector{
									LocalObjectReference: corev1.LocalObjectReference{
										Name: "overrideConfigMapSecretName",
									},
									Key: "Key",
								},
							},
						},
					},
				},
			},
		},
	}
}

func getSingleOverrideCR() *vzapi.Verrazzano {
	return &vzapi.Verrazzano{
		Spec: vzapi.VerrazzanoSpec{
			Components: vzapi.ComponentSpec{
				CertManager: &vzapi.CertManagerComponent{
					Certificate: vzapi.Certificate{
						CA: vzapi.CA{
							SecretName:               secretName,
							ClusterResourceNamespace: secretNamespace,
						},
					},
					InstallOverrides: vzapi.InstallOverrides{
						ValueOverrides: []vzapi.Overrides{
							{
								Values: &apiextensionsv1.JSON{
									Raw: []byte("certManagerCROverride"),
								},
							},
						},
					},
				},
			},
		},
	}
}
