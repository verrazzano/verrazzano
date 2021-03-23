// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// The VerrazzanoManagedCluster custom resource contains information about a
// kubernetes cluster where Verrazzano managed applications are deployed.

// VerrazzanoManagedClusterSpec defines the desired state of VerrazzanoManagedCluster
type VerrazzanoManagedClusterSpec struct {
	// The description of the managed cluster.
	Description string `json:"description,omitempty"`

	// The name of a secret that contains the credentials for scraping from
	// the prometheus endpoint on the managed cluster.  The secret contains
	// the endpoint, username and password.
	PrometheusSecret string `json:"prometheusSecret"`

	// The name of the ServiceAccount that was generated for the managed cluster.
	// This field is managed by a Verrazzano Kubernetes operator.
	ServiceAccount string `json:"serviceAccount,omitempty"`

	// The name of the secret containing generated YAML manifest to be applied by the user to the managed cluster.
	// This field is managed by a Verrazzano Kubernetes operator.
	ManagedClusterManifestSecret string `json:"managedClusterManifestSecret,omitempty"`
}

// VerrazzanoManagedClusterStatus defines the observed state of VerrazzanoManagedCluster
type VerrazzanoManagedClusterStatus struct {
	// Verrazzano API Server URL for the managed cluster.
	APIUrl string `json:"apiUrl,omitempty"`
	// Last time the agent from this managed cluster connected to the admin cluster.
	// +optional
	LastAgentConnectTime metav1.Time `json:"lastAgentConnectTime,omitempty"`
}

// VerrazzanoManagedCluster is the Schema for the Verrazzanomanagedclusters API
// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:shortName=vmc;vmcs
// +genclient
type VerrazzanoManagedCluster struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   VerrazzanoManagedClusterSpec   `json:"spec,omitempty"`
	Status VerrazzanoManagedClusterStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// VerrazzanoManagedClusterList contains a list of VerrazzanoManagedCluster
type VerrazzanoManagedClusterList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []VerrazzanoManagedCluster `json:"items"`
}

func init() {
	SchemeBuilder.Register(&VerrazzanoManagedCluster{}, &VerrazzanoManagedClusterList{})
}
