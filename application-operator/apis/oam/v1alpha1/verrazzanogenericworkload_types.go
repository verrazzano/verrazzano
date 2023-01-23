// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// VerrazzanoGenericWorkloadSpec wraps a Helidon application deployment.
type VerrazzanoGenericWorkloadSpec VerrazzanoHelidonWorkloadSpec

// VerrazzanoGenericWorkloadStatus defines the observed state of Verrazzano Helidon workload.
type VerrazzanoGenericWorkloadStatus VerrazzanoHelidonWorkloadStatus

// VerrazzanoGenericWorkload specifies the Verrazzano Helidon workload API.
type VerrazzanoGenericWorkload struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	// The desired state of a Verrazzano Helidon workload.
	// +kubebuilder:validation:Required
	Spec VerrazzanoGenericWorkloadSpec `json:"spec"`
	// The observed state of a Verrazzano Generic workload.
	Status VerrazzanoGenericWorkloadStatus `json:"status,omitempty"`
}

// VerrazzanoGenericWorkloadList contains a list of Verrazzano Helidon workload resources.
// +kubebuilder:object:root=true
type VerrazzanoGenericWorkloadList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []VerrazzanoGenericWorkload `json:"items"`
}

func init() {
	SchemeBuilder.Register(&VerrazzanoGenericWorkload{}, &VerrazzanoGenericWorkloadList{})
}
