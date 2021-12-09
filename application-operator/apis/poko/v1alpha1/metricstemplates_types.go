package v1alpha1

import (
	oamrt "github.com/crossplane/crossplane-runtime/apis/core/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// MetricsTemplateKind is the Kind of the MetricsTemplate
const MetricsTemplateKind string = "MetricsTemplate"

func init() {
	SchemeBuilder.Register(&MetricsTemplate{}, &MetricsTemplateList{})
}

// MetricsTemplateList contains a list of metrics templates.
// +kubebuilder:object:root=true
type MetricsTemplateList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []MetricsTemplate `json:"items"`
}

// MetricsTemplate specifies the metrics template API
// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +genclient
type MetricsTemplate struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   MetricsTemplateSpec   `json:"spec,omitempty"`
	Status MetricsTemplateStatus `json:"status,omitempty"`
}

// MetricsTemplateSpec specifies the desired state of a metrics template.
type MetricsTemplateSpec struct {
	WorkloadSelector TargetWorkload   `json:"workloadSelector,omitempty"`
	PrometheusConfig PrometheusConfig `json:"prometheusConfig,omitempty"`
}

// MetricsTemplateStatus defines the observed state of MetricsTemplate and related resources.
type MetricsTemplateStatus struct {
	// Important: Run code generation after modifying this file.

	// The reconcile status of this metrics template
	oamrt.ConditionedStatus `json:",inline"`

	// Related resources affected by this metrics template
	Resources []QualifiedResourceRelation `json:"resources,omitempty"`
}

// TargetWorkload identifies the workloads to which this template applies.
type TargetWorkload struct {
	// The label selector used to match the workload
	Selector metav1.LabelSelector `json:"selector,omitempty"`

	// Allows for explicit control of template selection.
	// Workloads with highest priority selected
	Priority int `json:"priority,omitempty"`
}

// PrometheusConfig refers to the templated metrics scraping configuration
type PrometheusConfig struct {
	TargetConfigMap TargetConfigMap `json:"targetConfigMap,omitempty"`

	// ScrapeConfigTemplate is the prometheus scrape target template to be added to the Prometheus Configmap
	ScrapeConfigTemplate string `json:"scrapeConfigTemplate,omitempty"`
}

// TargetConfigMap contains metadata about the Prometheus ConfigMap
type TargetConfigMap struct {
	// Namespace containing the Prometheus ConfigMap
	Namespace string `json:"namespace,omitempty"`

	// Name of the Prometheus ConfigMap
	Name string `json:"port,omitempty"`
}
