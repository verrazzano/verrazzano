// Copyright (c) 2020, 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package v8

// ServerHealth describes the current status and health of a specific WebLogic server.
// +k8s:openapi-gen=true
type ServerHealth struct {
	// RFC 3339 date and time at which the server started
	ActivationTime string `json:"activationTime,omitempty"`

	// Server health of this WebLogic server. If the value is "Not available", the operator has
	// failed to read the health. If the value is "Not available (possibly overloaded)", the
	// operator has failed to read the health of the server possibly due to the server is
	// in overloaded state"
	OverallHealth string `json:"overallHealth,omitempty"`

	// Status of unhealthy subsystems, if any
	// +x-kubernetes-list-type=set
	Subsystems []SubsystemHealth `json:"subsystems,omitempty"`
}
