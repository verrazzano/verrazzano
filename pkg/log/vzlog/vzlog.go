// Copyright (c) 2022, 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package vzlog

import (
	modulelog "github.com/verrazzano/verrazzano-modules/pkg/vzlog"

	"go.uber.org/zap"
)

// ResourceConfig is the configuration of a logger for a resource that is being reconciled
type ResourceConfig struct {
	// Name is the name of the resource
	Name string

	// Namespace is the namespace of the resource
	Namespace string

	// ID is the resource uid
	ID string

	// Generation is the resource generation
	Generation int64

	// Controller name is the name of the controller
	ControllerName string
}

// VerrazzanoLogger is a logger interface that provides sugared and progress logging
type VerrazzanoLogger modulelog.VerrazzanoLogger

// DefaultLogger ensures the default logger exists.  This is typically used for testing
func DefaultLogger() VerrazzanoLogger {
	return modulelog.DefaultLogger()
}

// EnsureResourceLogger ensures that a logger exists for a specific generation of a Kubernetes resource.
// When a resource is getting reconciled, the status may frequently get updated during
// the reconciliation.  This is the case for the Verrazzano resource.  As a result,
// the controller-runtime queue gets filled with updated instances of a resource that
// have the same generation. The side-effect is that after a resource is completely reconciled,
// the controller Reconcile method may still be called many times. In this case, the existing
// context must be used so that 'once' and 'progress' messages don't start from a new context,
// causing them to be displayed when they shouldn't.  This mehod ensures that the same
// logger is used for a given resource and generation.
func EnsureResourceLogger(config *ResourceConfig) (VerrazzanoLogger, error) {
	return modulelog.EnsureResourceLogger(copyModuleConfig(config))
}

func ForZapLogger(config *ResourceConfig, zaplog *zap.SugaredLogger) VerrazzanoLogger {
	return modulelog.ForZapLogger(copyModuleConfig(config), zaplog)
}

func copyModuleConfig(conf *ResourceConfig) *modulelog.ResourceConfig {
	return &modulelog.ResourceConfig{
		Name:           conf.Name,
		Namespace:      conf.Namespace,
		ID:             conf.ID,
		Generation:     conf.Generation,
		ControllerName: conf.ControllerName,
	}
}
