// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package certmanager

import (
	"fmt"
	"github.com/stretchr/testify/assert"
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"testing"
)

// TestGetModuleSpec tests the GetModuleConfigAsHelmValues function impl for this component
// GIVEN a call to GetModuleConfigAsHelmValues
//
//	WHEN for various Verrazzano CR configurations
//	THEN the generated helm values JSON snippet is valid
func TestGetModuleSpec(t *testing.T) {
	trueValue := true
	const issuerResourceNamespace = "issuerResourceNamespace"
	tests := []struct {
		name        string
		effectiveCR *vzapi.Verrazzano
		want        string
		wantErr     assert.ErrorAssertionFunc
	}{
		{
			name: "BasicCertManagerConfig",
			effectiveCR: &vzapi.Verrazzano{
				Spec: vzapi.VerrazzanoSpec{
					Components: vzapi.ComponentSpec{
						CertManager: &vzapi.CertManagerComponent{
							Enabled: &trueValue,
							Certificate: vzapi.Certificate{
								CA: vzapi.CA{
									SecretName:               secretName,
									ClusterResourceNamespace: secretNamespace,
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
							"clusterResourceNamespace": "ns"
						  }
						}
					  }
					}`,
		},
		{
			name: "ClusterIssuerConfigNamespace",
			effectiveCR: &vzapi.Verrazzano{
				Spec: vzapi.VerrazzanoSpec{
					Components: vzapi.ComponentSpec{
						CertManager: &vzapi.CertManagerComponent{
							Enabled:     &trueValue,
							Certificate: vzapi.Certificate{},
						},
						ClusterIssuer: &vzapi.ClusterIssuerComponent{
							Enabled:                  &trueValue,
							ClusterResourceNamespace: issuerResourceNamespace,
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
			want: fmt.Sprintf(`{
					  "verrazzano": {
						"module": {
						  "spec": {
							"clusterResourceNamespace": "%v"
						  }
						}
					  }
					}`, issuerResourceNamespace),
		},
		{
			name: "OtherComponentConfigsHaveNoEffect",
			effectiveCR: &vzapi.Verrazzano{
				Spec: vzapi.VerrazzanoSpec{
					Components: vzapi.ComponentSpec{
						Keycloak: &vzapi.KeycloakComponent{
							Enabled: &trueValue,
							InstallOverrides: vzapi.InstallOverrides{
								MonitorChanges: &trueValue,
								ValueOverrides: []vzapi.Overrides{
									{Values: &apiextensionsv1.JSON{Raw: []byte("somevalue")}},
								},
							},
							KeycloakInstallArgs: nil,
							MySQL:               vzapi.MySQLComponent{},
						},
						CertManager: &vzapi.CertManagerComponent{
							Enabled: &trueValue,
							Certificate: vzapi.Certificate{
								CA: vzapi.CA{
									SecretName:               secretName,
									ClusterResourceNamespace: secretNamespace,
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
							"clusterResourceNamespace": "ns"
						  }
						}
					  }
					}`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := NewComponent().(certManagerComponent)
			got, err := c.GetModuleConfigAsHelmValues(tt.effectiveCR)
			if !tt.wantErr(t, err, fmt.Sprintf("GetModuleConfigAsHelmValues(%v)", tt.effectiveCR)) {
				return
			}
			assert.JSONEq(t, tt.want, string(got.Raw))
		})
	}
}
