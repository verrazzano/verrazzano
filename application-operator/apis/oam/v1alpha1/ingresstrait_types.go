// Copyright (c) 2020, 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package v1alpha1

import (
	oamrt "github.com/crossplane/crossplane-runtime/apis/core/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// IngressTraitSpec specifies the desired state of an ingress trait.
type IngressTraitSpec struct {
	// Rules specifies a list of ingress rules to for an ingress trait.
	Rules []IngressRule `json:"rules,omitempty"`

	// The WorkloadReference to the workload to which this trait applies.
	// This value is populated by the OAM runtime when a ApplicationConfiguration
	// resource is processed.  When the ApplicationConfiguration is processed a trait and
	// a workload resource are created from the content of the ApplicationConfiguration.
	// The WorkloadReference is provided in the trait by OAM to ensure the trait controller
	// can find the workload associated with the component containing the trait within the
	// original ApplicationConfiguration.
	WorkloadReference oamrt.TypedReference `json:"workloadRef"`
}

// IngressRule specifies the hosts and paths to be exposed for an ingress trait.
type IngressRule struct {
	Hosts []string      `json:"hosts,omitempty"`
	Paths []IngressPath `json:"paths,omitempty"`
}

// IngressPath specifies a specific path to be exposed for an ingress trait.
type IngressPath struct {
	Path     string `json:"path,omitempty"`
	PathType string `json:"pathType,omitempty"`
}

// IngressTraitStatus specifies the observed state of an ingress trait and related resources.
type IngressTraitStatus struct {
	// The reconcile status of this ingress trait
	oamrt.ConditionedStatus `json:",inline"`
	// The resources managed by this ingress trait
	Resources []oamrt.TypedReference `json:"resources,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// IngressTrait specifies the ingress traits API
type IngressTrait struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   IngressTraitSpec   `json:"spec,omitempty"`
	Status IngressTraitStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// IngressTraitList contains a list of IngressTrait
type IngressTraitList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []IngressTrait `json:"items"`
}

func init() {
	SchemeBuilder.Register(&IngressTrait{}, &IngressTraitList{})
}
