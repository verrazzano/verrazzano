// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package v1alpha1

import (
	"github.com/crossplane/oam-kubernetes-runtime/apis/core/v1alpha2"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

const MultiClusterAppConfigKind = "MultiClusterApplicationConfiguration"
const MultiClusterAppConfigResource = "multiclusterapplicationconfigurations"

// MultiClusterApplicationConfigurationSpec defines the desired state of MultiClusterApplicationConfiguration
type MultiClusterApplicationConfigurationSpec struct {
	// The embedded OAM ApplicationConfiguration
	Template ApplicationConfigurationTemplate `json:"template"`

	// Clusters in which the secret is to be placed
	Placement Placement `json:"placement"`
}

// ApplicationConfigurationTemplate has the metadata and spec of the underlying
// OAM ApplicationConfiguration
type ApplicationConfigurationTemplate struct {
	// +optional
	Metadata metav1.ObjectMeta                     `json:"metadata,omitempty"`
	Spec     v1alpha2.ApplicationConfigurationSpec `json:"spec,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:resource:shortName=mcappconf;mcappconfs
// +kubebuilder:subresource:status

// MultiClusterApplicationConfiguration is the Schema for the multiclusterapplicationconfigurations API
type MultiClusterApplicationConfiguration struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   MultiClusterApplicationConfigurationSpec `json:"spec,omitempty"`
	Status MultiClusterResourceStatus               `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// MultiClusterApplicationConfigurationList contains a list of MultiClusterApplicationConfiguration
type MultiClusterApplicationConfigurationList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []MultiClusterApplicationConfiguration `json:"items"`
}

func init() {
	SchemeBuilder.Register(&MultiClusterApplicationConfiguration{}, &MultiClusterApplicationConfigurationList{})
}

// GetStatus returns the MultiClusterResourceStatus of this resource
func (in *MultiClusterApplicationConfiguration) GetStatus() MultiClusterResourceStatus {
	return in.Status
}

// GetItems returns the list of MultiClusterComponents
func (in *MultiClusterApplicationConfigurationList) GetItems() []runtime.Object {
	objects := []runtime.Object{}
	for _, item := range in.Items {
		objects = append(objects, item.DeepCopyObject())
	}
	return objects
}
