// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package config

import (
	"github.com/stretchr/testify/assert"
	"github.com/verrazzano/verrazzano/pkg/k8sutil"
	"github.com/verrazzano/verrazzano/pkg/log/vzlog"
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1beta1"
	cmcommon "github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/certmanager/common"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	k8sfake "k8s.io/client-go/kubernetes/fake"
	v1 "k8s.io/client-go/kubernetes/typed/core/v1"
	"testing"
)

func createFakeClient(objs ...runtime.Object) *k8sfake.Clientset {
	return k8sfake.NewSimpleClientset(objs...)
}

const emailAddress = "joeblow@foo.com"
const secretName = "newsecret"
const secretNamespace = "ns"

var simpleValidationTests = []validationTestStruct{
	{
		name:    "No OCI DNS or CertManager present",
		old:     &vzapi.Verrazzano{},
		new:     &vzapi.Verrazzano{},
		wantErr: false,
	},
	{
		name: "CertManager and OCI DNS webhook enabled",
		old:  &vzapi.Verrazzano{},
		new: &vzapi.Verrazzano{
			Spec: vzapi.VerrazzanoSpec{
				Components: vzapi.ComponentSpec{
					CertManager: &vzapi.CertManagerComponent{
						Enabled: getBoolPtr(true),
					},
					ClusterIssuer: &vzapi.ClusterIssuerComponent{
						IssuerConfig: vzapi.IssuerConfig{
							LetsEncrypt: &vzapi.LetsEncryptACMEIssuer{
								EmailAddress: emailAddress,
								Environment:  letsEncryptStaging,
							},
						},
					},
					DNS: &vzapi.DNSComponent{
						OCI: &vzapi.OCI{
							DNSScope:               "GLOBAL",
							DNSZoneCompartmentOCID: "ocid",
							DNSZoneOCID:            "zoneOcid",
							DNSZoneName:            "zoneName",
							OCIConfigSecret:        "oci",
						},
					},
				},
			},
		},
		wantErr: false,
	},
	{
		name: "CertManager Disabled and OCI DNS webhook enabled",
		old:  &vzapi.Verrazzano{},
		new: &vzapi.Verrazzano{
			Spec: vzapi.VerrazzanoSpec{
				Components: vzapi.ComponentSpec{
					CertManager: &vzapi.CertManagerComponent{
						Enabled: getBoolPtr(false),
					},
					ClusterIssuer: &vzapi.ClusterIssuerComponent{
						IssuerConfig: vzapi.IssuerConfig{
							LetsEncrypt: &vzapi.LetsEncryptACMEIssuer{
								EmailAddress: emailAddress,
								Environment:  letsEncryptStaging,
							},
						},
					},
					DNS: &vzapi.DNSComponent{
						OCI: &vzapi.OCI{
							DNSScope:               "GLOBAL",
							DNSZoneCompartmentOCID: "ocid",
							DNSZoneOCID:            "zoneOcid",
							DNSZoneName:            "zoneName",
							OCIConfigSecret:        "oci",
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
	//{
	//	name: "CertManager and ClusterIssuer both explicitly configured",
	//	old:  &vzapi.Verrazzano{},
	//	new: &vzapi.Verrazzano{
	//		Spec: vzapi.VerrazzanoSpec{
	//			Components: vzapi.ComponentSpec{
	//				CertManager: &vzapi.CertManagerComponent{
	//					Enabled: getBoolPtr(false),
	//					Certificate: vzapi.Certificate{
	//						CA: vzapi.CA{
	//							ClusterResourceNamespace: secretNamespace,
	//							SecretName:               secretName,
	//						},
	//					},
	//				},
	//				ClusterIssuer: &vzapi.ClusterIssuerComponent{
	//					IssuerConfig: vzapi.IssuerConfig{
	//						LetsEncrypt: &vzapi.LetsEncryptACMEIssuer{
	//							EmailAddress: emailAddress,
	//							Environment:  letsEncryptStaging,
	//						},
	//					},
	//				},
	//				DNS: &vzapi.DNSComponent{
	//					OCI: &vzapi.OCI{
	//						DNSScope:               "GLOBAL",
	//						DNSZoneCompartmentOCID: "ocid",
	//						DNSZoneOCID:            "zoneOcid",
	//						DNSZoneName:            "zoneName",
	//						OCIConfigSecret:        "oci",
	//					},
	//				},
	//			},
	//		},
	//	},
	//	caSecret: &corev1.Secret{
	//		ObjectMeta: metav1.ObjectMeta{Name: secretName, Namespace: secretNamespace},
	//	},
	//	wantErr: true,
	//},
	{
		name: "CertManager Component Custom CA",
		old:  &vzapi.Verrazzano{},
		new:  getCaSecretCR(),
		caSecret: &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{Name: secretName, Namespace: secretNamespace},
		},
		wantErr: false,
	},
}

var issuerConfigurationTests = []validationTestStruct{
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
		name: "updateInvalidCertificateBothConfigured",
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

func validationTests(t *testing.T, isUpdate bool) {
	defer func() { k8sutil.ResetGetAPIExtV1ClientFunc() }()

	tests := simpleValidationTests
	tests = append(tests, issuerConfigurationTests...)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.name == "Cert Manager Namespace already exists" && isUpdate { // will throw error only during installation
				tt.wantErr = false
			}
			c := NewComponent()
			cmcommon.GetClientFunc = getTestClient(tt)
			runValidationTest(t, tt, isUpdate, c)
		})
	}
}

func runValidationTest(t *testing.T, tt validationTestStruct, isUpdate bool, c spi.Component) {
	//	defer func() { k8sutil.GetCoreV1Func = k8sutil.GetCoreV1Client }()
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
			cmcommon.GetClientFunc = tt.coreV1Cli
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
				ClusterIssuer: &vzapi.ClusterIssuerComponent{
					ClusterResourceNamespace: secretNamespace,
					IssuerConfig: vzapi.IssuerConfig{
						CA: &vzapi.CAIssuer{
							SecretName: secretName,
						},
					},
				},
			},
		},
	}
}
