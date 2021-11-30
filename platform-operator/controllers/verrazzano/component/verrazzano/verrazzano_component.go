// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
package verrazzano

import (
	globalconst "github.com/verrazzano/verrazzano/pkg/constants"
	ctrlerrors "github.com/verrazzano/verrazzano/pkg/controller/errors"
	"github.com/verrazzano/verrazzano/platform-operator/constants"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/helm"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/istio"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/nginx"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	"github.com/verrazzano/verrazzano/platform-operator/internal/config"
	"github.com/verrazzano/verrazzano/platform-operator/internal/k8s/status"
	"k8s.io/apimachinery/pkg/types"
	"path/filepath"
)

type verrazzanoComponent struct {
	helm.HelmComponent
}

const vzImagePullSecretKeyName = "global.imagePullSecrets[0]"

func NewComponent() spi.Component {
	return verrazzanoComponent{
		helm.HelmComponent{
			ReleaseName:             componentName,
			ChartDir:                filepath.Join(config.GetHelmChartsDir(), componentName),
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
	vzLog(ctx).Debugf("Verrazzano pre-install")
	if err := createAndLabelNamespaces(ctx); err != nil {
		return ctrlerrors.RetryableError{Source: componentName, Cause: err}
	}
	if err := loggingPreInstall(ctx); err != nil {
		return ctrlerrors.RetryableError{Source: componentName, Cause: err}
	}
	return nil
}

// PreUpgrade Verrazzano component pre-upgrade processing
func (c verrazzanoComponent) PreUpgrade(ctx spi.ComponentContext) error {
	return verrazzanoPreUpgrade(vzLog(ctx), ctx.Client(),
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
	if !status.DeploymentsReady(vzLog(ctx), ctx.Client(), deployments, 1) {
		return false
	}
	return isVerrazzanoSecretReady(ctx)
}

// PostInstall - post-install, clean up temp files
func (c verrazzanoComponent) PostInstall(ctx spi.ComponentContext) error {
	cleanTempFiles(ctx)
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
	if err := fixupElasticSearchReplicaCount(ctx, c.ChartNamespace); err != nil {
		return err
	}
	return nil
}
