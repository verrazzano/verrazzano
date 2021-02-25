// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// VerrazzanoProjectSpec defines the desired state of VerrazzanoProject - a VerrazzanoProject
// contains a list of Kubernetes namespaces which are part of the project
type VerrazzanoProjectSpec struct {
	Namespaces []string `json:"namespaces,omitempty"`
}

// VerrazzanoProjectStatus defines the observed state of VerrazzanoProject
type VerrazzanoProjectStatus struct {
	// The latest available observations of an object's current state.
	Conditions []Condition `json:"conditions,omitempty"`

	// State of the MultiClusterSecret custom resource
	State StateType `json:"state,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:resource:shortName=vp;vps
// +kubebuilder:subresource:status

// VerrazzanoProject is the Schema for the verrazzanoprojects API
type VerrazzanoProject struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   VerrazzanoProjectSpec   `json:"spec,omitempty"`
	Status VerrazzanoProjectStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// VerrazzanoProjectList contains a list of VerrazzanoProject
type VerrazzanoProjectList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []VerrazzanoProject `json:"items"`
}

func init() {
	SchemeBuilder.Register(&VerrazzanoProject{}, &VerrazzanoProjectList{})
}
