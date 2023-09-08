// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package errors

import (
	"fmt"
	"strings"
)

// ErrorAggregator provides an interface for building composite error strings, separated by a delimiter.
// The aggregator implements the error interface, allowing it to be returned from error-returning functions.
type ErrorAggregator struct {
	delim  string
	errors []error
}

func NewAggregator(delim string) *ErrorAggregator {
	return &ErrorAggregator{
		delim:  delim,
		errors: []error{},
	}
}

// Error builds the aggregated error string.
func (e *ErrorAggregator) Error() string {
	sb := strings.Builder{}
	for i, err := range e.errors {
		sb.WriteString(err.Error())
		if i != len(e.errors)-1 {
			sb.WriteString(e.delim)
		}
	}
	return sb.String()
}

// Add appends a new error to the aggregation.
func (e *ErrorAggregator) Add(err error) {
	e.errors = append(e.errors, err)
}

// Addf is equivalent to Add(fmt.Errorf(...)).
func (e *ErrorAggregator) Addf(format string, args ...any) {
	e.Add(fmt.Errorf(format, args...))
}

// HasError returns true if any errors have been added to the aggregator.
func (e *ErrorAggregator) HasError() bool {
	return len(e.errors) > 0
}
