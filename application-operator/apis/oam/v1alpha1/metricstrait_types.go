// Copyright (c) 2020, 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package v1alpha1

import (
	oamrt "github.com/crossplane/crossplane-runtime/apis/core/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// MetricsTraitKind identifies the Kind for the metrics trait.
const MetricsTraitKind string = "MetricsTrait"

func init() {
	SchemeBuilder.Register(&MetricsTrait{}, &MetricsTraitList{})
}

// MetricsTraitList contains a list of MetricsTrait.
// +kubebuilder:object:root=true
type MetricsTraitList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []MetricsTrait `json:"items"`
}

// +genclient
// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// MetricsTrait specifies the metrics trait API.
type MetricsTrait struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec MetricsTraitSpec `json:"spec,omitempty"`
	// The observed state of a metrics trait and related resources.
	Status MetricsTraitStatus `json:"status,omitempty"`
}

// MetricsTraitSpec specifies the desired state of a metrics trait.
type MetricsTraitSpec struct {
	// Specifies whether metrics collection is enabled. Defaults to `true`.
	//+optional
	Enabled *bool `json:"enabled,omitempty"`

	// The HTTP path for the related metrics endpoint. Defaults to `/metrics`.
	// +optional
	Path *string `json:"path,omitempty"`

	// The HTTP port for the related metrics trait. Defaults to `8080`.
	// +optional
	Port *int `json:"port,omitempty"`

	// The HTTP ports for the related metrics trait. Defaults to `8080`.
	// +optional
	Ports []PortSpec `json:"ports,omitempty"`

	// The Prometheus deployment used to scrape the related metrics endpoints. By default, the Verrazzano-supplied
	// Prometheus component is used to scrape the endpoint.
	// +optional
	Scraper *string `json:"scraper,omitempty"`

	// The name of an opaque secret (for example, username and password) within the workload’s namespace for metrics
	// endpoint access.
	// +optional
	Secret *string `json:"secret,omitempty"`

	// The WorkloadReference of the workload to which this trait applies.
	// This value is populated by the OAM runtime when an ApplicationConfiguration
	// resource is processed.  When the ApplicationConfiguration is processed, a trait and
	// a workload resource are created from the content of the ApplicationConfiguration.
	// The WorkloadReference is provided in the trait by OAM to ensure that the trait controller
	// can find the workload associated with the component containing the trait within the
	// original ApplicationConfiguration.
	WorkloadReference oamrt.TypedReference `json:"workloadRef"`
}

// PortSpec defines an HTTP port and path combination
type PortSpec struct {
	// The HTTP path for the related metrics endpoint. Defaults to `/metrics`.
	// +optional
	Path *string `json:"path,omitempty"`

	// The HTTP port for the related metrics trait. Defaults to `8080`.
	// +optional
	Port *int `json:"port,omitempty"`
}

// MetricsTraitStatus defines the observed state of a metrics trait and related resources.
type MetricsTraitStatus struct {
	// Reconcile status of this metrics trait.
	oamrt.ConditionedStatus `json:",inline"`

	// Related resources affected by this metrics trait.
	Resources []QualifiedResourceRelation `json:"resources,omitempty"`
}
