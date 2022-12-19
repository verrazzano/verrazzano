// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package common

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	ctrlerrors "github.com/verrazzano/verrazzano/pkg/controller/errors"
)

var fakeCompName = "fake-component-name"

func TestMonitorType_IsRunning(t *testing.T) {
	a := assert.New(t)

	m := &MonitorType{ComponentName: fakeCompName}
	blocker := make(chan int)
	finished := make(chan int)
	operation := func() error {
		defer func() { finished <- 0 }()
		<-blocker
		return nil
	}

	m.Run(operation)
	a.True(m.IsRunning())

	// send to the channel to unblock operation
	blocker <- 0

	// block until the operation says it's finished
	<-finished

	a.False(m.IsRunning())
}

func TestMonitorType_CheckResultWhileRunning(t *testing.T) {
	a := assert.New(t)

	m := &MonitorType{ComponentName: fakeCompName}
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

func TestMonitorType_CheckResult(t *testing.T) {
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
		m := &MonitorType{ComponentName: fakeCompName}
		finished := make(chan int)
		operation := func() error {
			defer func() { finished <- 0 }()
			if tt.success {
				return nil
			}
			return fmt.Errorf(errMsg)
		}

		// Run the background goroutine, and block until it returns
		m.Run(operation)
		<-finished

		res, err := m.CheckResult()
		a.Equal(tt.expectedResult, res)
		a.Equal(nil, err)
	}
}

func TestMonitorType_Reset(t *testing.T) {
	a := assert.New(t)

	m := &MonitorType{ComponentName: fakeCompName}
	finished := make(chan int)
	operation := func() error {
		defer func() { finished <- 0 }()
		return nil
	}

	// Run the background goroutine, and block until it returns
	m.Run(operation)
	<-finished

	res, _ := m.CheckResult()
	a.True(res)

	m.Reset()
	res, _ = m.CheckResult()
	a.False(res)
}

func TestMonitorType_ResetWhileRunning(t *testing.T) {
	a := assert.New(t)

	m := &MonitorType{ComponentName: fakeCompName}
	blocker := make(chan int)
	operation := func() error {
		<-blocker
		return nil
	}

	m.Run(operation)
	a.True(m.IsRunning())
	m.Reset()
	a.False(m.IsRunning())
	blocker <- 0
}

func TestFakeMonitorType_CheckResult(t *testing.T) {
	a := assert.New(t)

	f := &FakeMonitorType{Result: true, Err: nil}
	res, err := f.CheckResult()
	a.True(res)
	a.NoError(err)

	f = &FakeMonitorType{Result: false, Err: fmt.Errorf("an unexpected error")}
	res, err = f.CheckResult()
	a.False(res)
	a.Error(err)
}

func TestFakeMonitorType_IsRunning(t *testing.T) {
	a := assert.New(t)

	f := &FakeMonitorType{Running: true}
	a.True(f.IsRunning())

	f = &FakeMonitorType{Running: false}
	a.False(f.IsRunning())
}
