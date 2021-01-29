// Copyright (c) 2020, 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

// VerrazzanoCoherenceWorkloadSpec wraps a Coherence resource. The Coherence object must include apiVersion,
// kind, and spec fields. It may include a metadata field.
type VerrazzanoCoherenceWorkloadSpec struct {
	// +kubebuilder:validation:EmbeddedResource
	// +kubebuilder:pruning:PreserveUnknownFields
	Coherence runtime.RawExtension `json:"coherence"`
}

// VerrazzanoCoherenceWorkloadStatus defines the observed state of VerrazzanoCoherenceWorkload
type VerrazzanoCoherenceWorkloadStatus struct {
}

// +kubebuilder:object:root=true

// VerrazzanoCoherenceWorkload is the Schema for the verrazzanocoherenceworkloads API
type VerrazzanoCoherenceWorkload struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   VerrazzanoCoherenceWorkloadSpec   `json:"spec,omitempty"`
	Status VerrazzanoCoherenceWorkloadStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// VerrazzanoCoherenceWorkloadList contains a list of VerrazzanoCoherenceWorkload
type VerrazzanoCoherenceWorkloadList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []VerrazzanoCoherenceWorkload `json:"items"`
}

func init() {
	SchemeBuilder.Register(&VerrazzanoCoherenceWorkload{}, &VerrazzanoCoherenceWorkloadList{})
}
