// Copyright (c) 2020, 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package v1alpha1

import (
	runtimev1alpha1 "github.com/crossplane/crossplane-runtime/apis/core/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// LoggingScopeKind is the Kind of the LoggingScope
const LoggingScopeKind string = "LoggingScope"

// LoggingScopeSpec defines the desired state of LoggingScope
type LoggingScopeSpec struct {
	// The fluentd image
	FluentdImage string `json:"fluentdImage"`

	// Host for ElasticSearch
	ElasticSearchHost string `json:"elasticSearchHost"`

	// Port for ElasticSearch
	ElasticSearchPort uint32 `json:"elasticSearchPort"`

	// Name of secret with ElasticSearch credentials
	SecretName string `json:"secretName"`

	// WorkloadReferences to the workloads this scope applies to.
	WorkloadReferences []runtimev1alpha1.TypedReference `json:"workloadRefs"`
}

// LoggingScopeStatus defines the observed state of LoggingScope
type LoggingScopeStatus struct {
	// Reconcile status of this scope
	runtimev1alpha1.ConditionedStatus `json:",inline"`

	// Related resources affected by this trait
	Resources []QualifiedResourceRelation `json:"resources,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// LoggingScope is the Schema for the loggingscopes API
type LoggingScope struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   LoggingScopeSpec   `json:"spec,omitempty"`
	Status LoggingScopeStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// LoggingScopeList contains a list of LoggingScope
type LoggingScopeList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []LoggingScope `json:"items"`
}

func init() {
	SchemeBuilder.Register(&LoggingScope{}, &LoggingScopeList{})
}
