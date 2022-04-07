// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package certmanager

import (
	"context"
	cmutil "github.com/jetstack/cert-manager/pkg/api/util"
	certv1 "github.com/jetstack/cert-manager/pkg/apis/certmanager/v1"
	cmmeta "github.com/jetstack/cert-manager/pkg/apis/meta/v1"
	certv1fake "github.com/jetstack/cert-manager/pkg/client/clientset/versioned/fake"
	certv1client "github.com/jetstack/cert-manager/pkg/client/clientset/versioned/typed/certmanager/v1"
	"github.com/stretchr/testify/assert"
	"github.com/verrazzano/verrazzano/pkg/k8sutil"
	"github.com/verrazzano/verrazzano/pkg/log/vzlog"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	v1 "k8s.io/client-go/kubernetes/typed/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"testing"

	k8sfake "k8s.io/client-go/kubernetes/fake"

	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
)

func Test_certManagerComponent_ValidateUpdate(t *testing.T) {
	disabled := false
	const emailAddress = "joeblow@foo.com"
	const secretName = "newsecret"
	const secretNamespace = "ns"
	var tests = []struct {
		name     string
		old      *vzapi.Verrazzano
		new      *vzapi.Verrazzano
		caSecret *corev1.Secret
		wantErr  bool
	}{
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
			new:     &vzapi.Verrazzano{},
			wantErr: false,
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
			wantErr: true,
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
			new: &vzapi.Verrazzano{
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
			},
			caSecret: &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{Name: secretName, Namespace: secretNamespace},
			},
			wantErr: false,
		},
		{
			name: "updateCustomCASecretNotFound",
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
							},
						},
					},
				},
			},
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
			name: "validLetsEncryptStaging",
			old:  &vzapi.Verrazzano{},
			new: &vzapi.Verrazzano{
				Spec: vzapi.VerrazzanoSpec{
					Components: vzapi.ComponentSpec{
						CertManager: &vzapi.CertManagerComponent{
							Certificate: vzapi.Certificate{
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
			name: "validLetsEncryptStagingCaseInsensitivity",
			old:  &vzapi.Verrazzano{},
			new: &vzapi.Verrazzano{
				Spec: vzapi.VerrazzanoSpec{
					Components: vzapi.ComponentSpec{
						CertManager: &vzapi.CertManagerComponent{
							Certificate: vzapi.Certificate{
								Acme: vzapi.Acme{
									Provider:     vzapi.LetsEncrypt,
									EmailAddress: emailAddress,
									Environment:  "STAGING",
								},
							},
						},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "validLetsEncryptProdCaseInsensitivity",
			old:  &vzapi.Verrazzano{},
			new: &vzapi.Verrazzano{
				Spec: vzapi.VerrazzanoSpec{
					Components: vzapi.ComponentSpec{
						CertManager: &vzapi.CertManagerComponent{
							Certificate: vzapi.Certificate{
								Acme: vzapi.Acme{
									Provider:     vzapi.LetsEncrypt,
									EmailAddress: emailAddress,
									Environment:  "PRODUCTION",
								},
							},
						},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "validLetsEncryptDefaultStagingEnv",
			old:  &vzapi.Verrazzano{},
			new: &vzapi.Verrazzano{
				Spec: vzapi.VerrazzanoSpec{
					Components: vzapi.ComponentSpec{
						CertManager: &vzapi.CertManagerComponent{
							Certificate: vzapi.Certificate{
								Acme: vzapi.Acme{
									Provider:     vzapi.LetsEncrypt,
									EmailAddress: emailAddress,
								},
							},
						},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "validLetsEncryptProd",
			old:  &vzapi.Verrazzano{},
			new: &vzapi.Verrazzano{
				Spec: vzapi.VerrazzanoSpec{
					Components: vzapi.ComponentSpec{
						CertManager: &vzapi.CertManagerComponent{
							Certificate: vzapi.Certificate{
								Acme: vzapi.Acme{
									Provider:     vzapi.LetsEncrypt,
									EmailAddress: emailAddress,
									Environment:  letsencryptProduction,
								},
							},
						},
					},
				},
			},
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
			name: "invalidLetsEncryptEnv",
			old:  &vzapi.Verrazzano{},
			new: &vzapi.Verrazzano{
				Spec: vzapi.VerrazzanoSpec{
					Components: vzapi.ComponentSpec{
						CertManager: &vzapi.CertManagerComponent{
							Certificate: vzapi.Certificate{
								Acme: vzapi.Acme{
									Provider:     vzapi.LetsEncrypt,
									EmailAddress: emailAddress,
									Environment:  "myenv",
								},
							},
						},
					},
				},
			},
			wantErr: true,
		},
		{
			name: "invalidACMEEmail",
			old:  &vzapi.Verrazzano{},
			new: &vzapi.Verrazzano{
				Spec: vzapi.VerrazzanoSpec{
					Components: vzapi.ComponentSpec{
						CertManager: &vzapi.CertManagerComponent{
							Certificate: vzapi.Certificate{
								Acme: vzapi.Acme{
									Provider:     vzapi.LetsEncrypt,
									EmailAddress: "joeblow",
									Environment:  letsEncryptStaging,
								},
							},
						},
					},
				},
			},
			wantErr: true,
		},
	}

	defer func() { getClientFunc = k8sutil.GetCoreV1Client }()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := NewComponent()
			getClientFunc = func(log ...vzlog.VerrazzanoLogger) (v1.CoreV1Interface, error) {
				if tt.caSecret != nil {
					return createFakeClient(tt.caSecret).CoreV1(), nil
				}
				return createFakeClient().CoreV1(), nil
			}
			if err := c.ValidateUpdate(tt.old, tt.new); (err != nil) != tt.wantErr {
				t.Errorf("ValidateUpdate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// TestPostInstallCA tests the PostInstall function
// GIVEN a call to PostInstall
//  WHEN the cert type is CA
//  THEN no error is returned
func TestPostInstallCA(t *testing.T) {
	localvz := vz.DeepCopy()
	localvz.Spec.Components.CertManager.Certificate.CA = ca

	defer func() { getClientFunc = k8sutil.GetCoreV1Client }()
	getClientFunc = createClientFunc(localvz.Spec.Components.CertManager.Certificate.CA)

	client := fake.NewFakeClientWithScheme(testScheme)
	err := fakeComponent.PostInstall(spi.NewFakeContext(client, localvz, false))
	assert.NoError(t, err)
}

// TestPostUpgradeUpdateCA tests the PostUpgrade function
// GIVEN a call to PostUpgrade
//  WHEN the type is CA and the CA is updated
//  THEN the ClusterIssuer is updated correctly and no error is returned
func TestPostUpgradeUpdateCA(t *testing.T) {
	runCAUpdateTest(t, true)
}

// TestPostInstallUpdateCA tests the PostInstall function
// GIVEN a call to PostInstall
//  WHEN the type is CA and the CA is updated
//  THEN the ClusterIssuer is updated correctly and no error is returned
func TestPostInstallUpdateCA(t *testing.T) {
	runCAUpdateTest(t, false)
}

func runCAUpdateTest(t *testing.T, upgrade bool) {
	localvz := vz.DeepCopy()
	localvz.Spec.Components.CertManager.Certificate.CA = ca

	updatedVZ := vz.DeepCopy()
	newCA := vzapi.CA{
		SecretName:               "newsecret",
		ClusterResourceNamespace: "newnamespace",
	}
	updatedVZ.Spec.Components.CertManager.Certificate.CA = newCA

	defer func() { getClientFunc = k8sutil.GetCoreV1Client }()
	getClientFunc = createClientFunc(updatedVZ.Spec.Components.CertManager.Certificate.CA)

	expectedIssuer := &certv1.ClusterIssuer{
		Spec: certv1.IssuerSpec{
			IssuerConfig: certv1.IssuerConfig{
				CA: &certv1.CAIssuer{
					SecretName: newCA.SecretName,
				},
			},
		},
	}

	client := fake.NewFakeClientWithScheme(testScheme, localvz)
	ctx := spi.NewFakeContext(client, updatedVZ, false)

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
//  WHEN the cert type is Acme
//  THEN no error is returned
func TestPostInstallAcme(t *testing.T) {
	localvz := vz.DeepCopy()
	localvz.Spec.Components.CertManager.Certificate.Acme = acme
	client := fake.NewFakeClientWithScheme(testScheme)
	// set OCI DNS secret value and create secret
	localvz.Spec.Components.DNS = &vzapi.DNSComponent{
		OCI: &vzapi.OCI{
			OCIConfigSecret: "ociDNSSecret",
			DNSZoneName:     "example.dns.io",
		},
	}
	client.Create(context.TODO(), &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "ociDNSSecret",
			Namespace: ComponentNamespace,
		},
	})
	err := fakeComponent.PostInstall(spi.NewFakeContext(client, localvz, false))
	assert.NoError(t, err)
}

// TestPostUpgradeAcmeUpdate tests the PostUpgrade function
// GIVEN a call to PostUpgrade
//  WHEN the cert type is Acme and the config has been updated
//  THEN the ClusterIssuer is updated as expected no error is returned
func TestPostUpgradeAcmeUpdate(t *testing.T) {
	runAcmeUpdateTest(t, true)
}

// TestPostInstallAcme tests the PostInstall function
// GIVEN a call to PostInstall
//  WHEN the cert type is Acme and the config has been updated
//  THEN the ClusterIssuer is updated as expected no error is returned
func TestPostInstallAcmeUpdate(t *testing.T) {
	runAcmeUpdateTest(t, false)
}

func runAcmeUpdateTest(t *testing.T, upgrade bool) {
	localvz := vz.DeepCopy()
	localvz.Spec.Components.CertManager.Certificate.Acme = acme
	// set OCI DNS secret value and create secret
	oci := &vzapi.OCI{
		OCIConfigSecret: "ociDNSSecret",
		DNSZoneName:     "example.dns.io",
	}
	localvz.Spec.Components.DNS = &vzapi.DNSComponent{
		OCI: oci,
	}

	oldSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "ociDNSSecret",
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
		Email:       acme.EmailAddress,
		Server:      acme.Environment,
		SecretName:  oci.OCIConfigSecret,
		OCIZoneName: oci.DNSZoneName,
	})

	updatedVz := vz.DeepCopy()
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
		Email:       newAcme.EmailAddress,
		Server:      letsEncryptProdEndpoint,
		SecretName:  newOCI.OCIConfigSecret,
		OCIZoneName: newOCI.DNSZoneName,
	})

	client := fake.NewFakeClientWithScheme(testScheme, localvz, oldSecret, newSecret, existingIssuer)
	ctx := spi.NewFakeContext(client, updatedVz, false)

	var err error
	if upgrade {
		err = fakeComponent.PostInstall(ctx)
	} else {
		err = fakeComponent.PostInstall(ctx)
	}
	assert.NoError(t, err)

	actualIssuer, _ := createAcmeClusterIssuer(vzlog.DefaultLogger(), templateData{})
	assert.NoError(t, client.Get(context.TODO(), types.NamespacedName{Name: verrazzanoClusterIssuerName}, actualIssuer))
	assert.Equal(t, expectedIssuer.Object["spec"], actualIssuer.Object["spec"])
}

// TestClusterIssuerUpdated tests the createOrUpdateClusterIssuer function
// GIVEN a call to createOrUpdateClusterIssuer
// WHEN the ClusterIssuer is updated and there are existing certificates with failed and successful CertificateRequests
// THEN the Cert status is updated to request a renewal, and any failed CertificateRequests are cleaned up beforehand
func TestClusterIssuerUpdated(t *testing.T) {
	asserts := assert.New(t)

	localvz := vz.DeepCopy()
	localvz.Spec.Components.CertManager.Certificate.Acme = acme
	// set OCI DNS secret value and create secret
	oci := &vzapi.OCI{
		OCIConfigSecret: "ociDNSSecret",
		DNSZoneName:     "example.dns.io",
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
	certificate := &certv1.Certificate{
		ObjectMeta: metav1.ObjectMeta{Name: "mycert", Namespace: "certns"},
		Spec: certv1.CertificateSpec{
			IssuerRef: cmmeta.ObjectReference{
				Name: verrazzanoClusterIssuerName,
			},
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
			Name:      "ociDNSSecret",
			Namespace: ComponentNamespace,
		},
	}

	// Fake controllerruntime client and ComponentContext for the call
	client := fake.NewFakeClientWithScheme(testScheme, existingClusterIssuer, certificate, ociSecret)
	ctx := spi.NewFakeContext(client, localvz, false)

	// Fake Go client for the CertManager clientset
	cmClient := certv1fake.NewSimpleClientset(certificate, certificateRequest1, certificateRequest2, otherCertRequest)

	defer func() { getCMClientFunc = k8sutil.GetCertManagerClientset }()
	getCMClientFunc = func(...string) (certv1client.CertmanagerV1Interface, error) {
		return cmClient.CertmanagerV1(), nil
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
//  WHEN the ComponentContext has DryRun set to true
//  THEN no error is returned
func TestDryRun(t *testing.T) {
	client := fake.NewFakeClientWithScheme(testScheme)
	ctx := spi.NewFakeContext(client, vz, true)

	assert.NoError(t, fakeComponent.PreInstall(ctx))
	assert.NoError(t, fakeComponent.PostInstall(ctx))
	assert.NoError(t, fakeComponent.PostUpgrade(ctx))
}

func createFakeClient(objs ...runtime.Object) *k8sfake.Clientset {
	return k8sfake.NewSimpleClientset(objs...)
}
