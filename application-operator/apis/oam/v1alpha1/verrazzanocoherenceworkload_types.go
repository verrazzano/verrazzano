// Copyright (c) 2020, 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// VerrazzanoCoherenceWorkloadSpec defines the desired state of VerrazzanoCoherenceWorkload
type VerrazzanoCoherenceWorkloadSpec struct {
	// +kubebuilder:validation:EmbeddedResource
	// +kubebuilder:pruning:PreserveUnknownFields
	Coherence runtime.RawExtension `json:"coherence"`
}

// VerrazzanoCoherenceWorkloadStatus defines the observed state of VerrazzanoCoherenceWorkload
type VerrazzanoCoherenceWorkloadStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
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
