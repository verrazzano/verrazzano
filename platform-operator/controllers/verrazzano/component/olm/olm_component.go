// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

// Operator Lifecycle Manager
package olm

import (
	"path/filepath"

	"github.com/verrazzano/verrazzano/platform-operator/constants"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/helm"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	"github.com/verrazzano/verrazzano/platform-operator/internal/config"
)

// ComponentName is the name of the OLM Operator
const ComponentName = "olm-operator"

// CatalogComponentName is the name of the OLM Catalog
const CatalogComponentName = "olm-catalog"

// ComponentNamespace is the namespace of the component
const ComponentNamespace = "olm"

// OperatorsNamespace is the namespace used for deploying operators by OLM
const OperatorNamespace = "operators"

// Directory containing CRDs
const crdDirectory = "operator-lifecycle-manager"

// File containing CRDs for operator-lifecycle-manager
const crdFile = "operator-lifecycle-manager.crds.yaml"

type olmComponent struct {
	helm.HelmComponent
}

// Verify that OLMComponent implements Component
var _ spi.Component = olmComponent{}

// NewComponent returns a new olmComponent component
func NewComponent() spi.Component {
	return olmComponent{
		helm.HelmComponent{
			ReleaseName:             ComponentName,
			ChartDir:                filepath.Join(config.GetHelmChartsDir(), ComponentName),
			ChartNamespace:          ComponentNamespace,
			AppendOverridesFunc:     AppendOverrides,
			IgnoreNamespaceOverride: true,
			SupportsOperatorInstall: true,
			MinVerrazzanoVersion:    constants.VerrazzanoVersion1_3_0,
			ImagePullSecretKeyname:  "global.imagePullSecrets[0]",
		},
	}
}

// IsEnabled olmComponent-specific enabled check for installation
func (c olmComponent) IsEnabled(ctx spi.ComponentContext) bool {
	comp := ctx.EffectiveCR().Spec.Components.OLM
	if comp == nil || comp.Enabled == nil {
		return true
	}
	return *comp.Enabled
}

// IsReady component check
func (c olmComponent) IsReady(ctx spi.ComponentContext) bool {
	if c.HelmComponent.IsReady(ctx) {
		return isOLMReady(ctx)
	}
	return false
}

// PreInstall runs before operator-lifecycle-manager components are installed
// The operator-lifecycle-manager namespace is created
// The operator-lifecycle-manager manifest is patched if needed and applied to create necessary CRDs
func (c olmComponent) PreInstall(compContext spi.ComponentContext) error {
	// If it is a dry-run, do nothing
	if compContext.IsDryRun() {
		compContext.Log().Debug("operator-lifecycle-manager PreInstall dry run")
		return nil
	}

	// create operator-lifecycle-manager namespace
	compContext.Log().Debug("Creating namespaces needed by operator-lifecycle-manager")
	err := c.createOrUpdateNamespace(compContext, ComponentNamespace)
	if err != nil {
		return compContext.Log().ErrorfNewErr("Failed to create the operator-lifecycle-manager namespace: %v", err)
	}
	err = c.createOrUpdateNamespace(compContext, OperatorNamespace)
	if err != nil {
		return compContext.Log().ErrorfNewErr("Failed to create the operator-lifecycle-manager operators namespace: %v", err)
	}

	// Apply the operator-lifecycle-manager manifest, patching if needed
	compContext.Log().Debug("Applying operator-lifecycle-manager crds")
	err = c.applyManifest(compContext)
	if err != nil {
		return compContext.Log().ErrorfNewErr("Failed to apply the operator-lifecycle-manager manifest: %v", err)
	}
	return nil
}
