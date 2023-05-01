// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package ocidns

import (
	"github.com/verrazzano/verrazzano/pkg/bom"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/verrazzano/verrazzano/pkg/constants"
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	k8scheme "k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

// All of this below is to make Sonar happy
type appendOverridesTest struct {
	testName  string
	config    *vzapi.Verrazzano
	expectedNamespace         string
	exectedSA                 string
	expectedResourceNamespace string
	expectErr bool
}

// Test_appendOCIDNSOverrides tests the appendOCIDNSOverrides function
// GIVEN a call to appendOCIDNSOverrides
// WHEN CertManager or ExternalCertManager are configured
// THEN the appropriate chart overrides are returned
func Test_appendOCIDNSOverrides(t *testing.T) {
	asserts := assert.New(t)

	const (
		mycmNamespace       = "mycm"
		mySA                = "my-service-account"
		myResourceNamespace = "my-cluster-resource-ns"
	)

	tests := []appendOverridesTest{
		{
			testName:                  "Default CM",
			config:                    &vzapi.Verrazzano{},
			expectedNamespace:         constants.CertManagerNamespace,
			exectedSA:                 constants.CertManagerNamespace,
			expectedResourceNamespace: constants.CertManagerNamespace,
		},
		{
			testName: "Enabled CM",
			config: &vzapi.Verrazzano{
				Spec: vzapi.VerrazzanoSpec{
					Components: vzapi.ComponentSpec{
						CertManager: &vzapi.CertManagerComponent{
							Enabled: &enabled,
						},
					},
				},
			},
			expectedNamespace:         constants.CertManagerNamespace,
			exectedSA:                 constants.CertManagerNamespace,
			expectedResourceNamespace: constants.CertManagerNamespace,
		},
		{
			testName: "External CM",
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
		},
		{
			testName: "External CM Namespace Override",
			config: &vzapi.Verrazzano{
				Spec: vzapi.VerrazzanoSpec{
					Components: vzapi.ComponentSpec{
						CertManager: &vzapi.CertManagerComponent{
							Enabled: &disabled,
						},
						ExternalCertManager: &vzapi.ExternalCertManagerComponent{
							Enabled:   &enabled,
							Namespace: mycmNamespace,
						},
					},
				},
			},
			expectedNamespace:         mycmNamespace,
			exectedSA:                 constants.CertManagerNamespace,
			expectedResourceNamespace: mycmNamespace,
		},
		{
			testName: "External CM All Override",
			config: &vzapi.Verrazzano{
				Spec: vzapi.VerrazzanoSpec{
					Components: vzapi.ComponentSpec{
						CertManager: &vzapi.CertManagerComponent{
							Enabled: &disabled,
						},
						ExternalCertManager: &vzapi.ExternalCertManagerComponent{
							Enabled:                  &enabled,
							Namespace:                mycmNamespace,
							ClusterResourceNamespace: myResourceNamespace,
							ServiceAccountName:       mySA,
						},
					},
				},
			},
			expectedNamespace:         mycmNamespace,
			exectedSA:                 mySA,
			expectedResourceNamespace: myResourceNamespace,
		},
		{
			testName:  "OCI DNS Not configured",
			config:    &vzapi.Verrazzano{},
			expectErr: true,
		},
		{
			testName: "OCI DNS Secret Not configured ",
			config: &vzapi.Verrazzano{
				Spec: vzapi.VerrazzanoSpec{
					Components: vzapi.ComponentSpec{
						CertManager: &vzapi.CertManagerComponent{
							Enabled: &enabled,
						},
						DNS: &vzapi.DNSComponent{
							OCI: &vzapi.OCI{},
						},
					},
				},
			},
			expectErr: true,
		},
		{
			testName: "OCI DNS Secret configured ",
			config: &vzapi.Verrazzano{
				Spec: vzapi.VerrazzanoSpec{
					Components: vzapi.ComponentSpec{
						CertManager: &vzapi.CertManagerComponent{
							Enabled: &enabled,
						},
						DNS: &vzapi.DNSComponent{
							OCI: &vzapi.OCI{
								OCIConfigSecret: "oci",
							},
						},
					},
				},
			},
			expectedNamespace:         constants.CertManagerNamespace,
			exectedSA:                 constants.CertManagerNamespace,
			expectedResourceNamespace: constants.CertManagerNamespace,
			expectErr: false,
		},
		{
			testName: "OCI DNS Secret configured alternate name ",
			config: &vzapi.Verrazzano{
				Spec: vzapi.VerrazzanoSpec{
					Components: vzapi.ComponentSpec{
						CertManager: &vzapi.CertManagerComponent{
							Enabled: &enabled,
						},
						DNS: &vzapi.DNSComponent{
							OCI: &vzapi.OCI{
								OCIConfigSecret: "mysecret",
							},
						},
					},
				},
			},
			expectedNamespace:         constants.CertManagerNamespace,
			exectedSA:                 constants.CertManagerNamespace,
			expectedResourceNamespace: constants.CertManagerNamespace,
			expectErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.testName, func(t *testing.T) {
			client := fake.NewClientBuilder().WithScheme(k8scheme.Scheme).Build()
			fakeContext := spi.NewFakeContext(client, tt.config, nil, false)
			resolvedNamespace := resolveCertManagerNamespace(fakeContext, "cert-manager")

			var overrides []bom.KeyValue
			var err error
			overrides, err = appendOCIDNSOverrides(fakeContext, "", "", "", overrides)

			// Check error condition
			if tt.expectErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}

			// if it failed skip the remaining checks
			if err != nil {
				return
			}

			assert.NoError(t, err)

			assert.Len(t, overrides, 4)

			asserts.Equal("ociAuthSecrets[0]", overrides[0].Key)
			asserts.Equal(tt.config.Spec.Components.DNS.OCI.OCIConfigSecret, overrides[0].Value)
			asserts.Equal("certManager.namespace", overrides[1].Key)
			asserts.Equal(tt.expectedNamespace, overrides[1].Value)
			asserts.Equal("certManager.clusterResourceNamespace", overrides[2].Key)
			asserts.Equal(tt.expectedResourceNamespace, overrides[2].Value)
			asserts.Equal("certManager.serviceAccountName", overrides[3].Key)
			asserts.Equal(tt.exectedSA, overrides[3].Value)
		})
	}
}
