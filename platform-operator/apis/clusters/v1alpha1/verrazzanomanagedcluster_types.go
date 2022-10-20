// Copyright (c) 2021, 2022, Oracle and/or its affiliates.
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

	// The name of a secret that contains the ca certificate for accessing console
	// and api endpoints on the managed cluster.
	CASecret string `json:"caSecret,omitempty"`

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

	// ManagedCARetrieved = true means that the managed cluster CA cert has been retrieved and
	// populated. This is done by the VMC controller via the Rancher API proxy for the managed cluster.
	ConditionManagedCARetrieved ConditionType = "ManagedCARetrieved"

	// ConditionManifestPushed = true means the the agent and registration secrets have been successfully transfered
	// to the managed cluster on a multicluster install
	ConditionManifestPushed ConditionType = "ManifestPushed"
)

// StateType identifies the state of the VMC which is shown in Verrazzano Dashboard.
type StateType string

const (
	StateActive   StateType = "Active"
	StateInactive StateType = "Inactive"
	StatePending  StateType = "Pending"
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

type RancherRegistrationStatus string

const (
	RegistrationCompleted RancherRegistrationStatus = "Completed"
	RegistrationFailed    RancherRegistrationStatus = "Failed"
)

// RancherRegistration defines the Rancher registration state for a managed cluster
type RancherRegistration struct {
	// The status of the Rancher registration
	Status RancherRegistrationStatus `json:"status"`
	// Supporting message related to the Rancher registration status
	// +optional
	Message string `json:"message,omitempty"`
	// ClusterID is the Rancher cluster ID for this cluster
	ClusterID string `json:"clusterID,omitempty"`
}

// VerrazzanoManagedClusterStatus defines the observed state of VerrazzanoManagedCluster
type VerrazzanoManagedClusterStatus struct {
	// The latest available observations of an object's current state.
	Conditions []Condition `json:"conditions,omitempty"`
	// Last time the agent from this managed cluster connected to the admin cluster.
	// +optional
	LastAgentConnectTime *metav1.Time `json:"lastAgentConnectTime,omitempty"`
	// State of the Cluster to determine if it is Active, Pending, or Inactive.
	State StateType `json:"state"`
	// Verrazzano API Server URL for the managed cluster.
	APIUrl string `json:"apiUrl,omitempty"`
	// Prometheus Host for the managed cluster.
	PrometheusHost string `json:"prometheusHost,omitempty"`
	// State of Rancher registration for a managed cluster
	RancherRegistration RancherRegistration `json:"rancherRegistration,omitempty"`
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
