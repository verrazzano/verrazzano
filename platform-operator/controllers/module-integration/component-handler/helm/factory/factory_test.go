// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package factory

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestNewModuleHandlerInfo tests the NewModuleHandlerInfo function
func TestNewModuleHandlerInfo(t *testing.T) {
	asserts := assert.New(t)
	info := NewModuleHandlerInfo()

	// GIVEN a call to NewModuleHandlerInfo
	// WHEN each of the action handlers are inspected
	// THEN the handlers return the expected work names
	asserts.Equal("install", info.InstallActionHandler.GetWorkName())
	asserts.Equal("uninstall", info.DeleteActionHandler.GetWorkName())
	asserts.Equal("update", info.UpdateActionHandler.GetWorkName())
	asserts.Equal("upgrade", info.UpgradeActionHandler.GetWorkName())
}
