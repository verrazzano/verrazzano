// Copyright (c) 2021, 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package v1alpha1

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const MultiClusterSecretKind = "MultiClusterSecret"
const MultiClusterSecretResource = "multiclustersecrets"

// MultiClusterSecretSpec defines the desired state of a Multi Cluster Secret.
type MultiClusterSecretSpec struct {
	// Clusters in which the secret is to be created.
	Placement Placement `json:"placement"`

	// The embedded Kubernetes secret.
	Template SecretTemplate `json:"template"`
}

// SecretTemplate has the metadata and spec of the Kubernetes Secret.
type SecretTemplate struct {
	// Corresponds to the data field of the struct Secret defined in
	// <a href="https://github.com/kubernetes/api/blob/master/core/v1/types.go">types.go</a>.
	Data map[string][]byte `json:"data,omitempty"`

	// Metadata describing the secret.
	Metadata EmbeddedObjectMeta `json:"metadata,omitempty"`

	// Corresponds to the `stringData` field of the `struct` Secret defined in
	// <a href="https://github.com/kubernetes/api/blob/master/core/v1/types.go">types.go</a>.
	StringData map[string]string `json:"stringData,omitempty"`

	// The type of secret.
	Type corev1.SecretType `json:"type,omitempty"`
}

// +genclient
// +kubebuilder:object:root=true
// +kubebuilder:resource:shortName=mcsecret;mcsecrets
// +kubebuilder:subresource:status

// MultiClusterSecret specifies the Multi Cluster Secret API.
type MultiClusterSecret struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// The desired state of a Multi Cluster Secret resource.
	Spec MultiClusterSecretSpec `json:"spec,omitempty"`
	// The observed state of a Multi Cluster Secret resource.
	Status MultiClusterResourceStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// MultiClusterSecretList contains a list of MultiClusterSecret resources.
type MultiClusterSecretList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []MultiClusterSecret `json:"items"`
}

func init() {
	SchemeBuilder.Register(&MultiClusterSecret{}, &MultiClusterSecretList{})
}

// GetStatus returns the MultiClusterResourceStatus of this resource.
func (in *MultiClusterSecret) GetStatus() MultiClusterResourceStatus {
	return in.Status
}

// GetPlacement returns the Placement of this resource.
func (in *MultiClusterSecret) GetPlacement() Placement {
	return in.Spec.Placement
}
