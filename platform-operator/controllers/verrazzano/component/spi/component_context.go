// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
package spi

// Default implementation of the ComponentContext interface

import (
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/internal/config"
	vzyaml "github.com/verrazzano/verrazzano/platform-operator/internal/yaml"
	"go.uber.org/zap"
	clipkg "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/yaml"
	"strings"
)

const (
	// implicit base profile (defaults)
	baseProfile = "base"
)

// NewContext creates a ComponentContext from a raw CR
func NewContext(log *zap.SugaredLogger, c clipkg.Client, actualCR *vzapi.Verrazzano, dryRun bool) (ComponentContext, error) {
	// Generate the effective CR based ond the declared profile and any overrides in the user-supplied one
	effectiveCR, err := getEffectiveCR(actualCR)
	if err != nil {
		return nil, err
	}
	return componentContext{
		log:         log,
		client:      c,
		dryRun:      dryRun,
		cr:          actualCR,
		effectiveCR: effectiveCR,
	}, nil
}

// getEffectiveCR Creates an "effective" Verrazzano CR based on the user defined resource merged with the profile definitions
// - Effective CR == base profile + declared profiles + ActualCR (in order)
// - lat definition wins
func getEffectiveCR(actualCR *vzapi.Verrazzano) (*vzapi.Verrazzano, error) {
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
	mergedProfilesYaml, err := vzyaml.StrategicMergeFiles(&vzapi.Verrazzano{}, profileFiles...)
	if err != nil {
		return nil, err
	}
	// Marshal the ActualCR entity into a YAML string
	crYaml, err := yaml.Marshal(actualCR)
	if err != nil {
		return nil, err
	}
	// Create a YAML of the effective CR
	effectiveCRYaml, err := vzyaml.StrategicMerge(&vzapi.Verrazzano{}, mergedProfilesYaml, string(crYaml))
	if err != nil {
		return nil, err
	}
	// Unmarshall the EffectiveCR YAML into a VZ struct
	effectiveCR := &vzapi.Verrazzano{}
	if err := yaml.Unmarshal([]byte(effectiveCRYaml), effectiveCR); err != nil {
		return nil, err
	}
	return effectiveCR, nil
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
		log:         c.log,
		client:      c.client,
		dryRun:      c.dryRun,
		cr:          c.cr,
		effectiveCR: c.effectiveCR,
	}
}
