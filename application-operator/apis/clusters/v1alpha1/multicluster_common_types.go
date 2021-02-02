// Copyright (c) 2021 Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package v1alpha1

import corev1 "k8s.io/api/core/v1"

// Placement information for multi cluster resources
type Placement struct {
	Clusters []Cluster `json:"clusters"`
}

// Cluster where multi cluster resources are placed
type Cluster struct {
	// the name of the cluster
	Name string `json:"name"`
}

// Condition describes current state of a multi cluster resource.
type Condition struct {
	// Type of condition.
	Type ConditionType `json:"type"`
	// Status of the condition, one of True, False, Unknown.
	Status corev1.ConditionStatus `json:"status"`
	// Last time the condition transitioned from one status to another.
	// +optional
	LastTransitionTime string `json:"lastTransitionTime,omitempty"`
	// Human readable message indicating details about last transition.
	// +optional
	Message string `json:"message,omitempty"`
}

// ConditionType identifies the condition of the multi-cluster resource which can be checked with kubectl wait
type ConditionType string

const (
	// DeployStarted means deployment to specified cluster is in progress.
	DeployStarted ConditionType = "DeployStarted"

	// DeployComplete means deployment to specified cluster completed successfully
	DeployComplete ConditionType = "DeployComplete"

	// DeployFailed means the deployment to specified cluster has failed.
	DeployFailed ConditionType = "DeployFailed"
)

// StateType identifies the state of a multi-cluster resource
type StateType string

const (
	// Deploying is the state when deploy to specified cluster is in progress
	Deploying StateType = "Deploying"

	// Ready is the state when deploy to specified cluster is completed
	Ready StateType = "Ready"

	// Failed is the state when deploy to specified cluster has failed
	Failed StateType = "Failed"
)
