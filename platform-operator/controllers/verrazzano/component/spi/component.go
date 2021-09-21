// Copyright (c) 2020, 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package spi

import (
	"go.uber.org/zap"
	clipkg "sigs.k8s.io/controller-runtime/pkg/client"
)

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
	PreUpgrade(log *zap.SugaredLogger, client clipkg.Client, namespace string, dryRun bool) error

	// Upgrade will upgrade the Verrazzano component specified in the CR.Version field
	Upgrade(log *zap.SugaredLogger, client clipkg.Client, namespace string, dryRun bool) error

	// PostUpgrade allows components to perform any post-processing required after upgrading
	PostUpgrade(log *zap.SugaredLogger, client clipkg.Client, namespace string, dryRun bool) error

	// PreInstall allows components to perform any pre-processing required prior to initial install
	PreInstall(log *zap.SugaredLogger, client clipkg.Client, namespace string, dryRun bool) error

	// Install performs the initial install of a component
	Install(log *zap.SugaredLogger, client clipkg.Client, namespace string, dryRun bool) error

	// PostInstall allows components to perform any post-processing required after initial install
	PostInstall(log *zap.SugaredLogger, client clipkg.Client, namespace string, dryRun bool) error

	// IsInstalled Indicates whether or not the component is installed
	IsInstalled(log *zap.SugaredLogger, client clipkg.Client, namespace string) (bool, error)

	// IsReady Indicates whether or not a component is available and ready
	IsReady(log *zap.SugaredLogger, client clipkg.Client, namespace string) bool
}
