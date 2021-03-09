// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package v1alpha1

import (
	"github.com/crossplane/oam-kubernetes-runtime/apis/core/v1alpha2"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// MultiClusterComponentSpec defines the desired state of MultiClusterComponent
type MultiClusterComponentSpec struct {
	// The embedded OAM Component
	Template ComponentTemplate `json:"template"`

	// Clusters in which the secret is to be placed
	Placement Placement `json:"placement"`
}

// ComponentTemplate has the metadata and spec of the underlying OAM component
type ComponentTemplate struct {
	// +kubebuilder:pruning:PreserveUnknownFields
	Metadata metav1.ObjectMeta      `json:"metadata,omitempty"`
	Spec     v1alpha2.ComponentSpec `json:"spec,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:resource:shortName=mccomp;mccomps
// +kubebuilder:subresource:status

// MultiClusterComponent is the Schema for the multiclustercomponents API, which will be used
// in the management cluster, to create an OAM Component targeted at one or more managed clusters
type MultiClusterComponent struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   MultiClusterComponentSpec  `json:"spec,omitempty"`
	Status MultiClusterResourceStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// MultiClusterComponentList contains a list of MultiClusterComponent
type MultiClusterComponentList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []MultiClusterComponent `json:"items"`
}

func init() {
	SchemeBuilder.Register(&MultiClusterComponent{}, &MultiClusterComponentList{})
}

func (in *MultiClusterComponent) GetStatus() MultiClusterResourceStatus {
	return in.Status
}