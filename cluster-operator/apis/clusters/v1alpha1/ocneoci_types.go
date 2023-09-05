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
		// +optional
		// +kubebuilder:default:={clusterNetwork:{podCIDR: "10.244.0.0/16",serviceCIDR: "10.96.0.0/16"}}

		// Kubernetes settings.
		KubernetesBase `json:"kubernetes"`
		// OCNE settings.
		OCNE OCNE `json:"ocne"`
		// OCI infrastructure settings.
		OCI OCI `json:"oci"`
	}
	OCI struct {
		CommonOCI `json:",inline"`
		// Control Plane node settings.
		ControlPlane OCINode `json:"controlPlane"`
		// List of worker nodes.
		Workers []NamedOCINode `json:"workers"`
		// +kubebuilder:default:={createVCN: false}
		// +optional

		// OCI Network settings.
		Network *Network `json:"network"`
	}
	Network struct {
		// +optional
		// +kubebuilder:default:=false

		// If true, a new VCN is created for the cluster.
		// The default is false.
		CreateVCN bool `json:"createVCN"`
		// +optional
		// +kubebuilder:validation:pattern:="^([0-9a-zA-Z-_]+[.:])([0-9a-zA-Z-_]*[.:]){3,}([0-9a-zA-Z-_]+)$"

		// OCID of an existing VCN to create the cluster inside.
		VCN string `json:"vcn,omitempty"`
		// +optional

		// List of existing subnets that will be used by the cluster.
		Subnets []Subnet `json:"subnets,omitempty"`
	}
	OCNEOCIQuickCreateStatus struct {
		Phase QuickCreatePhase `json:"phase"`
	}
)
