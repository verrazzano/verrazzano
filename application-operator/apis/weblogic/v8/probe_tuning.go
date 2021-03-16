// Copyright (c) 2020, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package v8

//
// +k8s:openapi-gen=true
type ProbeTuning struct {
	// The number of seconds before the first check is performed.
	InitialDelaySeconds *int32 `json:"initialDelaySeconds,omitempty"`

	// The number of seconds between checks.
	PeriodSeconds *int32 `json:"periodSeconds,omitempty"`

	// The number of seconds with no response that indicates a failure.
	TimeoutSeconds *int32 `json:"timeoutSeconds,omitempty"`
}
