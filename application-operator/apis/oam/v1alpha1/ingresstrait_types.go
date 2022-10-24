// Copyright (c) 2020, 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package v1alpha1

import (
	"time"

	oamrt "github.com/crossplane/crossplane-runtime/apis/core/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// IngressTraitSpec specifies the desired state of an ingress trait.
type IngressTraitSpec struct {
	// A list of ingress rules to for an ingress trait.
	Rules []IngressRule `json:"rules,omitempty"`

	// The security parameters for an ingress trait.
	// This is required only if specific hosts are given in an [IngressRule](#oam.verrazzano.io/v1alpha1.IngressRule).
	// +optional
	TLS IngressSecurity `json:"tls,omitempty"`

	// The WorkloadReference to the workload to which this trait applies.
	// This value is populated by the OAM runtime when a ApplicationConfiguration
	// resource is processed.  When the ApplicationConfiguration is processed a trait and
	// a workload resource are created from the content of the ApplicationConfiguration.
	// The WorkloadReference is provided in the trait by OAM to ensure the trait controller
	// can find the workload associated with the component containing the trait within the
	// original ApplicationConfiguration.
	WorkloadReference oamrt.TypedReference `json:"workloadRef"`
}

// IngressRule specifies a rule for an ingress trait.
type IngressRule struct {
	// The destination host and port for the ingress paths.
	// +optional
	Destination IngressDestination `json:"destination,omitempty"`
	// One or more hosts exposed by the ingress trait. Wildcard hosts or hosts that are
	// empty are filtered out. If there are no valid hosts provided, then a DNS host name
	// is automatically generated and used.
	// +optional
	Hosts []string `json:"hosts,omitempty"`
	// The paths to be exposed for an ingress trait.
	Paths []IngressPath `json:"paths,omitempty"`
}

// IngressSecurity specifies the secret containing the certificate securing the transport for an ingress trait.
type IngressSecurity struct {
	// The name of a secret containing the certificate securing the transport.  The specification of a secret here
	// implies that a certificate was created for specific hosts, as specified in an [IngressRule](#oam.verrazzano.io/v1alpha1.IngressRule).
	SecretName string `json:"secretName,omitempty"`
}

// IngressPath specifies a specific path to be exposed for an ingress trait.
type IngressPath struct {
	// If no path is provided, it defaults to forward slash (/).
	// +optional
	Path string `json:"path,omitempty"`
	// Path type values are case-sensitive and formatted as follows:
	// <ul><li>`exact`: exact string match</li><li>`prefix`: prefix-based match</li><li>`regex`: regex-based match</li></ul>
	// If the provided ingress path doesn't contain a `pathType`, it defaults to `prefix` if the path is `/` and `exact`
	// otherwise.
	// +optional
	PathType string `json:"pathType,omitempty"`
	// Defines the set of rules for authorizing a request.
	// +optional
	Policy *AuthorizationPolicy `json:"authorizationPolicy,omitempty"`
}

// IngressDestination specifies a specific destination host and port for the ingress paths.
// <div class="alert alert-warning" role="alert">
// <h4 class="alert-heading">NOTE</h4>
// If there are multiple ports defined for a service, then the destination port must be specified OR
// the service port name must have the prefix `http`.
// </div>
type IngressDestination struct {
	// Destination host.
	// +optional
	Host string `json:"host,omitempty"`
	// Session affinity cookie.
	// +optional
	HTTPCookie *IngressDestinationHTTPCookie `json:"httpCookie,omitempty"`
	// Destination port.
	// +optional
	Port uint32 `json:"port,omitempty"`
}

// IngressDestinationHTTPCookie specifies a session affinity cookie for an ingress trait.
type IngressDestinationHTTPCookie struct {
	// The name of the HTTP cookie.
	// +optional
	Name string `json:"name,omitempty"`
	// The name of the HTTP cookie.
	// +optional
	Path string `json:"path,omitempty"`
	// The lifetime of the HTTP cookie (in seconds).
	// +optional
	TTL time.Duration `json:"ttl,omitempty"`
}

// IngressTraitStatus specifies the observed state of an ingress trait and related resources.
type IngressTraitStatus struct {
	// Reconcile status of this ingress trait
	oamrt.ConditionedStatus `json:",inline"`
	// The resources managed by this ingress trait
	Resources []oamrt.TypedReference `json:"resources,omitempty"`
}

// +genclient
// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// IngressTrait specifies the ingress traits API.
type IngressTrait struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec IngressTraitSpec `json:"spec,omitempty"`
	// The observed state of an ingress trait and related resources.
	Status IngressTraitStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// IngressTraitList contains a list of IngressTrait.
type IngressTraitList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []IngressTrait `json:"items"`
}

func init() {
	SchemeBuilder.Register(&IngressTrait{}, &IngressTraitList{})
}
