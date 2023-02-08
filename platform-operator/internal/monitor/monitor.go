// Copyright (c) 2022, 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package monitor

import (
	ctrlerrors "github.com/verrazzano/verrazzano/pkg/controller/errors"
)

// BackgroundProcessMonitor - Represents a monitor object used by the component to monitor a background goroutine used for running
// install, uninstall, etc. operations asynchronously.
type BackgroundProcessMonitor interface {
	// CheckResult - Checks for a result from the goroutine; returns either the result of the operation, or an error indicating
	// the operation is still in progress
	CheckResult() (bool, error)
	// Reset - Resets the monitor and closes any open channels
	Reset()
	// IsRunning - returns true of the monitor/goroutine are active
	IsRunning() bool
	// IsCompleted - returns true if the monitor has run and marked as completed
	IsCompleted() bool
	// SetCompleted - sets the monitor thread to true
	SetCompleted()
	// Run - Run the operation with the specified args in a background goroutine
	Run(operation BackgroundFunc)
}

// BackgroundFunc - the operation to be called in the background goroutine
type BackgroundFunc func() error

// BackgroundProcessMonitorType - a monitor. &BackgroundProcessMonitorType acts as an implementation of BackgroundProcessMonitor
type BackgroundProcessMonitorType struct {
	ComponentName string
	running       bool
	completed     bool
	resultCh      chan bool
}

// CheckResult - checks for a result from the goroutine
// - returns false and a retry error if it's still running, or the result from the channel and nil if an answer was received
func (m *BackgroundProcessMonitorType) CheckResult() (bool, error) {
	select {
	case result := <-m.resultCh:
		return result, nil
	default:
		return false, ctrlerrors.RetryableError{Source: m.ComponentName}
	}
}

// Reset - reset the monitor and close the channel
func (m *BackgroundProcessMonitorType) Reset() {
	m.running = false
	m.completed = false
	close(m.resultCh)
}

// IsRunning - returns true if the monitor/goroutine are active
func (m *BackgroundProcessMonitorType) IsRunning() bool {
	return m.running
}

// IsCompleted - returns true if the monitor/goroutine is completed
func (m *BackgroundProcessMonitorType) IsCompleted() bool {
	return m.completed
}

// SetCompleted - sets the monitor thread as completed
func (m *BackgroundProcessMonitorType) SetCompleted() {
	m.completed = true
	m.running = false
}

// Run - calls the operation in a background goroutine
func (m *BackgroundProcessMonitorType) Run(operation BackgroundFunc) {
	m.running = true
	m.resultCh = make(chan bool, 2)

	go func(outputChan chan bool) {
		// The function will execute once, sending true on success, false on failure to the channel reader
		err := operation()
		if err != nil {
			outputChan <- false
			return
		}
		outputChan <- true
	}(m.resultCh)
}

// Check that &BackgroundProcessMonitorType implements BackgroundProcessMonitor
var _ BackgroundProcessMonitor = &BackgroundProcessMonitorType{}
