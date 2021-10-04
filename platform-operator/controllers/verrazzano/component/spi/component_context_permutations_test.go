// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
package spi

import (
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	basicDevMerged                 = "testdata/basicDevMerged.yaml"
	basicProdMerged                = "testdata/basicProdMerged.yaml"
	basicManagedClusterMerged      = "testdata/basicManagedClusterMerged.yaml"
	devAllDisabledMerged           = "testdata/devAllDisabledMerged.yaml"
	devOCIDNSOverrideMerged        = "testdata/devOCIOverrideMerged.yaml"
	devCertManagerOverrideMerged   = "testdata/devCertManagerOverrideMerged.yaml"
	devElasticSearchOveridesMerged = "testdata/devESArgsStorageOverride.yaml"
	devKeycloakOveridesMerged      = "testdata/devKeycloakInstallArgsStorageOverride.yaml"
)

var falseValue = false

//var trueValue = true
//var defaultPVC50Gi, _ = resource.ParseQuantity("50Gi")
var pvc100Gi, _ = resource.ParseQuantity("100Gi")

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
			Console:           &vzapi.ConsoleComponent{MonitoringComponent: vzapi.MonitoringComponent{Enabled: &falseValue}},
			CoherenceOperator: &vzapi.CoherenceOperatorComponent{Enabled: &falseValue},
			Elasticsearch:     &vzapi.ElasticsearchComponent{MonitoringComponent: vzapi.MonitoringComponent{Enabled: &falseValue}},
			Fluentd:           &vzapi.FluentdComponent{Enabled: &falseValue},
			Grafana:           &vzapi.GrafanaComponent{MonitoringComponent: vzapi.MonitoringComponent{Enabled: &falseValue}},
			Kiali:             &vzapi.KialiComponent{Enabled: &falseValue},
			Keycloak:          &vzapi.KeycloakComponent{Enabled: &falseValue},
			Kibana:            &vzapi.KibanaComponent{MonitoringComponent: vzapi.MonitoringComponent{Enabled: &falseValue}},
			Prometheus:        &vzapi.PrometheusComponent{MonitoringComponent: vzapi.MonitoringComponent{Enabled: &falseValue}},
			Rancher:           &vzapi.RancherComponent{Enabled: &falseValue},
			WebLogicOperator:  &vzapi.WebLogicOperatorComponent{Enabled: &falseValue},
		},
	},
}
