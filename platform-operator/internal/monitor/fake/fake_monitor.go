// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package fake

import "github.com/verrazzano/verrazzano/platform-operator/internal/monitor"

// FakeMonitorType - a fake monitor object, useful for unit testing.
type FakeBackgroundProcessMonitorType struct {
	Result  bool
	Err     error
	Running bool
}

func (f *FakeBackgroundProcessMonitorType) CheckResult() (bool, error)           { return f.Result, f.Err }
func (f *FakeBackgroundProcessMonitorType) Reset()                               {}
func (f *FakeBackgroundProcessMonitorType) IsRunning() bool                      { return f.Running }
func (f *FakeBackgroundProcessMonitorType) Run(operation monitor.BackgroundFunc) {}

// Check that &FakeMonitorType implements BackgroundProcessMonitor
var _ monitor.BackgroundProcessMonitor = &FakeBackgroundProcessMonitorType{}
