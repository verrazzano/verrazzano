// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package fake

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFakeMonitorType_CheckResult(t *testing.T) {
	a := assert.New(t)

	f := &FakeBackgroundProcessMonitorType{Result: true, Err: nil}
	res, err := f.CheckResult()
	a.True(res)
	a.NoError(err)

	f = &FakeBackgroundProcessMonitorType{Result: false, Err: fmt.Errorf("an unexpected error")}
	res, err = f.CheckResult()
	a.False(res)
	a.Error(err)
}

func TestFakeMonitorType_IsRunning(t *testing.T) {
	a := assert.New(t)

	f := &FakeBackgroundProcessMonitorType{Running: true}
	a.True(f.IsRunning())

	f = &FakeBackgroundProcessMonitorType{Running: false}
	a.False(f.IsRunning())
}
