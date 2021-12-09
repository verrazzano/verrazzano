package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// MetricsTemplateKind is the Kind of the MetricsTemplate
const MetricsTemplateKind string = "MetricsTemplate"

/*func init() {
	SchemeBuilder.Register(&MetricsTemplate{}, &MetricsTemplateList{})
}*/

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
type MetricsTemplate struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec MetricsTemplateSpec `json:"spec,omitempty"`
	//Status MetricsTemplateStatus `json:"status,omitempty"`
}

// MetricsTemplateSpec specifies the desired state of a metrics template.
type MetricsTemplateSpec struct {
	WorkloadSelector *TargetWorkload   `json:"workloadSelector,omitempty"`
	PrometheusConfig *PrometheusConfig `json:"prometheusConfig,omitempty"`
}

type TargetWorkload struct {
	// Selector to match workloads with template
	Selector *metav1.LabelSelector `json:"selector,omitempty"`

	// Priority of the the Template
	Priority *float32 `json:"priority,omitempty"`
}

type PrometheusConfig struct {
	TargetConfigMap      *TargetConfigMap `json:"targetConfigMap,omitempty"`
	ScrapeConfigTemplate *string          `json:"scrapeConfigTemplate,omitempty"`
}

type TargetConfigMap struct {
	Namespace *string `json:"namespace,omitempty"`
	Name      *string `json:"port,omitempty"`
}
