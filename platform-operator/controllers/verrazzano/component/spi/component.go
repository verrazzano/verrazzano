// Copyright (c) 2020, 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package spi

import (
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"go.uber.org/zap"
	clipkg "sigs.k8s.io/controller-runtime/pkg/client"
)

// ComponentContext Defines the context objects required for Component operations
type ComponentContext struct {
	// Client Kubernetes client
	Client clipkg.Client
	// DryRun If true, do a dry run of operations
	DryRun bool
	// Config Represents the current Verrazzano object state
	Config *vzapi.Verrazzano
	// EffectiveConfig Represents the actual configuration resulting from the named profiles
	// used and any configured overrides
	EffectiveConfig *vzapi.Verrazzano
}

// Component interface defines the methods implemented by components
type Component interface {
	// Name returns the name of the Verrazzano component
	Name() string

	// IsOperatorInstallSupported Returns true if the component supports install directly via the platform operator
	// - scaffolding while we move components from the scripts to the operator
	IsOperatorInstallSupported() bool

	// GetDependencies returns the dependencies of this component
	GetDependencies() []string

	// PreUpgrade allows components to perform any pre-processing required prior to upgrading
	PreUpgrade(log *zap.SugaredLogger, context *ComponentContext) error

	// Upgrade will upgrade the Verrazzano component specified in the CR.Version field
	Upgrade(log *zap.SugaredLogger, context *ComponentContext) error

	// PostUpgrade allows components to perform any post-processing required after upgrading
	PostUpgrade(log *zap.SugaredLogger, context *ComponentContext) error

	// PreInstall allows components to perform any pre-processing required prior to initial install
	PreInstall(log *zap.SugaredLogger, context *ComponentContext) error

	// Install performs the initial install of a component
	Install(log *zap.SugaredLogger, context *ComponentContext) error

	// PostInstall allows components to perform any post-processing required after initial install
	PostInstall(log *zap.SugaredLogger, context *ComponentContext) error

	// IsInstalled Indicates whether or not the component is installed
	IsInstalled(log *zap.SugaredLogger, context *ComponentContext) (bool, error)

	// IsReady Indicates whether or not a component is available and ready
	IsReady(log *zap.SugaredLogger, context *ComponentContext) bool
}
