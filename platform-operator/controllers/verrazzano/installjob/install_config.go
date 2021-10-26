// Copyright (c) 2020, 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

// NOTE: the code in this source file is specifically for transforming data from
// the verrazzano custom resource to the json config format needed by the bash installer scripts.

package installjob

import (
	"fmt"
	"github.com/verrazzano/verrazzano/platform-operator/constants"
	"strconv"

	installv1alpha1 "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"go.uber.org/zap"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

const (
	defaultCAClusterResourceName string = "cert-manager"
	defaultCASecretName          string = "verrazzano-ca-certificate-secret" //nolint:gosec //#gosec G101

	defaultElasticsearchURL    string = "http://verrazzano-authproxy-elasticsearch:8775"
	defaultElasticsearchSecret string = "verrazzano"

	// Verrazzano Helm chart value names
	esStorageValueName         string = "elasticSearch.nodes.data.requests.storage"
	grafanaStorageValueName    string = "grafana.requests.storage"
	prometheusStorageValueName string = "prometheus.requests.storage"
	grafanaEnabledValueName    string = "grafana.enabled"
	esEnabledValueName         string = "elasticSearch.enabled"
	promEnabledValueName       string = "prometheus.enabled"
	kibanaEnabledValueName     string = "kibana.enabled"
	consoleEnabledValueName    string = "console.enabled"
)

// DNSType identifies the DNS type
type DNSType string

const (
	// DNSTypeWildcard is for the dns type wildcard (magic dns)
	DNSTypeWildcard DNSType = "wildcard"
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

// WildcardDNS configuration
type WildcardDNS struct {
	Domain string `json:"domain"`
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
	Wildcard *WildcardDNS `json:"wildcard,omitempty"`
	External *ExternalDNS `json:"external,omitempty"`
	Oci      *DNSOCI      `json:"oci,omitempty"`
}

// Keycloak configuration
type Keycloak struct {
	KeycloakInstallArgs []InstallArg `json:"keycloakInstallArgs,omitempty"`
	MySQL               MySQL        `json:"mysql,omitempty"`
	Enabled             string       `json:"enabled,omitempty"`
}

// MySQL configuration
type MySQL struct {
	MySQLInstallArgs []InstallArg `json:"mySqlInstallArgs,omitempty"`
}

type Rancher struct {
	Enabled string `json:"enabled,omitempty"`
}

// Fluentd configuration
type Fluentd struct {
	Enabled             string       `json:"enabled,omitempty"`
	FluentdInstallArgs  []InstallArg `json:"fluentdInstallArgs,omitempty"`
	ElasticsearchURL    string       `json:"elasticsearchURL,omitempty"`
	ElasticsearchSecret string       `json:"elasticsearchSecret,omitempty"`
}

// InstallConfiguration - Verrazzano installation configuration options
type InstallConfiguration struct {
	EnvironmentName string                      `json:"environmentName"`
	Profile         installv1alpha1.ProfileType `json:"profile"`
	DNS             DNS                         `json:"dns"`
	Ingress         Ingress                     `json:"ingress"`
	Certificates    Certificate                 `json:"certificates"`
	Keycloak        Keycloak                    `json:"keycloak"`
	Rancher         Rancher                     `json:"rancher"`
	Fluentd         Fluentd                     `json:"fluentd"`
	VzInstallArgs   []InstallArg                `json:"verrazzanoInstallArgs,omitempty"`
}

// GetInstallConfig returns an install configuration in the json format required by the
// bash installer scripts.
func GetInstallConfig(vz *installv1alpha1.Verrazzano) (*InstallConfiguration, error) {
	vzArgs, err := getVerrazzanoInstallArgs(&vz.Spec)
	if err != nil {
		return nil, err
	}
	keycloak, err := getKeycloak(vz.Spec.Components.Keycloak, vz.Spec.VolumeClaimSpecTemplates, vz.Spec.DefaultVolumeSource)
	if err != nil {
		return nil, err
	}
	return &InstallConfiguration{
		EnvironmentName: getEnvironmentName(vz.Spec.EnvironmentName),
		Profile:         getProfile(vz.Spec.Profile),
		VzInstallArgs:   vzArgs,
		DNS:             getDNSConfig(vz),
		Ingress:         getIngress(vz.Spec.Components.Ingress, vz.Spec.Components.Istio),
		Certificates:    getCertificateConfig(vz),
		Keycloak:        keycloak,
		Rancher:         getRancher(vz.Spec.Components.Rancher),
		Fluentd:         getFluentd(vz.Spec.Components.Fluentd),
	}, nil
}

func getDNSConfig(vz *installv1alpha1.Verrazzano) DNS {
	dns := vz.Spec.Components.DNS
	if dns != nil {
		if dns.Wildcard != nil {
			return DNS{
				Type: DNSTypeWildcard,
				Wildcard: &WildcardDNS{
					Domain: vz.Spec.Components.DNS.Wildcard.Domain,
				},
			}
		}
		if dns.External != nil {
			return DNS{
				Type: DNSTypeExternal,
				External: &ExternalDNS{
					Suffix: vz.Spec.Components.DNS.External.Suffix,
				},
			}
		}
		if dns.OCI != nil {
			return DNS{
				Type: DNSTypeOci,
				Oci: &DNSOCI{
					OCIConfigSecret:        vz.Spec.Components.DNS.OCI.OCIConfigSecret,
					DNSZoneCompartmentOcid: vz.Spec.Components.DNS.OCI.DNSZoneCompartmentOCID,
					DNSZoneOcid:            vz.Spec.Components.DNS.OCI.DNSZoneOCID,
					DNSZoneName:            vz.Spec.Components.DNS.OCI.DNSZoneName,
				},
			}
		}
	}

	return DNS{
		Type: DNSTypeWildcard,
		Wildcard: &WildcardDNS{
			Domain: "nip.io",
		},
	}
}

func getCertificateConfig(vz *installv1alpha1.Verrazzano) Certificate {
	if vz.Spec.Components.CertManager != nil && (vz.Spec.Components.CertManager.Certificate.Acme != installv1alpha1.Acme{}) {
		return Certificate{
			IssuerType: CertIssuerTypeAcme,
			ACME: &CertificateACME{
				Provider:     string(vz.Spec.Components.CertManager.Certificate.Acme.Provider),
				EmailAddress: vz.Spec.Components.CertManager.Certificate.Acme.EmailAddress,
				Environment:  vz.Spec.Components.CertManager.Certificate.Acme.Environment,
			},
		}
	}
	return Certificate{
		IssuerType: CertIssuerTypeCA,
		CA: &CertificateCA{
			ClusterResourceNamespace: getCAClusterResourceNamespace(vz.Spec.Components.CertManager),
			SecretName:               getCASecretName(vz.Spec.Components.CertManager),
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

func getRancher(rancher *installv1alpha1.RancherComponent) Rancher {
	if rancher == nil {
		return Rancher{Enabled: "true"}
	}

	var enabled = "true"
	if rancher.Enabled != nil {
		enabled = strconv.FormatBool(*rancher.Enabled)
	}
	return Rancher{Enabled: enabled}
}

// getKeycloak returns the json representation for the keycloak configuration
func getKeycloak(keycloak *installv1alpha1.KeycloakComponent, templates []installv1alpha1.VolumeClaimSpecTemplate, defaultVolumeSpec *corev1.VolumeSource) (Keycloak, error) {
	// keycloak was not specified in CR so return defaults
	if keycloak == nil {
		keycloakConfig := Keycloak{Enabled: "true"}
		if defaultVolumeSpec != nil && defaultVolumeSpec.EmptyDir != nil {
			var mySQLArgs []InstallArg
			mySQLArgs = append(mySQLArgs, InstallArg{
				Name:  "persistence.enabled",
				Value: "false",
			})
			keycloakConfig.MySQL.MySQLInstallArgs = mySQLArgs
		}
		return keycloakConfig, nil
	}

	// Get the explicit helm args for MySQL
	mySQLArgs := getInstallArgs(keycloak.MySQL.MySQLInstallArgs)

	var enabled string
	if keycloak.Enabled != nil {
		enabled = strconv.FormatBool(*keycloak.Enabled)
	} else {
		enabled = "true"
	}
	keycloakConfig := Keycloak{
		KeycloakInstallArgs: getInstallArgs(keycloak.KeycloakInstallArgs),
		MySQL: MySQL{
			MySQLInstallArgs: mySQLArgs,
		},
		Enabled: enabled,
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
			Name:  "persistence.enabled",
			Value: "false",
		})
	} else if mysqlVolumeSource.PersistentVolumeClaim != nil {
		// Configured for persistence, adapt the PVC Spec template to the appropriate Helm args
		pvcs := mysqlVolumeSource.PersistentVolumeClaim
		storageSpec, found := findVolumeTemplate(pvcs.ClaimName, templates)
		if !found {
			err := fmt.Errorf("No VolumeClaimTemplate found for %s", pvcs.ClaimName)
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
		// Enable MySQL persistence
		mySQLArgs = append(mySQLArgs, InstallArg{
			Name:  "persistence.enabled",
			Value: "true",
		})
	}
	// Update the MySQL Install args
	keycloakConfig.MySQL.MySQLInstallArgs = mySQLArgs
	return keycloakConfig, nil
}

// getIngress returns the representation for the ingress for the installer scripts
func getIngress(ingress *installv1alpha1.IngressNginxComponent, istio *installv1alpha1.IstioComponent) Ingress {
	ingressConfig := Ingress{Type: getIngressType("")}
	if ingress != nil {
		ingressConfig.Type = getIngressType(ingress.Type)
		ingressConfig.Verrazzano = Verrazzano{
			NginxInstallArgs: getInstallArgs(ingress.NGINXInstallArgs),
			Ports:            getIngressPorts(ingress.Ports),
		}
	}
	if istio != nil {
		ingressConfig.Application = Application{
			IstioInstallArgs: getInstallArgs(istio.IstioInstallArgs),
		}
	}
	return ingressConfig
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
func getCAClusterResourceNamespace(cmConfig *installv1alpha1.CertManagerComponent) string {
	if cmConfig == nil {
		return defaultCAClusterResourceName
	}
	ca := cmConfig.Certificate.CA
	// Use default value if not specified
	if ca.ClusterResourceNamespace == "" {
		return defaultCAClusterResourceName
	}
	return ca.ClusterResourceNamespace
}

// getCASecretName returns the secret name for a CA certificate
func getCASecretName(cmConfig *installv1alpha1.CertManagerComponent) string {
	if cmConfig == nil {
		return defaultCASecretName
	}
	ca := cmConfig.Certificate.CA
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
		return constants.DefaultEnvironmentName
	}

	return envName
}

// getProfile returns the install profile name
func getProfile(profileType installv1alpha1.ProfileType) installv1alpha1.ProfileType {
	// Use the prod install profile if not specified
	switch profileType {
	case installv1alpha1.Prod, installv1alpha1.Dev, installv1alpha1.ManagedCluster:
		return profileType
	default:
		zap.S().Infof("Using profile %s, profile in resource either not found or unspecified: %s",
			installv1alpha1.Prod, profileType)
		return installv1alpha1.Prod
	}
}

// getVerrazzanoInstallArgs Set custom helm args for the Verrazzano internal component as needed
func getVerrazzanoInstallArgs(vzSpec *installv1alpha1.VerrazzanoSpec) ([]InstallArg, error) {
	args := []InstallArg{}
	if vzSpec.DefaultVolumeSource != nil {
		if vzSpec.DefaultVolumeSource.EmptyDir != nil {
			args = append(args, []InstallArg{
				{
					Name:      esStorageValueName,
					Value:     "",
					SetString: true,
				},
				{
					Name:      grafanaStorageValueName,
					Value:     "",
					SetString: true,
				},
				{
					Name:      prometheusStorageValueName,
					Value:     "",
					SetString: true,
				},
			}...)
		} else if vzSpec.DefaultVolumeSource.PersistentVolumeClaim != nil {
			pvcs := vzSpec.DefaultVolumeSource.PersistentVolumeClaim
			storageSpec, found := findVolumeTemplate(pvcs.ClaimName, vzSpec.VolumeClaimSpecTemplates)
			if !found {
				err := fmt.Errorf("No VolumeClaimTemplate found for %s", pvcs.ClaimName)
				return []InstallArg{}, err
			}
			args = append(args, []InstallArg{
				{
					Name:      esStorageValueName,
					Value:     storageSpec.Resources.Requests.Storage().String(),
					SetString: true,
				},
				{
					Name:      grafanaStorageValueName,
					Value:     storageSpec.Resources.Requests.Storage().String(),
					SetString: true,
				},
				{
					Name:      prometheusStorageValueName,
					Value:     storageSpec.Resources.Requests.Storage().String(),
					SetString: true,
				},
			}...)
		}
	}
	if len(vzSpec.Security.AdminSubjects) > 0 {
		for i, v := range vzSpec.Security.AdminSubjects {
			if err := validateRoleBindingSubject(v, fmt.Sprintf("adminSubjects[%d]", i)); err != nil {
				return []InstallArg{}, err
			}
			args = append(args, InstallArg{
				Name:      fmt.Sprintf("security.adminSubjects.subject-%d.name", i),
				Value:     v.Name,
				SetString: true,
			})
			args = append(args, InstallArg{
				Name:      fmt.Sprintf("security.adminSubjects.subject-%d.kind", i),
				Value:     v.Kind,
				SetString: true,
			})
			if len(v.Namespace) > 0 {
				args = append(args, InstallArg{
					Name:      fmt.Sprintf("security.adminSubjects.subject-%d.namespace", i),
					Value:     v.Namespace,
					SetString: true,
				})
			}
			if len(v.APIGroup) > 0 {
				args = append(args, InstallArg{
					Name:      fmt.Sprintf("security.adminSubjects.subject-%d.apiGroup", i),
					Value:     v.APIGroup,
					SetString: true,
				})
			}
		}
	}
	if len(vzSpec.Security.MonitorSubjects) > 0 {
		for i, v := range vzSpec.Security.MonitorSubjects {
			if err := validateRoleBindingSubject(v, fmt.Sprintf("adminSubjects[%d]", i)); err != nil {
				return []InstallArg{}, err
			}
			args = append(args, InstallArg{
				Name:      fmt.Sprintf("security.monitorSubjects.subject-%d.name", i),
				Value:     v.Name,
				SetString: true,
			})
			args = append(args, InstallArg{
				Name:      fmt.Sprintf("security.monitorSubjects.subject-%d.kind", i),
				Value:     v.Kind,
				SetString: true,
			})
			if len(v.Namespace) > 0 {
				args = append(args, InstallArg{
					Name:      fmt.Sprintf("security.monitorSubjects.subject-%d.namespace", i),
					Value:     v.Namespace,
					SetString: true,
				})
			}
			if len(v.APIGroup) > 0 {
				args = append(args, InstallArg{
					Name:      fmt.Sprintf("security.monitorSubjects.subject-%d.apiGroup", i),
					Value:     v.APIGroup,
					SetString: true,
				})
			}
		}
	}

	args = append(args, getVMIInstallArgs(vzSpec)...)

	// Console
	if vzSpec.Components.Console != nil && vzSpec.Components.Console.Enabled != nil {
		args = append(args, InstallArg{
			Name:  consoleEnabledValueName,
			Value: strconv.FormatBool(*vzSpec.Components.Console.Enabled),
		})
	}

	return args, nil
}

func validateRoleBindingSubject(subject rbacv1.Subject, name string) error {
	if len(subject.Name) < 1 {
		err := fmt.Errorf("no name for %s", name)
		return err
	}
	if subject.Kind != "User" && subject.Kind != "Group" && subject.Kind != "ServiceAccount" {
		err := fmt.Errorf("invalid kind '%s' for %s", subject.Kind, name)
		return err
	}
	if (subject.Kind == "User" || subject.Kind == "Group") && len(subject.APIGroup) > 0 && subject.APIGroup != "rbac.authorization.k8s.io" {
		err := fmt.Errorf("invalid apiGroup '%s' for %s", subject.APIGroup, name)
		return err
	}
	if subject.Kind == "ServiceAccount" && (len(subject.APIGroup) > 0 || subject.APIGroup != "") {
		err := fmt.Errorf("invalid apiGroup '%s' for %s", subject.APIGroup, name)
		return err
	}
	if subject.Kind == "ServiceAccount" && len(subject.Namespace) < 1 {
		err := fmt.Errorf("no namespace for ServiceAccount in %s", name)
		return err
	}
	return nil
}

func getVMIInstallArgs(vzSpec *installv1alpha1.VerrazzanoSpec) []InstallArg {
	const helmValuePrefix = "elasticSearch."
	vmiArgs := []InstallArg{}
	if vzSpec.Components.Elasticsearch != nil {
		if vzSpec.Components.Elasticsearch.Enabled != nil {
			vmiArgs = append(vmiArgs, InstallArg{
				Name:  esEnabledValueName,
				Value: strconv.FormatBool(*vzSpec.Components.Elasticsearch.Enabled),
			})
		}
		// Add the set of args specified in the yaml, prefixing the elasticSearch string
		// For example, the following YAML will result in `elasticSearch.nodes.master.replicas`
		// elasticsearch:
		//   installArgs:
		//	   - name: nodes.master.replicas
		//       value: "2"
		for _, esArg := range vzSpec.Components.Elasticsearch.ESInstallArgs {
			vmiArgs = append(vmiArgs, InstallArg{
				Name:      helmValuePrefix + esArg.Name,
				Value:     esArg.Value,
				SetString: esArg.SetString,
			})
		}
	}
	if vzSpec.Components.Prometheus != nil && vzSpec.Components.Prometheus.Enabled != nil {
		vmiArgs = append(vmiArgs, InstallArg{
			Name:  promEnabledValueName,
			Value: strconv.FormatBool(*vzSpec.Components.Prometheus.Enabled),
		})
	}

	if vzSpec.Components.Kibana != nil && vzSpec.Components.Kibana.Enabled != nil {
		vmiArgs = append(vmiArgs, InstallArg{
			Name:  kibanaEnabledValueName,
			Value: strconv.FormatBool(*vzSpec.Components.Kibana.Enabled),
		})
	}

	if vzSpec.Components.Grafana != nil && vzSpec.Components.Grafana.Enabled != nil {
		vmiArgs = append(vmiArgs, InstallArg{
			Name:  grafanaEnabledValueName,
			Value: strconv.FormatBool(*vzSpec.Components.Grafana.Enabled),
		})
	}

	return vmiArgs
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

func getFluentd(comp *installv1alpha1.FluentdComponent) Fluentd {
	if comp == nil {
		return Fluentd{
			Enabled:             "true",
			ElasticsearchURL:    defaultElasticsearchURL,
			ElasticsearchSecret: defaultElasticsearchSecret,
		}
	}
	fluentd := Fluentd{}
	fluentd.FluentdInstallArgs = []InstallArg{}
	for i, vm := range comp.ExtraVolumeMounts {
		fluentd.FluentdInstallArgs = append(fluentd.FluentdInstallArgs, InstallArg{
			Name:  fmt.Sprintf("logging.extraVolumeMounts[%d].source", i),
			Value: vm.Source,
		})
		dest := vm.Source
		if vm.Destination != "" {
			dest = vm.Destination
		}
		fluentd.FluentdInstallArgs = append(fluentd.FluentdInstallArgs, InstallArg{
			Name:  fmt.Sprintf("logging.extraVolumeMounts[%d].destination", i),
			Value: dest,
		})
		readOnly := true
		if vm.ReadOnly != nil {
			readOnly = *vm.ReadOnly
		}
		fluentd.FluentdInstallArgs = append(fluentd.FluentdInstallArgs, InstallArg{
			Name:  fmt.Sprintf("logging.extraVolumeMounts[%d].readOnly", i),
			Value: strconv.FormatBool(readOnly),
		})
	}

	var enabled string
	if comp.Enabled != nil {
		enabled = strconv.FormatBool(*comp.Enabled)
	} else {
		enabled = "true"
	}
	fluentd.Enabled = enabled

	if len(comp.ElasticsearchURL) > 0 {
		fluentd.ElasticsearchURL = comp.ElasticsearchURL
	} else {
		fluentd.ElasticsearchURL = defaultElasticsearchURL
	}

	if len(comp.ElasticsearchSecret) > 0 {
		fluentd.ElasticsearchSecret = comp.ElasticsearchSecret
	} else {
		fluentd.ElasticsearchSecret = defaultElasticsearchSecret
	}

	return fluentd
}
