// Copyright (c) 2021, 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package kiali

import (
	"context"
	"fmt"
	"path/filepath"

	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/constants"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/certmanager"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/common"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/helm"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/istio"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/nginx"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/secret"
	"github.com/verrazzano/verrazzano/platform-operator/internal/config"
	"github.com/verrazzano/verrazzano/platform-operator/internal/vzconfig"

	appv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
)

// ComponentName is the name of the component
const ComponentName = "kiali-server"

// ComponentNamespace is the namespace of the component
const ComponentNamespace = constants.VerrazzanoSystemNamespace

// ComponentJSONName is the josn name of the verrazzano component in CRD
const ComponentJSONName = "kiali"

type kialiComponent struct {
	helm.HelmComponent
}

var _ spi.Component = kialiComponent{}

const kialiOverridesFile = "kiali-server-values.yaml"

var certificates = []types.NamespacedName{
	{Name: "system-tls-kiali", Namespace: ComponentNamespace},
}

func NewComponent() spi.Component {
	return kialiComponent{
		helm.HelmComponent{
			ReleaseName:               ComponentName,
			JSONName:                  ComponentJSONName,
			ChartDir:                  filepath.Join(config.GetThirdPartyDir(), ComponentName),
			ChartNamespace:            ComponentNamespace,
			IgnoreNamespaceOverride:   true,
			SupportsOperatorInstall:   true,
			SupportsOperatorUninstall: true,
			ImagePullSecretKeyname:    secret.DefaultImagePullSecretKeyName,
			ValuesFile:                filepath.Join(config.GetHelmOverridesDir(), kialiOverridesFile),
			Dependencies:              []string{istio.ComponentName, nginx.ComponentName, certmanager.ComponentName},
			AppendOverridesFunc:       AppendOverrides,
			MinVerrazzanoVersion:      constants.VerrazzanoVersion1_1_0,
			Certificates:              certificates,
			IngressNames: []types.NamespacedName{
				{
					Namespace: ComponentNamespace,
					Name:      constants.KialiIngress,
				},
			},
			GetInstallOverridesFunc: GetOverrides,
		},
	}
}

// PostInstall Kiali-post-install processing, create or update the Kiali ingress
func (c kialiComponent) PostInstall(ctx spi.ComponentContext) error {
	ctx.Log().Debugf("Kiali post-install")
	if err := c.createOrUpdateKialiResources(ctx); err != nil {
		return err
	}
	return c.HelmComponent.PostInstall(ctx)
}

// PreUpgrade Kiali-pre-upgrade processing
func (c kialiComponent) PreUpgrade(ctx spi.ComponentContext) error {
	ctx.Log().Debugf("Kiali pre-upgrade")
	if err := removeDeploymentAndService(ctx); err != nil {
		return err
	}
	return common.ApplyCRDYaml(ctx, config.GetHelmKialiChartsDir())
}

// removeDeploymentAndService removes the Kiali deployment and service during pre-upgrade.
// The match selector for Kiali was changed in 1.42.0 from the previous Kiali version (1.34.1) that Verrazzano installed.
// The match selector is an immutable field so this was a workaround to avoid a failure during Kiali upgrade.
func removeDeploymentAndService(ctx spi.ComponentContext) error {
	deployment := &appv1.Deployment{}
	if err := ctx.Client().Get(context.TODO(), types.NamespacedName{Namespace: ComponentNamespace, Name: kialiSystemName}, deployment); err != nil {
		return ctx.Log().ErrorfNewErr("Failed to get deployment %s/%s: %v", ComponentNamespace, kialiSystemName, err)
	}
	// Remove the Kiali deployment only if the match selector is not what is expected.
	if deployment.Spec.Selector != nil && len(deployment.Spec.Selector.MatchExpressions) == 0 && len(deployment.Spec.Selector.MatchLabels) == 2 {
		instance, ok := deployment.Spec.Selector.MatchLabels["app.kubernetes.io/instance"]
		if ok && instance == kialiSystemName {
			name, ok := deployment.Spec.Selector.MatchLabels["app.kubernetes.io/name"]
			if ok && name == "kiali" {
				return nil
			}
		}
	}
	service := &corev1.Service{}
	if err := ctx.Client().Get(context.TODO(), types.NamespacedName{Namespace: ComponentNamespace, Name: kialiSystemName}, service); err != nil {
		return ctx.Log().ErrorfNewErr("Failed to get service %s/%s: %v", ComponentNamespace, kialiSystemName, err)
	}
	if err := ctx.Client().Delete(context.TODO(), service); err != nil {
		return ctx.Log().ErrorfNewErr("Failed to delete service %s/%s: %v", ComponentNamespace, kialiSystemName, err)
	}
	if err := ctx.Client().Delete(context.TODO(), deployment); err != nil {
		return ctx.Log().ErrorfNewErr("Failed to delete deployment %s/%s: %v", ComponentNamespace, kialiSystemName, err)
	}

	return nil
}

// PostUpgrade Kiali-post-upgrade processing, create or update the Kiali ingress
func (c kialiComponent) PostUpgrade(ctx spi.ComponentContext) error {
	ctx.Log().Debugf("Kiali post-upgrade")
	if err := c.HelmComponent.PostUpgrade(ctx); err != nil {
		return err
	}
	return c.createOrUpdateKialiResources(ctx)
}

// IsReady Kiali-specific ready-check
func (c kialiComponent) IsReady(context spi.ComponentContext) bool {
	if c.HelmComponent.IsReady(context) {
		return isKialiReady(context)
	}
	return false
}

// IsEnabled Kiali-specific enabled check for installation
func (c kialiComponent) IsEnabled(effectiveCR *vzapi.Verrazzano) bool {
	comp := effectiveCR.Spec.Components.Kiali
	if comp == nil || comp.Enabled == nil {
		return true
	}
	return *comp.Enabled
}

// createOrUpdateKialiResources create or update related Kiali resources
func (c kialiComponent) createOrUpdateKialiResources(ctx spi.ComponentContext) error {
	if vzconfig.IsNGINXEnabled(ctx.EffectiveCR()) {
		if err := createOrUpdateKialiIngress(ctx, c.ChartNamespace); err != nil {
			return err
		}
	}
	if err := createOrUpdateAuthPolicy(ctx); err != nil {
		return err
	}
	return nil
}

// ValidateUpdate checks if the specified new Verrazzano CR is valid for this component to be updated
func (c kialiComponent) ValidateUpdate(old *vzapi.Verrazzano, new *vzapi.Verrazzano) error {
	// Do not allow any changes except to enable the component post-install
	if c.IsEnabled(old) && !c.IsEnabled(new) {
		return fmt.Errorf("Disabling component %s is not allowed", ComponentJSONName)
	}
	return c.HelmComponent.ValidateUpdate(old, new)
}

// ValidateUpdate checks if the specified new Verrazzano CR is valid for this component to be updated
func (c kialiComponent) ValidateUpdateV1Beta1(old *vzapi.Verrazzano, new *vzapi.Verrazzano) error {
	return nil
}

// MonitorOverrides checks whether monitoring of install overrides is enabled or not
func (c kialiComponent) MonitorOverrides(ctx spi.ComponentContext) bool {
	if ctx.EffectiveCR().Spec.Components.Kiali != nil {
		if ctx.EffectiveCR().Spec.Components.Kiali.MonitorChanges != nil {
			return *ctx.EffectiveCR().Spec.Components.Kiali.MonitorChanges
		}
		return true
	}
	return false
}
