// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package fluentd

import (
	"fmt"
	"github.com/stretchr/testify/assert"
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	corev1 "k8s.io/api/core/v1"
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
	falseValue := false

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
						Fluentd: &vzapi.FluentdComponent{
							ElasticsearchSecret: "esSecret",
							ElasticsearchURL:    "esURL",
							Enabled:             &trueValue,
							ExtraVolumeMounts: []vzapi.VolumeMount{
								{Source: "asource", ReadOnly: &trueValue},
								{Source: "bsource", ReadOnly: &falseValue},
							},
							OCI: &vzapi.OciLoggingConfiguration{
								APISecret:       "ociSecret",
								DefaultAppLogID: "defaultApp",
								SystemLogID:     "systemLogID",
							},
							InstallOverrides: vzapi.InstallOverrides{
								MonitorChanges: &trueValue,
								ValueOverrides: []vzapi.Overrides{
									{Values: &apiextensionsv1.JSON{Raw: []byte(`someJSON{}`)}},
									{ConfigMapRef: &corev1.ConfigMapKeySelector{LocalObjectReference: corev1.LocalObjectReference{Name: "cfgmap"}}},
								},
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
					"elasticsearchSecret": "esSecret",
					"elasticsearchURL": "esURL",
					"extraVolumeMounts": [
					  {
						"readOnly": true,
						"source": "asource"
					  },
					  {
						"readOnly": false,
						"source": "bsource"
					  }
					],
					"oci": {
					  "apiSecret": "ociSecret",
					  "defaultAppLogId": "defaultApp",
					  "systemLogId": "systemLogID"
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
			c := NewComponent().(fluentdComponent)
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
