// Copyright (c) 2020, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package v8

// Current service state of domain
// +k8s:openapi-gen=true
type DomainCondition struct {
	// Last time we probed the condition
	LastProbeTime string `json:"lastProbeTime,omitempty"`

	// Last time the condition transitioned from one status to another
	LastTransitionTime string `json:"lastTransitionTime,omitempty"`

	// Human-readable message indicating details about last transition
	Message string `json:"message,omitempty"`

	// Unique, one-word, CamelCase reason for the condition's last transition
	Reason string `json:"reason,omitempty"`

	// Status is the status of the condition. Can be True, False, Unknown
	Status string `json:"status"`

	// The type of the condition. Valid types are Progressing,
	// Available, and Failed. Required
	Type string `json:"type"`
}
