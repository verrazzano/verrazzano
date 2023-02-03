// Copyright (c) 2021, 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package coherence

import (
	"fmt"
	"github.com/verrazzano/verrazzano/pkg/k8s/ready"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/verrazzano/verrazzano/pkg/vzcr"

	installv1beta1 "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1beta1"
	"k8s.io/apimachinery/pkg/runtime"

	"github.com/verrazzano/verrazzano/pkg/k8s/webhook"
	modulesv1alpha1 "github.com/verrazzano/verrazzano/platform-operator/apis/modules/v1alpha1"
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/constants"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/module/modules"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/module/reconciler"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/helm"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/secret"
	"github.com/verrazzano/verrazzano/platform-operator/internal/config"
)

// ComponentName is the name of the component
const ComponentName = "coherence-operator"

// ComponentNamespace is the namespace of the component
const ComponentNamespace = constants.VerrazzanoSystemNamespace

// ComponentJSONName is the JSON name of the verrazzano component in CRD
const ComponentJSONName = "coherenceOperator"

type coherenceComponent struct {
	helm.HelmComponent
}

func (c coherenceComponent) ReconcileModule(ctx spi.ComponentContext) error {
	return nil
}

func (c coherenceComponent) SetStatusWriter(statusWriter client.StatusWriter) {}

func NewComponent(module *modulesv1alpha1.Module) modules.DelegateReconciler {
	h := helm.HelmComponent{
		ChartDir:               config.GetThirdPartyDir(),
		ImagePullSecretKeyname: secret.DefaultImagePullSecretKeyName,
		AvailabilityObjects: &ready.AvailabilityObjects{
			DeploymentNames: []types.NamespacedName{
				{
					Name:      ComponentName,
					Namespace: ComponentNamespace,
				},
			},
		},
	}
	helm.SetForModule(&h, module)
	return &reconciler.Reconciler{
		Component: coherenceComponent{
			HelmComponent: h,
		},
	}
	//return coherenceComponent{
	//	helm.HelmComponent{
	//		ReleaseName:               ComponentName,
	//		JSONName:                  ComponentJSONName,
	//		ChartDir:                  filepath.Join(config.GetThirdPartyDir(), ComponentName),
	//		ChartNamespace:            ComponentNamespace,
	//		IgnoreNamespaceOverride:   true,
	//		SupportsOperatorInstall:   true,
	//		SupportsOperatorUninstall: true,
	//		ImagePullSecretKeyname:    secret.DefaultImagePullSecretKeyName,
	//		ValuesFile:                filepath.Join(config.GetHelmOverridesDir(), "coherence-values.yaml"),
	//		Dependencies:              []string{networkpolicies.ComponentName},
	//		GetInstallOverridesFunc:   GetOverrides,
	//		AvailabilityObjects: &ready.AvailabilityObjects{
	//			DeploymentNames: []types.NamespacedName{
	//				{
	//					Name:      ComponentName,
	//					Namespace: ComponentNamespace,
	//				},
	//			},
	//		},
	//	},
	//}
}

// IsEnabled Coherence-specific enabled check for installation
func (c coherenceComponent) IsEnabled(effectiveCR runtime.Object) bool {
	return vzcr.IsCoherenceOperatorEnabled(effectiveCR)
}

// IsReady checks if the Coherence deployment is ready
func (c coherenceComponent) IsReady(ctx spi.ComponentContext) bool {
	if c.HelmComponent.IsReady(ctx) {
		return c.isCoherenceOperatorReady(ctx)
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
func (c coherenceComponent) ValidateUpdateV1Beta1(old *installv1beta1.Verrazzano, new *installv1beta1.Verrazzano) error {
	// Do not allow any changes except to enable the component post-install
	if c.IsEnabled(old) && !c.IsEnabled(new) {
		return fmt.Errorf("Disabling component %s is not allowed", ComponentJSONName)
	}
	return c.HelmComponent.ValidateUpdateV1Beta1(old, new)
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

func (c coherenceComponent) Name() string {
	if c.HelmComponent.ReleaseName == "" {
		return ComponentName
	}
	return c.HelmComponent.ReleaseName
}
