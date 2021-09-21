// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
package spi

// Default implementation of the ComponentContext interface

import (
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"go.uber.org/zap"
	clipkg "sigs.k8s.io/controller-runtime/pkg/client"
)

// NewContext creates a ComponentContext from a raw CR
func NewContext(log *zap.SugaredLogger, c clipkg.Client, cr *vzapi.Verrazzano, dryRun bool) ComponentContext {
	return componentContext{
		log:         log,
		client:      c,
		dryRun:      dryRun,
		cr:          cr,
		effectiveCR: cr, // Eventually we will compute the merged effective CR
	}
}

type componentContext struct {
	// log logger for the execution context
	log *zap.SugaredLogger
	// client Kubernetes client
	client clipkg.Client
	// dryRun If true, do a dry run of operations
	dryRun bool
	// cr Represents the current Verrazzano object state in the cluster
	cr *vzapi.Verrazzano
	// effectiveCR Represents the configuration resulting from any named profiles used and any configured overrides in the CR
	effectiveCR *vzapi.Verrazzano
}

func (c componentContext) Log() *zap.SugaredLogger {
	return c.log
}

func (c componentContext) GetClient() clipkg.Client {
	return c.client
}

func (c componentContext) IsDryRun() bool {
	return c.dryRun
}

func (c componentContext) GetCR() *vzapi.Verrazzano {
	return c.cr
}

func (c componentContext) GetEffectiveCR() *vzapi.Verrazzano {
	return c.effectiveCR
}

func (c componentContext) Copy() ComponentContext {
	return componentContext{
		log:         c.log,
		client:      c.client,
		dryRun:      c.dryRun,
		cr:          c.cr,
		effectiveCR: c.effectiveCR,
	}
}
