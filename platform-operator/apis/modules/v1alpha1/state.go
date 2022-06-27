// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package v1alpha1

// ModuleState describes the current reconciling stage of a Module
type ModuleState string

const (
	StatePreinstall   ModuleState = "PreInstalling"
	StateInstalling   ModuleState = "Installing"
	StateUninstalling ModuleState = "Uninstalling"
	StateReady        ModuleState = "Ready"
	StatePreUpgrade   ModuleState = "PreUpgrading"
	StateUpgrading    ModuleState = "Upgrading"
)

type ModuleCondition string

const (
	ConditionArrayLimit = 5

	CondPreInstall      ModuleCondition = "PreInstall"
	CondInstallStarted  ModuleCondition = "InstallStarted"
	CondInstallComplete ModuleCondition = "InstallComplete"
	CondUninstall       ModuleCondition = "Uninstall"
	CondPreUpgrade      ModuleCondition = "PreUpgrade"
	CondUpgradeStarted  ModuleCondition = "UpgradeStarted"
	CondUpgradeComplete ModuleCondition = "UpgradeComplete"
)

func (m *Module) SetState(state ModuleState) {
	m.Status.State = &state
}

func State(condition ModuleCondition) ModuleState {
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
	default: // CondUpgradeComplete, CondInstallComplete
		return StateReady
	}
}
