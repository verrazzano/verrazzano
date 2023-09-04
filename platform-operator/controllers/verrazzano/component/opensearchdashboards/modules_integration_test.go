// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package opensearchdashboards

import (
	"fmt"
	"github.com/stretchr/testify/assert"
	vmov1 "github.com/verrazzano/verrazzano-monitoring-operator/pkg/apis/vmcontroller/v1"
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"testing"
)

var pvc100Gi, _ = resource.ParseQuantity("100Gi")

// TestGetModuleSpec tests the GetModuleConfigAsHelmValues function impl for this component
// GIVEN a call to GetModuleConfigAsHelmValues
//
//	WHEN for various Verrazzano CR configurations
//	THEN the generated helm values JSON snippet is valid
func TestGetModuleSpec(t *testing.T) {
	trueValue := true
	var replicas int32 = 3

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
					DefaultVolumeSource: &corev1.VolumeSource{
						PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
							ClaimName: "vmi",
						},
					},
					VolumeClaimSpecTemplates: []vzapi.VolumeClaimSpecTemplate{
						{
							ObjectMeta: metav1.ObjectMeta{Name: "vmi"},
							Spec: corev1.PersistentVolumeClaimSpec{
								Resources: corev1.ResourceRequirements{
									Requests: corev1.ResourceList{
										"storage": pvc100Gi,
									},
								},
							},
						},
					},
					Components: vzapi.ComponentSpec{
						Kibana: &vzapi.KibanaComponent{
							Enabled:  &trueValue,
							Replicas: &replicas,
							Plugins: vmov1.OpenSearchDashboardsPlugins{
								Enabled: false,
								InstallList: []string{
									"foo",
									"bar",
								},
							},
						},
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
						Prometheus: &vzapi.PrometheusComponent{
							Enabled: &trueValue,
						},
						Thanos: &vzapi.ThanosComponent{
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
					"replicas": 3,
					"plugins": {
					  "enabled": false,
					  "installList": [
						"foo",
						"bar"
					  ]
					},
					"ingress": {
					  "enabled": true,
					  "ingressClassName": "myclass",
					  "ports": [
						{
						  "name": "myport",
						  "protocol": "tcp",
						  "port": 8000,
						  "targetPort": 0,
						  "nodePort": 80
						}
					  ],
					  "type": "LoadBalancer"
					},
					"dns": {
					  "oci": {
						"dnsScope": "global",
						"dnsZoneCompartmentOCID": "ocid..compartment.mycomp",
						"dnsZoneOCID": "ocid..zone.myzone",
						"dnsZoneName": "myzone",
						"ociConfigSecret": "oci"
					  }
					},
					"environmentName": "Myenv",
					"defaultVolumeSource": {
					  "persistentVolumeClaim": {
						"claimName": "vmi"
					  }
					},
					"volumeClaimSpecTemplates": [
					  {
						"metadata": {
						  "name": "vmi",
						  "creationTimestamp": null
						},
						"spec": {
						  "resources": {
							"requests": {
							  "storage": "100Gi"
							}
						  }
						}
					  }
					]
				  }
				}
			  }
			}
			`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := NewComponent()
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
