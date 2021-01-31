// Copyright (c) 2020, 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package v1alpha1

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ProfileType is the type of install profile.
type ProfileType string

const (
	// Dev identifies the development install profile
	Dev ProfileType = "dev"
	// Prod identifies the production install profile
	Prod ProfileType = "prod"
)
const (
	// LoadBalancer is an ingress type of LoadBalancer.  This is the default value.
	LoadBalancer IngressType = "LoadBalancer"
	// NodePort is an ingress type of NodePort.
	NodePort IngressType = "NodePort"
)

// +kubebuilder:object:root=true
// +kubebuilder:resource:path=verrazzanos
// +kubebuilder:subresource:status
// +kubebuilder:resource:shortName=vz;vzs
// +kubebuilder:printcolumn:name="Status",type="string",JSONPath=".status.conditions[-1:].type",description="The current status of the install/uninstall"
// +kubebuilder:printcolumn:name="Version",type="string",JSONPath=".status.version",description="The current version of the Verrazzano installation"
// +genclient

// Verrazzano is the Schema for the verrazzanos API
type Verrazzano struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   VerrazzanoSpec   `json:"spec,omitempty"`
	Status VerrazzanoStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// VerrazzanoList contains a list of Verrazzano
type VerrazzanoList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Verrazzano `json:"items"`
}

// VerrazzanoSpec defines the desired state of Verrazzano
type VerrazzanoSpec struct {
	// Version is the Verrazzano version
	// +optional
	Version string `json:"version,omitempty"`
	// Profile is the name of the profile to install.  Default is "prod".
	// +optional
	Profile ProfileType `json:"profile,omitempty"`
	// EnvironmentName identifies install environment.  Default environment name is "default".
	// +optional
	EnvironmentName string `json:"environmentName,omitempty"`
	// Core specifies core Verrazzano configuration
	// +optional
	Components ComponentSpec `json:"components,omitempty"`

	// DefaultVolumeSource Defines the type of volume to be used for persistence, if not explicitly declared by a component;
	// at present only EmptyDirVolumeSource or PersistentVolumeClaimVolumeSource are supported. If PersistentVolumeClaimVolumeSource
	// is used, it must reference a VolumeClaimSpecTemplate in the VolumeClaimSpecTemplates section.
	// +optional
	DefaultVolumeSource *corev1.VolumeSource `json:"defaultVolumeSource,omitempty"`

	// VolumeClaimSpecTemplates Defines a named set of PVC configurations that can be referenced from components using persistent volumes.
	// +optional
	VolumeClaimSpecTemplates []VolumeClaimSpecTemplate `json:"volumeClaimSpecTemplates,omitempty"`
}

// VolumeClaimSpecTemplate Contains common PVC configuration that can be referenced from Components; these
// do not actually result in generated PVCs, but can used to provide common configuration to components that
// declare a PersistentVolumeClaimVolumeSource
type VolumeClaimSpecTemplate struct {
	// Name The name of the configuration template
	Name string `json:"name"`
	// Spec The configuration specs for the template
	Spec corev1.PersistentVolumeClaimSpec `json:"spec,omitempty"`
}

// VerrazzanoStatus defines the observed state of Verrazzano
type VerrazzanoStatus struct {
	// The version of Verrazzano that is installed
	Version string `json:"version,omitempty"`
	// The latest available observations of an object's current state.
	Conditions []Condition `json:"conditions,omitempty"`
	// State of the Verrazzano custom resource
	State StateType `json:"state,omitempty"`
}

// ConditionType identifies the condition of the install/uninstall/upgrade which can be checked with kubectl wait
type ConditionType string

const (
	// InstallStarted means an install is in progress.
	InstallStarted ConditionType = "InstallStarted"

	// InstallComplete means the install job has completed its execution successfully
	InstallComplete ConditionType = "InstallComplete"

	// InstallFailed means the install job has failed during execution.
	InstallFailed ConditionType = "InstallFailed"

	// UninstallStarted means an uninstall is in progress.
	UninstallStarted ConditionType = "UninstallStarted"

	// UninstallComplete means the uninstall job has completed its execution successfully
	UninstallComplete ConditionType = "UninstallComplete"

	// UninstallFailed means the uninstall job has failed during execution.
	UninstallFailed ConditionType = "UninstallFailed"

	// UpgradeStarted means that an upgrade has been started.
	UpgradeStarted ConditionType = "UpgradeStarted"

	// UpgradeFailed means the upgrade has failed during execution.
	UpgradeFailed ConditionType = "UpgradeFailed"

	// UpgradeComplete means the upgrade has completed successfully
	UpgradeComplete ConditionType = "UpgradeComplete"
)

// Condition describes current state of an install.
type Condition struct {
	// Type of condition.
	Type ConditionType `json:"type"`
	// Status of the condition, one of True, False, Unknown.
	Status corev1.ConditionStatus `json:"status"`
	// Last time the condition transitioned from one status to another.
	// +optional
	LastTransitionTime string `json:"lastTransitionTime,omitempty"`
	// Human readable message indicating details about last transition.
	// +optional
	Message string `json:"message,omitempty"`
}

// StateType identifies the state of an install/uninstall/upgrade
type StateType string

const (
	// Installing is the state when an install is in progress
	Installing StateType = "Installing"

	// Uninstalling is the state when an uninstall is in progress
	Uninstalling StateType = "Uninstalling"

	// Upgrading is the state when an upgrade is in progress
	Upgrading StateType = "Upgrading"

	// Ready is the state when a Verrazzano resource can perform an uninstall or upgrade
	Ready StateType = "Ready"

	// Failed is the state when an install/uninstall/upgrade has failed
	Failed StateType = "Failed"
)

// ComponentSpec contains a set of components used by Verrazzano
type ComponentSpec struct {
	// CertManager contains the CertManager component configuration
	// +optional
	CertManager CertManagerComponent `json:"certManager,omitempty"`
	// DNS contains the DNS component configuration
	// +optional
	DNS DNSComponent `json:"dns,omitempty"`
	// Ingress contains the ingress-nginx component configuration
	// +optional
	Ingress IngressNginxComponent `json:"ingress,omitempty"`
	// Istio contains the istio component configuration
	// +optional
	Istio IstioComponent `json:"istio,omitempty"`
	// Keycloak contains the Keycloak component configuration
	// +optional
	Keycloak KeycloakComponent `json:"keycloak,omitempty"`
	// OAM contains the OAM component configuration
	// +optional
	OAM OAMComponent `json:"oam,omitempty"`
}

// CertManagerComponent specifies the core CertManagerComponent config.
type CertManagerComponent struct {
	// Certificate used for an install
	// +optional
	Certificate Certificate `json:"certificate,omitempty"`
}

// DNSComponent specifies the DNS configuration
type DNSComponent struct {
	// DNS type of xip.io.  This is the default.
	// +optional
	XIPIO XIPIO `json:"xip.io,omitempty"`
	// DNS type of OCI (Oracle Cloud Infrastructure)
	// +optional
	OCI OCI `json:"oci,omitempty"`
	// DNS type of external. For example, OLCNE uses this type.
	// +optional
	External External `json:"external,omitempty"`
}

// IngressNginxComponent specifies the ingress-nginx configuration
type IngressNginxComponent struct {
	// Type of ingress.  Default is LoadBalancer
	// +optional
	Type IngressType `json:"type,omitempty"`
	// Arguments for installing NGINX
	// +optional
	NGINXInstallArgs []InstallArgs `json:"nginxInstallArgs,omitempty"`
	// Ports to be used for NGINX
	// +optional
	Ports []corev1.ServicePort `json:"ports,omitempty"`
}

// IstioComponent specifies the Istio configuration
type IstioComponent struct {
	// Arguments for installing Istio
	// +optional
	IstioInstallArgs []InstallArgs `json:"istioInstallArgs,omitempty"`
}

// KeycloakComponent specifies the Keycloak configuration
type KeycloakComponent struct {
	// Arguments for installing Keycloak
	// +optional
	KeycloakInstallArgs []InstallArgs `json:"keycloakInstallArgs,omitempty"`
	// MySQL contains the MySQL component configuration needed for Keycloak
	// +optional
	MySQL MySQLComponent `json:"mysql,omitempty"`
}

// MySQLComponent specifies the MySQL configuration
type MySQLComponent struct {
	// Arguments for installing MySQL
	// +optional
	MySQLInstallArgs []InstallArgs `json:"mysqlInstallArgs,omitempty"`
	// VolumeSource Defines the type of volume to be used for persistence; at present only EmptyDirVolumeSource or
	// PersistentVolumeClaimVolumeSource are supported. If PersistentVolumeClaimVolumeSource
	// is used, it must reference a VolumeClaimSpecTemplate in the VolumeClaimSpecTemplates section.
	// +optional
	VolumeSource *corev1.VolumeSource `json:"volumeSource,omitempty"`
}

// OAMComponent specifies the OAM configuration
type OAMComponent struct {
	// Argument to enable installation of OAM components
	// +optional
	Enabled bool `json:"enabled,omitempty"`
}

// InstallArgs identifies a name/value or name/value list needed for install.
// Value and ValueList cannot both be specified.
type InstallArgs struct {
	// Name of install argument
	Name string `json:"name"`
	// Value for named install argument
	// +optional
	Value string `json:"value,omitempty"`
	// If the Value is a literal string
	// +optional
	SetString bool `json:"setString,omitempty"`
	// List of values for named install argument
	// +optional
	ValueList []string `json:"valueList,omitempty"`
}

// ProviderType identifies Acme provider type.
type ProviderType string

const (
	// LetsEncrypt is a Let's Encrypt provider
	LetsEncrypt ProviderType = "LetsEncrypt"
)

// Acme identifies the ACME cert issuer.
type Acme struct {
	// Type of provider for ACME cert issuer.
	Provider ProviderType `json:"provider"`
	// email address
	// +optional
	EmailAddress string `json:"emailAddress,omitempty"`
	// environment
	// +optional
	Environment string `json:"environment,omitempty"`
}

// CA identifies the CA cert issuer.
type CA struct {
	// Name of secret for CA cert issuer
	SecretName string `json:"secretName"`
	// Namespace where secret is located for CA cert issuer
	ClusterResourceNamespace string `json:"clusterResourceNamespace"`
}

// Certificate represents the type of cert issuer for an install
// Only one of its members may be specified.
type Certificate struct {
	// ACME cert issuer
	// +optional
	Acme Acme `json:"acme,omitempty"`
	// CA cert issuer
	// +optional
	CA CA `json:"ca,omitempty"`
}

// OciPrivateKeyFileName is the private key file name
const OciPrivateKeyFileName = "oci_api_key.pem"

// OciPrivateKeyFilePath is the private key mount path
const OciPrivateKeyFilePath = "/config/" + OciPrivateKeyFileName

// OciConfigSecretFile is the name of the OCI configuration yaml file
const OciConfigSecretFile = "oci.yaml"

// XIPIO is xip.io DNS type
type XIPIO struct {
}

// OCI DNS type
type OCI struct {
	OCIConfigSecret        string `json:"ociConfigSecret"`
	DNSZoneCompartmentOCID string `json:"dnsZoneCompartmentOCID"`
	DNSZoneOCID            string `json:"dnsZoneOCID"`
	DNSZoneName            string `json:"dnsZoneName"`
}

// External DNS type
type External struct {
	// DNS suffix appended to EnviromentName to form DNS name
	Suffix string `json:"suffix"`
}

// IngressType is the type of ingress.
type IngressType string

func init() {
	SchemeBuilder.Register(&Verrazzano{}, &VerrazzanoList{})
}
