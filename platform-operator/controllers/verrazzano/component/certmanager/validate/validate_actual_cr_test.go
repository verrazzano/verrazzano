// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package validate

import (
	"github.com/stretchr/testify/assert"
	"github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1beta1"
	"testing"
)

const (
	emailAddress    = "foo@bar.com"
	customNamespace = "myns"
	customSecret    = "mySecret"

	production = "production"
	staging    = "staging"
)

// TestValidateRawCertifcatesConfigurationV1Beta1V1Beta1 tests the validateActualConfigurationV1Beta1 function
// GIVEN a call to validateActualConfigurationV1Beta1 for various configurations
// THEN the call returns an error IFF both the ClusterIssuerComponent.IssuerConfig and CertManagerComponent.Certificate
// fields are both explicitly configured
func TestValidateRawCertifcatesConfigurationV1Beta1V1Beta1(t *testing.T) {
	tests := []struct {
		testName  string
		config    *v1beta1.Verrazzano
		expectErr bool
	}{
		{
			testName: "Both Nil",
			config:   &v1beta1.Verrazzano{},
		},
		{
			testName: "Both ACME fields configured",
			config: &v1beta1.Verrazzano{
				Spec: v1beta1.VerrazzanoSpec{
					Components: v1beta1.ComponentSpec{
						CertManager: &v1beta1.CertManagerComponent{
							Certificate: v1beta1.Certificate{
								Acme: v1beta1.Acme{
									EmailAddress: "joe@blow.com",
									Environment:  production,
									Provider:     v1beta1.LetsEncrypt,
								},
							},
						},
						ClusterIssuer: &v1beta1.ClusterIssuerComponent{
							IssuerConfig: v1beta1.IssuerConfig{
								LetsEncrypt: &v1beta1.LetsEncryptACMEIssuer{
									EmailAddress: emailAddress,
									Environment:  staging,
								},
							},
						},
					},
				},
			},
			expectErr: true,
		},
		{
			testName: "Deprecated CA and LetsEncrypt Issuer",
			config: &v1beta1.Verrazzano{
				Spec: v1beta1.VerrazzanoSpec{
					Components: v1beta1.ComponentSpec{
						CertManager: &v1beta1.CertManagerComponent{
							Certificate: v1beta1.Certificate{
								CA: v1beta1.CA{
									SecretName:               customSecret,
									ClusterResourceNamespace: customNamespace,
								},
							},
						},
						ClusterIssuer: &v1beta1.ClusterIssuerComponent{
							IssuerConfig: v1beta1.IssuerConfig{
								LetsEncrypt: &v1beta1.LetsEncryptACMEIssuer{
									EmailAddress: emailAddress,
									Environment:  staging,
								},
							},
						},
					},
				},
			},
			expectErr: true,
		},
		{
			testName: "Deprecated Custom CA and Issuer CA",
			config: &v1beta1.Verrazzano{
				Spec: v1beta1.VerrazzanoSpec{
					Components: v1beta1.ComponentSpec{
						CertManager: &v1beta1.CertManagerComponent{
							Certificate: v1beta1.Certificate{
								CA: v1beta1.CA{
									SecretName:               customSecret,
									ClusterResourceNamespace: customNamespace,
								},
							},
						},
						ClusterIssuer: &v1beta1.ClusterIssuerComponent{
							IssuerConfig: v1beta1.IssuerConfig{
								CA: &v1beta1.CAIssuer{
									SecretName: customSecret,
								},
							},
						},
					},
				},
			},
			expectErr: true,
		},
		{
			testName: "Deprecated Empty Cert and Issuer CA",
			config: &v1beta1.Verrazzano{
				Spec: v1beta1.VerrazzanoSpec{
					Components: v1beta1.ComponentSpec{
						CertManager: &v1beta1.CertManagerComponent{},
						ClusterIssuer: &v1beta1.ClusterIssuerComponent{
							IssuerConfig: v1beta1.IssuerConfig{
								CA: &v1beta1.CAIssuer{
									SecretName: customSecret,
								},
							},
						},
					},
				},
			},
		},
		{
			testName: "Deprecated Empty Cert and Issuer LetsEncrypt",
			config: &v1beta1.Verrazzano{
				Spec: v1beta1.VerrazzanoSpec{
					Components: v1beta1.ComponentSpec{
						CertManager: &v1beta1.CertManagerComponent{},
						ClusterIssuer: &v1beta1.ClusterIssuerComponent{
							IssuerConfig: v1beta1.IssuerConfig{
								LetsEncrypt: &v1beta1.LetsEncryptACMEIssuer{
									EmailAddress: emailAddress,
									Environment:  staging,
								},
							},
						},
					},
				},
			},
		},
		{
			testName: "Deprecated Custom CA and Issuer Not set",
			config: &v1beta1.Verrazzano{
				Spec: v1beta1.VerrazzanoSpec{
					Components: v1beta1.ComponentSpec{
						CertManager: &v1beta1.CertManagerComponent{
							Certificate: v1beta1.Certificate{
								CA: v1beta1.CA{
									SecretName:               customSecret,
									ClusterResourceNamespace: customNamespace,
								},
							},
						},
						ClusterIssuer: &v1beta1.ClusterIssuerComponent{},
					},
				},
			},
		},
		{
			testName: "Deprecated Custom CA and Issuer Nil",
			config: &v1beta1.Verrazzano{
				Spec: v1beta1.VerrazzanoSpec{
					Components: v1beta1.ComponentSpec{
						CertManager: &v1beta1.CertManagerComponent{
							Certificate: v1beta1.Certificate{
								CA: v1beta1.CA{
									SecretName:               customSecret,
									ClusterResourceNamespace: customNamespace,
								},
							},
						},
					},
				},
			},
		},
		{
			testName: "Nil CertManager and Issuer CA",
			config: &v1beta1.Verrazzano{
				Spec: v1beta1.VerrazzanoSpec{
					Components: v1beta1.ComponentSpec{
						ClusterIssuer: &v1beta1.ClusterIssuerComponent{
							IssuerConfig: v1beta1.IssuerConfig{
								CA: &v1beta1.CAIssuer{
									SecretName: customSecret,
								},
							},
						},
					},
				},
			},
		},
		{
			testName: "Nil CertManager and Issuer LetsEcnrypt",
			config: &v1beta1.Verrazzano{
				Spec: v1beta1.VerrazzanoSpec{
					Components: v1beta1.ComponentSpec{
						ClusterIssuer: &v1beta1.ClusterIssuerComponent{
							IssuerConfig: v1beta1.IssuerConfig{
								LetsEncrypt: &v1beta1.LetsEncryptACMEIssuer{
									EmailAddress: emailAddress,
									Environment:  staging,
								},
							},
						},
					},
				},
			},
		},
	}
	for _, tt := range tests {
		errs := ValidateActualConfigurationV1Beta1(tt.config)
		if tt.expectErr {
			assert.Len(t, errs, 1)
			assert.EqualError(t, errs[0], bothConfiguredErr)
		} else {
			assert.Len(t, errs, 0)
		}
	}
}

// TestValidateRawCertifcatesConfigurationV1Alpha1 tests the validateActualConfigurationV1Alpha1 function
// GIVEN a call to validateActualConfigurationV1Alpha1 for various configurations
// THEN the call returns an error IFF both the ClusterIssuerComponent.IssuerConfig and CertManagerComponent.Certificate
// fields are both explicitly configured
func TestValidateRawCertifcatesConfigurationV1Alpha1(t *testing.T) {
	tests := []struct {
		name      string
		config    *v1alpha1.Verrazzano
		expectErr bool
	}{
		{
			name:   "Both Nil",
			config: &v1alpha1.Verrazzano{},
		},
		{
			name: "Both ACME fields configured",
			config: &v1alpha1.Verrazzano{
				Spec: v1alpha1.VerrazzanoSpec{
					Components: v1alpha1.ComponentSpec{
						CertManager: &v1alpha1.CertManagerComponent{
							Certificate: v1alpha1.Certificate{
								Acme: v1alpha1.Acme{
									EmailAddress: "joe@blow.com",
									Environment:  production,
									Provider:     v1alpha1.LetsEncrypt,
								},
							},
						},
						ClusterIssuer: &v1alpha1.ClusterIssuerComponent{
							IssuerConfig: v1alpha1.IssuerConfig{
								LetsEncrypt: &v1alpha1.LetsEncryptACMEIssuer{
									EmailAddress: emailAddress,
									Environment:  staging,
								},
							},
						},
					},
				},
			},
			expectErr: true,
		},
		{
			name: "Deprecated CA and LetsEncrypt Issuer",
			config: &v1alpha1.Verrazzano{
				Spec: v1alpha1.VerrazzanoSpec{
					Components: v1alpha1.ComponentSpec{
						CertManager: &v1alpha1.CertManagerComponent{
							Certificate: v1alpha1.Certificate{
								CA: v1alpha1.CA{
									SecretName:               customSecret,
									ClusterResourceNamespace: customNamespace,
								},
							},
						},
						ClusterIssuer: &v1alpha1.ClusterIssuerComponent{
							IssuerConfig: v1alpha1.IssuerConfig{
								LetsEncrypt: &v1alpha1.LetsEncryptACMEIssuer{
									EmailAddress: emailAddress,
									Environment:  staging,
								},
							},
						},
					},
				},
			},
			expectErr: true,
		},
		{
			name: "Deprecated Custom CA and Issuer CA",
			config: &v1alpha1.Verrazzano{
				Spec: v1alpha1.VerrazzanoSpec{
					Components: v1alpha1.ComponentSpec{
						CertManager: &v1alpha1.CertManagerComponent{
							Certificate: v1alpha1.Certificate{
								CA: v1alpha1.CA{
									SecretName:               customSecret,
									ClusterResourceNamespace: customNamespace,
								},
							},
						},
						ClusterIssuer: &v1alpha1.ClusterIssuerComponent{
							IssuerConfig: v1alpha1.IssuerConfig{
								CA: &v1alpha1.CAIssuer{
									SecretName: customSecret,
								},
							},
						},
					},
				},
			},
			expectErr: true,
		},
		{
			name: "Deprecated Empty Cert and Issuer CA",
			config: &v1alpha1.Verrazzano{
				Spec: v1alpha1.VerrazzanoSpec{
					Components: v1alpha1.ComponentSpec{
						CertManager: &v1alpha1.CertManagerComponent{},
						ClusterIssuer: &v1alpha1.ClusterIssuerComponent{
							IssuerConfig: v1alpha1.IssuerConfig{
								CA: &v1alpha1.CAIssuer{
									SecretName: customSecret,
								},
							},
						},
					},
				},
			},
		},
		{
			name: "Deprecated Empty Cert and Issuer LetsEncrypt",
			config: &v1alpha1.Verrazzano{
				Spec: v1alpha1.VerrazzanoSpec{
					Components: v1alpha1.ComponentSpec{
						CertManager: &v1alpha1.CertManagerComponent{},
						ClusterIssuer: &v1alpha1.ClusterIssuerComponent{
							IssuerConfig: v1alpha1.IssuerConfig{
								LetsEncrypt: &v1alpha1.LetsEncryptACMEIssuer{
									EmailAddress: emailAddress,
									Environment:  staging,
								},
							},
						},
					},
				},
			},
		},
		{
			name: "Deprecated Custom CA and Issuer Not set",
			config: &v1alpha1.Verrazzano{
				Spec: v1alpha1.VerrazzanoSpec{
					Components: v1alpha1.ComponentSpec{
						CertManager: &v1alpha1.CertManagerComponent{
							Certificate: v1alpha1.Certificate{
								CA: v1alpha1.CA{
									SecretName:               customSecret,
									ClusterResourceNamespace: customNamespace,
								},
							},
						},
						ClusterIssuer: &v1alpha1.ClusterIssuerComponent{},
					},
				},
			},
		},
		{
			name: "Deprecated Custom CA and Issuer Nil",
			config: &v1alpha1.Verrazzano{
				Spec: v1alpha1.VerrazzanoSpec{
					Components: v1alpha1.ComponentSpec{
						CertManager: &v1alpha1.CertManagerComponent{
							Certificate: v1alpha1.Certificate{
								CA: v1alpha1.CA{
									SecretName:               customSecret,
									ClusterResourceNamespace: customNamespace,
								},
							},
						},
					},
				},
			},
		},
		{
			name: "Nil CertManager and Issuer CA",
			config: &v1alpha1.Verrazzano{
				Spec: v1alpha1.VerrazzanoSpec{
					Components: v1alpha1.ComponentSpec{
						ClusterIssuer: &v1alpha1.ClusterIssuerComponent{
							IssuerConfig: v1alpha1.IssuerConfig{
								CA: &v1alpha1.CAIssuer{
									SecretName: customSecret,
								},
							},
						},
					},
				},
			},
		},
		{
			name: "Nil CertManager and Issuer LetsEcnrypt",
			config: &v1alpha1.Verrazzano{
				Spec: v1alpha1.VerrazzanoSpec{
					Components: v1alpha1.ComponentSpec{
						ClusterIssuer: &v1alpha1.ClusterIssuerComponent{
							IssuerConfig: v1alpha1.IssuerConfig{
								LetsEncrypt: &v1alpha1.LetsEncryptACMEIssuer{
									EmailAddress: emailAddress,
									Environment:  staging,
								},
							},
						},
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Logf("Running test %s", tt.name)
		errs := ValidateActualConfigurationV1Alpha1(tt.config)
		if tt.expectErr {
			assert.Len(t, errs, 1)
			assert.EqualError(t, errs[0], bothConfiguredErr)
		} else {
			assert.Len(t, errs, 0)
		}
	}
}
