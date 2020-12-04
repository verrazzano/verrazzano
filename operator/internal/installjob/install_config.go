// Copyright (c) 2020, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

// NOTE: the code in this source file is specifically for transforming data from
// the verrazzano custom resource to the json config format needed by the bash installer scripts.

package installjob

import (
	"fmt"
	"strconv"

	installv1alpha1 "github.com/verrazzano/verrazzano/operator/api/v1alpha1"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

// DNSType identifies the DNS type
type DNSType string

const (
	// DNSTypeXip is for the dns type xip (magic dns)
	DNSTypeXip DNSType = "xip.io"
	// DNSTypeOci is for the dns type OCI
	DNSTypeOci DNSType = "oci"
	// DNSTypeExternal is for dns type external (e.g. olcne)
	DNSTypeExternal DNSType = "external"
)

// IngressType identifies the ingress type
type IngressType string

const (
	// IngressTypeLoadBalancer is for ingress type load balancer
	IngressTypeLoadBalancer IngressType = "LoadBalancer"
	// IngressTypeNodePort is for ingress type node port
	IngressTypeNodePort IngressType = "NodePort"
)

// CertIssuerType identifies the certificate issuer type
type CertIssuerType string

const (
	// CertIssuerTypeCA is for certificate issuer ca
	CertIssuerTypeCA CertIssuerType = "ca"

	// CertIssuerTypeAcme is for certificate issuer acme
	CertIssuerTypeAcme CertIssuerType = "acme"
)

// CertificateACME configuration
type CertificateACME struct {
	Provider     string `json:"provider"`
	EmailAddress string `json:"emailAddress,omitempty"`
	Environment  string `json:"environment,omitempty"`
}

// CertificateCA configuration
type CertificateCA struct {
	ClusterResourceNamespace string `json:"clusterResourceNamespace"`
	SecretName               string `json:"secretName"`
}

// Certificate configuration
type Certificate struct {
	IssuerType CertIssuerType   `json:"issuerType"`
	CA         *CertificateCA   `json:"ca,omitempty"`
	ACME       *CertificateACME `json:"acme,omitempty"`
}

// IngressPort configuration
type IngressPort struct {
	Name       string `json:"name,omitempty"`
	Port       int32  `json:"port,omitempty"`
	NodePort   int32  `json:"nodePort,omitempty"`
	Protocol   string `json:"protocol,omitempty"`
	TargetPort int32  `json:"targetPort,omitempty"`
}

// IngressArg configuration
type IngressArg struct {
	Name      string `json:"name"`
	Value     string `json:"value"`
	SetString string `json:"setString,omitempty"`
}

// Application configuration
type Application struct {
	IstioInstallArgs []IngressArg `json:"istioInstallArgs,omitempty"`
}

// Verrazzano configuration
type Verrazzano struct {
	NginxInstallArgs []IngressArg  `json:"nginxInstallArgs,omitempty"`
	Ports            []IngressPort `json:"ports,omitempty"`
}

// Ingress configuration for a Verrazzano installation
type Ingress struct {
	Type        IngressType `json:"type"`
	Verrazzano  Verrazzano  `json:"verrazzano,omitempty"`
	Application Application `json:"application,omitempty"`
}

// ExternalDNS configuration
type ExternalDNS struct {
	Suffix string `json:"suffix"`
}

// OCIConfigAuth defines the OCI authentication fields
type OCIConfigAuth struct {
	Region      string `yaml:"region"`
	Tenancy     string `yaml:"tenancy"`
	User        string `yaml:"user"`
	Key         string `yaml:"key"`
	Fingerprint string `yaml:"fingerprint"`
	Passphrase  string `yaml:"passphrase"`
}

// DNSAuth provides the authentication structures for external DNS
type DNSAuth struct {
	PrivateKeyAuth OCIConfigAuth `yaml:"privateKeyAuth"`
}

// DNSOCI configuration
type DNSOCI struct {
	Region                 string `json:"region"`
	TenancyOcid            string `json:"tenancyOcid"`
	UserOcid               string `json:"userOcid"`
	DNSZoneCompartmentOcid string `json:"dnsZoneCompartmentOcid"`
	Fingerprint            string `json:"fingerprint"`
	PrivateKeyFile         string `json:"privateKeyFile"`
	PrivateKeyPassphrase   string `json:"privateKeyPassphrase,omitempty"`
	DNSZoneOcid            string `json:"dnsZoneOcid"`
	DNSZoneName            string `json:"dnsZoneName"`
}

// DNS configuration for a Verrazzano installation
type DNS struct {
	Type     DNSType      `json:"type"`
	External *ExternalDNS `json:"external,omitempty"`
	Oci      *DNSOCI      `json:"oci,omitempty"`
}

// InstallProfile type
type InstallProfile string

const (
	// InstallProfileProd - production profile
	InstallProfileProd InstallProfile = "prod"

	// InstallProfileDev - development profile
	InstallProfileDev InstallProfile = "dev"
)

// InstallConfiguration - Verrazzano installation configuration options
type InstallConfiguration struct {
	EnvironmentName string         `json:"environmentName"`
	Profile         InstallProfile `json:"profile"`
	DNS             DNS            `json:"dns"`
	Ingress         Ingress        `json:"ingress"`
	Certificates    Certificate    `json:"certificates"`
}

// GetInstallConfig returns an install configuration in the json format required by the
// bash installer scripts.
func GetInstallConfig(vz *installv1alpha1.Verrazzano, auth *DNSAuth) (*InstallConfiguration, error) {
	if vz.Spec.DNS.External != (installv1alpha1.External{}) {
		return newExternalDNSInstallConfig(vz), nil
	}

	if vz.Spec.DNS.OCI != (installv1alpha1.OCI{}) {
		return newOCIDNSInstallConfig(vz, auth)
	}

	return newXipIoInstallConfig(vz), nil
}

func newOCIDNSInstallConfig(vz *installv1alpha1.Verrazzano, auth *DNSAuth) (*InstallConfiguration, error) {
	return &InstallConfiguration{
		EnvironmentName: getEnvironmentName(vz.Spec.EnvironmentName),
		Profile:         getProfile(vz.Spec.Profile),
		DNS: DNS{
			Type: DNSTypeOci,
			Oci: &DNSOCI{
				Region:                 auth.PrivateKeyAuth.Region,
				TenancyOcid:            auth.PrivateKeyAuth.Tenancy,
				UserOcid:               auth.PrivateKeyAuth.User,
				DNSZoneCompartmentOcid: vz.Spec.DNS.OCI.DNSZoneCompartmentOCID,
				Fingerprint:            auth.PrivateKeyAuth.Fingerprint,
				PrivateKeyFile:         installv1alpha1.OciPrivateKeyFilePath,
				PrivateKeyPassphrase:   auth.PrivateKeyAuth.Passphrase,
				DNSZoneOcid:            vz.Spec.DNS.OCI.DNSZoneOCID,
				DNSZoneName:            vz.Spec.DNS.OCI.DNSZoneName,
			},
		},
		Ingress: getIngress(vz.Spec.Ingress),
		Certificates: Certificate{
			IssuerType: CertIssuerTypeAcme,
			ACME: &CertificateACME{
				Provider:     string(vz.Spec.Certificate.Acme.Provider),
				EmailAddress: vz.Spec.Certificate.Acme.EmailAddress,
				Environment:  vz.Spec.Certificate.Acme.Environment,
			},
		},
	}, nil
}

// newXipIoInstallConfig returns an install configuration for a xip.io install in the
// json format required by the bash installer scripts.
func newXipIoInstallConfig(vz *installv1alpha1.Verrazzano) *InstallConfiguration {
	return &InstallConfiguration{
		EnvironmentName: getEnvironmentName(vz.Spec.EnvironmentName),
		Profile:         getProfile(vz.Spec.Profile),
		DNS: DNS{
			Type: DNSTypeXip,
		},
		Ingress: getIngress(vz.Spec.Ingress),
		Certificates: Certificate{
			IssuerType: CertIssuerTypeCA,
			CA: &CertificateCA{
				ClusterResourceNamespace: "cattle-system",
				SecretName:               "tls-rancher",
			},
		},
	}
}

// newExternalDNSInstallConfig returns an install configuration for an external DNS install
// in the json format required by the bash installer scripts.
// This type of install configuration would be used for an OLCNE install.
func newExternalDNSInstallConfig(vz *installv1alpha1.Verrazzano) *InstallConfiguration {
	return &InstallConfiguration{
		EnvironmentName: getEnvironmentName(vz.Spec.EnvironmentName),
		Profile:         getProfile(vz.Spec.Profile),
		DNS: DNS{
			Type: DNSTypeExternal,
			External: &ExternalDNS{
				Suffix: vz.Spec.DNS.External.Suffix,
			},
		},
		Ingress: getIngress(vz.Spec.Ingress),
		Certificates: Certificate{
			IssuerType: CertIssuerTypeCA,
			CA: &CertificateCA{
				ClusterResourceNamespace: "cattle-system",
				SecretName:               "tls-rancher",
			},
		},
	}
}

// getIngressPorts returns the list of ingress ports in the json format required by the bash installer scripts
func getIngressPorts(ports []corev1.ServicePort) []IngressPort {
	ingressPorts := []IngressPort{}

	for _, port := range ports {
		ingressPort := IngressPort{}
		ingressPort.Port = port.Port
		if port.Name != "" {
			ingressPort.Name = port.Name
		}
		if port.Protocol == corev1.ProtocolTCP {
			ingressPort.Protocol = "TCP"
		} else if port.Protocol == corev1.ProtocolUDP {
			ingressPort.Protocol = "UDP"
		} else if port.Protocol == corev1.ProtocolSCTP {
			ingressPort.Protocol = "SCTP"
		}
		if port.TargetPort.Type == intstr.String {
			intVal, _ := strconv.ParseInt(port.TargetPort.StrVal, 10, 32)
			ingressPort.TargetPort = int32(intVal)
		} else if port.TargetPort.Type == intstr.Int {
			ingressPort.TargetPort = port.TargetPort.IntVal
		}
		if port.NodePort != 0 {
			ingressPort.NodePort = port.NodePort
		}
		ingressPorts = append(ingressPorts, ingressPort)

	}
	return ingressPorts
}

// getIngressArgs returns the list of ingress args in the json format required by the bash installer scripts
func getIngressArgs(args []installv1alpha1.InstallArgs) []IngressArg {
	ingressArgs := []IngressArg{}

	for _, arg := range args {
		ingressArg := IngressArg{}
		if arg.Value != "" {
			ingressArg.Name = arg.Name
			ingressArg.Value = arg.Value
			if arg.SetString == "true" {
				ingressArg.SetString = arg.SetString
			}
			ingressArgs = append(ingressArgs, ingressArg)
			continue
		}

		for i, value := range arg.ValueList {
			ingressArg.Name = fmt.Sprintf("%s[%d]", arg.Name, i)
			ingressArg.Value = value
			ingressArgs = append(ingressArgs, ingressArg)
		}
	}

	return ingressArgs
}

// getIngressType returns the install ingress type
func getIngressType(ingressType installv1alpha1.IngressType) IngressType {
	// Use ingress type of LoadBalancer if not specified
	if ingressType == "" || ingressType == installv1alpha1.LoadBalancer {
		return IngressTypeLoadBalancer
	}

	return IngressTypeNodePort
}

func getIngress(ingress installv1alpha1.Ingress) Ingress {
	return Ingress{
		Type: getIngressType(ingress.Type),
		Verrazzano: Verrazzano{
			NginxInstallArgs: getIngressArgs(ingress.Verrazzano.NGINXInstallArgs),
			Ports:            getIngressPorts(ingress.Verrazzano.Ports),
		},
		Application: Application{
			IstioInstallArgs: getIngressArgs(ingress.Application.IstioInstallArgs),
		},
	}
}

// getEnvironmentName returns the install environment name
func getEnvironmentName(envName string) string {
	// Use env name of default if not specified
	if envName == "" {
		return "default"
	}

	return envName
}

// getProfile returns the install profile name
func getProfile(profileType installv1alpha1.ProfileType) InstallProfile {
	// Use the prod install profile if not specified
	if profileType == "" || profileType == installv1alpha1.Prod {
		return InstallProfileProd
	}

	return InstallProfileDev
}
