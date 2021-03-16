// Copyright (c) 2020, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package v8

// +k8s:openapi-gen=true
type ClusterStatus struct {
	// WebLogic cluster name.
	ClusterName string `json:"clusterName,omitempty"`

	// The maximum number of cluster members.
	MaximumReplicas int `json:"maximumReplicas,omitempty"`

	// The minimum number of cluster members.
	MinimumReplicas int `json:"minimumReplicas,omitempty"`

	// The number of ready cluster members.
	ReadyReplicas int `json:"readyReplicas,omitempty"`

	// The number of currently running cluster members.
	Replicas int `json:"replicas,omitempty"`

	// The requested number of cluster members. Cluster members will be started by the operator if this value
	// is larger than zero.
	ReplicasGoal int `json:"replicasGoal,omitempty"`
}