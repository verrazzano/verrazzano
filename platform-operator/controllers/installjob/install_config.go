// Copyright (c) 2020, 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

// NOTE: the code in this source file is specifically for transforming data from
// the verrazzano custom resource to the json config format needed by the bash installer scripts.

package installjob

import (
	"fmt"
	"github.com/pkg/errors"
	"go.uber.org/zap"
	"strconv"

	installv1alpha1 "github.com/verrazzano/verrazzano/platform-operator/api/verrazzano/v1alpha1"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

const (
	defaultCAClusterResourceName string = "cattle-system"
	defaultCASecretName          string = "tls-rancher"
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

// InstallArg configuration
type InstallArg struct {
	Name      string `json:"name"`
	Value     string `json:"value"`
	SetString bool   `json:"setString,omitempty"`
}

// Application configuration
type Application struct {
	IstioInstallArgs []InstallArg `json:"istioInstallArgs,omitempty"`
}

// Verrazzano configuration
type Verrazzano struct {
	NginxInstallArgs []InstallArg  `json:"nginxInstallArgs,omitempty"`
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
	PrivateKeyAuth OCIConfigAuth `yaml:"auth"`
}

// DNSOCI configuration
type DNSOCI struct {
	OCIConfigSecret        string `json:"ociConfigSecret"`
	DNSZoneCompartmentOcid string `json:"dnsZoneCompartmentOcid"`
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

// Keycloak configuration
type Keycloak struct {
	KeycloakInstallArgs []InstallArg `json:"keycloakInstallArgs,omitempty"`
	MySQL               MySQL        `json:"mysql,omitempty"`
}

// MySQL configuration
type MySQL struct {
	MySQLInstallArgs []InstallArg `json:"mySqlInstallArgs,omitempty"`
}

// OAM configuration for a Verrazzano installation
type OAM struct {
	Enabled bool `json:"enabled"`
}

// InstallConfiguration - Verrazzano installation configuration options
type InstallConfiguration struct {
	EnvironmentName string         `json:"environmentName"`
	Profile         InstallProfile `json:"profile"`
	DNS             DNS            `json:"dns"`
	Ingress         Ingress        `json:"ingress"`
	Certificates    Certificate    `json:"certificates"`
	Keycloak        Keycloak       `json:"keycloak"`
	OAM             OAM            `json:"oam"`
	VzInstallArgs   []InstallArg   `json:"verrazzanoInstallArgs,omitempty"`
}

// GetInstallConfig returns an install configuration in the json format required by the
// bash installer scripts.
func GetInstallConfig(vz *installv1alpha1.Verrazzano, log *zap.SugaredLogger) (*InstallConfiguration, error) {
	if vz.Spec.Components.DNS.External != (installv1alpha1.External{}) {
		return newExternalDNSInstallConfig(vz, log)
	}

	if vz.Spec.Components.DNS.OCI != (installv1alpha1.OCI{}) {
		return newOCIDNSInstallConfig(vz, log)
	}

	return newXipIoInstallConfig(vz, log)
}

func newOCIDNSInstallConfig(vz *installv1alpha1.Verrazzano, log *zap.SugaredLogger) (*InstallConfiguration, error) {
	vzArgs, err := getVerrazzanoInstallArgs(&vz.Spec, log)
	if err != nil {
		return nil, err
	}
	keycloak, err := getKeycloak(vz.Spec.Components.Keycloak, vz.Spec.VolumeClaimSpecTemplates, vz.Spec.DefaultVolumeSource, log)
	if err != nil {
		return nil, err
	}
	return &InstallConfiguration{
		EnvironmentName: getEnvironmentName(vz.Spec.EnvironmentName),
		Profile:         getProfile(vz.Spec.Profile),
		VzInstallArgs:   vzArgs,
		DNS: DNS{
			Type: DNSTypeOci,
			Oci: &DNSOCI{
				OCIConfigSecret:        vz.Spec.Components.DNS.OCI.OCIConfigSecret,
				DNSZoneCompartmentOcid: vz.Spec.Components.DNS.OCI.DNSZoneCompartmentOCID,
				DNSZoneOcid:            vz.Spec.Components.DNS.OCI.DNSZoneOCID,
				DNSZoneName:            vz.Spec.Components.DNS.OCI.DNSZoneName,
			},
		},
		Ingress: getIngress(vz.Spec.Components.Ingress, vz.Spec.Components.Istio),
		Certificates: Certificate{
			IssuerType: CertIssuerTypeAcme,
			ACME: &CertificateACME{
				Provider:     string(vz.Spec.Components.CertManager.Certificate.Acme.Provider),
				EmailAddress: vz.Spec.Components.CertManager.Certificate.Acme.EmailAddress,
				Environment:  vz.Spec.Components.CertManager.Certificate.Acme.Environment,
			},
		},
		Keycloak: keycloak,
		OAM:      getOAM(vz.Spec.Components.OAM),
	}, nil
}

// newXipIoInstallConfig returns an install configuration for a xip.io install in the
// json format required by the bash installer scripts.
func newXipIoInstallConfig(vz *installv1alpha1.Verrazzano, log *zap.SugaredLogger) (*InstallConfiguration, error) {
	vzArgs, err := getVerrazzanoInstallArgs(&vz.Spec, log)
	if err != nil {
		return nil, err
	}
	keycloak, err := getKeycloak(vz.Spec.Components.Keycloak, vz.Spec.VolumeClaimSpecTemplates, vz.Spec.DefaultVolumeSource, log)
	if err != nil {
		return nil, err
	}
	return &InstallConfiguration{
		EnvironmentName: getEnvironmentName(vz.Spec.EnvironmentName),
		Profile:         getProfile(vz.Spec.Profile),
		VzInstallArgs:   vzArgs,
		DNS: DNS{
			Type: DNSTypeXip,
		},
		Ingress: getIngress(vz.Spec.Components.Ingress, vz.Spec.Components.Istio),
		Certificates: Certificate{
			IssuerType: CertIssuerTypeCA,
			CA: &CertificateCA{
				ClusterResourceNamespace: getCAClusterResourceNamespace(vz.Spec.Components.CertManager.Certificate.CA),
				SecretName:               getCASecretName(vz.Spec.Components.CertManager.Certificate.CA),
			},
		},
		Keycloak: keycloak,
		OAM:      getOAM(vz.Spec.Components.OAM),
	}, nil
}

// newExternalDNSInstallConfig returns an install configuration for an external DNS install
// in the json format required by the bash installer scripts.
// This type of install configuration would be used for an OLCNE install.
func newExternalDNSInstallConfig(vz *installv1alpha1.Verrazzano, log *zap.SugaredLogger) (*InstallConfiguration, error) {
	vzArgs, err := getVerrazzanoInstallArgs(&vz.Spec, log)
	if err != nil {
		return nil, err
	}
	keycloak, err := getKeycloak(vz.Spec.Components.Keycloak, vz.Spec.VolumeClaimSpecTemplates, vz.Spec.DefaultVolumeSource, log)
	if err != nil {
		return nil, err
	}
	return &InstallConfiguration{
		EnvironmentName: getEnvironmentName(vz.Spec.EnvironmentName),
		Profile:         getProfile(vz.Spec.Profile),
		VzInstallArgs:   vzArgs,
		DNS: DNS{
			Type: DNSTypeExternal,
			External: &ExternalDNS{
				Suffix: vz.Spec.Components.DNS.External.Suffix,
			},
		},
		Ingress: getIngress(vz.Spec.Components.Ingress, vz.Spec.Components.Istio),
		Certificates: Certificate{
			IssuerType: CertIssuerTypeCA,
			CA: &CertificateCA{
				ClusterResourceNamespace: getCAClusterResourceNamespace(vz.Spec.Components.CertManager.Certificate.CA),
				SecretName:               getCASecretName(vz.Spec.Components.CertManager.Certificate.CA),
			},
		},
		Keycloak: keycloak,
		OAM:      getOAM(vz.Spec.Components.OAM),
	}, nil
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

// getInstallArgs returns the list of install args in the json format required by the bash installer scripts
func getInstallArgs(args []installv1alpha1.InstallArgs) []InstallArg {
	installArgs := []InstallArg{}

	for _, arg := range args {
		installArg := InstallArg{}
		if arg.Value != "" {
			installArg.Name = arg.Name
			installArg.Value = arg.Value
			if arg.SetString {
				installArg.SetString = arg.SetString
			}
			installArgs = append(installArgs, installArg)
			continue
		}

		for i, value := range arg.ValueList {
			installArg.Name = fmt.Sprintf("%s[%d]", arg.Name, i)
			installArg.Value = value
			installArgs = append(installArgs, installArg)
		}
	}

	return installArgs
}

// getKeycloak returns the json representation for the keycloak configuration
func getKeycloak(keycloak installv1alpha1.KeycloakComponent, templates []installv1alpha1.VolumeClaimSpecTemplate, defaultVolumeSpec *corev1.VolumeSource, log *zap.SugaredLogger) (Keycloak, error) {

	// Get the explicit helm args for MySQL
	mySQLArgs := getInstallArgs(keycloak.MySQL.MySQLInstallArgs)

	keycloakConfig := Keycloak{
		KeycloakInstallArgs: getInstallArgs(keycloak.KeycloakInstallArgs),
		MySQL: MySQL{
			MySQLInstallArgs: mySQLArgs,
		},
	}

	// Use a volume source specified in the Keycloak config, otherwise use the default spec
	mysqlVolumeSource := keycloak.MySQL.VolumeSource
	if mysqlVolumeSource == nil {
		mysqlVolumeSource = defaultVolumeSpec
	}

	// No volumes to process, return what we have
	if mysqlVolumeSource == nil {
		return keycloakConfig, nil
	}

	if mysqlVolumeSource.EmptyDir != nil {
		// EmptyDir, disable persistence
		mySQLArgs = append(mySQLArgs, InstallArg{
			Name:      "persistence.enabled",
			Value:     "false",
		})
	} else if mysqlVolumeSource.PersistentVolumeClaim != nil {
		// Configured for persistence, adapt the PVC Spec template to the appropriate Helm args
		pvcs := mysqlVolumeSource.PersistentVolumeClaim
		storageSpec, found := findVolumeTemplate(pvcs.ClaimName, templates)
		if !found {
			err := errors.Errorf("No VolumeClaimTemplate found for %s", pvcs.ClaimName)
			return Keycloak{}, err
		}
		storageClass := storageSpec.StorageClassName
		if storageClass != nil && len(*storageClass) > 0 {
			mySQLArgs = append(mySQLArgs, InstallArg{
				Name:      "persistence.storageClass",
				Value:     *storageClass,
				SetString: true,
			})
		}
		storage := storageSpec.Resources.Requests.Storage()
		if storageSpec.Resources.Requests != nil && !storage.IsZero() {
			mySQLArgs = append(mySQLArgs, InstallArg{
				Name:      "persistence.size",
				Value:     storage.String(),
				SetString: true,
			})
		}
		accessModes := storageSpec.AccessModes
		if len(accessModes) > 0 {
			// MySQL only allows a single AccessMode value, so just choose the first
			mySQLArgs = append(mySQLArgs, InstallArg{
				Name:      "persistence.accessMode",
				Value:     string(accessModes[0]),
				SetString: true,
			})
		}
	}
	// Update the MySQL Install args
	keycloakConfig.MySQL.MySQLInstallArgs = mySQLArgs
	return keycloakConfig, nil
}

// getIngress returns the json representation for the ingress
func getIngress(ingress installv1alpha1.IngressNginxComponent, istio installv1alpha1.IstioComponent) Ingress {
	return Ingress{
		Type: getIngressType(ingress.Type),
		Verrazzano: Verrazzano{
			NginxInstallArgs: getInstallArgs(ingress.NGINXInstallArgs),
			Ports:            getIngressPorts(ingress.Ports),
		},
		Application: Application{
			IstioInstallArgs: getInstallArgs(istio.IstioInstallArgs),
		},
	}
}

// getIngressType returns the install ingress type
func getIngressType(ingressType installv1alpha1.IngressType) IngressType {
	// Use ingress type of LoadBalancer if not specified
	if ingressType == "" || ingressType == installv1alpha1.LoadBalancer {
		return IngressTypeLoadBalancer
	}

	return IngressTypeNodePort
}

// getCAClusterResourceNamespace returns the cluster resource name for a CA certificate
func getCAClusterResourceNamespace(ca installv1alpha1.CA) string {
	// Use default value if not specified
	if ca.ClusterResourceNamespace == "" {
		return defaultCAClusterResourceName
	}

	return ca.ClusterResourceNamespace
}

// getCASecretName returns the secret name for a CA certificate
func getCASecretName(ca installv1alpha1.CA) string {
	// Use default value if not specified
	if ca.SecretName == "" {
		return defaultCASecretName
	}

	return ca.SecretName
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

// getVerrazzanoInstallArgs Set custom helm args for the Verrazzano internal component as needed
func getVerrazzanoInstallArgs(vzSpec *installv1alpha1.VerrazzanoSpec, log *zap.SugaredLogger) ([]InstallArg, error) {
	if vzSpec.DefaultVolumeSource == nil {
		return []InstallArg{}, nil
	}
	if vzSpec.DefaultVolumeSource.EmptyDir != nil {
		return []InstallArg{
			{
				Name:      "verrazzanoOperator.esDataStorageSize",
				Value:     "",
				SetString: true,
			},
			{
				Name:      "verrazzanoOperator.grafanaDataStorageSize",
				Value:     "",
				SetString: true,
			},
			{
				Name:      "verrazzanoOperator.prometheusDataStorageSize",
				Value:     "",
				SetString: true,
			},
		}, nil
	} else if vzSpec.DefaultVolumeSource.PersistentVolumeClaim != nil {
		pvcs := vzSpec.DefaultVolumeSource.PersistentVolumeClaim
		storageSpec, found := findVolumeTemplate(pvcs.ClaimName, vzSpec.VolumeClaimSpecTemplates)
		if !found {
			err := errors.Errorf("No VolumeClaimTemplate found for %s", pvcs.ClaimName)
			return []InstallArg{}, err
		}
		return []InstallArg{
			{
				Name:      "verrazzanoOperator.esDataStorageSize",
				Value:     storageSpec.Resources.Requests.Storage().String(),
				SetString: true,
			},
			{
				Name:      "verrazzanoOperator.grafanaDataStorageSize",
				Value:     storageSpec.Resources.Requests.Storage().String(),
				SetString: true,
			},
			{
				Name:      "verrazzanoOperator.prometheusDataStorageSize",
				Value:     storageSpec.Resources.Requests.Storage().String(),
				SetString: true,
			},
		}, nil
	}
	return []InstallArg{}, nil
}

// findVolumeTemplate Find a named VolumeClaimTemplate in the list
func findVolumeTemplate(templateName string, templates []installv1alpha1.VolumeClaimSpecTemplate) (*corev1.PersistentVolumeClaimSpec, bool) {
	for i, template := range templates {
		if templateName == template.Name {
			return &templates[i].Spec, true
		}
	}
	return nil, false
}

// getOAM returns the install config for OAM
func getOAM(oam installv1alpha1.OAMComponent) OAM {
	config := OAM{Enabled: oam.Enabled}
	return config
}
