// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package v1alpha1

import (
	corev1 "k8s.io/api/core/v1"
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

// ConditionType identifies the condition of the VMC which can be checked with kubectl wait
type ConditionType string

const (
	// Ready = true means the VMC is ready to be used and all resources needed have been generated
	ConditionReady ConditionType = "Ready"
)

// Condition describes a condition that occurred on the VerrazzanoManagedCluster resource
type Condition struct {
	// Type of condition.
	Type ConditionType `json:"type"`
	// Status of the condition, one of True, False, Unknown.
	Status corev1.ConditionStatus `json:"status"`
	// Last time the condition transitioned from one status to another.
	// +optional
	LastTransitionTime *metav1.Time `json:"lastTransitionTime,omitempty"`
	// Human readable message indicating details about last transition.
	// +optional
	Message string `json:"message,omitempty"`
}

// VerrazzanoManagedClusterStatus defines the observed state of VerrazzanoManagedCluster
type VerrazzanoManagedClusterStatus struct {
	// The latest available observations of an object's current state.
	Conditions []Condition `json:"conditions,omitempty"`
	// Last time the agent from this managed cluster connected to the admin cluster.
	// +optional
	LastAgentConnectTime *metav1.Time `json:"lastAgentConnectTime,omitempty"`
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
