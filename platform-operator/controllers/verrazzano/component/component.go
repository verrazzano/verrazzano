// Copyright (c) 2020, 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package component

import (
	"go.uber.org/zap"
	clipkg "sigs.k8s.io/controller-runtime/pkg/client"
)

// Component interface defines the methods implemented by components
type Component interface {
	// Name returns the name of the Verrazzano component
	Name() string

	// Upgrade will upgrade the Verrazzano component specified in the CR.Version field
	Upgrade(log *zap.SugaredLogger, client clipkg.Client, namespace string, dryRun bool) error

	// Install performs the initial install of a component
	Install(log *zap.SugaredLogger, client clipkg.Client, namespace string, dryRun bool) error

	// IsOperatorInstallSupported Returns true if the component supports install directly via the platform operator
	// - scaffolding while we move components from the scripts to the operator
	IsOperatorInstallSupported() bool

	// IsInstalled Indicates whether or not the component is installed
	IsInstalled() bool
}
