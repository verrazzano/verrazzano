// Copyright (c) 2020, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package installjob

import (
	"fmt"
	"testing"

	"k8s.io/apimachinery/pkg/util/intstr"

	"github.com/stretchr/testify/assert"
	installv1alpha1 "github.com/verrazzano/verrazzano/api/v1alpha1"
	corev1 "k8s.io/api/core/v1"
)

// TestXipIoInstallDefaults tests the creation of an xip.io install default configuration
// GIVEN a verrazzano.install.verrazzano.io custom resource
//  WHEN I call GetInstallConfig
//  THEN the xip.io install configuration is created and verified
func TestXipIoInstallDefaults(t *testing.T) {
	vz := installv1alpha1.Verrazzano{}
	config, _ := GetInstallConfig(&vz, nil)
	assert.Equalf(t, "default", config.EnvironmentName, "Expected environment name did not match")
	assert.Equalf(t, InstallProfileProd, config.Profile, "Expected profile did not match")
	assert.Equalf(t, DNSTypeXip, config.DNS.Type, "Expected DNS type did not match")
	assert.Equalf(t, IngressTypeLoadBalancer, config.Ingress.Type, "Expected Ingress type did not match")
	assert.Equalf(t, CertIssuerTypeCA, config.Certificates.IssuerType, "Expected certification issuer type did not match")
	assert.Equalf(t, "cattle-system", config.Certificates.CA.ClusterResourceNamespace, "Expected namespace did not match")
	assert.Equalf(t, "tls-rancher", config.Certificates.CA.SecretName, "Expected CA secret name did not match")
}

// TestXipIoInstallNonDefaults tests the creation of an xip.io install non-default configuration
// GIVEN a verrazzano.install.verrazzano.io custom resource
//  WHEN I call GetInstallConfig
//  THEN the xip.io install configuration is created and verified
func TestXipIoInstallNonDefaults(t *testing.T) {
	vz := installv1alpha1.Verrazzano{
		Spec: installv1alpha1.VerrazzanoSpec{
			Profile:         "dev",
			EnvironmentName: "testEnv",
			DNS: installv1alpha1.DNS{
				XIPIO: installv1alpha1.XIPIO{},
			},
			Ingress: installv1alpha1.Ingress{
				Type: installv1alpha1.LoadBalancer,
				Verrazzano: installv1alpha1.VerrazzanoInstall{
					NGINXInstallArgs: []installv1alpha1.InstallArgs{
						{
							Name:  "name1",
							Value: "value1",
						},
					},
					Ports: []corev1.ServicePort{
						{
							Name:       "port1",
							Protocol:   corev1.ProtocolTCP,
							Port:       8000,
							TargetPort: intstr.FromInt(8000),
							NodePort:   30500,
						},
					},
				},
				Application: installv1alpha1.ApplicationInstall{
					IstioInstallArgs: []installv1alpha1.InstallArgs{
						{
							Name:  "name2",
							Value: "value2",
						},
					},
				},
			},
		},
	}

	config, _ := GetInstallConfig(&vz, nil)
	assert.Equalf(t, "testEnv", config.EnvironmentName, "Expected environment name did not match")
	assert.Equalf(t, InstallProfileDev, config.Profile, "Expected profile did not match")
	assert.Equalf(t, DNSTypeXip, config.DNS.Type, "Expected DNS type did not match")

	assert.Equalf(t, IngressTypeLoadBalancer, config.Ingress.Type, "Expected Ingress type did not match")
	assert.Equalf(t, 1, len(config.Ingress.Verrazzano.NginxInstallArgs), "Expected nginxInstallArgs length did not match")
	assert.Equalf(t, "name1", config.Ingress.Verrazzano.NginxInstallArgs[0].Name, "Expected nginxInstallArg name did not match")
	assert.Equalf(t, "value1", config.Ingress.Verrazzano.NginxInstallArgs[0].Value, "Expected nginxInstallArg value did not match")
	assert.Equalf(t, 1, len(config.Ingress.Verrazzano.Ports), "Expected ports length did not match")
	assert.Equalf(t, "port1", config.Ingress.Verrazzano.Ports[0].Name, "Expected port name did not match")
	assert.Equalf(t, "TCP", config.Ingress.Verrazzano.Ports[0].Protocol, "Expected port protocol did not match")
	assert.Equalf(t, int32(8000), config.Ingress.Verrazzano.Ports[0].Port, "Expected port did not match")
	assert.Equalf(t, int32(8000), config.Ingress.Verrazzano.Ports[0].TargetPort, "Expected target port did not match")
	assert.Equalf(t, int32(30500), config.Ingress.Verrazzano.Ports[0].NodePort, "Expected node port did not match")
	assert.Equalf(t, 1, len(config.Ingress.Application.IstioInstallArgs), "Expected istioInstallArgs length did not match")
	assert.Equalf(t, "name2", config.Ingress.Application.IstioInstallArgs[0].Name, "Expected istioInstallArg name did not match")
	assert.Equalf(t, "value2", config.Ingress.Application.IstioInstallArgs[0].Value, "Expected istioInstallArg name did not match")

	assert.Equalf(t, CertIssuerTypeCA, config.Certificates.IssuerType, "Expected certification issuer type did not match")
	assert.Equalf(t, "cattle-system", config.Certificates.CA.ClusterResourceNamespace, "Expected namespace did not match")
	assert.Equalf(t, "tls-rancher", config.Certificates.CA.SecretName, "Expected CA secret name did not match")
}

// TestExternalInstall tests the creation of an external install configuration
// GIVEN a verrazzano.install.verrazzano.io custom resource
//  WHEN I call GetInstallConfig
//  THEN the external install configuration is created and verified
func TestExternalInstall(t *testing.T) {
	vz := installv1alpha1.Verrazzano{
		Spec: installv1alpha1.VerrazzanoSpec{
			Profile:         "prod",
			EnvironmentName: "external",
			DNS: installv1alpha1.DNS{
				External: installv1alpha1.External{
					Suffix: "abc.def.com",
				},
			},
			Ingress: installv1alpha1.Ingress{
				Type: installv1alpha1.LoadBalancer,
				Verrazzano: installv1alpha1.VerrazzanoInstall{
					NGINXInstallArgs: []installv1alpha1.InstallArgs{
						{
							Name:  "name1",
							Value: "value1",
						},
						{
							Name:  "name2",
							Value: "value2",
						},
						{
							Name: "name3",
							ValueList: []string{
								"valueList3-1",
								"valueList3-2",
							},
						},
					},
					Ports: []corev1.ServicePort{
						{
							Name:       "port1",
							Protocol:   corev1.ProtocolTCP,
							Port:       8000,
							TargetPort: intstr.FromInt(8000),
							NodePort:   30500,
						},
						{
							Name:       "port2",
							Protocol:   corev1.ProtocolUDP,
							Port:       8010,
							TargetPort: intstr.FromString("8011"),
						},
						{
							Name:     "port3",
							Protocol: corev1.ProtocolSCTP,
							Port:     8020,
							NodePort: 30600,
						},
					},
				},
				Application: installv1alpha1.ApplicationInstall{
					IstioInstallArgs: []installv1alpha1.InstallArgs{
						{
							Name:  "name4",
							Value: "value4",
						},
						{
							Name: "name5",
							ValueList: []string{
								"valueList5-1",
							},
						},
					},
				},
			},
		},
	}

	config, _ := GetInstallConfig(&vz, nil)
	assert.Equalf(t, "external", config.EnvironmentName, "Expected environment name did not match")
	assert.Equalf(t, InstallProfileProd, config.Profile, "Expected profile did not match")

	assert.Equalf(t, DNSTypeExternal, config.DNS.Type, "Expected DNS type did not match")
	assert.Equalf(t, "abc.def.com", config.DNS.External.Suffix, "Expected DNS external suffix did not match")

	assert.Equalf(t, IngressTypeLoadBalancer, config.Ingress.Type, "Expected Ingress type did not match")
	assert.Equalf(t, 4, len(config.Ingress.Verrazzano.NginxInstallArgs), "Expected nginxInstallArgs length did not match")
	assert.Equalf(t, "name1", config.Ingress.Verrazzano.NginxInstallArgs[0].Name, "Expected nginxInstallArg name did not match")
	assert.Equalf(t, "value1", config.Ingress.Verrazzano.NginxInstallArgs[0].Value, "Expected nginxInstallArg value did not match")
	assert.Equalf(t, "name2", config.Ingress.Verrazzano.NginxInstallArgs[1].Name, "Expected nginxInstallArg name did not match")
	assert.Equalf(t, "value2", config.Ingress.Verrazzano.NginxInstallArgs[1].Value, "Expected nginxInstallArg value did not match")
	assert.Equalf(t, fmt.Sprintf("%s[0]", "name3"), config.Ingress.Verrazzano.NginxInstallArgs[2].Name, "Expected nginxInstallArg name did not match")
	assert.Equalf(t, "valueList3-1", config.Ingress.Verrazzano.NginxInstallArgs[2].Value, "Expected nginxInstallArg value did not match")
	assert.Equalf(t, fmt.Sprintf("%s[1]", "name3"), config.Ingress.Verrazzano.NginxInstallArgs[3].Name, "Expected nginxInstallArg name did not match")
	assert.Equalf(t, "valueList3-2", config.Ingress.Verrazzano.NginxInstallArgs[3].Value, "Expected nginxInstallArg value did not match")
	assert.Equalf(t, 3, len(config.Ingress.Verrazzano.Ports), "Expected ports length did not match")
	assert.Equalf(t, "port1", config.Ingress.Verrazzano.Ports[0].Name, "Expected port name did not match")
	assert.Equalf(t, "TCP", config.Ingress.Verrazzano.Ports[0].Protocol, "Expected port protocol did not match")
	assert.Equalf(t, int32(8000), config.Ingress.Verrazzano.Ports[0].Port, "Expected port did not match")
	assert.Equalf(t, int32(8000), config.Ingress.Verrazzano.Ports[0].TargetPort, "Expected target port did not match")
	assert.Equalf(t, int32(30500), config.Ingress.Verrazzano.Ports[0].NodePort, "Expected node port did not match")
	assert.Equalf(t, "port2", config.Ingress.Verrazzano.Ports[1].Name, "Expected port name did not match")
	assert.Equalf(t, "UDP", config.Ingress.Verrazzano.Ports[1].Protocol, "Expected port protocol did not match")
	assert.Equalf(t, int32(8010), config.Ingress.Verrazzano.Ports[1].Port, "Expected port did not match")
	assert.Equalf(t, int32(8011), config.Ingress.Verrazzano.Ports[1].TargetPort, "Expected target port did not match")
	assert.Equalf(t, "port3", config.Ingress.Verrazzano.Ports[2].Name, "Expected port name did not match")
	assert.Equalf(t, "SCTP", config.Ingress.Verrazzano.Ports[2].Protocol, "Expected port protocol did not match")
	assert.Equalf(t, int32(8020), config.Ingress.Verrazzano.Ports[2].Port, "Expected port did not match")
	assert.Equalf(t, int32(30600), config.Ingress.Verrazzano.Ports[2].NodePort, "Expected node port did not match")
	assert.Equalf(t, 2, len(config.Ingress.Application.IstioInstallArgs), "Expected istioInstallArgs length did not match")
	assert.Equalf(t, "name4", config.Ingress.Application.IstioInstallArgs[0].Name, "Expected istioInstallArg name did not match")
	assert.Equalf(t, "value4", config.Ingress.Application.IstioInstallArgs[0].Value, "Expected istioInstallArg name did not match")
	assert.Equalf(t, fmt.Sprintf("%s[0]", "name5"), config.Ingress.Application.IstioInstallArgs[1].Name, "Expected istioInstallArg name did not match")
	assert.Equalf(t, "valueList5-1", config.Ingress.Application.IstioInstallArgs[1].Value, "Expected istioInstallArg name did not match")

	assert.Equalf(t, CertIssuerTypeCA, config.Certificates.IssuerType, "Expected certification issuer type did not match")
	assert.Equalf(t, "cattle-system", config.Certificates.CA.ClusterResourceNamespace, "Expected namespace did not match")
	assert.Equalf(t, "tls-rancher", config.Certificates.CA.SecretName, "Expected CA secret name did not match")
}

// TestOCIDNSInstall tests the creation of an OCI DNS install configuration
// GIVEN a verrazzano.install.verrazzano.io custom resource
//  WHEN I call GetInstallConfig
//  THEN the OCI DNS install configuration is created and verified
func TestOCIDNSInstall(t *testing.T) {
	vz := installv1alpha1.Verrazzano{
		Spec: installv1alpha1.VerrazzanoSpec{
			Profile:         "prod",
			EnvironmentName: "oci",
			DNS: installv1alpha1.DNS{
				OCI: installv1alpha1.OCI{
					OCIConfigSecret:        "oci-config-secret",
					DNSZoneCompartmentOCID: "test-dns-zone-compartment-ocid",
					DNSZoneOCID:            "test-dns-zone-ocid",
					DNSZoneName:            "test-dns-zone-name",
				},
			},
			Ingress: installv1alpha1.Ingress{
				Type: installv1alpha1.NodePort,
				Verrazzano: installv1alpha1.VerrazzanoInstall{
					NGINXInstallArgs: []installv1alpha1.InstallArgs{
						{
							Name:  "name1",
							Value: "value1",
						},
					},
					Ports: []corev1.ServicePort{
						{
							Name:       "port1",
							Protocol:   corev1.ProtocolTCP,
							Port:       8000,
							TargetPort: intstr.FromInt(8000),
							NodePort:   30500,
						},
					},
				},
				Application: installv1alpha1.ApplicationInstall{
					IstioInstallArgs: []installv1alpha1.InstallArgs{
						{
							Name:  "name2",
							Value: "value2",
						},
					},
				},
			},
			Certificate: installv1alpha1.Certificate{
				Acme: installv1alpha1.Acme{
					Provider:     installv1alpha1.LetsEncrypt,
					EmailAddress: "someguy@foo.com",
				},
			},
		},
	}

	dnsAuth := DNSAuth{
		PrivateKeyAuth: OCIConfigAuth{
			Region:      "test-region",
			Tenancy:     "test-tenancy-ocid",
			User:        "test-user-ocid",
			Key:         "privateKeyData",
			Fingerprint: "test-fingerprint",
			Passphrase:  "passphraseValue",
		},
	}

	config, _ := GetInstallConfig(&vz, &dnsAuth)
	assert.Equalf(t, "oci", config.EnvironmentName, "Expected environment name did not match")
	assert.Equalf(t, InstallProfileProd, config.Profile, "Expected profile did not match")

	assert.Equalf(t, DNSTypeOci, config.DNS.Type, "Expected DNS type did not match")
	assert.Equalf(t, "test-region", config.DNS.Oci.Region, "Expected region did not match")
	assert.Equalf(t, "test-tenancy-ocid", config.DNS.Oci.TenancyOcid, "Expected tenancy ocid did not match")
	assert.Equalf(t, "test-user-ocid", config.DNS.Oci.UserOcid, "Expected user ocid did not match")
	assert.Equalf(t, "test-dns-zone-compartment-ocid", config.DNS.Oci.DNSZoneCompartmentOcid, "Expected dns zone compartment ocid did not match")
	assert.Equalf(t, "test-fingerprint", config.DNS.Oci.Fingerprint, "Expected fingerprint did not match")
	assert.Equalf(t, "test-dns-zone-ocid", config.DNS.Oci.DNSZoneOcid, "Expected dns zone ocid did not match")
	assert.Equalf(t, "test-dns-zone-name", config.DNS.Oci.DNSZoneName, "Expected dns zone name did not match")
	assert.Equalf(t, installv1alpha1.OciPrivateKeyFilePath, config.DNS.Oci.PrivateKeyFile, "Expected private key file name did not match")
	assert.Equalf(t, "passphraseValue", config.DNS.Oci.PrivateKeyPassphrase, "Expected passphrase did not match")

	assert.Equalf(t, IngressTypeNodePort, config.Ingress.Type, "Expected Ingress type did not match")
	assert.Equalf(t, 1, len(config.Ingress.Verrazzano.NginxInstallArgs), "Expected nginxInstallArgs length did not match")
	assert.Equalf(t, "name1", config.Ingress.Verrazzano.NginxInstallArgs[0].Name, "Expected nginxInstallArg name did not match")
	assert.Equalf(t, "value1", config.Ingress.Verrazzano.NginxInstallArgs[0].Value, "Expected nginxInstallArg value did not match")
	assert.Equalf(t, 1, len(config.Ingress.Verrazzano.Ports), "Expected ports length did not match")
	assert.Equalf(t, "port1", config.Ingress.Verrazzano.Ports[0].Name, "Expected port name did not match")
	assert.Equalf(t, "TCP", config.Ingress.Verrazzano.Ports[0].Protocol, "Expected port protocol did not match")
	assert.Equalf(t, int32(8000), config.Ingress.Verrazzano.Ports[0].Port, "Expected port did not match")
	assert.Equalf(t, int32(8000), config.Ingress.Verrazzano.Ports[0].TargetPort, "Expected target port did not match")
	assert.Equalf(t, int32(30500), config.Ingress.Verrazzano.Ports[0].NodePort, "Expected node port did not match")
	assert.Equalf(t, 1, len(config.Ingress.Application.IstioInstallArgs), "Expected istioInstallArgs length did not match")
	assert.Equalf(t, "name2", config.Ingress.Application.IstioInstallArgs[0].Name, "Expected istioInstallArg name did not match")
	assert.Equalf(t, "value2", config.Ingress.Application.IstioInstallArgs[0].Value, "Expected istioInstallArg name did not match")

	assert.Equalf(t, CertIssuerTypeAcme, config.Certificates.IssuerType, "Expected certification issuer type did not match")
	assert.Equalf(t, "LetsEncrypt", config.Certificates.ACME.Provider, "Expected cert provider did not match")
	assert.Equalf(t, "someguy@foo.com", config.Certificates.ACME.EmailAddress, "Expected email address did not match")
}
