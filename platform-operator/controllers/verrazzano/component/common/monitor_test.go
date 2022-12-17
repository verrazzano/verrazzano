// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package common

import (
	"fmt"
	"github.com/stretchr/testify/assert"
	ctrlerrors "github.com/verrazzano/verrazzano/pkg/controller/errors"
	"testing"
	"time"
)

var fakeCompName = "fake-component-name"

func TestMonitorType_IsRunning(t *testing.T) {
	a := assert.New(t)

	m := &MonitorType{ComponentName: fakeCompName}
	c := make(chan int)
	operation := func() error {
		_ = <-c
		return nil
	}

	m.Run(operation)
	a.True(m.IsRunning())

	// send to the channel, and wait a short while for the goroutine to finish
	c <- 0
	time.Sleep(10 * time.Millisecond)

	a.False(m.IsRunning())
}

func TestMonitorType_CheckResultWhileRunning(t *testing.T) {
	a := assert.New(t)

	m := &MonitorType{ComponentName: fakeCompName}
	c := make(chan int)
	operation := func() error {
		_ = <-c
		return nil
	}

	m.Run(operation)
	res, err := m.CheckResult()
	a.Equal(false, res)
	a.Equal(ctrlerrors.RetryableError{Source: fakeCompName}, err)
	c <- 0
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
			expectedError:  nil,
		},
		{
			success:        false,
			expectedResult: false,
			expectedError:  fmt.Errorf(errMsg),
		},
	}

	for _, tt := range tests {
		m := &MonitorType{ComponentName: fakeCompName}
		c := make(chan int)
		operation := func() error {
			defer func() { c <- 0 }()
			if tt.success {
				return nil
			}
			return fmt.Errorf(errMsg)
		}

		// Run the background goroutine, and block until it returns
		m.Run(operation)
		<-c

		res, err := m.CheckResult()
		a.Equal(tt.expectedResult, res)
		a.Equal(tt.expectedError, err)
	}
}

func TestMonitorType_Reset(t *testing.T) {
	a := assert.New(t)

	m := &MonitorType{ComponentName: fakeCompName}
	c := make(chan int)
	operation := func() error {
		defer func() { c <- 0 }()
		return nil
	}

	// Run the background goroutine, and block until it returns
	m.Run(operation)
	<-c

	res, _ := m.CheckResult()
	a.True(res)

	m.Reset()
	res, _ = m.CheckResult()
	a.False(res)
}

func TestMonitorType_ResetWhileRunning(t *testing.T) {
	a := assert.New(t)

	m := &MonitorType{ComponentName: fakeCompName}
	c := make(chan int)
	operation := func() error {
		<-c
		return nil
	}

	m.Run(operation)
	a.True(m.IsRunning())
	m.Reset()
	a.False(m.IsRunning())
	c <- 0
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
