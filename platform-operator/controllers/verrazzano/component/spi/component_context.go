// Copyright (c) 2021, 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
package spi

// Default implementation of the ComponentContext interface

import (
	vzctx "github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/context"

	"github.com/verrazzano/verrazzano/pkg/log/vzlog"

	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	clipkg "sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	// implicit base profile (defaults)
	baseProfile = "base"
)

var _ ComponentContext = componentContext{}

// componentContext has the context needed to reconcile a component
type componentContext struct {
	// log logger for the execution context
	log vzlog.VerrazzanoLogger
	// client Kubernetes client
	client clipkg.Client
	// dryRun If true, do a dry run of operations
	dryRun bool
	// cr Represents the current Verrazzano object state in the cluster
	cr *vzapi.Verrazzano
	// effectiveCR Represents the configuration resulting from any named profiles used and any configured overrides in the CR
	effectiveCR *vzapi.Verrazzano
	// operation is the defined operation field for the logger. Defaults to nil if not present
	operation string
	// componentName is the defined componentName field for the logger. Defaults to nil if not present
	componentName string
}

// NewComponentContext creates a ComponentContext
// func NewComponentContext(log vzlog.VerrazzanoLogger, c clipkg.Client, effectiveCR *vzapi.Verrazzano, dryRun bool) ComponentContext
func NewComponentContext(vzContext *vzctx.VerrazzanoContext, compName string, operation string) ComponentContext {
	log := vzContext.Log
	if len(operation) > 0 {
		// Get zap logger, add "with" field for this componentName name and operator
		zapLogger := vzContext.Log.GetRootZapLogger().With("componentName", len(compName))
		zapLogger = zapLogger.With("operation", operation)

		// Ensure that there is a logger for this componentName, inject the new zap logger
		log = vzContext.Log.GetContext().EnsureLogger(compName, zapLogger, zapLogger)
	}

	return componentContext{
		componentName: compName,
		log:           log,
		client:        vzContext.Client,
		dryRun:        vzContext.DryRun,
		cr:            vzContext.ActualCR,
		effectiveCR:   vzContext.EffectiveCR,
	}
}

func (c componentContext) Log() vzlog.VerrazzanoLogger {
	return c.log
}

func (c componentContext) Client() clipkg.Client {
	return c.client
}

func (c componentContext) IsDryRun() bool {
	return c.dryRun
}

func (c componentContext) ActualCR() *vzapi.Verrazzano {
	return c.cr
}

func (c componentContext) EffectiveCR() *vzapi.Verrazzano {
	return c.effectiveCR
}

func (c componentContext) Copy() ComponentContext {
	return componentContext{
		log:           c.log,
		client:        c.client,
		dryRun:        c.dryRun,
		cr:            c.cr,
		effectiveCR:   c.effectiveCR,
		operation:     c.operation,
		componentName: c.componentName,
	}
}

func (c componentContext) GetOperation() string {
	return c.operation
}

func (c componentContext) GetComponent() string {
	return c.componentName
}
