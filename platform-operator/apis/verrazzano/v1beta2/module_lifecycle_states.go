// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package v1beta2

// ModuleLifecycleState describes the current reconciling stage of a ModuleLifecycle instance
type ModuleLifecycleState string

const (
	StatePreinstall   ModuleLifecycleState = "PreInstalling"
	StateInstalling   ModuleLifecycleState = "Installing"
	StateUninstalling ModuleLifecycleState = "Uninstalling"
	StateReady        ModuleLifecycleState = "Ready"
	StatePreUpgrade   ModuleLifecycleState = "PreUpgrading"
	StateUpgrading    ModuleLifecycleState = "Upgrading"
	StateFailed       ModuleLifecycleState = "Failed"
)

type LifecycleCondition string

const (
	ConditionArrayLimit = 5

	CondPreInstall      LifecycleCondition = "PreInstall"
	CondInstallStarted  LifecycleCondition = "InstallStarted"
	CondInstallComplete LifecycleCondition = "InstallComplete"
	CondUninstall       LifecycleCondition = "Uninstall"
	CondPreUpgrade      LifecycleCondition = "PreUpgrade"
	CondUpgradeStarted  LifecycleCondition = "UpgradeStarted"
	CondUpgradeComplete LifecycleCondition = "UpgradeComplete"
	CondFailed          LifecycleCondition = "Failed"
)

func (m *ModuleLifecycle) SetState(state ModuleLifecycleState) {
	m.Status.State = state
}

func LifecycleState(condition LifecycleCondition) ModuleLifecycleState {
	switch condition {
	case CondPreInstall:
		return StatePreinstall
	case CondInstallStarted:
		return StateInstalling
	case CondUninstall:
		return StateUninstalling
	case CondPreUpgrade:
		return StatePreUpgrade
	case CondUpgradeStarted:
		return StateUpgrading
	case CondFailed:
		return StateFailed
	default: // CondUpgradeComplete, CondInstallComplete
		return StateReady
	}
}
