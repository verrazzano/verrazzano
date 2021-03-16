// Copyright (c) 2020, 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package v8

// Cluster contains details of a WebLogic cluster
// +k8s:openapi-gen=true
type Cluster struct {
	// Specifies whether the number of running cluster members is allowed to drop below the minimum dynamic cluster
	// size configured in the WebLogic domain configuration. Otherwise, the operator will ensure that the number of
	// running cluster members is not less than the minimum dynamic cluster setting. This setting applies to dynamic
	// clusters only. Defaults to true.
	AllowReplicasBelowMinDynClusterSize bool `json:"allowReplicasBelowMinDynClusterSize,omitempty"`

	// The name of this cluster
	ClusterName string `json:"clusterName"`

	// Customization affecting ClusterIP Kubernetes services for the WebLogic cluster.
	ClusterService KubernetesResource `json:"clusterService,omitempty"`

	// The maximum number of Managed Servers instances that the operator will start in parallel for this cluster in
	// response to a change in the replicas count. If more Managed Server instances must be started, the operator
	// will wait until a Managed Server Pod is in the Ready state before starting the next Managed Server instance.
	// A value of 0 means all Managed Server instances will start in parallel. Defaults to 0.
	MaxConcurrentStartup int32 `json:"maxConcurrentStartup,omitempty"`

	// The maximum number of cluster members that can be temporarily unavailable. Defaults to 1.
	MaxUnavailable int32 `json:"maxUnavailable,omitempty"`

	// The number of managed servers to run in this cluster
	// Note: this value is required by WebLogic Operator, but is marked optional because Verrazzano can provide a default value.
	Replicas int `json:"replicas,omitempty"`

	// If present, every time this value is updated the operator will restart
	// the required servers.
	RestartVersion string `json:"restartVersion,omitempty"`

	// Server Pod
	ServerPod ServerPod `json:"serverPod,omitempty"`

	// Customization affecting ClusterIP Kubernetes services for WebLogic Server instances.
	ServerService ServerService `json:"serverService,omitempty"`

	// The strategy for deciding whether to start a server.  Legal values are ADMIN_ONLY, NEVER, or IF_NEEDED.
	// Note: this value is required by WebLogic Operator, but is marked optional because Verrazzano can provide a
	// default value.
	ServerStartPolicy string `json:"serverStartPolicy,omitempty"`

	// The state in which the server is to be started.  Legal values are "RUNNING" or "ADMIN"
	ServerStartState string `json:"serverStartState,omitempty"`
}
