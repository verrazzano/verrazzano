// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package reconcile

import "sync"

var vzControllerContextMutex sync.Mutex
var vzControllerContext *VerrazzanoControllerContext

// VerrazzanoControllerContext is used to synchronize the two Verrazzano controllers, the legacy controller and
// the module-based controller.  This will be removed when we finally have a single controller.
type VerrazzanoControllerContext struct {
	ModuleUninstallDone bool
}

// SetModuleUninstallDone returns true if the Module uninstall is not done
func SetModuleUninstallDone() {
	vzControllerContextMutex.Lock()
	defer vzControllerContextMutex.Unlock()
	if vzControllerContext == nil {
		vzControllerContext = &VerrazzanoControllerContext{}
	}
	vzControllerContext.ModuleUninstallDone = true
}

// IsModuleUninstallDone returns true if the Module uninstall is done
func IsModuleUninstallDone() bool {
	vzControllerContextMutex.Lock()
	defer vzControllerContextMutex.Unlock()
	if vzControllerContext == nil {
		vzControllerContext = &VerrazzanoControllerContext{}
	}
	return vzControllerContext.ModuleUninstallDone
}

// ClearControllerContext clears the controller context
func ClearControllerContext() {
	vzControllerContextMutex.Lock()
	defer vzControllerContextMutex.Unlock()
	vzControllerContext = nil
}
