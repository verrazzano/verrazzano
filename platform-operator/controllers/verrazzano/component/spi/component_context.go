// Copyright (c) 2021, 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
package spi

// Default implementation of the ComponentContext interface

import (
	"crypto/sha256"
	"fmt"

	"github.com/verrazzano/verrazzano/pkg/log/vzlog"
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/transform"
	"github.com/verrazzano/verrazzano/platform-operator/internal/config"
	clipkg "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/yaml"
)

var _ ComponentContext = componentContext{}

// NewContext creates a ComponentContext from a raw CR
func NewContext(log vzlog.VerrazzanoLogger, c clipkg.Client, actualCR *vzapi.Verrazzano, dryRun bool) (ComponentContext, error) {

	// Generate the effective CR based ond the declared profile and any overrides in the user-supplied one
	effectiveCR, err := transform.GetEffectiveCR(actualCR)
	if err != nil {
		return nil, err
	}
	return componentContext{
		log:             log,
		client:          c,
		dryRun:          dryRun,
		cr:              actualCR,
		effectiveCR:     effectiveCR,
		effectiveConfig: make(map[string]interface{}),
	}, nil
}

// NewFakeContext creates a fake ComponentContext for unit testing purposes
// c Kubernetes client
// actualCR The user-supplied Verrazzano CR
// dryRun Dry-run indicator
// profilesDir Optional override to the location of the profiles dir; if not provided, EffectiveCR == ActualCR
func NewFakeContext(c clipkg.Client, actualCR *vzapi.Verrazzano, dryRun bool, profilesDir ...string) ComponentContext {
	effectiveCR := actualCR
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
	}
	return componentContext{
		log:             log,
		client:          c,
		dryRun:          dryRun,
		cr:              actualCR,
		effectiveCR:     effectiveCR,
		operation:       "",
		component:       "",
		effectiveConfig: make(map[string]interface{}),
	}
}

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
	// component is the defined component field for the logger. Defaults to nil if not present
	component string
	// cache of effective config of each component
	effectiveConfig map[string]interface{}
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
		log:             c.log,
		client:          c.client,
		dryRun:          c.dryRun,
		cr:              c.cr,
		effectiveCR:     c.effectiveCR,
		operation:       c.operation,
		component:       c.component,
		effectiveConfig: c.effectiveConfig,
	}
}

// Clone the component context, initializing the zap logger from the resource
// logger. This makes sure that we get the
func (c componentContext) Init(compName string) ComponentContext {
	// Get zap logger, add "with" field for this component name
	zapLogger := c.log.GetRootZapLogger().With("component", compName)
	// Ensure that there is a logger for this component, inject the new zap logger
	log := c.log.GetContext().EnsureLogger(compName, zapLogger, zapLogger)

	// c.log.
	return componentContext{
		log:             log,
		client:          c.client,
		dryRun:          c.dryRun,
		cr:              c.cr,
		effectiveCR:     c.effectiveCR,
		operation:       c.operation,
		component:       compName,
		effectiveConfig: c.effectiveConfig,
	}
}

func (c componentContext) Operation(op string) ComponentContext {
	// Get zap logger, add "with" field for this component operation
	zapLogger := c.log.GetZapLogger().With("operation", op)
	c.log.SetZapLogger(zapLogger)

	return componentContext{
		log:             c.log,
		client:          c.client,
		dryRun:          c.dryRun,
		cr:              c.cr,
		effectiveCR:     c.effectiveCR,
		operation:       op,
		component:       c.component,
		effectiveConfig: c.effectiveConfig,
	}
}

func (c componentContext) GetOperation() string {
	return c.operation
}

func (c componentContext) GetComponent() string {
	return c.component
}

func (c componentContext) GetConfigHashByJSONName(name string) string {
	if len(c.effectiveConfig) == 0 {
		data, _ := yaml.Marshal(c.effectiveCR.Spec.Components)
		yaml.Unmarshal(data, &c.effectiveConfig)
	}
	config := c.effectiveConfig[name]
	if config == nil {
		return ""
	}
	return HashSum(config)
}

// HashSum returns the hash sum of the config object
func HashSum(config ...interface{}) string {
	sha := sha256.New()
	for _, c := range config {
		if data, err := yaml.Marshal(c); err == nil {
			sha.Write(data)
		}
	}
	return fmt.Sprintf("%x", sha.Sum(nil))
}
