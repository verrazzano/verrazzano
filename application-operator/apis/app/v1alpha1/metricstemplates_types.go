// Copyright (c) 2021, 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// MetricsTemplateKind identifies the Kind for the MetricsTemplate.
const MetricsTemplateKind string = "MetricsTemplate"

func init() {
	SchemeBuilder.Register(&MetricsTemplate{}, &MetricsTemplateList{})
}

// MetricsTemplateList contains a list of metrics template resources.
// +kubebuilder:object:root=true
type MetricsTemplateList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []MetricsTemplate `json:"items"`
}

// +genclient
// +kubebuilder:object:root=true
// MetricsTemplate specifies the metrics template API.
type MetricsTemplate struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec MetricsTemplateSpec `json:"spec"`
}

// MetricsTemplateSpec specifies the desired state of a metrics template.
type MetricsTemplateSpec struct {
	// Prometheus configuration details.
	// +optional
	PrometheusConfig PrometheusConfig `json:"prometheusConfig,omitempty"`
	// Selector for target workloads.
	// +optional
	WorkloadSelector WorkloadSelector `json:"workloadSelector,omitempty"`
}

// WorkloadSelector identifies the workloads to which a template applies.
type WorkloadSelector struct {
	// Scopes the template to given API Groups.
	// +optional
	APIGroups []string `json:"apiGroups,omitempty"`

	// Scopes the template to given API Versions.
	// +optional
	APIVersions []string `json:"apiVersions,omitempty"`

	// Scopes the template to a namespace.
	// +optional
	NamespaceSelector metav1.LabelSelector `json:"namespaceSelector,omitempty"`

	// Scopes the template to a specifically labelled object instance.
	// +optional
	ObjectSelector metav1.LabelSelector `json:"objectSelector,omitempty"`

	// Scopes the template to given API Resources.
	// +optional
	Resources []string `json:"resources,omitempty"`
}

// PrometheusConfig refers to the templated metrics scraping configuration.
type PrometheusConfig struct {
	// Scrape configuration template to be added to the Prometheus configuration.
	ScrapeConfigTemplate string `json:"scrapeConfigTemplate"`

	// Identity of the ConfigMap to be updated with the scrape configuration specified in `scrapeConfigTemplate`.
	TargetConfigMap TargetConfigMap `json:"targetConfigMap"`
}

// TargetConfigMap contains metadata about the Prometheus ConfigMap.
type TargetConfigMap struct {
	// Name of the ConfigMap to be updated with the scrape target configuration.
	Name string `json:"name"`

	// Namespace of the ConfigMap to be updated with the scrape target configuration.
	Namespace string `json:"namespace"`
}
