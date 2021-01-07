// Copyright (c) 2020, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package v1alpha1

import (
	oamrt "github.com/crossplane/crossplane-runtime/apis/core/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// MetricsTraitKind is the Kind of the MetricsTrait
const MetricsTraitKind string = "MetricsTrait"

func init() {
	SchemeBuilder.Register(&MetricsTrait{}, &MetricsTraitList{})
}

// MetricsTraitList contains a list of MetricsTrait
// +kubebuilder:object:root=true
type MetricsTraitList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []MetricsTrait `json:"items"`
}

// MetricsTrait is the Schema for the metricstraits API
// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
type MetricsTrait struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   MetricsTraitSpec   `json:"spec,omitempty"`
	Status MetricsTraitStatus `json:"status,omitempty"`
}

// MetricsTraitSpec defines the desired state of the MetricsTrait
type MetricsTraitSpec struct {
	// The HTTP port for the related metrics endpoint. Defaults to 8080.
	Port *int `json:"port,omitempty"`

	// The HTTP path for the related metrics endpoint. Defaults to /metrics.
	Path *string `json:"path,omitempty"`

	// The name of an opaque secret (i.e. username and password) within the workload's namespace for metrics endpoint access.
	Secret *string `json:"secret,omitempty"`

	// The prometheus deployment used to scrape the related metrics endpoints.
	// Defaults to istio-system/prometheus
	Scraper *string `json:"scraper,omitempty"`

	// A reference to the workload used to generate this metrics trait.
	WorkloadReference oamrt.TypedReference `json:"workloadRef"`
}

// MetricsTraitStatus defines the observed state of MetricsTrait
type MetricsTraitStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// Reconcile status of this trait
	oamrt.ConditionedStatus `json:",inline"`

	// Related resources affected by this trait
	Resources []QualifiedResourceRelation `json:"resources,omitempty"`
}
