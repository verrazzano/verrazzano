// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package v1alpha1

import corev1 "k8s.io/api/core/v1"

// This file contains common types and functions used by all MultiCluster Custom Resource Types

// Placement contains the name of each cluster where a resource will be located.
type Placement struct {
	// List of clusters.
	Clusters []Cluster `json:"clusters"`
}

// Cluster contains the name of a single cluster.
type Cluster struct {
	// The name of a cluster.
	Name string `json:"name"`
}

// Condition describes current state of a multi cluster resource.
type Condition struct {
	// Last time the condition transitioned from one status to another.
	// +optional
	LastTransitionTime string `json:"lastTransitionTime,omitempty"`
	// A message with details about the last transition.
	// +optional
	Message string `json:"message,omitempty"`
	// Status of the condition: one of `True`, `False`, or `Unknown`.
	Status corev1.ConditionStatus `json:"status"`
	// Type of condition.
	Type ConditionType `json:"type"`
}

// ClusterLevelStatus describes the status of the multi cluster resource in a specific cluster.
type ClusterLevelStatus struct {
	// Last update time of the resource state in this cluster.
	LastUpdateTime string `json:"lastUpdateTime"`
	// Message details about the status in this cluster.
	Message string `json:"message,omitempty"`
	// Name of the cluster.
	Name string `json:"name"`
	// State of the resource in this cluster.
	State StateType `json:"state"`
}

// ConditionType identifies the condition of the multi-cluster resource which can be checked with `kubectl wait`.
type ConditionType string

const (
	// DeployComplete means deployment to specified cluster completed successfully.
	DeployComplete ConditionType = "DeployComplete"

	// DeployFailed means the deployment to specified cluster has failed.
	DeployFailed ConditionType = "DeployFailed"

	// DeployPending means deployment to specified cluster is in progress.
	DeployPending ConditionType = "DeployPending"
)

// StateType identifies the state of a multi-cluster resource.
type StateType string

const (
	// Failed is the state when deploy to specified cluster has failed.
	Failed StateType = "Failed"

	// Pending is the state when deploy to specified cluster is in progress.
	Pending StateType = "Pending"

	// Succeeded is the state when deploy to specified cluster is completed.
	Succeeded StateType = "Succeeded"
)

// MultiClusterResourceStatus is the runtime status of a multi-cluster resource.
type MultiClusterResourceStatus struct {
	// Status information for each cluster.
	Clusters []ClusterLevelStatus `json:"clusters,omitempty"`

	// The current state of a multicluster resource.
	Conditions []Condition `json:"conditions,omitempty"`

	// The state of the multicluster resource. State values are case-sensitive and formatted as follows:
	// <ul><li>`Failed`: rdeployment to cluster failed</li><li>`Pending`: deployment to cluster is in progress</li><li>`Succeeded`: deployment to cluster successfully completed</li></ul>
	State StateType `json:"state,omitempty"`
}

// EmbeddedObjectMeta is metadata describing a resource.
type EmbeddedObjectMeta struct {
	// Name of the resource.
	// +optional
	Name string `json:"name,omitempty" protobuf:"bytes,1,opt,name=name"`

	// Namespace of the resource.
	// +optional
	Namespace string `json:"namespace,omitempty" protobuf:"bytes,3,opt,name=namespace"`

	// Labels for the resource.
	// +optional
	Labels map[string]string `json:"labels,omitempty" protobuf:"bytes,11,rep,name=labels"`

	// Annotations for the resource.
	// +optional
	Annotations map[string]string `json:"annotations,omitempty" protobuf:"bytes,12,rep,name=annotations"`
}
