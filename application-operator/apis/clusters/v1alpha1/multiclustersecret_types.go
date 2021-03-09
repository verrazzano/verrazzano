// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package v1alpha1

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// MultiClusterSecretSpec defines the desired state of MultiClusterSecret
type MultiClusterSecretSpec struct {
	// The embedded Kubernetes secret
	Template SecretTemplate `json:"template"`

	// Clusters in which the secret is to be placed
	Placement Placement `json:"placement"`
}

// SecretTemplate has the metadata and spec of the underlying secret
// Note that K8S does not define a "SecretSpec" data type, so the 3 fields in Secret are copied here
type SecretTemplate struct {
	// +kubebuilder:pruning:PreserveUnknownFields
	Metadata metav1.ObjectMeta `json:"metadata,omitempty"`

	// Data corresponds to the Data field of K8S corev1.Secret
	Data map[string][]byte `json:"data,omitempty"`

	// StringData corresponds to the StringData field of K8S corev1.Secret
	StringData map[string]string `json:"stringData,omitempty"`

	// Type corresponds to the Type field of K8S corev1.Secret
	Type corev1.SecretType `json:"type,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:resource:shortName=mcsecret;mcsecrets
// +kubebuilder:subresource:status

// MultiClusterSecret is the Schema for the multiclustersecrets API, which will be used by a user
// in the management cluster, to create a Kubernetes secret targeted at one or more managed clusters
type MultiClusterSecret struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   MultiClusterSecretSpec     `json:"spec,omitempty"`
	Status MultiClusterResourceStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// MultiClusterSecretList contains a list of MultiClusterSecret
type MultiClusterSecretList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []MultiClusterSecret `json:"items"`
}

func init() {
	SchemeBuilder.Register(&MultiClusterSecret{}, &MultiClusterSecretList{})
}

func (in *MultiClusterSecret) GetStatus() MultiClusterResourceStatus {
	return in.Status
}
