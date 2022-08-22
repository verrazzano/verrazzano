// Copyright (c) 2021, 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
package spi

import (
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
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

//var defaultPVC50Gi, _ = resource.ParseQuantity("50Gi")
var pvc100Gi, _ = resource.ParseQuantity("100Gi")
var pvc500Gi, _ = resource.ParseQuantity("2T")

var basicDevWithStatus = vzapi.Verrazzano{
	ObjectMeta: metav1.ObjectMeta{
		Name: "default-dev",
	},
	Spec: vzapi.VerrazzanoSpec{
		Profile: "dev",
	},
	Status: vzapi.VerrazzanoStatus{
		Version:            "v1.0.1",
		VerrazzanoInstance: &vzapi.InstanceInfo{},
	},
}

var basicProdWithStatus = vzapi.Verrazzano{
	ObjectMeta: metav1.ObjectMeta{
		Name: "default-prod",
	},
	Spec: vzapi.VerrazzanoSpec{
		Profile: "prod",
	},
	Status: vzapi.VerrazzanoStatus{
		Version:            "v1.0.1",
		VerrazzanoInstance: &vzapi.InstanceInfo{},
	},
}

var basicMgdClusterWithStatus = vzapi.Verrazzano{
	ObjectMeta: metav1.ObjectMeta{
		Name: "default-mgd",
	},
	Spec: vzapi.VerrazzanoSpec{
		Profile: "managed-cluster",
	},
	Status: vzapi.VerrazzanoStatus{
		Version:            "v1.0.1",
		VerrazzanoInstance: &vzapi.InstanceInfo{},
	},
}

var devOCIDNSOverride = vzapi.Verrazzano{
	ObjectMeta: metav1.ObjectMeta{
		Name: "dev-dns-override",
	},
	Spec: vzapi.VerrazzanoSpec{
		Profile: "dev",
		Components: vzapi.ComponentSpec{
			DNS: &vzapi.DNSComponent{
				OCI: &vzapi.OCI{
					OCIConfigSecret:        "mysecret",
					DNSZoneCompartmentOCID: "compartment-ocid",
					DNSZoneOCID:            "zone-ocid",
					DNSZoneName:            "myzone.com",
				},
			},
		},
	},
}

var devCertManagerNoCert = vzapi.Verrazzano{
	ObjectMeta: metav1.ObjectMeta{
		Name: "default-dev",
	},
	Spec: vzapi.VerrazzanoSpec{
		Profile: "dev",
		Components: vzapi.ComponentSpec{
			CertManager: &vzapi.CertManagerComponent{
				Enabled: &trueValue,
			},
		},
	},
}

var devCertManagerOverride = vzapi.Verrazzano{
	ObjectMeta: metav1.ObjectMeta{
		Name: "dev-cm-override",
	},
	Spec: vzapi.VerrazzanoSpec{
		Profile: "dev",
		Components: vzapi.ComponentSpec{
			CertManager: &vzapi.CertManagerComponent{
				Certificate: vzapi.Certificate{
					Acme: vzapi.Acme{
						Provider:     "letsencrypt",
						EmailAddress: "myemail",
						Environment:  "production",
					},
				},
			},
		},
	},
}

var devElasticSearchOverrides = vzapi.Verrazzano{
	ObjectMeta: metav1.ObjectMeta{
		Name: "dev-es-override",
	},
	Spec: vzapi.VerrazzanoSpec{
		Profile: "dev",
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
				ESInstallArgs: []vzapi.InstallArgs{
					{Name: "nodes.master.replicas", Value: "3"},
					{Name: "nodes.master.requests.memory", Value: "3G"},
				},
			},
		},
	},
}

var devKeycloakOverrides = vzapi.Verrazzano{
	ObjectMeta: metav1.ObjectMeta{
		Name: "dev-keycloak-override",
	},
	Spec: vzapi.VerrazzanoSpec{
		Profile: "dev",
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
			Keycloak: &vzapi.KeycloakComponent{
				KeycloakInstallArgs: []vzapi.InstallArgs{
					{Name: "some.keycloak.arg1", Value: "val1"},
					{Name: "some.keycloak.arg2", Value: "val2"},
				},
				MySQL: vzapi.MySQLComponent{
					MySQLInstallArgs: []vzapi.InstallArgs{
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

var devAllDisabledOverride = vzapi.Verrazzano{
	ObjectMeta: metav1.ObjectMeta{
		Name: "dev-disable-all-override",
	},
	Spec: vzapi.VerrazzanoSpec{
		Profile: "dev",
		Components: vzapi.ComponentSpec{
			Console:           &vzapi.ConsoleComponent{Enabled: &falseValue},
			CoherenceOperator: &vzapi.CoherenceOperatorComponent{Enabled: &falseValue},
			Elasticsearch:     &vzapi.ElasticsearchComponent{Enabled: &falseValue},
			Fluentd:           &vzapi.FluentdComponent{Enabled: &falseValue},
			Grafana:           &vzapi.GrafanaComponent{Enabled: &falseValue},
			Kiali:             &vzapi.KialiComponent{Enabled: &falseValue},
			Keycloak:          &vzapi.KeycloakComponent{Enabled: &falseValue},
			MySQLOperator:     &vzapi.MySQLOperatorComponent{Enabled: &falseValue},
			Kibana:            &vzapi.KibanaComponent{Enabled: &falseValue},
			Prometheus:        &vzapi.PrometheusComponent{Enabled: &falseValue},
			Rancher:           &vzapi.RancherComponent{Enabled: &falseValue},
			WebLogicOperator:  &vzapi.WebLogicOperatorComponent{Enabled: &falseValue},
		},
	},
}

var prodElasticSearchOverrides = vzapi.Verrazzano{
	ObjectMeta: metav1.ObjectMeta{
		Name: "prod-es-override",
	},
	Spec: vzapi.VerrazzanoSpec{
		EnvironmentName: "prodenv",
		Profile:         "prod",
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
							"storage": pvc500Gi,
						},
					},
				},
			},
		},
		Components: vzapi.ComponentSpec{
			Elasticsearch: &vzapi.ElasticsearchComponent{
				ESInstallArgs: []vzapi.InstallArgs{
					{Name: "nodes.master.replicas", Value: "3"},
					{Name: "nodes.master.requests.memory", Value: "3G"},
					{Name: "nodes.ingest.replicas", Value: "6"},
					{Name: "nodes.ingest.requests.memory", Value: "32G"},
					{Name: "nodes.data.replicas", Value: "6"},
					{Name: "nodes.data.requests.memory", Value: "32G"},
				},
			},
		},
	},
}

var prodElasticSearchStorageArgs = vzapi.Verrazzano{
	ObjectMeta: metav1.ObjectMeta{
		Name: "prod-es-override",
	},
	Spec: vzapi.VerrazzanoSpec{
		EnvironmentName: "prodenv",
		Profile:         "prod",
		Components: vzapi.ComponentSpec{
			Elasticsearch: &vzapi.ElasticsearchComponent{
				ESInstallArgs: []vzapi.InstallArgs{
					{Name: "nodes.master.replicas", Value: "3"},
					{Name: "nodes.master.requests.memory", Value: "3G"},
					{Name: "nodes.master.requests.storage", Value: "100Gi"},
					{Name: "nodes.ingest.replicas", Value: "6"},
					{Name: "nodes.ingest.requests.memory", Value: "32G"},
					{Name: "nodes.data.replicas", Value: "6"},
					{Name: "nodes.data.requests.memory", Value: "32G"},
					{Name: "nodes.data.requests.storage", Value: "150Gi"},
				},
			},
		},
	},
}

var prodIngressIstioOverrides = vzapi.Verrazzano{
	ObjectMeta: metav1.ObjectMeta{
		Name: "prod-ingress-istio-override",
	},
	Spec: vzapi.VerrazzanoSpec{
		EnvironmentName: "prodenv",
		Profile:         "prod",
		Components: vzapi.ComponentSpec{
			Ingress: &vzapi.IngressNginxComponent{
				NGINXInstallArgs: []vzapi.InstallArgs{
					{Name: "controller.service.annotations.\"service\\.beta\\.kubernetes\\.io/oci-load-balancer-shape\"", Value: "10Mbps"},
					{Name: "controller.service.externalTrafficPolicy", Value: "Local"},
					{Name: "controller.service.externalIPs", ValueList: []string{"11.22.33.44"}},
				},
				Ports: []corev1.ServicePort{
					{Name: "http", NodePort: 30080, Port: 80, Protocol: corev1.ProtocolTCP, TargetPort: intstr.FromInt(0)},
					{Name: "https", NodePort: 30443, Port: 443, Protocol: corev1.ProtocolTCP, TargetPort: intstr.FromInt(0)},
				},
			},
			Istio: &vzapi.IstioComponent{
				IstioInstallArgs: []vzapi.InstallArgs{
					{Name: "gateways.istio-ingressgateway.serviceAnnotations.\"service\\.beta\\.kubernetes\\.io/oci-load-balancer-shape\"", Value: "10Mbps"},
					{Name: "gateways.istio-ingressgateway.replicaCount", Value: "3"},
					{Name: "gateways.istio-ingressgateway.externalIPs", ValueList: []string{"11.22.33.44"}},
				},
			},
		},
	},
}

var prodFluentdOverrides = vzapi.Verrazzano{
	ObjectMeta: metav1.ObjectMeta{
		Name: "prod-fluentd-override",
	},
	Spec: vzapi.VerrazzanoSpec{
		EnvironmentName: "prodenv",
		Profile:         "prod",
		Components: vzapi.ComponentSpec{
			Fluentd: &vzapi.FluentdComponent{
				ExtraVolumeMounts: []vzapi.VolumeMount{
					{Source: "/u01/datarw", ReadOnly: &falseValue},
					{Source: "/u01/dataro", ReadOnly: &trueValue},
				},
				ElasticsearchURL:    "https://myes.mycorp.com",
				ElasticsearchSecret: "mysecret",
			},
		},
	},
}

var managedClusterEnableAllOverride = vzapi.Verrazzano{
	ObjectMeta: metav1.ObjectMeta{
		Name: "managed-enable-all-override",
	},
	Spec: vzapi.VerrazzanoSpec{
		Profile: "managed-cluster",
		Components: vzapi.ComponentSpec{
			Console:       &vzapi.ConsoleComponent{Enabled: &trueValue},
			Elasticsearch: &vzapi.ElasticsearchComponent{Enabled: &trueValue},
			Grafana:       &vzapi.GrafanaComponent{Enabled: &trueValue},
			Kiali:         &vzapi.KialiComponent{Enabled: &trueValue},
			Keycloak:      &vzapi.KeycloakComponent{Enabled: &trueValue},
			Kibana:        &vzapi.KibanaComponent{Enabled: &trueValue},
			Prometheus:    &vzapi.PrometheusComponent{Enabled: &trueValue},
			Rancher:       &vzapi.RancherComponent{Enabled: &trueValue},
		},
	},
}
