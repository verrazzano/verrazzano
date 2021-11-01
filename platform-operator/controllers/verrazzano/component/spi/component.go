// Copyright (c) 2020, 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package spi

import (
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"go.uber.org/zap"
	clipkg "sigs.k8s.io/controller-runtime/pkg/client"
)

// ComponentContext Defines the context objects required for Component operations
type ComponentContext interface {
	// Log returns the logger for the context
	Log() *zap.SugaredLogger
	// GetClient returns the controller client for the context
	Client() clipkg.Client
	// ActualCR returns the actual unmerged Verrazzano resource
	ActualCR() *vzapi.Verrazzano
	// EffectiveCR returns the effective merged Verrazzano CR
	EffectiveCR() *vzapi.Verrazzano
	// IsDryRun indicates the component context is in DryRun mode
	IsDryRun() bool
	// Copy returns a copy of the current context
	Copy() ComponentContext
}

// ComponentInfo interface defines common information and metadata about components
type ComponentInfo interface {
	// Name returns the name of the Verrazzano component
	Name() string
	// GetDependencies returns the dependencies of this component
	GetDependencies() []string
	// IsReady Indicates whether or not a component is available and ready
	IsReady(context ComponentContext) bool
	// IsEnabled Indicates whether or a component is enabled for installation
	IsEnabled(context ComponentContext) bool
	// GetMinVerrazzanoVersion returns the minimum Verrazzano version required by the component
	GetMinVerrazzanoVersion() string
}

// ComponentInstaller interface defines installs operations for components that support it
type ComponentInstaller interface {
	// IsOperatorInstallSupported Returns true if the component supports install directly via the platform operator
	// - scaffolding while we move components from the scripts to the operator
	IsOperatorInstallSupported() bool
	// IsInstalled Indicates whether or not the component is installed
	IsInstalled(context ComponentContext) (bool, error)
	// PreInstall allows components to perform any pre-processing required prior to initial install
	PreInstall(context ComponentContext) error
	// Install performs the initial install of a component
	Install(context ComponentContext) error
	// PostInstall allows components to perform any post-processing required after initial install
	PostInstall(context ComponentContext) error
}

// ComponentUpgrader interface defines upgrade operations for components that support it
type ComponentUpgrader interface {
	// PreUpgrade allows components to perform any pre-processing required prior to upgrading
	PreUpgrade(context ComponentContext) error
	// Upgrade will upgrade the Verrazzano component specified in the CR.Version field
	Upgrade(context ComponentContext) error
	// PostUpgrade allows components to perform any post-processing required after upgrading
	PostUpgrade(context ComponentContext) error
}

// Component interface defines the methods implemented by components
type Component interface {
	ComponentInfo
	ComponentInstaller
	ComponentUpgrader
}
