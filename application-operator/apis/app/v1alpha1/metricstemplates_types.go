// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// MetricsTemplateKind is the Kind of the MetricsTemplate
const MetricsTemplateKind string = "MetricsTemplate"

func init() {
	SchemeBuilder.Register(&MetricsTemplate{}, &MetricsTemplateList{})
}

// MetricsTemplateList contains a list of metrics templates
// +kubebuilder:object:root=true
type MetricsTemplateList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []MetricsTemplate `json:"items"`
}

// MetricsTemplate specifies the metrics template API
// +kubebuilder:object:root=true
// +genclient
type MetricsTemplate struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec MetricsTemplateSpec `json:"spec"`
}

// MetricsTemplateSpec specifies the desired state of a metrics template
type MetricsTemplateSpec struct {
	WorkloadSelector WorkloadSelector `json:"workloadSelector,omitempty"`
	PrometheusConfig PrometheusConfig `json:"prometheusConfig,omitempty"`
}

// WorkloadSelector identifies the workloads to which this template applies
type WorkloadSelector struct {
	// NamespaceSelector scopes the template to a namespace
	NamespaceSelector metav1.LabelSelector `json:"namespaceSelector,omitempty"`

	// ObjectSelector scopes the template to a specifically labelled object instance
	ObjectSelector metav1.LabelSelector `json:"objectSelector,omitempty"`

	// APIGroups scopes the template to listed APIGroups
	APIGroups []string `json:"apiGroups,omitempty"`

	// APIVersions scopes the template to listed APIVersions
	APIVersions []string `json:"apiVersions,omitempty"`

	// Resources scopes the template to listed object kind
	Resources []string `json:"resources,omitempty"`
}

// PrometheusConfig refers to the templated metrics scraping configuration
type PrometheusConfig struct {
	TargetConfigMap TargetConfigMap `json:"targetConfigMap"`

	// ScrapeConfigTemplate is a template for the Prometheus scrape job to be added to the Prometheus Configmap
	ScrapeConfigTemplate string `json:"scrapeConfigTemplate"`
}

// TargetConfigMap contains metadata about the Prometheus ConfigMap
type TargetConfigMap struct {
	// Namespace containing the Prometheus ConfigMap
	Namespace string `json:"namespace"`

	// Name of the Prometheus ConfigMap
	Name string `json:"name"`
}
