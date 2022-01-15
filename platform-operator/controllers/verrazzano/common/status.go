// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
package common

import (
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
)

func CheckConditionType(currentCondition vzapi.ConditionType) vzapi.StateType {
	switch currentCondition {
	case vzapi.PreInstall:
		return vzapi.PreInstalling
	case vzapi.InstallStarted:
		return vzapi.Installing
	case vzapi.UninstallStarted:
		return vzapi.Uninstalling
	case vzapi.UpgradeStarted:
		return vzapi.Upgrading
	case vzapi.UninstallComplete:
		return vzapi.Ready
	case vzapi.InstallFailed, vzapi.UpgradeFailed, vzapi.UninstallFailed:
		return vzapi.Failed
	}
	// Return ready for vzapi.InstallComplete, vzapi.UpgradeComplete
	return vzapi.Ready
}
