// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
package common

import (
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
)

func CheckConditionType(currentCondition vzapi.ConditionType) vzapi.CompStateType {
	switch currentCondition {
	case vzapi.CondPreInstall:
		return vzapi.CompStatePreInstalling
	case vzapi.CondInstallStarted:
		return vzapi.CompStateInstalling
	case vzapi.CondUninstallStarted:
		return vzapi.CompStateUninstalling
	case vzapi.CondUpgradeStarted:
		return vzapi.CompStateUpgrading
	case vzapi.CondUpgradeComplete:
		return vzapi.CompStateReady
	case vzapi.CondInstallFailed, vzapi.CondUpgradeFailed, vzapi.CondUninstallFailed:
		return vzapi.CompStateFailed
	}
	// Return ready for vzapi.InstallComplete, vzapi.UpgradeComplete
	return vzapi.CompStateReady
}
