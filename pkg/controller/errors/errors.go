// Copyright (c) 2021, 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package spi

import (
	"fmt"
	"strings"

	controllerruntime "sigs.k8s.io/controller-runtime"
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
	builder.WriteString("Retryable error")
	if r.Source != "" {
		builder.WriteString(fmt.Sprintf(", source: %s", r.Source))
	}
	if r.Operation != "" {
		builder.WriteString(fmt.Sprintf(", operation: %s", r.Operation))
	}
	if r.Cause != nil {
		builder.WriteString(fmt.Sprintf(", cause: %s", r.Cause))
	}
	if !r.Result.IsZero() {
		builder.WriteString(fmt.Sprintf(", result: {requeue: %v, requeueAfter: %s}", r.Result.Requeue, r.Result.RequeueAfter))
	}
	return builder.String()
}

// IsRetryableError returns true if the error is a RetryableError.
func IsRetryableError(err error) bool {
	_, ok := err.(RetryableError)
	return ok
}

// IsUpdateConflict returns true if the error is an update conflict error. This is occurs when the controller-runtime cache
// is out of sync with the etcd database
func IsUpdateConflict(err error) bool {
	return strings.Contains(err.Error(), "the object has been modified; please apply your changes to the latest version")
}

// ShouldLogKubernetesAPIError returns true if error should be logged.  This is used
// when calling the Kubernetes API, so conflict and webhook
// errors are not logged, the controller will just retry.
func ShouldLogKubernetesAPIError(err error) bool {
	if err == nil {
		return false
	}
	if IsUpdateConflict(err) {
		return false
	}
	return true
}
