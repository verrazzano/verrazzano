// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package common

import (
	"github.com/stretchr/testify/assert"
	"github.com/verrazzano/verrazzano/pkg/constants"
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"testing"
)

var (
	enabled  = true
	disabled = false
)

type deriveNamespaceTestStruct struct {
	testName                  string
	config                    *vzapi.Verrazzano
	expectedNamespace         string
	exectedSA                 string
	expectedResourceNamespace string
	expectedCertConfig        vzapi.Certificate
}

// TestDerivedConfig tests the derived-config CertManager functions
// GIVEN various CertManager and ExternalCertManager configurations
// WHEN the DeriveXXXXX functions are called
// THEN the appropriate values are returned
func TestDerivedConfig(t *testing.T) {
	asserts := assert.New(t)

	const (
		mycmNamespace       = "mycm"
		mySA                = "my-service-account"
		myResourceNamespace = "my-cluster-resource-ns"
	)

	defaultCAConfig := vzapi.Certificate{
		CA: vzapi.CA{
			ClusterResourceNamespace: constants.CertManagerNamespace,
			SecretName:               DefaultCACertificateSecretName,
		},
	}
	myCAConfig := vzapi.Certificate{
		CA: vzapi.CA{
			ClusterResourceNamespace: "myresns",
			SecretName:               "mysecret",
		},
	}

	externalCMCertOverride := vzapi.Certificate{
		Acme: vzapi.Acme{
			EmailAddress: "myemail",
			Environment:  "staging",
			Provider:     "LetsEncrypt",
		},
	}

	tests := []deriveNamespaceTestStruct{
		{
			testName:                  "Default CM",
			config:                    &vzapi.Verrazzano{},
			expectedNamespace:         constants.CertManagerNamespace,
			exectedSA:                 constants.CertManagerNamespace,
			expectedResourceNamespace: constants.CertManagerNamespace,
		},
		{
			testName: "Enabled Default CM",
			config: &vzapi.Verrazzano{
				Spec: vzapi.VerrazzanoSpec{
					Components: vzapi.ComponentSpec{
						CertManager: &vzapi.CertManagerComponent{
							Enabled:     &enabled,
							Certificate: myCAConfig,
						},
					},
				},
			},
			expectedNamespace:         constants.CertManagerNamespace,
			exectedSA:                 constants.CertManagerNamespace,
			expectedResourceNamespace: constants.CertManagerNamespace,
			expectedCertConfig:        myCAConfig,
		},
		{
			testName: "External CM Defaults",
			config: &vzapi.Verrazzano{
				Spec: vzapi.VerrazzanoSpec{
					Components: vzapi.ComponentSpec{
						CertManager: &vzapi.CertManagerComponent{
							Enabled: &disabled,
						},
						ExternalCertManager: &vzapi.ExternalCertManagerComponent{
							Enabled: &enabled,
						},
					},
				},
			},
			expectedNamespace:         constants.CertManagerNamespace,
			exectedSA:                 constants.CertManagerNamespace,
			expectedResourceNamespace: constants.CertManagerNamespace,
			expectedCertConfig:        defaultCAConfig,
		},
		{
			testName: "External CM Namespace Only Override",
			config: &vzapi.Verrazzano{
				Spec: vzapi.VerrazzanoSpec{
					Components: vzapi.ComponentSpec{
						CertManager: &vzapi.CertManagerComponent{
							Enabled: &disabled,
						},
						ExternalCertManager: &vzapi.ExternalCertManagerComponent{
							Enabled:   &enabled,
							Namespace: mycmNamespace,
							Certificate: vzapi.Certificate{
								CA: vzapi.CA{
									ClusterResourceNamespace: mycmNamespace,
									SecretName:               DefaultCACertificateSecretName,
								},
							},
						},
					},
				},
			},
			expectedNamespace:         mycmNamespace,
			exectedSA:                 CertManagerComponentName,
			expectedResourceNamespace: mycmNamespace,
			expectedCertConfig: vzapi.Certificate{
				CA: vzapi.CA{
					ClusterResourceNamespace: mycmNamespace,
					SecretName:               DefaultCACertificateSecretName,
				},
			},
		},
		{
			testName: "External CertManager All Values Override",
			config: &vzapi.Verrazzano{
				Spec: vzapi.VerrazzanoSpec{
					Components: vzapi.ComponentSpec{
						CertManager: &vzapi.CertManagerComponent{
							Enabled:     &disabled,
							Certificate: myCAConfig,
						},
						ExternalCertManager: &vzapi.ExternalCertManagerComponent{
							Enabled:                  &enabled,
							Namespace:                mycmNamespace,
							ServiceAccountName:       mySA,
							ClusterResourceNamespace: myResourceNamespace,
							Certificate:              externalCMCertOverride,
						},
					},
				},
			},
			expectedNamespace:         mycmNamespace,
			exectedSA:                 mySA,
			expectedResourceNamespace: myResourceNamespace,
			expectedCertConfig:        externalCMCertOverride,
		},
	}
	for _, tt := range tests {
		t.Run(tt.testName, func(t *testing.T) {
			t.Logf("Running test %s", tt.testName)
			cmConfig, err := GetCertManagerConfiguration(tt.config)
			asserts.NoError(err)
			asserts.Equal(tt.expectedNamespace, cmConfig.Namespace)
			asserts.Equal(tt.exectedSA, cmConfig.ServiceAccountName)
			asserts.Equal(tt.expectedResourceNamespace, cmConfig.ClusterResourceNamespace)
			asserts.Equal(tt.expectedCertConfig, cmConfig.Certificate)
		})
	}
}
