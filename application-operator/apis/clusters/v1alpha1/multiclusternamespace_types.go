// Copyright (c) 2020, 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package v1alpha1

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// MultiClusterNamespaceSpec defines the desired state of MultiClusterNamespace
type MultiClusterNamespaceSpec struct {
	// The embedded Kubernetes namespace
	namespace corev1.Namespace `json:"namespace"`

	// Clusters in which the namespace is to be placed
	Clusters []Cluster `json:"clusters"`
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

// MultiClusterNamespace is the Schema for the multiclusternamespaces API
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
