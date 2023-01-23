// Copyright (c) 2021, 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const MultiClusterConfigMapKind = "MultiClusterConfigMap"
const MultiClusterConfigMapResource = "multiclusterconfigmaps"

// MultiClusterConfigMapSpec defines the desired state of a MultiCluster ConfigMap.
type MultiClusterConfigMapSpec struct {
	// Clusters in which the ConfigMap is to be created.
	Placement Placement `json:"placement"`

	// The embedded Kubernetes ConfigMap.
	Template ConfigMapTemplate `json:"template"`
}

// ConfigMapTemplate has the metadata and spec of the Kubernetes ConfigMap.
type ConfigMapTemplate struct {
	// Corresponds to the `binaryData` field of the `struct` ConfigMap defined in
	// <a href="https://github.com/kubernetes/api/blob/master/core/v1/types.go">types.go</a>.
	BinaryData map[string][]byte `json:"binaryData,omitempty"`

	// Corresponds to the `data` field of the `struct` ConfigMap defined in
	// <a href="https://github.com/kubernetes/api/blob/master/core/v1/types.go">types.go</a>.
	Data map[string]string `json:"data,omitempty"`

	// Corresponds to the `immutable` field of the `struct` ConfigMap defined in
	// <a href="https://github.com/kubernetes/api/blob/master/core/v1/types.go">types.go</a>.
	Immutable *bool `json:"immutable,omitempty"`

	// Metadata describing the ConfigMap.
	Metadata EmbeddedObjectMeta `json:"metadata,omitempty"`
}

// +genclient
// +kubebuilder:object:root=true
// +kubebuilder:resource:shortName=mccm;mccms
// +kubebuilder:subresource:status
// +kubebuilder:deprecatedversion:warning="clusters.verrazzano.io/v1alpha1 MultiClusterConfigMap is deprecated and will be removed in v2.0.0. See https://verrazzano.io/v1.5/docs/reference/migration/#multicluster."

// MultiClusterConfigMap specifies the MultiCluster ConfigMap API.
type MultiClusterConfigMap struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// The desired state of a MultiCluster ConfigMap resource.
	Spec MultiClusterConfigMapSpec `json:"spec,omitempty"`
	// The observed state of a MultiCluster ConfigMap resource.
	Status MultiClusterResourceStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// MultiClusterConfigMapList contains a list of MultiClusterConfigMap resources.
type MultiClusterConfigMapList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []MultiClusterConfigMap `json:"items"`
}

func init() {
	SchemeBuilder.Register(&MultiClusterConfigMap{}, &MultiClusterConfigMapList{})
}

// GetStatus returns the MultiClusterResourceStatus of this resource.
func (in *MultiClusterConfigMap) GetStatus() MultiClusterResourceStatus {
	return in.Status
}

// GetPlacement returns the Placement of this resource.
func (in *MultiClusterConfigMap) GetPlacement() Placement {
	return in.Spec.Placement
}
