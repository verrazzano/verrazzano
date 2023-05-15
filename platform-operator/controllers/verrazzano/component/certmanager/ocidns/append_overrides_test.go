// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package ocidns

import (
	"github.com/verrazzano/verrazzano/pkg/bom"
	"testing"

	"github.com/stretchr/testify/assert"
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	k8scheme "k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

// All of this below is to make Sonar happy
type appendOverridesTest struct {
	testName  string
	config    *vzapi.Verrazzano
	expectErr bool
}

var (
	enabled = true
)

// Test_appendOCIDNSOverrides tests the appendOCIDNSOverrides function
// GIVEN a call to appendOCIDNSOverrides
// WHEN CertManager or ExternalCertManager are configured
// THEN the appropriate chart overrides are returned
func Test_appendOCIDNSOverrides(t *testing.T) {
	asserts := assert.New(t)

	tests := []appendOverridesTest{
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
			expectErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.testName, func(t *testing.T) {
			client := fake.NewClientBuilder().WithScheme(k8scheme.Scheme).Build()
			fakeContext := spi.NewFakeContext(client, tt.config, nil, false)

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

			assert.Len(t, overrides, 1)

			asserts.Equal("ociAuthSecrets[0]", overrides[0].Key)
			asserts.Equal(tt.config.Spec.Components.DNS.OCI.OCIConfigSecret, overrides[0].Value)
		})
	}
}
