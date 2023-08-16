// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +genclient
// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// OKEQuickCreate specifies the API for quick-create OKE Clusters.
type OKEQuickCreate struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// The desired state of an OCIOCNECluster resource.
	Spec OKEQuickCreateSpec `json:"spec,omitempty"`
	// The observed state of an OCIOCNECluster resource.
	Status OKEQuickCreateStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// OKEQuickCreateList contains a list of OKEQuickCreate resources.
type OKEQuickCreateList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []OKEQuickCreate `json:"items"`
}

type (
	OKEQuickCreateSpec struct {
		CommonClusterSpec `json:",inline"`
		OKESpec           `json:"oke"`
	}
	OKESpec struct {
		CommonOCISpec    CommonOCISpec     `json:",inline"`
		NodePools        []NodeConfig      `json:"nodePools,omitempty"`
		VirtualNodePools []VirtualNodePool `json:"virtualNodePools,omitempty"`
		Network          Network           `json:"network"`
	}
	VirtualNodePool struct {
		// +patchMergeKey=name
		// +patchStrategy=merge,retainKeys
		Name string `json:"name" patchStrategy:"merge,retainKeys" patchMergeKey:"version"`
	}
	OKEQuickCreateStatus struct {
	}
)
