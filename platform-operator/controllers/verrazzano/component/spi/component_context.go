// Copyright (c) 2021, 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package spi

// Default implementation of the ComponentContext interface

import (
	"github.com/verrazzano/verrazzano/pkg/log/vzlog"
	"github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1beta1"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/transform"
	"github.com/verrazzano/verrazzano/platform-operator/internal/config"
	clipkg "sigs.k8s.io/controller-runtime/pkg/client"
)

var _ ComponentContext = componentContext{}

// NewContext creates a ComponentContext from a raw CR
func NewContext(log vzlog.VerrazzanoLogger, c clipkg.Client, actualCR *v1alpha1.Verrazzano, actualV1beta1CR *v1beta1.Verrazzano, dryRun bool) (ComponentContext, error) {
	// Generate the effective CR based on the declared profile and any overrides in the user-supplied one
	effectiveCR, err := transform.GetEffectiveCR(actualCR)
	if err != nil {
		return nil, err
	}
	effectiveV1beta1CR, err := transform.GetEffectiveV1beta1CR(actualV1beta1CR)
	if err != nil {
		return nil, err
	}
	return componentContext{
		log:                log,
		client:             c,
		dryRun:             dryRun,
		cr:                 actualCR,
		effectiveCR:        effectiveCR,
		crv1beta1:          actualV1beta1CR,
		effectiveCRv1beta1: effectiveV1beta1CR,
	}, nil
}

// NewMinimalContext creates a ComponentContext limited to kubernetes client interactions and logging
func NewMinimalContext(c clipkg.Client, log vzlog.VerrazzanoLogger) (ComponentContext, error) {
	// Generate the effective CR based ond the declared profile and any overrides in the user-supplied one
	return componentContext{
		log:    log,
		client: c,
	}, nil
}

// NewFakeContext creates a fake ComponentContext for unit testing purposes
// c Kubernetes client
// actualCR The user-supplied Verrazzano CR
// dryRun Dry-run indicator
// profilesDir Optional override to the location of the profiles dir; if not provided, EffectiveCR == ActualCR
func NewFakeContext(c clipkg.Client, actualCR *v1alpha1.Verrazzano, actualV1beta1CR *v1beta1.Verrazzano, dryRun bool, profilesDir ...string) ComponentContext {
	effectiveCR := actualCR
	effectiveV1beta1CR := actualV1beta1CR
	log := vzlog.DefaultLogger()
	if len(profilesDir) > 0 {
		config.TestProfilesDir = profilesDir[0]
		log.Debugf("Profiles location: %s", config.TestProfilesDir)
		defer func() { config.TestProfilesDir = "" }()

		var err error
		effectiveCR, err = transform.GetEffectiveCR(actualCR)
		if err != nil {
			log.Errorf("Failed, unexpected error building fake context: %v", err)
			return nil
		}
		effectiveV1beta1CR, err = transform.GetEffectiveV1beta1CR(actualV1beta1CR)
		if err != nil {
			log.Errorf("Failed, unexpected error building fake context: %v", err)
			return nil
		}
	}

	if effectiveCR != nil && effectiveV1beta1CR == nil {
		effectiveV1beta1CR = &v1beta1.Verrazzano{}
		effectiveCR.ConvertTo(effectiveV1beta1CR)
	}
	if effectiveV1beta1CR != nil && effectiveCR == nil {
		effectiveCR = &v1alpha1.Verrazzano{}
		effectiveCR.ConvertFrom(effectiveV1beta1CR)
	}

	return componentContext{
		log:                log,
		client:             c,
		dryRun:             dryRun,
		cr:                 actualCR,
		effectiveCR:        effectiveCR,
		crv1beta1:          actualV1beta1CR,
		effectiveCRv1beta1: effectiveV1beta1CR,
		operation:          "",
		component:          "",
	}
}

type componentContext struct {
	// log logger for the execution context
	log vzlog.VerrazzanoLogger
	// client Kubernetes client
	client clipkg.Client
	// dryRun If true, do a dry run of operations
	dryRun bool
	// cr Represents the current v1alpha1.Verrazzano object state in the cluster
	cr *v1alpha1.Verrazzano
	// crv1beta1 Represents the current v1beta1.Verrazzano object state in the cluster
	crv1beta1 *v1beta1.Verrazzano
	// effectiveCR Represents the configuration resulting from any named profiles used and any configured overrides in the v1alpha1.Verrazzano resource
	effectiveCR *v1alpha1.Verrazzano
	// effectiveCRv1beta1 effectiveCR in v1beta1 form
	effectiveCRv1beta1 *v1beta1.Verrazzano
	// operation is the defined operation field for the logger. Defaults to nil if not present
	operation string
	// component is the defined component field for the logger. Defaults to nil if not present
	component string
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

func (c componentContext) ActualCR() *v1alpha1.Verrazzano {
	return c.cr
}

func (c componentContext) EffectiveCR() *v1alpha1.Verrazzano {
	return c.effectiveCR
}

func (c componentContext) ActualCRV1Beta1() *v1beta1.Verrazzano {
	return c.crv1beta1
}

func (c componentContext) EffectiveCRV1Beta1() *v1beta1.Verrazzano {
	return c.effectiveCRv1beta1
}

func (c componentContext) Copy() ComponentContext {
	return componentContext{
		log:         c.log,
		client:      c.client,
		dryRun:      c.dryRun,
		cr:          c.cr,
		effectiveCR: c.effectiveCR,
		operation:   c.operation,
		component:   c.component,
	}
}

// Init clones the component context, initializing the zap logger from the resource
// logger. This makes sure that we get the component context with the right logger.
func (c componentContext) Init(compName string) ComponentContext {
	// Get zap logger, add "with" field for this component name
	zapLogger := c.log.GetRootZapLogger().With("component", compName)

	// Ensure that there is a logger for this component, inject the new zap logger
	log := c.log.GetContext().EnsureLogger(compName, zapLogger, zapLogger)

	// c.log.
	return componentContext{
		log:                log,
		client:             c.client,
		dryRun:             c.dryRun,
		cr:                 c.cr,
		effectiveCR:        c.effectiveCR,
		crv1beta1:          c.crv1beta1,
		effectiveCRv1beta1: c.effectiveCRv1beta1,
		operation:          c.operation,
		component:          compName,
	}
}

func (c componentContext) Operation(op string) ComponentContext {
	// Get zap logger, add "with" field for this component name
	zapLogger := c.log.GetRootZapLogger().With("component", c.component).With("operation", op)
	
	// Ensure that there is a logger for this component, inject the new zap logger
	log := c.log.GetContext().EnsureLogger(c.component, zapLogger, zapLogger)

	c.log.SetZapLogger(zapLogger)

	return componentContext{
		log:         log,
		client:      c.client,
		dryRun:      c.dryRun,
		cr:          c.cr,
		effectiveCR: c.effectiveCR,
		operation:   op,
		component:   c.component,
	}
}

func (c componentContext) GetOperation() string {
	return c.operation
}

func (c componentContext) GetComponent() string {
	return c.component
}
