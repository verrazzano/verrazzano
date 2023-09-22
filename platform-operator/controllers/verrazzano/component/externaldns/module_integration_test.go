// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package externaldns

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
)

// TestGetModuleSpec tests the GetModuleConfigAsHelmValues function impl for this component
// GIVEN a call to GetModuleConfigAsHelmValues
//
//	WHEN for various Verrazzano CR configurations
//	THEN the generated helm values JSON snippet is valid
func TestGetModuleSpec(t *testing.T) {
	trueValue := true
	falseValue := false

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
						DNS: &vzapi.DNSComponent{
							OCI: &vzapi.OCI{
								DNSScope:               "global",
								DNSZoneCompartmentOCID: "ocid..compartment.mycomp",
								DNSZoneOCID:            "ocid..zone.myzone",
								DNSZoneName:            "myzone",
								OCIConfigSecret:        "oci",
							},
						},
						Istio: &vzapi.IstioComponent{
							Enabled: &trueValue,
						},
					},
				},
			},
			wantErr: assert.NoError,
			want: `{
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
					}
				  }
				}
			  }
			}
			`,
		},
		{
			name: "IstioDisabled",
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
						Istio: &vzapi.IstioComponent{
							Enabled: &falseValue,
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
						}
					  }
					}
				  }
				}
				`,
		},
		{
			name: "IstioNotPresent",
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
						}
					  }
					}
				  }
				}`,
		},
		{
			name: "NoDNSConfig",
			effectiveCR: &vzapi.Verrazzano{
				Spec: vzapi.VerrazzanoSpec{
					Components: vzapi.ComponentSpec{
						Istio: &vzapi.IstioComponent{
							Enabled: &trueValue,
						},
					},
				},
			},
			wantErr: assert.NoError,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := NewComponent().(*externalDNSComponent)
			got, err := c.GetModuleConfigAsHelmValues(tt.effectiveCR)
			if !tt.wantErr(t, err, fmt.Sprintf("GetModuleConfigAsHelmValues(%v)", tt.effectiveCR)) {
				return
			}
			if len(tt.want) > 0 {
				assert.JSONEq(t, tt.want, string(got.Raw))
			} else {
				assert.Nil(t, got)
			}
		})
	}
}
