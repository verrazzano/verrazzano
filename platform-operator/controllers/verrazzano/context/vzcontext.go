// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package context

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

// VerrazzanoContext the context needed to reconcile a Verrazzano CR
type VerrazzanoContext struct {
	// log logger for the execution context
	Log vzlog.VerrazzanoLogger
	// client Kubernetes client
	Client clipkg.Client
	// dryRun If true, do a dry run of operations
	DryRun bool
	// ActualCR is the CR passed to top level Reconcile.  It epresents the desired Verrazzano state in the cluster
	ActualCR *vzapi.Verrazzano
	// effectiveCR Represents the configuration resulting from any named profiles used and any configured overrides in the CR
	EffectiveCR *vzapi.Verrazzano
}

// NewVerrazzanoContext creates a ComponentContext from a raw CR
func NewVerrazzanoContext(log vzlog.VerrazzanoLogger, c clipkg.Client, actualCR *vzapi.Verrazzano, dryRun bool) (VerrazzanoContext, error) {
	// Generate the effective CR based on the declared profile and any overrides in the user-supplied one
	effectiveCR, err := GetEffectiveCR(actualCR)
	if err != nil {
		return VerrazzanoContext{}, err
	}
	return VerrazzanoContext{
		Log:         log,
		Client:      c,
		DryRun:      dryRun,
		ActualCR:    actualCR,
		EffectiveCR: effectiveCR,
	}, nil
}

// GetEffectiveCR Creates an "effective" Verrazzano CR based on the user defined resource merged with the profile definitions
// - Effective CR == base profile + declared profiles + ActualCR (in order)
// - last definition wins
func GetEffectiveCR(actualCR *vzapi.Verrazzano) (*vzapi.Verrazzano, error) {
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

	// Copy actual status to effective CR
	actualCR.Status.DeepCopyInto(&effectiveCR.Status)

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
