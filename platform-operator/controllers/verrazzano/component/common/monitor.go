// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package common

import (
	ctrlerrors "github.com/verrazzano/verrazzano/pkg/controller/errors"
)

// Monitor - Represents a monitor object used by the component to monitor a background goroutine used for running
// install, uninstall, etc. operations asynchronously.
type Monitor interface {
	// CheckResult - Checks for a result from the goroutine; returns either the result of the operation, or an error indicating
	// the operation is still in progress
	CheckResult() (bool, error)
	// Reset - Resets the monitor and closes any open channels
	Reset()
	// IsRunning - returns true of the monitor/goroutine are active
	IsRunning() bool
	// Run - Run the operation with the specified args in a background goroutine
	Run(operation BackgroundFunc)
}

// BackgroundFunc - the operation to be called in the background goroutine
type BackgroundFunc func() error

// MonitorType - a monitor. &MonitorType acts as an implementation of Monitor
type MonitorType struct {
	ComponentName string
	running       bool
	resultCh      chan bool
}

// CheckResult - checks for a result from the goroutine
// - returns false and a retry error if it's still running, or the result from the channel and nil if an answer was received
func (m *MonitorType) CheckResult() (bool, error) {
	select {
	case result := <-m.resultCh:
		return result, nil
	default:
		return false, ctrlerrors.RetryableError{Source: m.ComponentName}
	}
}

// Reset - reset the monitor and close the channel
func (m *MonitorType) Reset() {
	m.running = false
	close(m.resultCh)
}

// IsRunning - returns true of the monitor/goroutine are active
func (m *MonitorType) IsRunning() bool {
	return m.running
}

// Run - calls the operation in a background goroutine
func (m *MonitorType) Run(operation BackgroundFunc) {
	m.running = true
	m.resultCh = make(chan bool, 2)

	go func(outputChan chan bool) {
		// The function will execute once, sending true on success, false on failure to the channel reader
		err := operation()
		defer func() { m.running = false }()
		if err != nil {
			outputChan <- false
			return
		}
		outputChan <- true
	}(m.resultCh)
}

// Check that &MonitorType implements Monitor
var _ Monitor = &MonitorType{}

// FakeMonitorType - a fake monitor object, useful for unit testing.
type FakeMonitorType struct {
	Result  bool
	Err     error
	Running bool
}

func (f *FakeMonitorType) CheckResult() (bool, error)   { return f.Result, f.Err }
func (f *FakeMonitorType) Reset()                       {}
func (f *FakeMonitorType) IsRunning() bool              { return f.Running }
func (f *FakeMonitorType) Run(operation BackgroundFunc) {}

// Check that &FakeMonitorType implements Monitor
var _ Monitor = &FakeMonitorType{}
