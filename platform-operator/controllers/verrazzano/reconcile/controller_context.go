// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package reconcile

import (
	"sync/atomic"
)

var vzControllerContext VerrazzanoControllerContext

// VerrazzanoControllerContext is used to synchronize the two Verrazzano controllers, the legacy controller and
// the module-based controller.  This will be removed when we finally have a single controller.
type VerrazzanoControllerContext struct {
	LegacyUninstallPreWorkDone atomic.Bool
	ModuleCreateOrUpdateDone   atomic.Bool
	ModuleUninstallDone        atomic.Bool
}

// SetModuleCreateOrUpdateDone sets the value of ModuleCreateOrUpdateDone
func SetModuleCreateOrUpdateDone(val bool) {
	vzControllerContext.ModuleCreateOrUpdateDone.Store(val)
}

// IsModuleCreateOrUpdateDone returns true if the Module createOrUpdate is done
func IsModuleCreateOrUpdateDone() bool {
	return vzControllerContext.ModuleCreateOrUpdateDone.Load()
}

// SetModuleUninstallDone set the value of ModuleUninstallDone
func SetModuleUninstallDone(val bool) {
	vzControllerContext.ModuleUninstallDone.Store(true)
}

// IsModuleUninstallDone returns true if the Module uninstall is done
func IsModuleUninstallDone() bool {
	return vzControllerContext.ModuleUninstallDone.Load()
}

// SetLegacyUninstallPreWorkDone set the value of LegacyUninstallPreWorkDone
func SetLegacyUninstallPreWorkDone(val bool) {
	vzControllerContext.LegacyUninstallPreWorkDone.Store(true)
}

// IsLegacyUninstallPreWorkDone returns true if the Legacy uninstall prework is done
func IsLegacyUninstallPreWorkDone() bool {
	return vzControllerContext.LegacyUninstallPreWorkDone.Load()
}
