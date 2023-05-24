// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package transform

import (
	"github.com/stretchr/testify/assert"
	"github.com/verrazzano/verrazzano/pkg/constants"
	"github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1beta1"
	"testing"
)

// Test_convertCertificateToClusterIssuerV1Beta1 tests the convertCertificateToClusterIssuerV1Beta1 function
// GIVEN a call to convertCertificateToClusterIssuerV1Beta1
// THEN the appropriate conversions from the deprecated Certificate object to the ClusterIssuer Component
func Test_convertCertificateToClusterIssuerV1Beta1(t *testing.T) {
	asserts := assert.New(t)
	tests := []struct {
		testName             string
		certConfig           v1beta1.Certificate
		issuerConfig         *v1beta1.ClusterIssuerComponent
		expectErr            bool
		expectedIssuerConfig *v1beta1.ClusterIssuerComponent
	}{
		{
			testName:             "Neither configured",
			certConfig:           v1beta1.Certificate{},
			issuerConfig:         v1beta1.NewDefaultClusterIssuer(),
			expectedIssuerConfig: v1beta1.NewDefaultClusterIssuer(),
			expectErr:            true,
		},
		{
			testName: "Non Default CA Certificate",
			certConfig: v1beta1.Certificate{
				CA: v1beta1.CA{
					ClusterResourceNamespace: "myns",
					SecretName:               "mySecret",
				},
			},
			issuerConfig: v1beta1.NewDefaultClusterIssuer(),
			expectedIssuerConfig: &v1beta1.ClusterIssuerComponent{
				ClusterResourceNamespace: "myns",
				IssuerConfig: v1beta1.IssuerConfig{
					CA: &v1beta1.CAIssuer{SecretName: "mySecret"},
				},
			},
			expectErr: false,
		},
		{
			testName: "LetsEncrypt Certificate",
			certConfig: v1beta1.Certificate{
				Acme: v1beta1.Acme{
					EmailAddress: "foo@bar.com",
					Environment:  "staging",
					Provider:     v1beta1.LetsEncrypt,
				},
			},
			issuerConfig: v1beta1.NewDefaultClusterIssuer(),
			expectedIssuerConfig: &v1beta1.ClusterIssuerComponent{
				ClusterResourceNamespace: constants.CertManagerNamespace,
				IssuerConfig: v1beta1.IssuerConfig{
					LetsEncrypt: &v1beta1.LetsEncryptACMEIssuer{
						EmailAddress: "foo@bar.com",
						Environment:  "staging",
					},
				},
			},
			expectErr: false,
		},
		{
			testName: "Illegal Certificate",
			certConfig: v1beta1.Certificate{
				CA: v1beta1.CA{
					ClusterResourceNamespace: "myns",
					SecretName:               "mySecret",
				},
				Acme: v1beta1.Acme{
					EmailAddress: "foo@bar.com",
					Environment:  "staging",
					Provider:     v1beta1.LetsEncrypt,
				},
			},
			issuerConfig: v1beta1.NewDefaultClusterIssuer(),
			expectErr:    true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.testName, func(t *testing.T) {
			vz := &v1beta1.Verrazzano{
				Spec: v1beta1.VerrazzanoSpec{
					Components: v1beta1.ComponentSpec{
						CertManager: &v1beta1.CertManagerComponent{
							Certificate: tt.certConfig,
						},
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

// Test_convertCertificateToClusterIssuerV1Alpha1 tests the convertCertificateToClusterIssuerV1Alpha1 function
// GIVEN a call to convertCertificateToClusterIssuerV1Beta1
// THEN the appropriate conversions from the deprecated Certificate object to the ClusterIssuer Component
func Test_convertCertificateToClusterIssuerV1Alpha1(t *testing.T) {
	asserts := assert.New(t)
	tests := []struct {
		testName             string
		certConfig           v1alpha1.Certificate
		issuerConfig         *v1alpha1.ClusterIssuerComponent
		expectErr            bool
		expectedIssuerConfig *v1alpha1.ClusterIssuerComponent
	}{
		{
			testName:     "Neither configured",
			certConfig:   v1alpha1.Certificate{},
			issuerConfig: v1alpha1.NewDefaultClusterIssuer(),
			expectErr:    true,
		},
		{
			testName: "Non Default CA Certificate",
			certConfig: v1alpha1.Certificate{
				CA: v1alpha1.CA{
					ClusterResourceNamespace: "myns",
					SecretName:               "mySecret",
				},
			},
			issuerConfig: v1alpha1.NewDefaultClusterIssuer(),
			expectedIssuerConfig: &v1alpha1.ClusterIssuerComponent{
				ClusterResourceNamespace: "myns",
				IssuerConfig: v1alpha1.IssuerConfig{
					CA: &v1alpha1.CAIssuer{SecretName: "mySecret"},
				},
			},
			expectErr: false,
		},
		{
			testName: "LetsEncrypt Certificate",
			certConfig: v1alpha1.Certificate{
				Acme: v1alpha1.Acme{
					EmailAddress: "foo@bar.com",
					Environment:  "staging",
					Provider:     v1alpha1.LetsEncrypt,
				},
			},
			issuerConfig: v1alpha1.NewDefaultClusterIssuer(),
			expectedIssuerConfig: &v1alpha1.ClusterIssuerComponent{
				ClusterResourceNamespace: constants.CertManagerNamespace,
				IssuerConfig: v1alpha1.IssuerConfig{
					LetsEncrypt: &v1alpha1.LetsEncryptACMEIssuer{
						EmailAddress: "foo@bar.com",
						Environment:  "staging",
					},
				},
			},
			expectErr: false,
		},
		{
			testName: "Illegal Certificate",
			certConfig: v1alpha1.Certificate{
				CA: v1alpha1.CA{
					ClusterResourceNamespace: "myns",
					SecretName:               "mySecret",
				},
				Acme: v1alpha1.Acme{
					EmailAddress: "foo@bar.com",
					Environment:  "staging",
					Provider:     v1alpha1.LetsEncrypt,
				},
			},
			issuerConfig: v1alpha1.NewDefaultClusterIssuer(),
			expectErr:    true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.testName, func(t *testing.T) {
			vz := &v1alpha1.Verrazzano{
				Spec: v1alpha1.VerrazzanoSpec{
					Components: v1alpha1.ComponentSpec{
						CertManager: &v1alpha1.CertManagerComponent{
							Certificate: tt.certConfig,
						},
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
