// Copyright (c) 2020, 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package v1alpha1

import (
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ProfileType is the type of install profile.
type ProfileType string

const (
	// Dev identifies the development install profile
	Dev ProfileType = "dev"
	// Prod identifies the production install profile
	Prod ProfileType = "prod"
	// ManagedCluster identifies the production managed-cluster install profile
	ManagedCluster ProfileType = "managed-cluster"
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

	// Security specifies Verrazzano security configuration
	// +optional
	Security SecuritySpec `json:"security,omitempty"`

	// DefaultVolumeSource Defines the type of volume to be used for persistence, if not explicitly declared by a component;
	// at present only EmptyDirVolumeSource or PersistentVolumeClaimVolumeSource are supported. If PersistentVolumeClaimVolumeSource
	// is used, it must reference a VolumeClaimSpecTemplate in the VolumeClaimSpecTemplates section.
	// +optional
	// +patchStrategy=replace
	DefaultVolumeSource *corev1.VolumeSource `json:"defaultVolumeSource,omitempty" patchStrategy:"replace"`

	// VolumeClaimSpecTemplates Defines a named set of PVC configurations that can be referenced from components using persistent volumes.
	// +optional
	// +patchStrategy=merge,retainKeys
	VolumeClaimSpecTemplates []VolumeClaimSpecTemplate `json:"volumeClaimSpecTemplates,omitempty" patchStrategy:"merge,retainKeys" patchMergeKey:"name"`
}

// SecuritySpec defines the security configuration for Verrazzano
type SecuritySpec struct {
	// AdminSubjects specifies subjects that should be bound to the verrazzano-admin role
	// +optional
	AdminSubjects []rbacv1.Subject `json:"adminSubjects,omitempty"`
	// MonitorSubjects specifies subjects that should be bound to the verrazzano-monitor role
	// +optional
	MonitorSubjects []rbacv1.Subject `json:"monitorSubjects,omitempty"`
}

// VolumeClaimSpecTemplate Contains common PVC configuration that can be referenced from Components; these
// do not actually result in generated PVCs, but can used to provide common configuration to components that
// declare a PersistentVolumeClaimVolumeSource
type VolumeClaimSpecTemplate struct {
	// Metadata about the PersistentVolumeClaimSpec template.
	// +kubebuilder:pruning:PreserveUnknownFields
	metav1.ObjectMeta `json:"metadata,omitempty"`
	// Spec The configuration specs for the template
	Spec corev1.PersistentVolumeClaimSpec `json:"spec,omitempty"`
}

// InstanceInfo details of installed Verrazzano instance maintained in status field
type InstanceInfo struct {
	// ConsoleURL The Console URL for this Verrazzano installation
	ConsoleURL *string `json:"consoleUrl,omitempty"`
	// KeyCloakURL The KeyCloak URL for this Verrazzano installation
	KeyCloakURL *string `json:"keyCloakUrl,omitempty"`
	// RancherURL The Rancher URL for this Verrazzano installation
	RancherURL *string `json:"rancherUrl,omitempty"`
	// ElasticURL The Elasticsearch URL for this Verrazzano installation
	ElasticURL *string `json:"elasticUrl,omitempty"`
	// KibanaURL The Kibana URL for this Verrazzano installation
	KibanaURL *string `json:"kibanaUrl,omitempty"`
	// GrafanaURL The Grafana URL for this Verrazzano installation
	GrafanaURL *string `json:"grafanaUrl,omitempty"`
	// PrometheusURL The Prometheus URL for this Verrazzano installation
	PrometheusURL *string `json:"prometheusUrl,omitempty"`
	// KialiURL The Kiali URL for this Verrazzano installation
	KialiURL *string `json:"kialiUrl,omitempty"`
}

// VerrazzanoStatus defines the observed state of Verrazzano
type VerrazzanoStatus struct {
	// The version of Verrazzano that is installed
	Version string `json:"version,omitempty"`
	// The Verrazzano instance info
	VerrazzanoInstance *InstanceInfo `json:"instance,omitempty"`
	// The latest available observations of an object's current state.
	Conditions []Condition `json:"conditions,omitempty"`
	// State of the Verrazzano custom resource
	State StateType `json:"state,omitempty"`
	// States of the individual installed components
	Components ComponentStatusMap `json:"components,omitempty"`
}

type ComponentStatusMap map[string]*ComponentStatusDetails

// ComponentStatusDetails defines the observed state of a Verrazzano component
type ComponentStatusDetails struct {
	// Name of the component
	Name string `json:"name,omitempty"`
	// Information about the current state of a component
	Conditions []Condition `json:"conditions,omitempty"`
	// The version of Verrazzano that is installed
	State StateType `json:"state,omitempty"`
	// The version of Verrazzano that is installed
	Version string `json:"version,omitempty"`
}

// ConditionType identifies the condition of the install/uninstall/upgrade which can be checked with kubectl wait
type ConditionType string

const (
	// PreInstall means an install about to start.
	PreInstall ConditionType = "PreInstall"

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
	// Disabled is the state for when a component is not currently installed
	Disabled StateType = "Disabled"

	// PreInstalling is the state when an install is about to be started
	PreInstalling StateType = "PreInstalling"

	// Installing is the state when an install is in progress
	Installing StateType = "Installing"

	// Uninstalling is the state when an uninstall is in progress
	Uninstalling StateType = "Uninstalling"

	// Upgrading is the state when an upgrade is in progress
	Upgrading StateType = "Upgrading"

	// Updating is the state when a component configuration update is being applied
	Updating StateType = "Updating"

	// Error is the state when a Verrazzano resource has experienced an error that may leave it in an unstable state
	Error StateType = "Error"

	// Ready is the state when a Verrazzano resource can perform an uninstall or upgrade
	Ready StateType = "Ready"

	// Quiescing state when waiting for stable condition to start uninstall
	Quiescing StateType = "Quiescing"

	// Failed is the state when an install/uninstall/upgrade has failed
	Failed StateType = "Failed"
)

// ComponentSpec contains a set of components used by Verrazzano
type ComponentSpec struct {
	// CertManager contains the CertManager component configuration
	// +optional
	CertManager *CertManagerComponent `json:"certManager,omitempty"`

	// CoherenceOperator configuration
	// +optional
	CoherenceOperator *CoherenceOperatorComponent `json:"coherenceOperator,omitempty"`

	// ApplicationOperator configuration
	// +optional
	ApplicationOperator *ApplicationOperatorComponent `json:"applicationOperator,omitempty"`

	// OAM configuration
	// +optional
	OAM *OAMComponent `json:"oam,omitempty"`

	// Console configuration
	// +optional
	Console *ConsoleComponent `json:"console,omitempty"`

	// DNS contains the DNS component configuration
	// +optional
	// +patchStrategy=replace
	DNS *DNSComponent `json:"dns,omitempty" patchStrategy:"replace"`

	// Elasticsearch configuration
	// +optional
	Elasticsearch *ElasticsearchComponent `json:"elasticsearch,omitempty"`

	// Fluentd configuration
	// +optional
	Fluentd *FluentdComponent `json:"fluentd,omitempty"`

	// Grafana configuration
	// +optional
	Grafana *GrafanaComponent `json:"grafana,omitempty"`

	// Ingress contains the ingress-nginx component configuration
	// +optional
	Ingress *IngressNginxComponent `json:"ingress,omitempty"`

	// Istio contains the istio component configuration
	// +optional
	Istio *IstioComponent `json:"istio,omitempty"`

	// Kiali contains the Kiali component configuration
	// +optional
	Kiali *KialiComponent `json:"kiali,omitempty"`

	// Keycloak contains the Keycloak component configuration
	// +optional
	Keycloak *KeycloakComponent `json:"keycloak,omitempty"`

	// Grafana configuration
	// +optional
	Kibana *KibanaComponent `json:"kibana,omitempty"`

	// Prometheus configuration
	// +optional
	Prometheus *PrometheusComponent `json:"prometheus,omitempty"`

	// Rancher configuration
	// +optional
	Rancher *RancherComponent `json:"rancher,omitempty"`

	// WebLogicOperator configuration
	// +optional
	WebLogicOperator *WebLogicOperatorComponent `json:"weblogicOperator,omitempty"`

	// Verrazzano configuration
	// +optional
	Verrazzano *VerrazzanoComponent `json:"verrazzano,omitempty"`
}

// MonitoringComponent Common configuration for monitoring components
type MonitoringComponent struct {
	// +optional
	Enabled *bool `json:"enabled,omitempty"`
}

// ElasticsearchComponent specifies the Elasticsearch configuration.
type ElasticsearchComponent struct {
	MonitoringComponent `json:",inline"`

	// Arguments for installing Elasticsearch
	// +optional
	// +patchMergeKey=name
	// +patchStrategy=merge,retainKeys
	ESInstallArgs []InstallArgs `json:"installArgs,omitempty" patchStrategy:"merge,retainKeys" patchMergeKey:"name"`
	// +optional
	RetentionPolicies RetentionPolicies `json:"retentionPolicies,omitempty"`
}

// KibanaComponent specifies the Kibana configuration.
type KibanaComponent struct {
	MonitoringComponent `json:",inline"`
}

// GrafanaComponent specifies the Grafana configuration.
type GrafanaComponent struct {
	MonitoringComponent `json:",inline"`
}

// PrometheusComponent specifies the Prometheus configuration.
type PrometheusComponent struct {
	MonitoringComponent `json:",inline"`
}

// CertManagerComponent specifies the core CertManagerComponent config.
type CertManagerComponent struct {
	// Certificate used for an install
	// +optional
	// +patchStrategy=replace
	Certificate Certificate `json:"certificate,omitempty" patchStrategy:"replace"`
	// +optional
	Enabled *bool `json:"enabled,omitempty"`
}

// CoherenceOperatorComponent specifies the Coherence Operator configuration
type CoherenceOperatorComponent struct {
	// +optional
	Enabled *bool `json:"enabled,omitempty"`
}

// ApplicationOperatorComponent specifies the Application Operator configuration
type ApplicationOperatorComponent struct {
	// +optional
	Enabled *bool `json:"enabled,omitempty"`
}

// OAMComponent specifies the OAM configuration
type OAMComponent struct {
	// +optional
	Enabled *bool `json:"enabled,omitempty"`
}

// VerrazzanoComponent specifies the Verrazzano configuration
type VerrazzanoComponent struct {
	// +optional
	Enabled *bool `json:"enabled,omitempty"`
}

// KialiComponent specifies the Kiali configuration
type KialiComponent struct {
	// +optional
	Enabled *bool `json:"enabled,omitempty"`
}

// ConsoleComponent specifies the Console UI configuration
type ConsoleComponent struct {
	MonitoringComponent `json:",inline"`
}

// DNSComponent specifies the DNS configuration
type DNSComponent struct {
	// DNS type of wildcard.  This is the default.
	// +optional
	Wildcard *Wildcard `json:"wildcard,omitempty"`
	// DNS type of OCI (Oracle Cloud Infrastructure)
	// +optional
	OCI *OCI `json:"oci,omitempty"`
	// DNS type of external. For example, OLCNE uses this type.
	// +optional
	External *External `json:"external,omitempty"`
}

// IngressNginxComponent specifies the ingress-nginx configuration
type IngressNginxComponent struct {
	// Type of ingress.  Default is LoadBalancer
	// +optional
	Type IngressType `json:"type,omitempty"`
	// Arguments for installing NGINX
	// +optional
	// +patchMergeKey=name
	// +patchStrategy=merge,retainKeys
	NGINXInstallArgs []InstallArgs `json:"nginxInstallArgs,omitempty" patchStrategy:"merge,retainKeys" patchMergeKey:"name"`
	// Ports to be used for NGINX
	// +optional
	Ports []corev1.ServicePort `json:"ports,omitempty"`
	// +optional
	Enabled *bool `json:"enabled,omitempty"`
}

// IstioComponent specifies the Istio configuration
type IstioComponent struct {
	// Arguments for installing Istio
	// +optional
	// +patchMergeKey=name
	// +patchStrategy=merge,retainKeys
	IstioInstallArgs []InstallArgs `json:"istioInstallArgs,omitempty" patchStrategy:"merge,retainKeys" patchMergeKey:"name"`
	// +optional
	Enabled *bool `json:"enabled,omitempty"`
}

// KeycloakComponent specifies the Keycloak configuration
type KeycloakComponent struct {
	// Arguments for installing Keycloak
	// +optional
	// +patchMergeKey=name
	// +patchStrategy=merge,retainKeys
	KeycloakInstallArgs []InstallArgs `json:"keycloakInstallArgs,omitempty" patchStrategy:"merge,retainKeys" patchMergeKey:"name"`
	// MySQL contains the MySQL component configuration needed for Keycloak
	// +optional
	MySQL MySQLComponent `json:"mysql,omitempty"`
	// +optional
	Enabled *bool `json:"enabled,omitempty"`
}

// MySQLComponent specifies the MySQL configuration
type MySQLComponent struct {
	// Arguments for installing MySQL
	// +optional
	// +patchMergeKey=name
	// +patchStrategy=merge,retainKeys
	MySQLInstallArgs []InstallArgs `json:"mysqlInstallArgs,omitempty" patchStrategy:"merge,retainKeys" patchMergeKey:"name"`
	// VolumeSource Defines the type of volume to be used for persistence; at present only EmptyDirVolumeSource or
	// PersistentVolumeClaimVolumeSource are supported. If PersistentVolumeClaimVolumeSource
	// is used, it must reference a VolumeClaimSpecTemplate in the VolumeClaimSpecTemplates section.
	// +optional
	// +patchStrategy=replace
	VolumeSource *corev1.VolumeSource `json:"volumeSource,omitempty" patchStrategy:"replace"`
}

// RancherComponent specifies the Rancher configuration
type RancherComponent struct {
	// +optional
	Enabled *bool `json:"enabled,omitempty"`
}

// FluentdComponent specifies the Fluentd DaemonSet configuration
type FluentdComponent struct {
	// Specifies whether Fluentd is deployed or not on a cluster.  Default is true.
	// +optional
	Enabled *bool `json:"enabled,omitempty"`
	// +optional
	// +patchStrategy=merge,retainKeys
	ExtraVolumeMounts []VolumeMount `json:"extraVolumeMounts,omitempty" patchStrategy:"merge,retainKeys" patchMergeKey:"source"`
	// +optional
	ElasticsearchURL string `json:"elasticsearchURL,omitempty"`
	// +optional
	ElasticsearchSecret string `json:"elasticsearchSecret,omitempty"`

	// Configuration for integration with OCI (Oracle Cloud Infrastructure) Logging Service
	// +optional
	OCI *OciLoggingConfiguration `json:"oci,omitempty"`
}

// WebLogicOperatorComponent specifies the WebLogic Operator configuration
type WebLogicOperatorComponent struct {
	// +optional
	Enabled *bool `json:"enabled,omitempty"`
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
	// +patchStrategy=replace
	ValueList []string `json:"valueList,omitempty" patchStrategy:"replace"`
}

//RetentionPolicies defines the policies which may be configued in Verrazzano
type RetentionPolicies struct {
	// +optional
	System RetentionPolicy `json:"system"`
	// +optional
	Application RetentionPolicy `json:"application"`
}

//RetentionPolicy defines configuration for an index retention policy
type RetentionPolicy struct {
	// If the RetentionPolicy should be applied
	// +optional
	Enabled *bool `json:"enabled"`
	// Min index age for Retention Policy
	// +optional
	MinAge *string `json:"minAge"`
}

type RolloverPolicy struct {
}

// VolumeMount defines a hostPath type Volume mount
type VolumeMount struct {
	// Source hostPath
	Source string `json:"source"`
	// Destination path on the Container, defaults to source hostPath
	// +optional
	Destination string `json:"destination,omitempty"`
	// ReadOnly defaults to true
	// +optional
	ReadOnly *bool `json:"readOnly,omitempty"`
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

// OciConfigSecretFile is the name of the OCI configuration yaml file
const OciConfigSecretFile = "oci.yaml"

// Wildcard DNS type
type Wildcard struct {
	// DNS wildcard domain (nip.io, sslip.io, etc.)
	Domain string `json:"domain"`
}

// OCI DNS type
type OCI struct {
	OCIConfigSecret        string `json:"ociConfigSecret"`
	DNSZoneCompartmentOCID string `json:"dnsZoneCompartmentOCID"`
	DNSZoneOCID            string `json:"dnsZoneOCID"`
	DNSZoneName            string `json:"dnsZoneName"`
	DNSScope               string `json:"dnsScope,omitempty"`
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

// OCI Logging configuration for Fluentd DaemonSet
type OciLoggingConfiguration struct {
	DefaultAppLogID string `json:"defaultAppLogId"`
	SystemLogID     string `json:"systemLogId"`
	APISecret       string `json:"apiSecret,omitempty"`
}
