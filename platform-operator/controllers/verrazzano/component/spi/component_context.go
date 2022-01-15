// Copyright (c) 2021, 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
package spi

// Default implementation of the ComponentContext interface

import (
	"strings"

	"github.com/verrazzano/verrazzano/pkg/log/vzlog"

	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/transform"
	"github.com/verrazzano/verrazzano/platform-operator/internal/config"
	clipkg "sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	// implicit base profile (defaults)
	baseProfile = "base"
)

var _ ComponentContext = &componentContext{}

// NewContext creates a ComponentContext from a raw CR
func NewContext(log vzlog.VerrazzanoLogger, c clipkg.Client, actualCR *vzapi.Verrazzano, registry ComponentRegistry, dryRun bool) (ComponentContext, error) {
	// Generate the effective CR based ond the declared profile and any overrides in the user-supplied one
	effectiveCR, err := getEffectiveCR(actualCR)
	if err != nil {
		return nil, err
	}
	return &componentContext{
		log:               log,
		client:            c,
		dryRun:            dryRun,
		cr:                actualCR,
		effectiveCR:       effectiveCR,
		componentRegistry: registry,
	}, nil
}

// NewFakeContext creates a fake ComponentContext for unit testing purposes without a Registry
func NewFakeContext(c clipkg.Client, actualCR *vzapi.Verrazzano, dryRun bool, profilesDir ...string) ComponentContext {
	return NewFakeContextWithRegistry(c, actualCR, &fakeRegistry{}, dryRun, profilesDir...)
}

// NewFakeContext creates a fake ComponentContext for unit testing purposes
// c Kubernetes client
// actualCR The user-supplied Verrazzano CR
// dryRun Dry-run indicator
// profilesDir Optional override to the location of the profiles dir; if not provided, EffectiveCR == ActualCR
func NewFakeContextWithRegistry(c clipkg.Client, actualCR *vzapi.Verrazzano, registry ComponentRegistry, dryRun bool, profilesDir ...string) ComponentContext {
	effectiveCR := actualCR
	log := vzlog.DefaultLogger()
	if len(profilesDir) > 0 {
		config.TestProfilesDir = profilesDir[0]
		log.Debugf("Profiles location: %s", config.TestProfilesDir)
		defer func() { config.TestProfilesDir = "" }()

		var err error
		effectiveCR, err = getEffectiveCR(actualCR)
		if err != nil {
			log.Errorf("Failed, unexpected error building fake context: %v", err)
			return nil
		}
	}
	return &componentContext{
		log:               log,
		client:            c,
		dryRun:            dryRun,
		cr:                actualCR,
		effectiveCR:       effectiveCR,
		componentRegistry: registry,
		operation:         "",
		component:         "",
	}
}

// getEffectiveCR Creates an "effective" Verrazzano CR based on the user defined resource merged with the profile definitions
// - Effective CR == base profile + declared profiles + ActualCR (in order)
// - last definition wins
func getEffectiveCR(actualCR *vzapi.Verrazzano) (*vzapi.Verrazzano, error) {
	if actualCR == nil {
		return nil, nil
	}
	// Identify the set of profiles, base + declared
	profiles := []string{baseProfile, string(vzapi.Prod)}
	if len(actualCR.Spec.Profile) > 0 {
		profiles = append([]string{baseProfile}, strings.Split(string(actualCR.Spec.Profile), ",")...)
	}
	var profileFiles []string
	for _, profile := range profiles {
		profileFiles = append(profileFiles, config.GetProfile(profile))
	}
	// Merge the profile files into an effective profile YAML string
	effectiveCR, err := transform.MergeProfiles(actualCR, profileFiles...)
	if err != nil {
		return nil, err
	}
	effectiveCR.Status = vzapi.VerrazzanoStatus{} // Don't replicate the CR status in the effective config
	// if Certificate in CertManager is empty, set it to default CA
	var emptyCertConfig = vzapi.Certificate{}
	if effectiveCR.Spec.Components.CertManager.Certificate == emptyCertConfig {
		effectiveCR.Spec.Components.CertManager.Certificate.CA = vzapi.CA{
			SecretName:               "verrazzano-ca-certificate-secret",
			ClusterResourceNamespace: "cert-manager",
		}
	}
	return effectiveCR, nil
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
	// component registry
	componentRegistry ComponentRegistry
}

func (c *componentContext) Log() vzlog.VerrazzanoLogger {
	return c.log
}

func (c *componentContext) Client() clipkg.Client {
	return c.client
}

func (c *componentContext) IsDryRun() bool {
	return c.dryRun
}

func (c *componentContext) ActualCR() *vzapi.Verrazzano {
	return c.cr
}

func (c *componentContext) EffectiveCR() *vzapi.Verrazzano {
	return c.effectiveCR
}

func (c *componentContext) Copy() ComponentContext {
	return &componentContext{
		log:               c.log,
		client:            c.client,
		dryRun:            c.dryRun,
		cr:                c.cr,
		effectiveCR:       c.effectiveCR,
		operation:         c.operation,
		component:         c.component,
		componentRegistry: c.componentRegistry,
	}
}

// Clone the component context, initializing the zap logger from the resource
// logger. This makes sure that we get the
func (c *componentContext) Init(compName string) ComponentContext {
	// Get zap logger, add "with" field for this component name
	zapLogger := c.log.GetRootZapLogger().With("component", compName)
	// Ensure that there is a logger for this component, inject the new zap logger
	log := c.log.GetContext().EnsureLogger(compName, zapLogger, zapLogger)

	return &componentContext{
		log:               log,
		client:            c.client,
		dryRun:            c.dryRun,
		cr:                c.cr,
		effectiveCR:       c.effectiveCR,
		operation:         c.operation,
		component:         compName,
		componentRegistry: c.componentRegistry,
	}
}

func (c *componentContext) Operation(op string) ComponentContext {
	// Get zap logger, add "with" field for this component operation
	zapLogger := c.log.GetZapLogger().With("operation", op)
	c.log.SetZapLogger(zapLogger)

	return &componentContext{
		log:               c.log,
		client:            c.client,
		dryRun:            c.dryRun,
		cr:                c.cr,
		effectiveCR:       c.effectiveCR,
		operation:         op,
		component:         c.component,
		componentRegistry: c.componentRegistry,
	}
}

func (c *componentContext) GetComponentRegistry() ComponentRegistry {
	return c.componentRegistry
}

func (c *componentContext) GetOperation() string {
	return c.operation
}

func (c *componentContext) GetComponent() string {
	return c.component
}

// For unit testing
type fakeRegistry struct {
	components []Component
}

func (f *fakeRegistry) GetComponents() []Component {
	return []Component{}
}

func (f *fakeRegistry) FindComponent(releaseName string) (bool, Component) {
	for _, comp := range f.components {
		if comp.Name() == releaseName {
			return true, comp
		}
	}
	return false, nil
}

var _ ComponentRegistry = &fakeRegistry{}
