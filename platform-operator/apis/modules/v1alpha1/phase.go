// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package v1alpha1

// ModulePhase describes the current reconciling stage of a Module
type ModulePhase string

const (
	PhasePreinstall   ModulePhase = "PreInstalling"
	PhaseInstalling   ModulePhase = "Installing"
	PhaseUninstalling ModulePhase = "Uninstalling"
	PhaseReady        ModulePhase = "Ready"
	PhasePreUpgrade   ModulePhase = "PreUpgrading"
	PhaseUpgrading    ModulePhase = "Upgrading"
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

func (m *Module) SetPhase(phase ModulePhase) {
	m.Status.Phase = &phase
}

func Phase(condition ModuleCondition) ModulePhase {
	switch condition {
	case CondPreInstall:
		return PhasePreinstall
	case CondInstallStarted:
		return PhaseInstalling
	case CondUninstall:
		return PhaseUninstalling
	case CondPreUpgrade:
		return PhasePreUpgrade
	case CondUpgradeStarted:
		return PhaseUpgrading
	default: // CondUpgradeComplete, CondInstallComplete
		return PhaseReady
	}
}
