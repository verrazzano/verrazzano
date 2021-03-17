// Copyright (c) 2020, 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package v8

// AdminServer represents the operator configuration for the admin server
// +k8s:openapi-gen=true
type AdminServer struct {
	// Configures which of the admin server's WebLogic admin channels should be exposed outside
	// the Kubernetes cluster via a node port service.
	AdminService AdminService `json:"adminService"`

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
