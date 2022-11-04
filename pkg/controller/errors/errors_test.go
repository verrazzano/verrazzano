// Copyright (c) 2021, 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
package spi

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	controllerruntime "sigs.k8s.io/controller-runtime"
)

// TestRetryableError Tests RetryableError
// GIVEN a RetryableError
// THEN the error is properly created and the Error() message returns the proper values
func TestRetryableError(t *testing.T) {
	tests := []struct {
		name        string
		description string
		err         RetryableError
	}{
		{
			name:        "EmptyError",
			description: "Retryable error with nothing provided",
			err:         RetryableError{},
		},
		{
			name:        "SourceOnly",
			description: "Retryable error with only a source",
			err: RetryableError{
				Source: "mySource",
			},
		},
		{
			name:        "SourceAndOp",
			description: "Retryable error with a source and an operation",
			err: RetryableError{
				Source:    "mySource",
				Operation: "myOp",
			},
		},
		{
			name:        "SourceAndOpAndCause",
			description: "Retryable error with a source, an op, and a cause",
			err: RetryableError{
				Source:    "mySource",
				Operation: "myOp",
				Cause:     fmt.Errorf("Custom error"),
			},
		},
		{
			name:        "SourceAndOpAndCauseAndResult",
			description: "Retryable error with a source, an op, a cause, and a Result",
			err: RetryableError{
				Source:    "mySource",
				Operation: "myOp",
				Cause:     fmt.Errorf("Custom error"),
				Result: controllerruntime.Result{
					Requeue:      true,
					RequeueAfter: time.Second * 30,
				},
			},
		},
	}
	for _, test := range tests {
		assert := assert.New(t)

		t.Log(test.description)

		err := test.err
		t.Logf("Error message: %s", err)

		if len(err.Operation) > 0 {
			assert.Contains(err.Error(), err.Source, "Source not found in message")
		}
		if len(err.Operation) > 0 {
			assert.Contains(err.Error(), err.Operation, "Operation not found in message")
		}
		if err.Cause != nil {
			assert.True(err.HasCause(), "HasCause should return true")
			assert.Contains(err.Error(), err.Cause.Error())
		} else {
			assert.False(err.HasCause(), "HasCause should return false")
		}
	}
}

// TestShouldLogKubenetesAPIError tests ShouldLogKubernetesAPIError
// Given an error
// Check whether it should be logged ot not
func TestShouldLogKubenetesAPIError(t *testing.T) {
	asserts := assert.New(t)
	err := fmt.Errorf("some kubernetes API error")

	asserts.True(ShouldLogKubenetesAPIError(err))

	err = fmt.Errorf(`operation cannot be fulfilled on configmaps "test": the object has been modified; please apply your changes to the latest version and try again`)
	asserts.False(ShouldLogKubenetesAPIError(err))
}
