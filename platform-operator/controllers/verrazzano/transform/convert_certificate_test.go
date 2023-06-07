// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package transform

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/verrazzano/verrazzano/pkg/constants"
	"github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1beta1"
)

const (
	emailAddress    = "foo@bar.com"
	environmentType = "staging"
	customNamespace = "myns"
	customSecret    = "mySecret"
)

var (
	nonDefaultCA = v1alpha1.Certificate{
		CA: v1alpha1.CA{
			ClusterResourceNamespace: customNamespace,
			SecretName:               customSecret,
		},
	}

	nonDefaultCAV1Beta1 = v1beta1.Certificate{
		CA: v1beta1.CA{
			ClusterResourceNamespace: customNamespace,
			SecretName:               customSecret,
		},
	}
)

// TestConvertCertificateToClusterIssuerV1Beta1 tests the convertCertificateToClusterIssuerV1Beta1 function
// GIVEN a call to convertCertificateToClusterIssuerV1Beta1
// THEN the appropriate conversions from the deprecated Certificate object to the ClusterIssuer Component
func TestConvertCertificateToClusterIssuerV1Beta1(t *testing.T) {
	asserts := assert.New(t)
	for _, tt := range v1beta1Tests {
		t.Run(tt.testName, func(t *testing.T) {
			t.Logf("Running test %s", tt.testName)
			vz := &v1beta1.Verrazzano{
				Spec: v1beta1.VerrazzanoSpec{
					Components: v1beta1.ComponentSpec{
						CertManager:   tt.certConfig,
						ClusterIssuer: tt.issuerConfig,
					},
				},
			}
			err := convertCertificateToClusterIssuerV1Beta1(vz)
			if tt.expectErr {
				asserts.Error(err)
			} else {
				asserts.NoError(err)
				asserts.Equal(tt.expectedIssuerConfig, vz.Spec.Components.ClusterIssuer)
			}
		})
	}
}

// TestConvertCertificateToClusterIssuerV1Alpha1 tests the convertCertificateToClusterIssuerV1Alpha1 function
// GIVEN a call to convertCertificateToClusterIssuerV1Beta1
// THEN the appropriate conversions from the deprecated Certificate object to the ClusterIssuer Component
func TestConvertCertificateToClusterIssuerV1Alpha1(t *testing.T) {
	asserts := assert.New(t)
	for _, tt := range v1alpha1Tests {
		t.Run(tt.testName, func(t *testing.T) {
			t.Logf("Running test %s", tt.testName)
			vz := &v1alpha1.Verrazzano{
				Spec: v1alpha1.VerrazzanoSpec{
					Components: v1alpha1.ComponentSpec{
						CertManager:   tt.certConfig,
						ClusterIssuer: tt.issuerConfig,
					},
				},
			}
			err := convertCertificateToClusterIssuerV1Alpha1(vz)
			if tt.expectErr {
				asserts.Error(err)
			} else {
				asserts.NoError(err)
				asserts.Equal(tt.expectedIssuerConfig, vz.Spec.Components.ClusterIssuer)
			}
		})
	}
}

var v1alpha1Tests = []struct {
	testName             string
	certConfig           *v1alpha1.CertManagerComponent
	issuerConfig         *v1alpha1.ClusterIssuerComponent
	expectErr            bool
	expectedIssuerConfig *v1alpha1.ClusterIssuerComponent
}{
	{
		testName:             "No CM or ClusterIssuer",
		certConfig:           nil,
		issuerConfig:         nil,
		expectedIssuerConfig: v1alpha1.NewDefaultClusterIssuer(),
		expectErr:            false,
	},
	{
		testName:             "Empty CM Certificate nil ClusterIssuer",
		certConfig:           &v1alpha1.CertManagerComponent{},
		issuerConfig:         nil,
		expectedIssuerConfig: v1alpha1.NewDefaultClusterIssuer(),
		expectErr:            false,
	},
	{
		testName:             "Default CM Certificate nil ClusterIssuer",
		certConfig:           &v1alpha1.CertManagerComponent{Certificate: defaultCertConfigV1Alpha1},
		issuerConfig:         nil,
		expectedIssuerConfig: v1alpha1.NewDefaultClusterIssuer(),
		expectErr:            false,
	},
	{
		testName:             "Empty CM Certificate default ClusterIssuer",
		certConfig:           &v1alpha1.CertManagerComponent{},
		issuerConfig:         v1alpha1.NewDefaultClusterIssuer(),
		expectedIssuerConfig: v1alpha1.NewDefaultClusterIssuer(),
		expectErr:            false,
	},
	{
		testName:             "Neither Certificate field configured",
		certConfig:           &v1alpha1.CertManagerComponent{},
		issuerConfig:         v1alpha1.NewDefaultClusterIssuer(),
		expectedIssuerConfig: v1alpha1.NewDefaultClusterIssuer(),
		expectErr:            false,
	},
	{
		testName: "Non Default CA Certificate",
		certConfig: &v1alpha1.CertManagerComponent{
			Certificate: nonDefaultCA,
		},
		issuerConfig: v1alpha1.NewDefaultClusterIssuer(),
		expectedIssuerConfig: &v1alpha1.ClusterIssuerComponent{
			ClusterResourceNamespace: customNamespace,
			IssuerConfig: v1alpha1.IssuerConfig{
				CA: &v1alpha1.CAIssuer{SecretName: customSecret},
			},
		},
		expectErr: false,
	},
	{
		testName: "Non Default CA Certificate nil ClusterIssuer",
		certConfig: &v1alpha1.CertManagerComponent{
			Certificate: nonDefaultCA,
		},
		issuerConfig: nil,
		expectedIssuerConfig: &v1alpha1.ClusterIssuerComponent{
			ClusterResourceNamespace: customNamespace,
			IssuerConfig: v1alpha1.IssuerConfig{
				CA: &v1alpha1.CAIssuer{SecretName: customSecret},
			},
		},
		expectErr: false,
	},
	{
		testName: "Non Default CA Certificate",
		certConfig: &v1alpha1.CertManagerComponent{
			Certificate: nonDefaultCA,
		},
		issuerConfig: v1alpha1.NewDefaultClusterIssuer(),
		expectedIssuerConfig: &v1alpha1.ClusterIssuerComponent{
			ClusterResourceNamespace: customNamespace,
			IssuerConfig: v1alpha1.IssuerConfig{
				CA: &v1alpha1.CAIssuer{SecretName: customSecret},
			},
		},
		expectErr: false,
	},
	{
		testName: "LetsEncrypt Certificate",
		certConfig: &v1alpha1.CertManagerComponent{
			Certificate: v1alpha1.Certificate{
				Acme: v1alpha1.Acme{
					EmailAddress: emailAddress,
					Environment:  environmentType,
					Provider:     v1alpha1.LetsEncrypt,
				},
			},
		},
		issuerConfig: v1alpha1.NewDefaultClusterIssuer(),
		expectedIssuerConfig: &v1alpha1.ClusterIssuerComponent{
			ClusterResourceNamespace: constants.CertManagerNamespace,
			IssuerConfig: v1alpha1.IssuerConfig{
				LetsEncrypt: &v1alpha1.LetsEncryptACMEIssuer{
					EmailAddress: emailAddress,
					Environment:  environmentType,
				},
			},
		},
		expectErr: false,
	},
	{
		testName:   "LetsEncrypt ClusterIssuer",
		certConfig: &v1alpha1.CertManagerComponent{Certificate: defaultCertConfigV1Alpha1},
		issuerConfig: &v1alpha1.ClusterIssuerComponent{
			ClusterResourceNamespace: constants.CertManagerNamespace,
			IssuerConfig: v1alpha1.IssuerConfig{
				LetsEncrypt: &v1alpha1.LetsEncryptACMEIssuer{
					EmailAddress: emailAddress,
					Environment:  environmentType,
				},
			},
		},
		expectedIssuerConfig: &v1alpha1.ClusterIssuerComponent{
			ClusterResourceNamespace: constants.CertManagerNamespace,
			IssuerConfig: v1alpha1.IssuerConfig{
				LetsEncrypt: &v1alpha1.LetsEncryptACMEIssuer{
					EmailAddress: emailAddress,
					Environment:  environmentType,
				},
			},
		},
		expectErr: false,
	},
	{
		testName:   "CA ClusterIssuer",
		certConfig: &v1alpha1.CertManagerComponent{Certificate: defaultCertConfigV1Alpha1},
		issuerConfig: &v1alpha1.ClusterIssuerComponent{
			ClusterResourceNamespace: customNamespace,
			IssuerConfig: v1alpha1.IssuerConfig{
				CA: &v1alpha1.CAIssuer{SecretName: customSecret},
			},
		},
		expectedIssuerConfig: &v1alpha1.ClusterIssuerComponent{
			ClusterResourceNamespace: customNamespace,
			IssuerConfig: v1alpha1.IssuerConfig{
				CA: &v1alpha1.CAIssuer{SecretName: customSecret},
			},
		},
		expectErr: false,
	},
	{
		testName: "Illegal Certificate",
		certConfig: &v1alpha1.CertManagerComponent{
			Certificate: v1alpha1.Certificate{
				CA: v1alpha1.CA{
					ClusterResourceNamespace: customNamespace,
					SecretName:               customSecret,
				},
				Acme: v1alpha1.Acme{
					EmailAddress: emailAddress,
					Environment:  environmentType,
					Provider:     v1alpha1.LetsEncrypt,
				},
			},
		},
		issuerConfig:         v1alpha1.NewDefaultClusterIssuer(),
		expectedIssuerConfig: v1alpha1.NewDefaultClusterIssuer(),
		expectErr:            false,
	},
	{
		testName:   "Both Certificate and ClusterIssuer Configured",
		certConfig: &v1alpha1.CertManagerComponent{Certificate: nonDefaultCA},
		issuerConfig: &v1alpha1.ClusterIssuerComponent{
			ClusterResourceNamespace: customNamespace,
			IssuerConfig: v1alpha1.IssuerConfig{
				CA: &v1alpha1.CAIssuer{SecretName: customSecret},
			},
		},
		expectedIssuerConfig: &v1alpha1.ClusterIssuerComponent{
			ClusterResourceNamespace: customNamespace,
			IssuerConfig: v1alpha1.IssuerConfig{
				CA: &v1alpha1.CAIssuer{SecretName: customSecret},
			},
		},
		expectErr: true,
	},
}

var v1beta1Tests = []struct {
	testName             string
	certConfig           *v1beta1.CertManagerComponent
	issuerConfig         *v1beta1.ClusterIssuerComponent
	expectErr            bool
	expectedIssuerConfig *v1beta1.ClusterIssuerComponent
}{
	{
		testName:             "No CM or ClusterIssuer",
		certConfig:           nil,
		issuerConfig:         nil,
		expectedIssuerConfig: v1beta1.NewDefaultClusterIssuer(),
		expectErr:            false,
	},
	{
		testName:             "Empty CM Certificate nil ClusterIssuer",
		certConfig:           &v1beta1.CertManagerComponent{},
		issuerConfig:         nil,
		expectedIssuerConfig: v1beta1.NewDefaultClusterIssuer(),
		expectErr:            false,
	},
	{
		testName:             "Default CM Certificate nil ClusterIssuer",
		certConfig:           &v1beta1.CertManagerComponent{Certificate: defaultCertConfigV1Beta1},
		issuerConfig:         nil,
		expectedIssuerConfig: v1beta1.NewDefaultClusterIssuer(),
		expectErr:            false,
	},
	{
		testName:             "Empty CM Certificate default ClusterIssuer",
		certConfig:           &v1beta1.CertManagerComponent{},
		issuerConfig:         v1beta1.NewDefaultClusterIssuer(),
		expectedIssuerConfig: v1beta1.NewDefaultClusterIssuer(),
		expectErr:            false,
	},
	{
		testName:             "Neither Certificate field configured",
		certConfig:           &v1beta1.CertManagerComponent{},
		issuerConfig:         v1beta1.NewDefaultClusterIssuer(),
		expectedIssuerConfig: v1beta1.NewDefaultClusterIssuer(),
		expectErr:            false,
	},
	{
		testName: "Non Default CA Certificate",
		certConfig: &v1beta1.CertManagerComponent{
			Certificate: nonDefaultCAV1Beta1,
		},
		issuerConfig: v1beta1.NewDefaultClusterIssuer(),
		expectedIssuerConfig: &v1beta1.ClusterIssuerComponent{
			ClusterResourceNamespace: customNamespace,
			IssuerConfig: v1beta1.IssuerConfig{
				CA: &v1beta1.CAIssuer{SecretName: customSecret},
			},
		},
		expectErr: false,
	},
	{
		testName: "Non Default CA Certificate nil ClusterIssuer",
		certConfig: &v1beta1.CertManagerComponent{
			Certificate: nonDefaultCAV1Beta1,
		},
		issuerConfig: nil,
		expectedIssuerConfig: &v1beta1.ClusterIssuerComponent{
			ClusterResourceNamespace: customNamespace,
			IssuerConfig: v1beta1.IssuerConfig{
				CA: &v1beta1.CAIssuer{SecretName: customSecret},
			},
		},
		expectErr: false,
	},
	{
		testName: "Non Default CA Certificate",
		certConfig: &v1beta1.CertManagerComponent{
			Certificate: nonDefaultCAV1Beta1,
		},
		issuerConfig: v1beta1.NewDefaultClusterIssuer(),
		expectedIssuerConfig: &v1beta1.ClusterIssuerComponent{
			ClusterResourceNamespace: customNamespace,
			IssuerConfig: v1beta1.IssuerConfig{
				CA: &v1beta1.CAIssuer{SecretName: customSecret},
			},
		},
		expectErr: false,
	},
	{
		testName: "LetsEncrypt Certificate",
		certConfig: &v1beta1.CertManagerComponent{
			Certificate: v1beta1.Certificate{
				Acme: v1beta1.Acme{
					EmailAddress: emailAddress,
					Environment:  environmentType,
					Provider:     v1beta1.LetsEncrypt,
				},
			},
		},
		issuerConfig: v1beta1.NewDefaultClusterIssuer(),
		expectedIssuerConfig: &v1beta1.ClusterIssuerComponent{
			ClusterResourceNamespace: constants.CertManagerNamespace,
			IssuerConfig: v1beta1.IssuerConfig{
				LetsEncrypt: &v1beta1.LetsEncryptACMEIssuer{
					EmailAddress: emailAddress,
					Environment:  environmentType,
				},
			},
		},
		expectErr: false,
	},
	{
		testName:   "LetsEncrypt ClusterIssuer",
		certConfig: &v1beta1.CertManagerComponent{Certificate: defaultCertConfigV1Beta1},
		issuerConfig: &v1beta1.ClusterIssuerComponent{
			ClusterResourceNamespace: constants.CertManagerNamespace,
			IssuerConfig: v1beta1.IssuerConfig{
				LetsEncrypt: &v1beta1.LetsEncryptACMEIssuer{
					EmailAddress: emailAddress,
					Environment:  environmentType,
				},
			},
		},
		expectedIssuerConfig: &v1beta1.ClusterIssuerComponent{
			ClusterResourceNamespace: constants.CertManagerNamespace,
			IssuerConfig: v1beta1.IssuerConfig{
				LetsEncrypt: &v1beta1.LetsEncryptACMEIssuer{
					EmailAddress: emailAddress,
					Environment:  environmentType,
				},
			},
		},
		expectErr: false,
	},
	{
		testName:   "CA ClusterIssuer",
		certConfig: &v1beta1.CertManagerComponent{Certificate: defaultCertConfigV1Beta1},
		issuerConfig: &v1beta1.ClusterIssuerComponent{
			ClusterResourceNamespace: customNamespace,
			IssuerConfig: v1beta1.IssuerConfig{
				CA: &v1beta1.CAIssuer{SecretName: customSecret},
			},
		},
		expectedIssuerConfig: &v1beta1.ClusterIssuerComponent{
			ClusterResourceNamespace: customNamespace,
			IssuerConfig: v1beta1.IssuerConfig{
				CA: &v1beta1.CAIssuer{SecretName: customSecret},
			},
		},
		expectErr: false,
	},
	{
		testName: "Illegal Certificate",
		certConfig: &v1beta1.CertManagerComponent{
			Certificate: v1beta1.Certificate{
				CA: v1beta1.CA{
					ClusterResourceNamespace: customNamespace,
					SecretName:               customSecret,
				},
				Acme: v1beta1.Acme{
					EmailAddress: emailAddress,
					Environment:  environmentType,
					Provider:     v1beta1.LetsEncrypt,
				},
			},
		},
		issuerConfig:         v1beta1.NewDefaultClusterIssuer(),
		expectedIssuerConfig: v1beta1.NewDefaultClusterIssuer(),
		expectErr:            false,
	},
	{
		testName:   "Both Certificate and ClusterIssuer Configured",
		certConfig: &v1beta1.CertManagerComponent{Certificate: nonDefaultCAV1Beta1},
		issuerConfig: &v1beta1.ClusterIssuerComponent{
			ClusterResourceNamespace: customNamespace,
			IssuerConfig: v1beta1.IssuerConfig{
				CA: &v1beta1.CAIssuer{SecretName: customSecret},
			},
		},
		expectedIssuerConfig: &v1beta1.ClusterIssuerComponent{
			ClusterResourceNamespace: customNamespace,
			IssuerConfig: v1beta1.IssuerConfig{
				CA: &v1beta1.CAIssuer{SecretName: customSecret},
			},
		},
		expectErr: true,
	},
}
