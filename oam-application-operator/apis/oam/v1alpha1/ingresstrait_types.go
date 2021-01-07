// Copyright (c) 2020, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package v1alpha1

import (
	oamrt "github.com/crossplane/crossplane-runtime/apis/core/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// IngressTraitSpec defines the desired state of IngressTrait
type IngressTraitSpec struct {
	// A set of rules to configure the IngressTrait.
	Rules []IngressRule `json:"rules,omitempty"`

	// WorkloadReference to the workload this trait applies to.
	// This value is populated by the OAM runtime when a ApplicationConfiguration
	// resource is processed.  When the ApplicationConfiguration is processed a trait and
	// a workload resource are created from the content of the ApplicationConfiguration.
	// The WorkloadReference is provided in the trait by OAM to ensure the trait controller
	// can find the workload associated with the component containing the trait within the
	// original ApplicationConfiguration.
	WorkloadReference oamrt.TypedReference `json:"workloadRef"`
}

// IngressRule defines hosts and paths to be exposed for ingress
type IngressRule struct {
	Hosts []string      `json:"hosts,omitempty"`
	Paths []IngressPath `json:"paths,omitempty"`
}

// IngressPath defines a specific path to be exposed for ingress
type IngressPath struct {
	Path     string `json:"path,omitempty"`
	PathType string `json:"pathType,omitempty"`
}

// IngressTraitStatus defines the observed state of IngressTrait
type IngressTraitStatus struct {
	// Reconcile status of this trait
	oamrt.ConditionedStatus `json:",inline"`
	// Resources managed by this trait
	Resources []oamrt.TypedReference `json:"resources,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// IngressTrait is the Schema for the ingresstraits API
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
