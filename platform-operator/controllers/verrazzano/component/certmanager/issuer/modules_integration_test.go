// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package issuer

import (
	"fmt"

	"github.com/stretchr/testify/assert"
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"testing"
)

func TestGetModuleSpec(t *testing.T) {
	trueValue := true
	tests := []struct {
		name        string
		effectiveCR *vzapi.Verrazzano
		want        string
		wantErr     assert.ErrorAssertionFunc
	}{
		{
			name: "BasicIssuerConfig",
			effectiveCR: &vzapi.Verrazzano{
				Spec: vzapi.VerrazzanoSpec{
					Components: vzapi.ComponentSpec{
						ClusterIssuer: &vzapi.ClusterIssuerComponent{
							Enabled:                  &trueValue,
							ClusterResourceNamespace: secretNamespace,
							IssuerConfig: vzapi.IssuerConfig{
								CA: &vzapi.CAIssuer{
									SecretName: secretName,
								},
							},
						},
					},
				},
			},
			wantErr: assert.NoError,
			want: `{
				  "verrazzano": {
					"module": {
					  "spec": {
						"issuerConfig": {
						  "ca": {
							"secretName": "newsecret"
						  }
						},
						"clusterResourceNamespace": "ns"
					  }
					}
				  }
				}
				`,
		},
		{
			name: "IssuerConfigWithOCIDNS",
			effectiveCR: &vzapi.Verrazzano{
				Spec: vzapi.VerrazzanoSpec{
					Components: vzapi.ComponentSpec{
						DNS: &vzapi.DNSComponent{
							OCI: &vzapi.OCI{
								DNSScope:               "global",
								DNSZoneCompartmentOCID: "ocid..compartment.mycomp",
								DNSZoneOCID:            "ocid..zone.myzone",
								DNSZoneName:            "myzone",
								OCIConfigSecret:        "oci",
							},
						},
						ClusterIssuer: &vzapi.ClusterIssuerComponent{
							Enabled:                  &trueValue,
							ClusterResourceNamespace: secretNamespace,
							IssuerConfig: vzapi.IssuerConfig{
								CA: &vzapi.CAIssuer{
									SecretName: secretName,
								},
							},
						},
					},
				},
			},
			wantErr: assert.NoError,
			want: `
				{
				  "verrazzano": {
					"module": {
					  "spec": {
						"dns": {
						  "oci": {
							"dnsScope": "global",
							"dnsZoneCompartmentOCID": "ocid..compartment.mycomp",
							"dnsZoneOCID": "ocid..zone.myzone",
							"dnsZoneName": "myzone",
							"ociConfigSecret": "oci"
						  }
						},
						"issuerConfig": {
						  "ca": {
							"secretName": "newsecret"
						  }
						},
						"clusterResourceNamespace": "ns"
					  }
					}
				  }
				}
				`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := NewComponent().(clusterIssuerComponent)
			got, err := c.GetModuleSpec(tt.effectiveCR)
			if !tt.wantErr(t, err, fmt.Sprintf("GetModuleSpec(%v)", tt.effectiveCR)) {
				return
			}
			assert.JSONEq(t, tt.want, string(got.Raw))
		})
	}
}
