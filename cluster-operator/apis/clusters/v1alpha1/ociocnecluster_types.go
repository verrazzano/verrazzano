// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +genclient
// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// OCIOCNECluster specifies the API for quick-create OCI OCNE Clusters.
type OCIOCNECluster struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// The desired state of an OCIOCNECluster resource.
	Spec OCIOCNEClusterSpec `json:"spec,omitempty"`
	// The observed state of an OCIOCNECluster resource.
	Status OCIOCNEClusterStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// OCIOCNEClusterList contains a list of OCIOCNECluster resources.
type OCIOCNEClusterList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []OCIOCNECluster `json:"items"`
}

type (
	OCIOCNEClusterSpec struct {
		CommonClusterSpec `json:",inline"`
		OCI               OCI `json:"oci"`
	}
	OCI struct {
		CommonOCISpec CommonOCISpec `json:",inline"`
		ControlPlane  NodeConfig    `json:"controlPlane"`
		Workers       []NodeConfig  `json:"workers,omitempty"`
		Network       Network       `json:"network"`
	}
	Network struct {
		PodCIDR     string   `json:"podCIDR"`
		ClusterCIDR string   `json:"clusterCIDR"`
		VCN         *string  `json:"string,omitempty"`
		Subnets     *Subnets `json:"subnets,omitempty"`
	}
	OCIOCNEClusterStatus struct {
	}
)
