// Copyright (c) 2020, 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

// VerrazzanoCoherenceWorkloadSpec wraps a Coherence resource.
type VerrazzanoCoherenceWorkloadSpec struct {
	// The metadata and spec for the underlying
	// <a href="https://oracle.github.io/coherence-operator/docs/3.2.8/#/docs/about/04_coherence_spec">Coherence</a> resource.
	// +kubebuilder:pruning:PreserveUnknownFields
	Template runtime.RawExtension `json:"template"`
}

// VerrazzanoCoherenceWorkloadStatus defines the observed state of a Verrazzano Coherence workload.
type VerrazzanoCoherenceWorkloadStatus struct {
	// The last generation of the Verrazzano Coherence workload that was reconciled.
	LastGeneration string `json:"lastGeneration,omitempty"`
	// The last value of the verrazzano.io/restart-version annotation.
	LastRestartVersion string `json:"lastRestartVersion,omitempty"`
}

// +genclient
// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// VerrazzanoCoherenceWorkload specifies the Verrazzano Coherence workload API.
type VerrazzanoCoherenceWorkload struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// The desired state of a Verrazzano Coherence workload.
	Spec   VerrazzanoCoherenceWorkloadSpec   `json:"spec,omitempty"`
	Status VerrazzanoCoherenceWorkloadStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// VerrazzanoCoherenceWorkloadList contains a list of VerrazzanoCoherenceWorkload resources.
type VerrazzanoCoherenceWorkloadList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []VerrazzanoCoherenceWorkload `json:"items"`
}

func init() {
	SchemeBuilder.Register(&VerrazzanoCoherenceWorkload{}, &VerrazzanoCoherenceWorkloadList{})
}
