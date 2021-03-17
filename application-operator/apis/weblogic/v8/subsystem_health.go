// Copyright (c) 2020, 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package v8

// SubsystemHealth describes the current health of a specific subsystem.
// +k8s:openapi-gen=true
type SubsystemHealth struct {
	// Server health of this WebLogic server
	Health string `json:"health"`

	// Name of subsystem providing symptom information
	SubsystemName string `json:"subsystemName"`

	// Symptoms provided by the reporting subsystem.
	// +x-kubernetes-list-type=set
	Symptoms []string `json:"symptoms,omitempty"`
}
