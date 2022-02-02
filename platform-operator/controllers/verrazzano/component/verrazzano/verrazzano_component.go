// Copyright (c) 2021, 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
package verrazzano

import (
	"fmt"
	"path/filepath"

	globalconst "github.com/verrazzano/verrazzano/pkg/constants"
	"github.com/verrazzano/verrazzano/platform-operator/constants"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/helm"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/istio"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/nginx"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	"github.com/verrazzano/verrazzano/platform-operator/internal/config"
	"github.com/verrazzano/verrazzano/platform-operator/internal/k8s/status"
	"github.com/verrazzano/verrazzano/platform-operator/internal/vzconfig"
	"k8s.io/apimachinery/pkg/types"
)

type verrazzanoComponent struct {
	helm.HelmComponent
}

const vzImagePullSecretKeyName = "global.imagePullSecrets[0]"

func NewComponent() spi.Component {
	return verrazzanoComponent{
		helm.HelmComponent{
			ReleaseName:             ComponentName,
			ChartDir:                filepath.Join(config.GetHelmChartsDir(), ComponentName),
			ChartNamespace:          constants.VerrazzanoSystemNamespace,
			IgnoreNamespaceOverride: true,
			ResolveNamespaceFunc:    resolveVerrazzanoNamespace,
			AppendOverridesFunc:     appendVerrazzanoOverrides,
			ImagePullSecretKeyname:  vzImagePullSecretKeyName,
			SupportsOperatorInstall: true,
			Dependencies:            []string{istio.ComponentName, nginx.ComponentName},
		},
	}
}

// PostInstall Verrazzano component pre-install processing; create and label required namespaces, copy any
// required secrets
func (c verrazzanoComponent) PreInstall(ctx spi.ComponentContext) error {
	ctx.Log().Debugf("Verrazzano pre-install")
	if err := createAndLabelNamespaces(ctx); err != nil {
		return ctx.Log().ErrorfNewErr("Failed creating/labeling namespaces for Verrazzano: %v", err)
	}
	if err := loggingPreInstall(ctx); err != nil {
		return ctx.Log().ErrorfNewErr("Failed copying logging secrets for Verrazzano: %v", err)
	}
	return nil
}

// PreUpgrade Verrazzano component pre-upgrade processing
func (c verrazzanoComponent) PreUpgrade(ctx spi.ComponentContext) error {
	return verrazzanoPreUpgrade(ctx.Log(), ctx.Client(),
		c.ReleaseName, resolveVerrazzanoNamespace(c.ChartNamespace), c.ChartDir)
}

// IsReady Verrazzano component ready-check
func (c verrazzanoComponent) IsReady(ctx spi.ComponentContext) bool {
	if !c.HelmComponent.IsReady(ctx) {
		return false
	}
	deployments := []types.NamespacedName{
		{Name: "verrazzano-authproxy", Namespace: globalconst.VerrazzanoSystemNamespace},
	}
	if isVMOEnabled(ctx.EffectiveCR()) {
		deployments = append(deployments, []types.NamespacedName{
			{Name: "verrazzano-operator", Namespace: globalconst.VerrazzanoSystemNamespace},
			{Name: "verrazzano-monitoring-operator", Namespace: globalconst.VerrazzanoSystemNamespace},
		}...)
	}
	prefix := fmt.Sprintf("Component %s", ctx.GetComponent())
	if !status.DeploymentsReady(ctx.Log(), ctx.Client(), deployments, 1, prefix) {
		return false
	}
	return isVerrazzanoSecretReady(ctx)
}

// PostInstall - post-install, clean up temp files
func (c verrazzanoComponent) PostInstall(ctx spi.ComponentContext) error {
	cleanTempFiles(ctx)
	// populate the ingress names before calling PostInstall on Helm component because those will be needed there
	c.HelmComponent.IngressNames = c.GetIngressNames(ctx)
	return c.HelmComponent.PostInstall(ctx)
}

// PostUpgrade Verrazzano-post-upgrade processing
func (c verrazzanoComponent) PostUpgrade(ctx spi.ComponentContext) error {
	ctx.Log().Debugf("Verrazzano component post-upgrade")
	if err := c.HelmComponent.PostUpgrade(ctx); err != nil {
		return err
	}
	cleanTempFiles(ctx)
	return c.updateElasticsearchResources(ctx)
}

// updateElasticsearchResources updates elasticsearch resources
func (c verrazzanoComponent) updateElasticsearchResources(ctx spi.ComponentContext) error {
	if err := fixupElasticSearchReplicaCount(ctx, resolveVerrazzanoNamespace(c.ChartNamespace)); err != nil {
		return err
	}
	return nil
}

// IsEnabled verrazzano-specific enabled check for installation
func (c verrazzanoComponent) IsEnabled(ctx spi.ComponentContext) bool {
	comp := ctx.EffectiveCR().Spec.Components.Verrazzano
	if comp == nil || comp.Enabled == nil {
		return true
	}
	return *comp.Enabled
}

// GetIngressNames - gets the names of the ingresses associated with this component
func (c verrazzanoComponent) GetIngressNames(ctx spi.ComponentContext) []types.NamespacedName {
	ingressNames := []types.NamespacedName{
		{
			Namespace: constants.VerrazzanoSystemNamespace,
			Name:      constants.VzConsoleIngress,
		},
	}

	if vzconfig.IsElasticsearchEnabled(ctx.EffectiveCR()) {
		ingressNames = append(ingressNames, types.NamespacedName{
			Namespace: constants.VerrazzanoSystemNamespace,
			Name:      constants.ElasticsearchIngress,
		})
	}

	if vzconfig.IsGrafanaEnabled(ctx.EffectiveCR()) {
		ingressNames = append(ingressNames, types.NamespacedName{
			Namespace: constants.VerrazzanoSystemNamespace,
			Name:      constants.GrafanaIngress,
		})
	}

	if vzconfig.IsKibanaEnabled(ctx.EffectiveCR()) {
		ingressNames = append(ingressNames, types.NamespacedName{
			Namespace: constants.VerrazzanoSystemNamespace,
			Name:      constants.KibanaIngress,
		})
	}

	if vzconfig.IsPrometheusEnabled(ctx.EffectiveCR()) {
		ingressNames = append(ingressNames, types.NamespacedName{
			Namespace: constants.VerrazzanoSystemNamespace,
			Name:      constants.PrometheusIngress,
		})
	}

	return ingressNames
}
