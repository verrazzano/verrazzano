// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package externaldns

import (
	"testing"

	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
)

func Test_externalDNSComponent_ValidateUpdate(t *testing.T) {
	tests := []struct {
		name    string
		old     *vzapi.Verrazzano
		new     *vzapi.Verrazzano
		wantErr bool
	}{
		{
			name: "enable",
			old:  &vzapi.Verrazzano{},
			new: &vzapi.Verrazzano{
				Spec: vzapi.VerrazzanoSpec{
					Components: vzapi.ComponentSpec{
						DNS: &vzapi.DNSComponent{
							OCI: &vzapi.OCI{
								OCIConfigSecret: "oci-config-secret",
							},
						},
					},
				},
			},
			wantErr: false, // For now, any changes to the DNS component are rejected
		},
		{
			name: "disable",
			old: &vzapi.Verrazzano{
				Spec: vzapi.VerrazzanoSpec{
					Components: vzapi.ComponentSpec{
						DNS: &vzapi.DNSComponent{
							OCI: &vzapi.OCI{
								OCIConfigSecret: "oci-config-secret",
							},
						},
					},
				},
			},
			new:     &vzapi.Verrazzano{},
			wantErr: true,
		},
		{
			name:    "no change",
			old:     &vzapi.Verrazzano{},
			new:     &vzapi.Verrazzano{},
			wantErr: false,
		},
		{
			name: "default-to-external",
			old:  &vzapi.Verrazzano{},
			new: &vzapi.Verrazzano{
				Spec: vzapi.VerrazzanoSpec{
					Components: vzapi.ComponentSpec{
						DNS: &vzapi.DNSComponent{
							External: &vzapi.External{Suffix: "foo.com"},
						},
					},
				},
			},
			wantErr: false, // For now, any changes to the DNS component are rejected
		},
		{
			name: "oci-to-external",
			old: &vzapi.Verrazzano{
				Spec: vzapi.VerrazzanoSpec{
					Components: vzapi.ComponentSpec{
						DNS: &vzapi.DNSComponent{
							OCI: &vzapi.OCI{
								OCIConfigSecret: "oci-config-secret",
							},
						},
					},
				},
			},
			new: &vzapi.Verrazzano{
				Spec: vzapi.VerrazzanoSpec{
					Components: vzapi.ComponentSpec{
						DNS: &vzapi.DNSComponent{
							External: &vzapi.External{Suffix: "foo.com"},
						},
					},
				},
			},
			wantErr: true, // For now, any changes to the DNS component are rejected
		},
		{
			name: "oci-to-wildcard",
			old: &vzapi.Verrazzano{
				Spec: vzapi.VerrazzanoSpec{
					Components: vzapi.ComponentSpec{
						DNS: &vzapi.DNSComponent{
							OCI: &vzapi.OCI{
								OCIConfigSecret: "oci-config-secret",
							},
						},
					},
				},
			},
			new: &vzapi.Verrazzano{
				Spec: vzapi.VerrazzanoSpec{
					Components: vzapi.ComponentSpec{
						DNS: &vzapi.DNSComponent{
							Wildcard: &vzapi.Wildcard{Domain: "xip.io"},
						},
					},
				},
			},
			wantErr: true, // For now, any changes to the DNS component are rejected
		},
		{
			name: "default-to-wildcard",
			old:  &vzapi.Verrazzano{},
			new: &vzapi.Verrazzano{
				Spec: vzapi.VerrazzanoSpec{
					Components: vzapi.ComponentSpec{
						DNS: &vzapi.DNSComponent{
							Wildcard: &vzapi.Wildcard{Domain: "xip.io"},
						},
					},
				},
			},
			wantErr: false, // For now, any changes to the DNS component are rejected
		},
		{
			name: "wildcard-to-wildcard",
			old: &vzapi.Verrazzano{
				Spec: vzapi.VerrazzanoSpec{
					Components: vzapi.ComponentSpec{
						DNS: &vzapi.DNSComponent{
							Wildcard: &vzapi.Wildcard{Domain: "sslip.io"},
						},
					},
				},
			},
			new: &vzapi.Verrazzano{
				Spec: vzapi.VerrazzanoSpec{
					Components: vzapi.ComponentSpec{
						DNS: &vzapi.DNSComponent{
							Wildcard: &vzapi.Wildcard{Domain: "xip.io"},
						},
					},
				},
			},
			wantErr: false, // For now, any changes to the DNS component are rejected
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := NewComponent()
			if err := c.ValidateUpdate(tt.old, tt.new); (err != nil) != tt.wantErr {
				t.Errorf("ValidateUpdate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
