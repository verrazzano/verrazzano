// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package istio

import (
	"fmt"
	"github.com/stretchr/testify/assert"
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"testing"
)

// TestGetModuleSpec tests the GetModuleConfigAsHelmValues function impl for this component
// GIVEN a call to GetModuleConfigAsHelmValues
//
//	WHEN for various Verrazzano CR configurations
//	THEN the generated helm values JSON snippet is valid
func TestGetModuleSpec(t *testing.T) {
	trueValue := true
	tests := []struct {
		name        string
		effectiveCR *vzapi.Verrazzano
		want        string
		wantErr     assert.ErrorAssertionFunc
	}{
		{
			name: "BasicConfig",
			effectiveCR: &vzapi.Verrazzano{
				Spec: vzapi.VerrazzanoSpec{
					Components: vzapi.ComponentSpec{
						ClusterIssuer: &vzapi.ClusterIssuerComponent{
							Enabled: &trueValue,
							IssuerConfig: vzapi.IssuerConfig{
								CA: &vzapi.CAIssuer{
									SecretName: "secret",
								},
							},
						},
						Istio: &vzapi.IstioComponent{
							Enabled:          &trueValue,
							InjectionEnabled: &trueValue,
						},
					},
				},
			},
			wantErr: assert.NoError,
			want: `{
				  "verrazzano": {
					"module": {
					  "spec": {
						"istio": {
      					  "enabled": true,
						  "injectionEnabled": true
						}
					  }
					}
				  }
				}
				`,
		},
		{
			name:        "EmptyConfig",
			effectiveCR: &vzapi.Verrazzano{},
			wantErr:     assert.NoError,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := NewComponent().(istioComponent)
			got, err := c.GetModuleConfigAsHelmValues(tt.effectiveCR)
			if !tt.wantErr(t, err, fmt.Sprintf("GetModuleConfigAsHelmValues(%v)", tt.effectiveCR)) {
				return
			}
			if len(tt.want) == 0 {
				assert.Nil(t, got)
			} else {
				assert.JSONEq(t, tt.want, string(got.Raw))
			}
		})
	}
}
