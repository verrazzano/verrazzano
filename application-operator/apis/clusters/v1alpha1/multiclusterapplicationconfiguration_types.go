// Copyright (c) 2021, 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package v1alpha1

import (
	"github.com/crossplane/oam-kubernetes-runtime/apis/core/v1alpha2"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const MultiClusterAppConfigKind = "MultiClusterApplicationConfiguration"
const MultiClusterAppConfigResource = "multiclusterapplicationconfigurations"

// MultiClusterApplicationConfigurationSpec defines the desired state of a multicluster application.
type MultiClusterApplicationConfigurationSpec struct {
	// Clusters in which the application is to be created.
	Placement Placement `json:"placement"`

	// List of secrets used by the application. These secrets must be created in the application’s namespace before
	// deploying a MultiClusterApplicationConfiguration resource.
	// +optional
	Secrets []string `json:"secrets,omitempty"`

	// Template containing the metadata and spec for an OAM applicationConfiguration resource.
	Template ApplicationConfigurationTemplate `json:"template"`
}

// ApplicationConfigurationTemplate has the metadata and embedded spec of the OAM applicationConfiguration resource.
type ApplicationConfigurationTemplate struct {
	// Metadata describing the application.
	Metadata EmbeddedObjectMeta `json:"metadata,omitempty"`
	// The embedded OAM application specification.
	Spec v1alpha2.ApplicationConfigurationSpec `json:"spec,omitempty"`
}

// +genclient
// +kubebuilder:object:root=true
// +kubebuilder:resource:shortName=mcappconf;mcappconfs
// +kubebuilder:subresource:status

// MultiClusterApplicationConfiguration specifies the multicluster application API.
type MultiClusterApplicationConfiguration struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// The desired state of a multicluster application resource.
	Spec MultiClusterApplicationConfigurationSpec `json:"spec,omitempty"`
	// The observed state of a multicluster application resource.
	Status MultiClusterResourceStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// MultiClusterApplicationConfigurationList contains a list of MultiClusterApplicationConfiguration resources.
type MultiClusterApplicationConfigurationList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []MultiClusterApplicationConfiguration `json:"items"`
}

func init() {
	SchemeBuilder.Register(&MultiClusterApplicationConfiguration{}, &MultiClusterApplicationConfigurationList{})
}

// GetStatus returns the MultiClusterResourceStatus of this resource.
func (in *MultiClusterApplicationConfiguration) GetStatus() MultiClusterResourceStatus {
	return in.Status
}

// GetPlacement returns the Placement of this resource.
func (in *MultiClusterApplicationConfiguration) GetPlacement() Placement {
	return in.Spec.Placement
}
