// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +genclient
// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// OCNEOCIQuickCreate specifies the API for quick-create OCI OCNE Clusters.
type OCNEOCIQuickCreate struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// The desired state of an OCNEOCIQuickCreate resource.
	Spec OCIOCNEClusterSpec `json:"spec,omitempty"`
	// The observed state of an OCNEOCIQuickCreate resource.
	Status OCNEOCIQuickCreateStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// OCNEOCIQuickCreateList contains a list of OCNEOCIQuickCreate resources.
type OCNEOCIQuickCreateList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []OCNEOCIQuickCreate `json:"items"`
}

type (
	OCIOCNEClusterSpec struct {
		CommonClusterSpec `json:",inline"`
		// OCNE settings.
		OCNE OCNE `json:"ocne"`
		// OCI infrastructure settings.
		OCI OCI `json:"oci"`
	}
	OCI struct {
		CommonOCI `json:",inline"`
		// Control Plane node settings.
		ControlPlane NodeConfig `json:"controlPlane"`
		// List of worker nodes.
		Workers []NodeConfig `json:"workers,omitempty"`
		// OCI Network settings.
		Network *Network `json:"network"`
	}
	Network struct {
		CreateVCN bool     `json:"createVCN"`
		VCN       string   `json:"vcn,omitempty"`
		Subnets   []Subnet `json:"subnets,omitempty"`
	}
	OCNEOCIQuickCreateStatus struct {
		Phase QuickCreatePhase `json:"phase"`
	}
)
