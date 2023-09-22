// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package opensearch

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
	age := "1d"

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
						Elasticsearch: &vzapi.ElasticsearchComponent{
							ESInstallArgs: createInstallArgs(3, 3, 3),
							Nodes: []vzapi.OpenSearchNode{
								createNG("a", 1, nil),
								createNG("b", 2, nil),
								createNG("c", 3, nil),
							},
							Policies: []vmov1.IndexManagementPolicy{
								{
									PolicyName:   "my-policy",
									IndexPattern: "pattern",
									MinIndexAge:  &age,
								},
							},
							Plugins: vmov1.OpenSearchPlugins{
								Enabled: false,
								InstallList: []string{
									"foo",
									"bar",
								},
							},
							DisableDefaultPolicy: true,
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
					"nodes": [
					  {
						"name": "a",
						"replicas": 1
					  },
					  {
						"name": "b",
						"replicas": 2
					  },
					  {
						"name": "c",
						"replicas": 3
					  }
					],
					"policies": [
					  {
						"policyName": "my-policy",
						"indexPattern": "pattern",
						"minIndexAge": "1d",
						"rollover": {}
					  }
					],
					"plugins": {
					  "enabled": false,
					  "installList": [
						"foo",
						"bar"
					  ]
					},
					"disableDefaultPolicy": true,
					"installArgs": [
					  {
						"name": "nodes.master.replicas",
						"value": "3"
					  },
					  {
						"name": "nodes.data.replicas",
						"value": "3"
					  },
					  {
						"name": "nodes.ingest.replicas",
						"value": "3"
					  }
					],
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
