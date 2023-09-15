// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package keycloak

//var pvc100Gi, _ = resource.ParseQuantity("100Gi")

// TestGetModuleSpec tests the GetModuleConfigAsHelmValues function impl for this component
// GIVEN a call to GetModuleConfigAsHelmValues
//
//	WHEN for various Verrazzano CR configurations
//	THEN the generated helm values JSON snippet is valid
//func TestGetModuleSpec(t *testing.T) {
//	trueValue := true
//	ingressClassName := "myclass"
//
//	tests := []struct {
//		name        string
//		effectiveCR *vzapi.Verrazzano
//		want        string
//		wantErr     assert.ErrorAssertionFunc
//	}{
//		{
//			name: "BasicConfig",
//			effectiveCR: &vzapi.Verrazzano{
//				Spec: vzapi.VerrazzanoSpec{
//					EnvironmentName: "Myenv",
//					DefaultVolumeSource: &corev1.VolumeSource{
//						PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
//							ClaimName: "vmi",
//						},
//					},
//					VolumeClaimSpecTemplates: []vzapi.VolumeClaimSpecTemplate{
//						{
//							ObjectMeta: metav1.ObjectMeta{Name: "vmi"},
//							Spec: corev1.PersistentVolumeClaimSpec{
//								Resources: corev1.ResourceRequirements{
//									Requests: corev1.ResourceList{
//										"storage": pvc100Gi,
//									},
//								},
//							},
//						},
//					},
//					Components: vzapi.ComponentSpec{
//						Ingress: &vzapi.IngressNginxComponent{
//							Enabled:          &trueValue,
//							IngressClassName: &ingressClassName,
//							Ports: []corev1.ServicePort{
//								{
//									Name:     "myport",
//									Protocol: "tcp",
//									Port:     8000,
//									NodePort: 80,
//								},
//							},
//							Type: vzapi.LoadBalancer,
//						},
//						DNS: &vzapi.DNSComponent{
//							OCI: &vzapi.OCI{
//								DNSScope:               "global",
//								DNSZoneCompartmentOCID: "ocid..compartment.mycomp",
//								DNSZoneOCID:            "ocid..zone.myzone",
//								DNSZoneName:            "myzone",
//								OCIConfigSecret:        "oci",
//							},
//						},
//						PrometheusOperator: &vzapi.PrometheusOperatorComponent{
//							Enabled: &trueValue,
//						},
//						Prometheus: &vzapi.PrometheusComponent{
//							Enabled: &trueValue,
//						},
//						Thanos: &vzapi.ThanosComponent{
//							Enabled: &trueValue,
//						},
//						AuthProxy: &vzapi.AuthProxyComponent{
//							Enabled: &trueValue,
//							Kubernetes: &vzapi.AuthProxyKubernetesSection{
//								CommonKubernetesSpec: vzapi.CommonKubernetesSpec{
//									Replicas: 3,
//								},
//							},
//						},
//						Keycloak: &vzapi.KeycloakComponent{
//							Enabled: &trueValue,
//							InstallOverrides: vzapi.InstallOverrides{
//								MonitorChanges: &trueValue,
//								ValueOverrides: []vzapi.Overrides{
//									{Values: &apiextensionsv1.JSON{Raw: []byte("somevalue")}},
//								},
//							},
//							KeycloakInstallArgs: nil,
//							MySQL:               vzapi.MySQLComponent{},
//						},
//					},
//				},
//			},
//			wantErr: assert.NoError,
//			want: `{
//			  "verrazzano": {
//				"module": {
//				  "spec": {
//					"ingress": {
//					  "enabled": true,
//					  "ingressClassName": "myclass",
//					  "ports": [
//						{
//						  "name": "myport",
//						  "protocol": "tcp",
//						  "port": 8000,
//						  "targetPort": 0,
//						  "nodePort": 80
//						}
//					  ],
//					  "type": "LoadBalancer"
//					},
//					"dns": {
//					  "oci": {
//						"dnsScope": "global",
//						"dnsZoneCompartmentOCID": "ocid..compartment.mycomp",
//						"dnsZoneOCID": "ocid..zone.myzone",
//						"dnsZoneName": "myzone",
//						"ociConfigSecret": "oci"
//					  }
//					},
//					"environmentName": "Myenv"
//				  }
//				}
//			  }
//			}
//			`,
//		},
//	}
//	for _, tt := range tests {
//		t.Run(tt.name, func(t *testing.T) {
//			c := NewComponent()
//			got, err := c.GetModuleConfigAsHelmValues(tt.effectiveCR)
//			if !tt.wantErr(t, err, fmt.Sprintf("GetModuleConfigAsHelmValues(%v)", tt.effectiveCR)) {
//				return
//			}
//			assert.JSONEq(t, tt.want, string(got.Raw))
//		})
//	}
//}
