// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package v1alpha1

import (
	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// VerrazzanoHelidonWorkloadSpec wraps meta/ObjectMeta & apps/DeploymentSpec.
type VerrazzanoHelidonWorkloadSpec struct {
	// The embedded apps/Deployment
	DeploymentTemplate DeploymentTemplate `json:"deploymentTemplate"`
}

// DeploymentTemplate should have the metadata and spec of the underlying apps/Deployment
type DeploymentTemplate struct {
	// +kubebuilder:pruning:PreserveUnknownFields
	Metadata metav1.ObjectMeta     `json:"metadata,omitempty"`
	Spec     appsv1.DeploymentSpec `json:"spec,omitempty"`
}

// VerrazzanoHelidonWorkloadStatus defines the observed state of VerrazzanoHelidonWorkload
type VerrazzanoHelidonWorkloadStatus struct {
}

// +kubebuilder:object:root=true

// VerrazzanoHelidonWorkload is the Schema for the verrazzanohelidonworkloads API
type VerrazzanoHelidonWorkload struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   VerrazzanoHelidonWorkloadSpec   `json:"spec,omitempty"`
	Status VerrazzanoHelidonWorkloadStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// VerrazzanoHelidonWorkloadList contains a list of VerrazzanoHelidonWorkload
type VerrazzanoHelidonWorkloadList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []VerrazzanoHelidonWorkload `json:"items"`
}

func init() {
	SchemeBuilder.Register(&VerrazzanoHelidonWorkload{}, &VerrazzanoHelidonWorkloadList{})
}
