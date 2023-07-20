// Copyright (c) 2022, 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package monitor

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	ctrlerrors "github.com/verrazzano/verrazzano/pkg/controller/errors"
)

var fakeCompName = "fake-component-name"

func TestMonitorTypeIsRunning(t *testing.T) {
	a := assert.New(t)

	m := &BackgroundProcessMonitorType{ComponentName: fakeCompName}
	blocker := make(chan int)
	finished := make(chan int)
	operation := func() error {
		defer func() { finished <- 0 }()
		<-blocker
		return nil
	}

	a.False(m.IsRunning())

	m.Run(operation)
	a.True(m.IsRunning())

	// send to the channel to unblock operation
	blocker <- 0

	// block until the operation says it's finished
	<-finished

	// even after the operation is finished, the monitor should still be "running" unless reset
	a.True(m.IsRunning())
}

func TestMonitorTypeCheckResultWhileRunning(t *testing.T) {
	a := assert.New(t)

	m := &BackgroundProcessMonitorType{ComponentName: fakeCompName}
	blocker := make(chan int)
	operation := func() error {
		<-blocker
		return nil
	}

	m.Run(operation)
	res, err := m.CheckResult()
	a.Equal(false, res)
	a.Equal(ctrlerrors.RetryableError{Source: fakeCompName}, err)
	blocker <- 0
}

func TestMonitorTypeCheckResult(t *testing.T) {
	a := assert.New(t)

	errMsg := "an error from the background operation"
	tests := []struct {
		success        bool
		expectedResult bool
		expectedError  error
	}{
		{
			success:        true,
			expectedResult: true,
		},
		{
			success:        false,
			expectedResult: false,
		},
	}

	for _, tt := range tests {
		m := &BackgroundProcessMonitorType{ComponentName: fakeCompName}
		operation := func() error {
			if tt.success {
				return nil
			}
			return fmt.Errorf(errMsg)
		}

		// Run the background goroutine
		m.Run(operation)

		// CheckResult returns an error if the operation has not yet completed, so keep checking until complete
		if res, ok := waitForResult(t, m); ok {
			a.Equal(tt.expectedResult, res)
		} else {
			a.Fail("Expected a result from CheckResult but never received one")
		}
	}
}

// waitForResult waits for the background process result to be available. Returns the value of
// the result and true if a result was received, or false if we timed out waiting.
func waitForResult(t *testing.T, monitor *BackgroundProcessMonitorType) (bool, bool) {
	var res, gotResult bool
	for i := 0; i < 100; i++ {
		var err error
		// CheckResult returns an error if the operation is still running
		res, err = monitor.CheckResult()
		if err != nil {
			t.Log("Waiting for result...")
			time.Sleep(100 * time.Millisecond)
			continue
		}
		gotResult = true
		break
	}
	return res, gotResult
}

func TestMonitorTypeReset(t *testing.T) {
	a := assert.New(t)

	m := &BackgroundProcessMonitorType{ComponentName: fakeCompName}
	operation := func() error {
		return nil
	}

	// Run the background goroutine
	m.Run(operation)

	// Wait for the operation result to be available
	if res, ok := waitForResult(t, m); ok {
		a.True(res)
	} else {
		a.Fail("Expected a result from CheckResult but never received one")
	}

	a.True(m.IsRunning())

	m.Reset()
	res, _ := m.CheckResult()
	a.False(res)
	a.False(m.IsRunning())
}

func TestMonitorTypeIsCompleted(t *testing.T) {
	a := assert.New(t)

	m := &BackgroundProcessMonitorType{ComponentName: fakeCompName}
	blocker := make(chan int)
	finished := make(chan int)
	operation := func() error {
		defer func() { finished <- 0 }()
		<-blocker
		return nil
	}

	m.Run(operation)
	a.True(m.IsRunning())
	m.SetCompleted()
	a.True(m.IsCompleted())
	a.False(m.IsRunning())

	// send to the channel to unblock operation
	blocker <- 0

	// block until the operation says it's finished
	<-finished
}
