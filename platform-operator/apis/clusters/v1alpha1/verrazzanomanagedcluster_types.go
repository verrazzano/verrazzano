// Copyright (c) 2021, 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package v1alpha1

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// The VerrazzanoManagedCluster custom resource contains information about a
// kubernetes cluster where Verrazzano managed applications are deployed.

// VerrazzanoManagedClusterSpec defines the desired state of a VerrazzanoManagedCluster.
type VerrazzanoManagedClusterSpec struct {
	// The name of a Secret that contains the CA certificate of the managed cluster. This is used to configure the
	// admin cluster to scrape metrics from the Prometheus endpoint on the managed cluster. See the pre-registration
	// <a href="../../../../../docs/setup/install/multicluster/#preregistration-setup">instructions</a>
	// for how to create this Secret.
	CASecret string `json:"caSecret,omitempty"`

	// The description of the managed cluster.
	// +optional
	Description string `json:"description,omitempty"`

	// The name of the Secret containing generated YAML manifest file to be applied by the user to the managed cluster.
	// This field is managed by a Verrazzano Kubernetes operator.
	// +optional
	ManagedClusterManifestSecret string `json:"managedClusterManifestSecret,omitempty"`

	// The name of the ServiceAccount that was generated for the managed cluster. This field is managed by a
	// Verrazzano Kubernetes operator.
	// +optional
	ServiceAccount string `json:"serviceAccount,omitempty"`
}

// ConditionType identifies the condition of the VMC which can be checked with kubectl wait
type ConditionType string

const (
	// Ready = true means the VMC is ready to be used and all resources needed have been generated
	ConditionReady ConditionType = "Ready"

	// ManagedCARetrieved = true means that the managed cluster CA cert has been retrieved and
	// populated. This is done by the VMC controller via the Rancher API proxy for the managed cluster.
	ConditionManagedCARetrieved ConditionType = "ManagedCARetrieved"

	// ConditionManifestPushed = true means the the agent and registration secrets have been successfully transferred
	// to the managed cluster on a multicluster install
	ConditionManifestPushed ConditionType = "ManifestPushed"
)

// StateType identifies the state of the VerrazzanoManagedCluster resource.
type StateType string

const (
	StateActive   StateType = "Active"
	StateInactive StateType = "Inactive"
	StatePending  StateType = "Pending"
)

// Condition describes a condition that occurred on the VerrazzanoManagedCluster resource.
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

// RancherRegistrationStatus identifies the status of Rancher registration.
type RancherRegistrationStatus string

const (
	RegistrationCompleted RancherRegistrationStatus = "Completed"
	RegistrationFailed    RancherRegistrationStatus = "Failed"
)

// RancherRegistration defines the Rancher registration state for a managed cluster.
type RancherRegistration struct {
	// The status of the Rancher registration
	Status RancherRegistrationStatus `json:"status"`
	// Supporting message related to the Rancher registration status
	// +optional
	Message string `json:"message,omitempty"`
	// ClusterID is the Rancher cluster ID for this cluster
	ClusterID string `json:"clusterID,omitempty"`
}

// VerrazzanoManagedClusterStatus defines the observed state of a VerrazzanoManagedCluster resource.
type VerrazzanoManagedClusterStatus struct {
	// The Verrazzano API server URL for this managed cluster.
	APIUrl string `json:"apiUrl,omitempty"`
	// The current state of this managed cluster.
	Conditions []Condition `json:"conditions,omitempty"`
	// The last time the agent from this managed cluster connected to the admin cluster.
	LastAgentConnectTime *metav1.Time `json:"lastAgentConnectTime,omitempty"`
	// The Prometheus host for this managed cluster.
	PrometheusHost string `json:"prometheusHost,omitempty"`
	// The state of Rancher registration for this managed cluster.
	RancherRegistration RancherRegistration `json:"rancherRegistration,omitempty"`
	// The state of this managed cluster.
	State StateType `json:"state"`
}

//	VerrazzanoManagedCluster specifies the Verrazzano Managed Cluster API.
//
// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:shortName=vmc;vmcs
// +genclient
type VerrazzanoManagedCluster struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// The desired state of a Verrazzano Managed Cluster resource.
	Spec VerrazzanoManagedClusterSpec `json:"spec,omitempty"`
	// The observed state of a Verrazzano Managed Cluster resource.
	Status VerrazzanoManagedClusterStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// VerrazzanoManagedClusterList contains a list of VerrazzanoManagedCluster resources.
type VerrazzanoManagedClusterList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []VerrazzanoManagedCluster `json:"items"`
}

func init() {
	SchemeBuilder.Register(&VerrazzanoManagedCluster{}, &VerrazzanoManagedClusterList{})
}
