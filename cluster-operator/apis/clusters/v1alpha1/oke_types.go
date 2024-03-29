// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +genclient
// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// OKEQuickCreate specifies the API for quick-create OKE clusters.
type OKEQuickCreate struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// The desired state of an OCNEOCIQuickCreate resource.
	Spec OKEQuickCreateSpec `json:"spec,omitempty"`
	// The observed state of an OCNEOCIQuickCreate resource.
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
		// Reference for cloud authentication.
		IdentityRef NamespacedRef `json:"identityRef"`
		// Kubernetes settings.
		Kubernetes `json:"kubernetes"`
		// OKE cluster settings.
		OKE `json:"oke,omitempty"`
	}
	OKE struct {
		CommonOCI `json:",inline"`
		// +optional

		// List of Node pools.
		NodePools []NamedOCINode `json:"nodePools,omitempty"`
		// +optional

		// List of Virtual Node pools.
		VirtualNodePools []VirtualNodePool `json:"virtualNodePools,omitempty"`
		// Network settings for the OKE cluster.
		Network *OKENetwork `json:"network,omitempty"`
	}
	OKENetwork struct {
		// +optional

		// VCN and subnet settings for existing networks.
		Config *Network `json:"config"`
		// CNI Type for cluster networking. May be FLANNEL_OVERLAY or OCI_VCN_IP_NATIVE.
		CNIType CNIType `json:"cniType"`
	}
	VirtualNodePool struct {
		// +patchMergeKey=name
		// +patchStrategy=merge,retainKeys
		Name     string `json:"name" patchStrategy:"merge,retainKeys" patchMergeKey:"name"`
		Replicas *int   `json:"replicas"`
	}
	OKEQuickCreateStatus struct {
		Phase QuickCreatePhase `json:"phase"`
	}
)

type CNIType string

const (
	FlannelOverlay CNIType = "FLANNEL_OVERLAY"
	VCNNative      CNIType = "OCI_VCN_IP_NATIVE"
)
