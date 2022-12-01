// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

type VerrazzanoWebLogicWorkloadTemplate struct {
	APIVersion string               `json:"apiVersion,omitempty"`
	// +kubebuilder:pruning:PreserveUnknownFields
	Metadata   runtime.RawExtension `json:"metadata,omitempty"`

	// +kubebuilder:pruning:PreserveUnknownFields
	Spec runtime.RawExtension `json:"spec"`
}

// VerrazzanoWebLogicWorkloadSpec wraps a WebLogic resource. The WebLogic domain specified
// in the template must contain a spec field and it may include a metadata field.
type VerrazzanoWebLogicWorkloadSpec struct {
	Template VerrazzanoWebLogicWorkloadTemplate   `json:"template"`
	Clusters []VerrazzanoWebLogicWorkloadTemplate `json:"clusters,omitempty"`
}

// VerrazzanoWebLogicWorkloadStatus defines the observed state of VerrazzanoWebLogicWorkload
type VerrazzanoWebLogicWorkloadStatus struct {
	// LastGeneration is the last generation of the spec that was reconciled
	LastGeneration string `json:"lastGeneration,omitempty"`
	// LastRestartVersion is the last value of the verrazzano.io/restart-version annotation
	LastRestartVersion string `json:"lastRestartVersion,omitempty"`
	// LastLifecycleAction is the last value of the verrazzano.io/lifecycle-action
	LastLifecycleAction string `json:"lastLifecycleAction,omitempty"`
}

// +kubebuilder:object:root=true

// VerrazzanoWebLogicWorkload is the Schema for the verrazzanoweblogicworkloads API
// +kubebuilder:subresource:status
type VerrazzanoWebLogicWorkload struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   VerrazzanoWebLogicWorkloadSpec   `json:"spec,omitempty"`
	Status VerrazzanoWebLogicWorkloadStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// VerrazzanoWebLogicWorkloadList contains a list of VerrazzanoWebLogicWorkload
type VerrazzanoWebLogicWorkloadList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []VerrazzanoWebLogicWorkload `json:"items"`
}

func init() {
	SchemeBuilder.Register(&VerrazzanoWebLogicWorkload{}, &VerrazzanoWebLogicWorkloadList{})
}
