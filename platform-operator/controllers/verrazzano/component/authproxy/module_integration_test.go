// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package authproxy

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
	ingressClassName := "myclass"
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
					EnvironmentName: "Myenv",
					Components: vzapi.ComponentSpec{
						Ingress: &vzapi.IngressNginxComponent{
							Enabled:          &trueValue,
							IngressClassName: &ingressClassName,
							Ports: []corev1.ServicePort{
								{
									Name:     "myport",
									Protocol: "tcp",
									Port:     8000,
									NodePort: 80,
								},
							},
							Type: vzapi.LoadBalancer,
						},
						DNS: &vzapi.DNSComponent{
							OCI: &vzapi.OCI{
								DNSScope:               "global",
								DNSZoneCompartmentOCID: "ocid..compartment.mycomp",
								DNSZoneOCID:            "ocid..zone.myzone",
								DNSZoneName:            "myzone",
								OCIConfigSecret:        "oci",
							},
						},
						PrometheusOperator: &vzapi.PrometheusOperatorComponent{
							Enabled: &trueValue,
						},
						AuthProxy: &vzapi.AuthProxyComponent{
							Enabled: &trueValue,
							Kubernetes: &vzapi.AuthProxyKubernetesSection{
								CommonKubernetesSpec: vzapi.CommonKubernetesSpec{
									Replicas: 3,
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
					"kubernetes": {
					  "replicas": 3
					}
				  }
				}
			  }
			}`,
		},
		{
			name: "OtherComponentConfigsHaveNoEffect",
			effectiveCR: &vzapi.Verrazzano{
				Spec: vzapi.VerrazzanoSpec{
					EnvironmentName: "Myenv",
					Components: vzapi.ComponentSpec{
						Ingress: &vzapi.IngressNginxComponent{
							Enabled:          &trueValue,
							IngressClassName: &ingressClassName,
							Ports: []corev1.ServicePort{
								{
									Name:     "myport",
									Protocol: "tcp",
									Port:     8000,
									NodePort: 80,
								},
							},
							Type: vzapi.LoadBalancer,
						},
						DNS: &vzapi.DNSComponent{
							OCI: &vzapi.OCI{
								DNSScope:               "global",
								DNSZoneCompartmentOCID: "ocid..compartment.mycomp",
								DNSZoneOCID:            "ocid..zone.myzone",
								DNSZoneName:            "myzone",
								OCIConfigSecret:        "oci",
							},
						},
						PrometheusOperator: &vzapi.PrometheusOperatorComponent{
							Enabled: &trueValue,
						},
						AuthProxy: &vzapi.AuthProxyComponent{
							Enabled: &trueValue,
							Kubernetes: &vzapi.AuthProxyKubernetesSection{
								CommonKubernetesSpec: vzapi.CommonKubernetesSpec{
									Replicas: 3,
								},
							},
						},
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
					},
				},
			},
			wantErr: assert.NoError,
			want: `{
			  "verrazzano": {
				"module": {
				  "spec": {
					"kubernetes": {
					  "replicas": 3
					}
				  }
				}
			  }
			}`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := NewComponent().(authProxyComponent)
			got, err := c.GetModuleConfigAsHelmValues(tt.effectiveCR)
			if !tt.wantErr(t, err, fmt.Sprintf("GetModuleConfigAsHelmValues(%v)", tt.effectiveCR)) {
				return
			}
			assert.JSONEq(t, tt.want, string(got.Raw))
		})
	}
}
