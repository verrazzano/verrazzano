// Copyright (c) 2021, 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package v1alpha1

import (
	"github.com/crossplane/oam-kubernetes-runtime/apis/core/v1alpha2"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const MultiClusterComponentKind = "MultiClusterComponent"
const MultiClusterComponentResource = "multiclustercomponents"

// MultiClusterComponentSpec defines the desired state of a Multi Cluster Component.
type MultiClusterComponentSpec struct {
	// Clusters in which the component is to be created.
	Placement Placement `json:"placement"`

	// Template containing the metadata and spec for an OAM component.
	Template ComponentTemplate `json:"template"`
}

// ComponentTemplate has the metadata and embedded spec of the OAM component.
type ComponentTemplate struct {
	// Metadata describing the component.
	Metadata EmbeddedObjectMeta `json:"metadata,omitempty"`

	// The embedded OAM component spec.
	Spec v1alpha2.ComponentSpec `json:"spec,omitempty"`
}

// +genclient
// +kubebuilder:object:root=true
// +kubebuilder:resource:shortName=mccomp;mccomps
// +kubebuilder:subresource:status

// MultiClusterComponent specifies the Multi Cluster Component API.
type MultiClusterComponent struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// The desired state of a Multi Cluster Component resource.
	Spec MultiClusterComponentSpec `json:"spec,omitempty"`
	// The observed state of a Multi Cluster Component resource.
	Status MultiClusterResourceStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// MultiClusterComponentList contains a list of MultiClusterComponent resources.
type MultiClusterComponentList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []MultiClusterComponent `json:"items"`
}

func init() {
	SchemeBuilder.Register(&MultiClusterComponent{}, &MultiClusterComponentList{})
}

// GetStatus returns the MultiClusterResourceStatus of this resource.
func (in *MultiClusterComponent) GetStatus() MultiClusterResourceStatus {
	return in.Status
}

// GetPlacement returns the Placement of this resource.
func (in *MultiClusterComponent) GetPlacement() Placement {
	return in.Spec.Placement
}
