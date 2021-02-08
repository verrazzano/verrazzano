// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package v1alpha1

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// MultiClusterNamespaceSpec defines the desired state of MultiClusterNamespace
type MultiClusterNamespaceSpec struct {
	// The embedded Kubernetes namespace
	Template NamespaceTemplate `json:"template"`

	// Clusters in which the namespace is to be placed
	Placement Placement `json:"placement"`
}

// NamespaceTemplate has the metadata and spec of the underlying namespace
type NamespaceTemplate struct {
	// +kubebuilder:pruning:PreserveUnknownFields
	Metadata metav1.ObjectMeta    `json:"metadata,omitempty"`
	Spec     corev1.NamespaceSpec `json:"spec,omitempty"`
}

// MultiClusterNamespaceStatus defines the observed state of MultiClusterNamespace
type MultiClusterNamespaceStatus struct {
	// The latest available observations of an object's current state.
	Conditions []Condition `json:"conditions,omitempty"`

	// State of the MultiClusterNamespace custom resource
	State StateType `json:"state,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:resource:shortName=mcns
// +kubebuilder:subresource:status

// MultiClusterNamespace is the Schema for the multiclusternamespaces API, which will be used
// by a user in the management cluster, to create a Kubernetes namespace targeted at one or more
// managed clusters
type MultiClusterNamespace struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   MultiClusterNamespaceSpec   `json:"spec,omitempty"`
	Status MultiClusterNamespaceStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// MultiClusterNamespaceList contains a list of MultiClusterNamespace
type MultiClusterNamespaceList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []MultiClusterNamespace `json:"items"`
}

func init() {
	SchemeBuilder.Register(&MultiClusterNamespace{}, &MultiClusterNamespaceList{})
}
