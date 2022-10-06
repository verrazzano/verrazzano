// Copyright (c) 2021, 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
package spi

import (
	"github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

const (
	basicDevMerged                  = "testdata/basicDevMerged.yaml"
	basicProdMerged                 = "testdata/basicProdMerged.yaml"
	basicManagedClusterMerged       = "testdata/basicManagedClusterMerged.yaml"
	devAllDisabledMerged            = "testdata/devAllDisabledMerged.yaml"
	devOCIDNSOverrideMerged         = "testdata/devOCIOverrideMerged.yaml"
	devCertManagerOverrideMerged    = "testdata/devCertManagerOverrideMerged.yaml"
	devElasticSearchOveridesMerged  = "testdata/devESArgsStorageOverride.yaml"
	devKeycloakOveridesMerged       = "testdata/devKeycloakInstallArgsStorageOverride.yaml"
	prodElasticSearchOveridesMerged = "testdata/prodESOverridesMerged.yaml"
	prodElasticSearchStorageMerged  = "testdata/prodESStorageArgsMerged.yaml"
	prodIngressIstioOverridesMerged = "testdata/prodIngressIstioOverridesMerged.yaml"
	prodFluentdOverridesMerged      = "testdata/prodFluentdOverridesMerged.yaml"
	managedClusterEnableAllMerged   = "testdata/managedClusterEnableAllOverrideMerged.yaml"
)

var falseValue = false

var trueValue = true

// var defaultPVC50Gi, _ = resource.ParseQuantity("50Gi")
var pvc100Gi, _ = resource.ParseQuantity("100Gi")
var pvc500Gi, _ = resource.ParseQuantity("2T")

var basicDevWithStatus = v1alpha1.Verrazzano{
	ObjectMeta: metav1.ObjectMeta{
		Name: "default-dev",
	},
	Spec: v1alpha1.VerrazzanoSpec{
		Profile: "dev",
	},
	Status: v1alpha1.VerrazzanoStatus{
		Version:            "v1.0.1",
		VerrazzanoInstance: &v1alpha1.InstanceInfo{},
	},
}

var basicProdWithStatus = v1alpha1.Verrazzano{
	ObjectMeta: metav1.ObjectMeta{
		Name: "default-prod",
	},
	Spec: v1alpha1.VerrazzanoSpec{
		Profile: "prod",
	},
	Status: v1alpha1.VerrazzanoStatus{
		Version:            "v1.0.1",
		VerrazzanoInstance: &v1alpha1.InstanceInfo{},
	},
}

var basicMgdClusterWithStatus = v1alpha1.Verrazzano{
	ObjectMeta: metav1.ObjectMeta{
		Name: "default-mgd",
	},
	Spec: v1alpha1.VerrazzanoSpec{
		Profile: "managed-cluster",
	},
	Status: v1alpha1.VerrazzanoStatus{
		Version:            "v1.0.1",
		VerrazzanoInstance: &v1alpha1.InstanceInfo{},
	},
}

var devOCIDNSOverride = v1alpha1.Verrazzano{
	ObjectMeta: metav1.ObjectMeta{
		Name: "dev-dns-override",
	},
	Spec: v1alpha1.VerrazzanoSpec{
		Profile: "dev",
		Components: v1alpha1.ComponentSpec{
			DNS: &v1alpha1.DNSComponent{
				OCI: &v1alpha1.OCI{
					OCIConfigSecret:        "mysecret",
					DNSZoneCompartmentOCID: "compartment-ocid",
					DNSZoneOCID:            "zone-ocid",
					DNSZoneName:            "myzone.com",
				},
			},
		},
	},
}

var devCertManagerNoCert = v1alpha1.Verrazzano{
	ObjectMeta: metav1.ObjectMeta{
		Name: "default-dev",
	},
	Spec: v1alpha1.VerrazzanoSpec{
		Profile: "dev",
		Components: v1alpha1.ComponentSpec{
			CertManager: &v1alpha1.CertManagerComponent{
				Enabled: &trueValue,
			},
		},
	},
}

var devCertManagerOverride = v1alpha1.Verrazzano{
	ObjectMeta: metav1.ObjectMeta{
		Name: "dev-cm-override",
	},
	Spec: v1alpha1.VerrazzanoSpec{
		Profile: "dev",
		Components: v1alpha1.ComponentSpec{
			CertManager: &v1alpha1.CertManagerComponent{
				Certificate: v1alpha1.Certificate{
					Acme: v1alpha1.Acme{
						Provider:     "letsencrypt",
						EmailAddress: "myemail",
						Environment:  "production",
					},
				},
			},
		},
	},
}

var devElasticSearchOverrides = v1alpha1.Verrazzano{
	ObjectMeta: metav1.ObjectMeta{
		Name: "dev-es-override",
	},
	Spec: v1alpha1.VerrazzanoSpec{
		Profile: "dev",
		DefaultVolumeSource: &corev1.VolumeSource{
			PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
				ClaimName: "vmi",
			},
		},
		VolumeClaimSpecTemplates: []v1alpha1.VolumeClaimSpecTemplate{
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
		Components: v1alpha1.ComponentSpec{
			Elasticsearch: &v1alpha1.ElasticsearchComponent{
				Nodes: []v1alpha1.OpenSearchNode{
					{
						Name:     "es-master",
						Replicas: 3,
						Resources: &corev1.ResourceRequirements{
							Requests: map[corev1.ResourceName]resource.Quantity{
								corev1.ResourceMemory: resource.MustParse("3G"),
							},
						},
					},
				},
			},
		},
	},
}

var devKeycloakOverrides = v1alpha1.Verrazzano{
	ObjectMeta: metav1.ObjectMeta{
		Name: "dev-keycloak-override",
	},
	Spec: v1alpha1.VerrazzanoSpec{
		Profile: "dev",
		VolumeClaimSpecTemplates: []v1alpha1.VolumeClaimSpecTemplate{
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
		Components: v1alpha1.ComponentSpec{
			Keycloak: &v1alpha1.KeycloakComponent{
				KeycloakInstallArgs: []v1alpha1.InstallArgs{
					{Name: "some.keycloak.arg1", Value: "val1"},
					{Name: "some.keycloak.arg2", Value: "val2"},
				},
				MySQL: v1alpha1.MySQLComponent{
					MySQLInstallArgs: []v1alpha1.InstallArgs{
						{Name: "some.mysql.arg1", Value: "val1"},
						{Name: "some.mysql.arg2", Value: "val2"},
					},
					VolumeSource: &corev1.VolumeSource{
						PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
							ClaimName: "vmi",
						},
					},
				},
			},
		},
	},
}

var devAllDisabledOverride = v1alpha1.Verrazzano{
	ObjectMeta: metav1.ObjectMeta{
		Name: "dev-disable-all-override",
	},
	Spec: v1alpha1.VerrazzanoSpec{
		Profile: "dev",
		Components: v1alpha1.ComponentSpec{
			Console:           &v1alpha1.ConsoleComponent{Enabled: &falseValue},
			CoherenceOperator: &v1alpha1.CoherenceOperatorComponent{Enabled: &falseValue},
			Elasticsearch:     &v1alpha1.ElasticsearchComponent{Enabled: &falseValue},
			Fluentd:           &v1alpha1.FluentdComponent{Enabled: &falseValue},
			Grafana:           &v1alpha1.GrafanaComponent{Enabled: &falseValue},
			Kiali:             &v1alpha1.KialiComponent{Enabled: &falseValue},
			Keycloak:          &v1alpha1.KeycloakComponent{Enabled: &falseValue},
			MySQLOperator:     &v1alpha1.MySQLOperatorComponent{Enabled: &falseValue},
			Kibana:            &v1alpha1.KibanaComponent{Enabled: &falseValue},
			Prometheus:        &v1alpha1.PrometheusComponent{Enabled: &falseValue},
			Rancher:           &v1alpha1.RancherComponent{Enabled: &falseValue},
			WebLogicOperator:  &v1alpha1.WebLogicOperatorComponent{Enabled: &falseValue},
		},
	},
}

var prodElasticSearchOverrides = v1alpha1.Verrazzano{
	ObjectMeta: metav1.ObjectMeta{
		Name: "prod-es-override",
	},
	Spec: v1alpha1.VerrazzanoSpec{
		EnvironmentName: "prodenv",
		Profile:         "prod",
		DefaultVolumeSource: &corev1.VolumeSource{
			PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
				ClaimName: "vmi",
			},
		},
		VolumeClaimSpecTemplates: []v1alpha1.VolumeClaimSpecTemplate{
			{
				ObjectMeta: metav1.ObjectMeta{Name: "vmi"},
				Spec: corev1.PersistentVolumeClaimSpec{
					Resources: corev1.ResourceRequirements{
						Requests: corev1.ResourceList{
							"storage": pvc500Gi,
						},
					},
				},
			},
		},
		Components: v1alpha1.ComponentSpec{
			Elasticsearch: &v1alpha1.ElasticsearchComponent{
				Nodes: []v1alpha1.OpenSearchNode{
					{
						Name:     "es-master",
						Replicas: 3,
						Resources: &corev1.ResourceRequirements{
							Requests: map[corev1.ResourceName]resource.Quantity{
								corev1.ResourceMemory: resource.MustParse("3G"),
							},
						},
						Storage: &v1alpha1.OpenSearchNodeStorage{
							Size: "50Gi",
						},
					},
					{
						Name:     "es-data",
						Replicas: 6,
						Resources: &corev1.ResourceRequirements{
							Requests: map[corev1.ResourceName]resource.Quantity{
								corev1.ResourceMemory: resource.MustParse("32G"),
							},
						},
						Storage: &v1alpha1.OpenSearchNodeStorage{
							Size: "50Gi",
						},
					},
					{
						Name:     "es-ingest",
						Replicas: 6,
						Resources: &corev1.ResourceRequirements{
							Requests: map[corev1.ResourceName]resource.Quantity{
								corev1.ResourceMemory: resource.MustParse("32G"),
							},
						},
					},
				},
			},
		},
	},
}

var prodElasticSearchStorageArgs = v1alpha1.Verrazzano{
	ObjectMeta: metav1.ObjectMeta{
		Name: "prod-es-override",
	},
	Spec: v1alpha1.VerrazzanoSpec{
		EnvironmentName: "prodenv",
		Profile:         "prod",
		Components: v1alpha1.ComponentSpec{
			Elasticsearch: &v1alpha1.ElasticsearchComponent{
				Nodes: []v1alpha1.OpenSearchNode{
					{
						Name:     "es-master",
						Replicas: 3,
						Resources: &corev1.ResourceRequirements{
							Requests: map[corev1.ResourceName]resource.Quantity{
								corev1.ResourceMemory: resource.MustParse("3G"),
							},
						},
						Storage: &v1alpha1.OpenSearchNodeStorage{
							Size: "100Gi",
						},
					},
					{
						Name:     "es-data",
						Replicas: 6,
						Resources: &corev1.ResourceRequirements{
							Requests: map[corev1.ResourceName]resource.Quantity{
								corev1.ResourceMemory: resource.MustParse("32G"),
							},
						},
						Storage: &v1alpha1.OpenSearchNodeStorage{
							Size: "150Gi",
						},
					},
					{
						Name:     "es-ingest",
						Replicas: 6,
						Resources: &corev1.ResourceRequirements{
							Requests: map[corev1.ResourceName]resource.Quantity{
								corev1.ResourceMemory: resource.MustParse("32G"),
							},
						},
					},
				},
			},
		},
	},
}

var prodIngressIstioOverrides = v1alpha1.Verrazzano{
	ObjectMeta: metav1.ObjectMeta{
		Name: "prod-ingress-istio-override",
	},
	Spec: v1alpha1.VerrazzanoSpec{
		EnvironmentName: "prodenv",
		Profile:         "prod",
		Components: v1alpha1.ComponentSpec{
			Ingress: &v1alpha1.IngressNginxComponent{
				NGINXInstallArgs: []v1alpha1.InstallArgs{
					{Name: "controller.service.annotations.\"service\\.beta\\.kubernetes\\.io/oci-load-balancer-shape\"", Value: "10Mbps"},
					{Name: "controller.service.externalTrafficPolicy", Value: "Local"},
					{Name: "controller.service.externalIPs", ValueList: []string{"11.22.33.44"}},
				},
				Ports: []corev1.ServicePort{
					{Name: "http", NodePort: 30080, Port: 80, Protocol: corev1.ProtocolTCP, TargetPort: intstr.FromInt(0)},
					{Name: "https", NodePort: 30443, Port: 443, Protocol: corev1.ProtocolTCP, TargetPort: intstr.FromInt(0)},
				},
			},
			Istio: &v1alpha1.IstioComponent{
				IstioInstallArgs: []v1alpha1.InstallArgs{
					{Name: "gateways.istio-ingressgateway.serviceAnnotations.\"service\\.beta\\.kubernetes\\.io/oci-load-balancer-shape\"", Value: "10Mbps"},
					{Name: "gateways.istio-ingressgateway.replicaCount", Value: "3"},
					{Name: "gateways.istio-ingressgateway.externalIPs", ValueList: []string{"11.22.33.44"}},
				},
			},
		},
	},
}

var prodFluentdOverrides = v1alpha1.Verrazzano{
	ObjectMeta: metav1.ObjectMeta{
		Name: "prod-fluentd-override",
	},
	Spec: v1alpha1.VerrazzanoSpec{
		EnvironmentName: "prodenv",
		Profile:         "prod",
		Components: v1alpha1.ComponentSpec{
			Fluentd: &v1alpha1.FluentdComponent{
				ExtraVolumeMounts: []v1alpha1.VolumeMount{
					{Source: "/u01/datarw", ReadOnly: &falseValue},
					{Source: "/u01/dataro", ReadOnly: &trueValue},
				},
				ElasticsearchURL:    "https://myes.mycorp.com",
				ElasticsearchSecret: "mysecret",
			},
		},
	},
}

var managedClusterEnableAllOverride = v1alpha1.Verrazzano{
	ObjectMeta: metav1.ObjectMeta{
		Name: "managed-enable-all-override",
	},
	Spec: v1alpha1.VerrazzanoSpec{
		Profile: "managed-cluster",
		Components: v1alpha1.ComponentSpec{
			Console:       &v1alpha1.ConsoleComponent{Enabled: &trueValue},
			Elasticsearch: &v1alpha1.ElasticsearchComponent{Enabled: &trueValue},
			Grafana:       &v1alpha1.GrafanaComponent{Enabled: &trueValue},
			Kiali:         &v1alpha1.KialiComponent{Enabled: &trueValue},
			Keycloak:      &v1alpha1.KeycloakComponent{Enabled: &trueValue},
			Kibana:        &v1alpha1.KibanaComponent{Enabled: &trueValue},
			Prometheus:    &v1alpha1.PrometheusComponent{Enabled: &trueValue},
			Rancher:       &v1alpha1.RancherComponent{Enabled: &trueValue},
		},
	},
}
