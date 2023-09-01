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
						"injectionEnabled": true
					  }
					}
				  }
				}
				`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := NewComponent().(istioComponent)
			got, err := c.GetModuleConfigAsHelmValues(tt.effectiveCR)
			if !tt.wantErr(t, err, fmt.Sprintf("GetModuleConfigAsHelmValues(%v)", tt.effectiveCR)) {
				return
			}
			assert.JSONEq(t, tt.want, string(got.Raw))
		})
	}
}

// TestGetWatchDescriptors tests the GetWatchDescriptors function impl for this component
// GIVEN a call to GetWatchDescriptors
//
//	WHEN a new component is created
//	THEN the watch descriptors have the correct number of watches
func TestGetWatchDescriptors(t *testing.T) {
	wd := NewComponent().GetWatchDescriptors()
	assert.Len(t, wd, 1)
}
