// Copyright (c) 2020, Oracle and/or its affiliates.
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

// XIPIO is xip.io DNS type
type XIPIO struct {
}

// PrivateKeyPassphraseSecretRef identifies the private key passphrase needed for an OCI DNS install
type PrivateKeyPassphraseSecretRef struct {
	// Name of secret
	Name string `json:"name"`
	// Key for value in secret
	Key string `json:"key"`
}

// OciPrivateKeyFileName is the private key file name
const OciPrivateKeyFileName = "oci_api_key.pem"

// OciPrivateKeyFilePath is the private key mount path
const OciPrivateKeyFilePath = "/config/" + OciPrivateKeyFileName

// OciConfigSecretFile is the name of the OCI configuration yaml file
const OciConfigSecretFile = "oci.yaml"

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

// DNS Represents the type of DNS for an install.
// Only one of its members may be specified.
type DNS struct {
	// DNS type of xio.io.  This is the default.
	// +optional
	XIPIO XIPIO `json:"xip.io,omitempty"`
	// DNS type of OCI (Oracle Cloud Infrastructure)
	// +optional
	OCI OCI `json:"oci,omitempty"`
	// DNS type of external. For example, OLCNE uses this type.
	// +optional
	External External `json:"external,omitempty"`
}

// IngressType is the type of ingress.
type IngressType string

const (
	// LoadBalancer is an ingress type of LoadBalancer.  This is the default value.
	LoadBalancer IngressType = "LoadBalancer"
	// NodePort is an ingress type of NodePort.
	NodePort IngressType = "NodePort"
)

// InstallArgs identifies a name/value or name/value list needed for install.
// Value and ValueList cannot both be specified.
type InstallArgs struct {
	// Name of install argument
	Name string `json:"name"`
	// Value for named install argument
	// +optional
	Value string `json:"value,omitempty"`
	// List of values for named install argument
	// +optional
	ValueList []string `json:"valueList,omitempty"`
}

// VerrazzanoInstall identifies the Verrazzano infrastructure install options.
type VerrazzanoInstall struct {
	// Arguments for installing nginx
	// +optional
	NGINXInstallArgs []InstallArgs `json:"nginxInstallArgs,omitempty"`
	// Ports to be used for nginx
	// +optional
	Ports []corev1.ServicePort `json:"ports,omitempty"`
}

// ApplicationInstall identifies non-Verrazzano infrastructure install options.
type ApplicationInstall struct {
	// Arguments for installing istio
	// +optional
	IstioInstallArgs []InstallArgs `json:"istioInstallArgs,omitempty"`
}

// Ingress identifies the ingress configuration.
type Ingress struct {
	// Type of ingress.  Default is LoadBalancer
	// +optional
	Type IngressType `json:"type,omitempty"`
	// Verrazzano infrastructure install options
	// +optional
	Verrazzano VerrazzanoInstall `json:"verrazzano,omitempty"`
	// Non-Verrazzano infrastructure install options
	// +optional
	Application ApplicationInstall `json:"application,omitempty"`
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

// VerrazzanoSpec defines the desired state of Verrazzano
type VerrazzanoSpec struct {
	// Profile is the name of the profile to install.  Default is "prod".
	// +optional
	Profile ProfileType `json:"profile,omitempty"`
	// Short name to identify install environment.  Default environment name is "default".
	// +optional
	EnvironmentName string `json:"environmentName,omitempty"`
	// Type of DNS for an install
	// +optional
	DNS DNS `json:"dns,omitempty"`
	// Type of Ingress for an install
	// +optional
	Ingress Ingress `json:"ingress,omitempty"`
	// Type of cert issuer for an install
	// +optional
	Certificate Certificate `json:"certificate,omitempty"`
}

// ConditionType identifies the condition of the install/uninstall which can be checked with kubectl wait
type ConditionType string

const (
	// InstallStarted is state when an install is in progress.
	InstallStarted ConditionType = "InstallStarted"

	// InstallComplete means the install job has completed its execution successfully
	InstallComplete ConditionType = "InstallComplete"

	// InstallFailed means the install job has failed during execution.
	InstallFailed ConditionType = "InstallFailed"

	// UninstallStarted is state when an uninstall is in progress.
	UninstallStarted ConditionType = "UninstallStarted"

	// UninstallComplete means the uninstall job has completed its execution successfully
	UninstallComplete ConditionType = "UninstallComplete"

	// UninstallFailed means the uninstall job has failed during execution.
	UninstallFailed ConditionType = "UninstallFailed"
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

// VerrazzanoStatus defines the observed state of Verrazzano
type VerrazzanoStatus struct {
	// The latest available observations of an object's current state.
	Conditions []Condition `json:"conditions,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:resource:path=verrazzanos
// +kubebuilder:subresource:status
// +kubebuilder:resource:shortName=vz;vzs
// +kubebuilder:printcolumn:name="Status",type="string",JSONPath=".status.conditions[-1:].type",description="The current status of the install/uninstall"

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

func init() {
	SchemeBuilder.Register(&Verrazzano{}, &VerrazzanoList{})
}
