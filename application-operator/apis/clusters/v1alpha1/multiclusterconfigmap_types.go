// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// MultiClusterConfigMapSpec defines the desired state of MultiClusterConfigMap
type MultiClusterConfigMapSpec struct {
	// The embedded Kubernetes ConfigMap
	Template ConfigMapTemplate `json:"template"`

	// Clusters in which the ConfigMap is to be placed
	Placement Placement `json:"placement"`
}

// ConfigMapTemplate has the metadata and spec of the underlying ConfigMap
// Note that K8S does not define a "ConfigMapSpec" data type, so fields in ConfigMap are copied here
type ConfigMapTemplate struct {
	// +kubebuilder:pruning:PreserveUnknownFields
	Metadata metav1.ObjectMeta `json:"metadata,omitempty"`

	// Immutable corresponds to the Immutable field of K8S corev1.ConfigMap
	Immutable *bool `json:"immutable,omitempty"`

	// Data corresponds to the Data field of K8S corev1.ConfigMap
	Data map[string]string `json:"data,omitempty"`

	// BinaryData corresponds to the BinaryData field of K8S corev1.ConfigMap
	BinaryData map[string][]byte `json:"binaryData,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:resource:shortName=mccm;mccms
// +kubebuilder:subresource:status

// MultiClusterConfigMap is the Schema for the multiclusterconfigmaps API, which will be used in
// the management cluster, to create a Kubernetes ConfigMap targeted at one or more managed clusters
type MultiClusterConfigMap struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   MultiClusterConfigMapSpec  `json:"spec,omitempty"`
	Status MultiClusterResourceStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// MultiClusterConfigMapList contains a list of MultiClusterConfigMap
type MultiClusterConfigMapList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []MultiClusterConfigMap `json:"items"`
}

func init() {
	SchemeBuilder.Register(&MultiClusterConfigMap{}, &MultiClusterConfigMapList{})
}

// GetStatus returns the MultiClusterResourceStatus of this resource
func (in *MultiClusterConfigMap) GetStatus() MultiClusterResourceStatus {
	return in.Status
}
