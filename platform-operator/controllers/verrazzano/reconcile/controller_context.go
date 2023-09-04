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
	LegacyUninstallPreWorkDone  atomic.Bool
	ModuleCreateOrUpdateDoneGen atomic.Int64
}

// SetLegacyUninstallPreWorkDone set the value of LegacyUninstallPreWorkDone
func SetLegacyUninstallPreWorkDone(val bool) {
	vzControllerContext.LegacyUninstallPreWorkDone.Store(true)
}

// IsLegacyUninstallPreWorkDone returns true if the Legacy uninstall prework is done
func IsLegacyUninstallPreWorkDone() bool {
	return vzControllerContext.LegacyUninstallPreWorkDone.Load()
}

// SetModuleCreateOrUpdateDoneGen set the value of ModuleCreateOrUpdateDoneGen
func SetModuleCreateOrUpdateDoneGen(gen int64) {
	vzControllerContext.ModuleCreateOrUpdateDoneGen.Store(gen)
}

// IsModuleCreateOrUpdateDoneGen returns the generation for ModuleCreateOrUpdateDoneGen
func GetModuleCreateOrUpdateDoneGen() int64 {
	return vzControllerContext.ModuleCreateOrUpdateDoneGen.Load()
}
