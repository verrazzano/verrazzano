// Copyright (c) 2021, 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package kiali

import (
	"fmt"
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"path/filepath"

	"github.com/verrazzano/verrazzano/platform-operator/constants"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/helm"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/nginx"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/secret"
	"github.com/verrazzano/verrazzano/platform-operator/internal/config"
	"k8s.io/apimachinery/pkg/types"
)

// ComponentName is the name of the component
const ComponentName = "kiali-server"

// ComponentNamespace is the namespace of the component
const ComponentNamespace = constants.VerrazzanoSystemNamespace

type kialiComponent struct {
	helm.HelmComponent
}

var _ spi.Component = kialiComponent{}

const kialiOverridesFile = "kiali-server-values.yaml"

func NewComponent() spi.Component {
	return kialiComponent{
		helm.HelmComponent{
			ReleaseName:             ComponentName,
			ChartDir:                filepath.Join(config.GetThirdPartyDir(), ComponentName),
			ChartNamespace:          ComponentNamespace,
			IgnoreNamespaceOverride: true,
			SupportsOperatorInstall: true,
			ImagePullSecretKeyname:  secret.DefaultImagePullSecretKeyName,
			ValuesFile:              filepath.Join(config.GetHelmOverridesDir(), kialiOverridesFile),
			Dependencies:            []string{nginx.ComponentName},
			AppendOverridesFunc:     AppendOverrides,
			MinVerrazzanoVersion:    constants.VerrazzanoVersion1_1_0,
			IngressNames: []types.NamespacedName{
				{
					Namespace: ComponentNamespace,
					Name:      constants.KialiIngress,
				},
			},
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
	if err := createOrUpdateKialiIngress(ctx, c.ChartNamespace); err != nil {
		return err
	}
	if err := createOrUpdateAuthPolicy(ctx); err != nil {
		return err
	}
	return nil
}

// ValidateUpdate checks if the specified new Verrazzano CR is valid for this component to be updated
func (c kialiComponent) ValidateUpdate(old *vzapi.Verrazzano, new *vzapi.Verrazzano) error {
	if c.IsEnabled(old) && !c.IsEnabled(new) {
		return fmt.Errorf("can not disable previously enabled kiali")
	}
	return nil
}
