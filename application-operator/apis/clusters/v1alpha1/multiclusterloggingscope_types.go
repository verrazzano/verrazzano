// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package v1alpha1

import (
	"github.com/verrazzano/verrazzano/application-operator/apis/oam/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// MultiClusterLoggingScopeSpec defines the desired state of MultiClusterLoggingScope
type MultiClusterLoggingScopeSpec struct {
	// The embedded LoggingScope
	Template LoggingScopeTemplate `json:"template"`

	// Clusters in which the secret is to be placed
	Placement Placement `json:"placement"`
}

// LoggingScopeTemplate has the metadata and spec of the underlying LoggingScope
type LoggingScopeTemplate struct {
	// +optional
	Metadata metav1.ObjectMeta         `json:"metadata,omitempty"`
	Spec     v1alpha1.LoggingScopeSpec `json:"spec,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:resource:shortName=mcloggingscope;mcloggingscopes
// +kubebuilder:subresource:status

// MultiClusterLoggingScope is the Schema for the multiclusterloggingscopes API, which will be used
// in the management cluster, to create a LoggingScope targeted at one or more managed clusters
type MultiClusterLoggingScope struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   MultiClusterLoggingScopeSpec `json:"spec,omitempty"`
	Status MultiClusterResourceStatus   `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// MultiClusterLoggingScopeList contains a list of MultiClusterLoggingScope
type MultiClusterLoggingScopeList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []MultiClusterLoggingScope `json:"items"`
}

func init() {
	SchemeBuilder.Register(&MultiClusterLoggingScope{}, &MultiClusterLoggingScopeList{})
}

// GetStatus returns the MultiClusterResourceStatus of this resource
func (in *MultiClusterLoggingScope) GetStatus() MultiClusterResourceStatus {
	return in.Status
}
