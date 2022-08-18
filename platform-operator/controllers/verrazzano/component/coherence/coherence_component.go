// Copyright (c) 2021, 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
package coherence

import (
	"fmt"
	"path/filepath"

	"github.com/verrazzano/verrazzano/pkg/k8s/webhook"
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/constants"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/helm"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/secret"
	"github.com/verrazzano/verrazzano/platform-operator/internal/config"
)

// ComponentName is the name of the component
const ComponentName = "coherence-operator"

// ComponentNamespace is the namespace of the component
const ComponentNamespace = constants.VerrazzanoSystemNamespace

// ComponentJSONName is the josn name of the verrazzano component in CRD
const ComponentJSONName = "coherenceOperator"

type coherenceComponent struct {
	helm.HelmComponent
}

func NewComponent() spi.Component {
	return coherenceComponent{
		helm.HelmComponent{
			ReleaseName:               ComponentName,
			JSONName:                  ComponentJSONName,
			ChartDir:                  filepath.Join(config.GetThirdPartyDir(), ComponentName),
			ChartNamespace:            ComponentNamespace,
			IgnoreNamespaceOverride:   true,
			SupportsOperatorInstall:   true,
			SupportsOperatorUninstall: true,
			ImagePullSecretKeyname:    secret.DefaultImagePullSecretKeyName,
			ValuesFile:                filepath.Join(config.GetHelmOverridesDir(), "coherence-values.yaml"),
			Dependencies:              []string{},
			GetInstallOverridesFunc:   GetOverrides,
		},
	}
}

// IsEnabled Coherence-specific enabled check for installation
func (c coherenceComponent) IsEnabled(effectiveCR *vzapi.Verrazzano) bool {
	comp := effectiveCR.Spec.Components.CoherenceOperator
	if comp == nil || comp.Enabled == nil {
		return true
	}
	return *comp.Enabled
}

// IsReady checks if the Coherence deployment is ready
func (c coherenceComponent) IsReady(ctx spi.ComponentContext) bool {
	if c.HelmComponent.IsReady(ctx) {
		return isCoherenceOperatorReady(ctx)
	}
	return false
}

// ValidateUpdate checks if the specified new Verrazzano CR is valid for this component to be updated
func (c coherenceComponent) ValidateUpdate(old *vzapi.Verrazzano, new *vzapi.Verrazzano) error {
	// Do not allow any changes except to enable the component post-install
	if c.IsEnabled(old) && !c.IsEnabled(new) {
		return fmt.Errorf("Disabling component %s is not allowed", ComponentJSONName)
	}
	return c.HelmComponent.ValidateUpdate(old, new)
}

// ValidateUpdate checks if the specified new Verrazzano CR is valid for this component to be updated
func (c coherenceComponent) ValidateUpdateV1Beta1(old *vzapi.Verrazzano, new *vzapi.Verrazzano) error {
	return nil
}

// MonitorOverrides checks whether monitoring of install overrides is enabled or not
func (c coherenceComponent) MonitorOverrides(ctx spi.ComponentContext) bool {
	if ctx.EffectiveCR().Spec.Components.CoherenceOperator != nil {
		if ctx.EffectiveCR().Spec.Components.CoherenceOperator.MonitorChanges != nil {
			return *ctx.EffectiveCR().Spec.Components.CoherenceOperator.MonitorChanges
		}
		return true
	}
	return false
}

func (c coherenceComponent) PostUninstall(context spi.ComponentContext) error {
	if err := webhook.DeleteValidatingWebhookConfiguration(context.Log(), context.Client(), "coherence-operator-validating-webhook-configuration"); err != nil {
		return err
	}
	if err := webhook.DeleteMutatingWebhookConfiguration(context.Log(), context.Client(), "coherence-operator-mutating-webhook-configuration"); err != nil {
		return err
	}
	return nil
}
