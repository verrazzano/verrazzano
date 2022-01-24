// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
package spi

import (
	"fmt"
	controllerruntime "sigs.k8s.io/controller-runtime"
	"strings"
)

// RetryableError an error that can be used to indicate to a controller that a requeue is needed, with an optional custom result
type RetryableError struct {
	// The source of the error
	Source string
	// The operation type
	Operation string
	// The root cause error
	Cause error
	// An optional Result type to return to the controllerruntime
	Result controllerruntime.Result
}

// HasCause indicates whether or not the error has a root cause
func (r RetryableError) HasCause() bool {
	return r.Cause != nil
}

var _ error = RetryableError{}

// Error implements the basic Go error contract
func (r RetryableError) Error() string {
	var builder strings.Builder
	builder.WriteString(fmt.Sprintf("Retryable error, source: %s, operation: %s", r.Operation, r.Source))
	if r.Cause != nil {
		builder.WriteString(fmt.Sprintf(", cause %s", r.Cause))
	}
	if !r.Result.IsZero() {
		builder.WriteString(fmt.Sprintf(", result: {requeue: %v, requeueAfter: %s}", r.Result.Requeue, r.Result.RequeueAfter))
	}
	return builder.String()
}
