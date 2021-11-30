// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
package verrazzano

import (
	"github.com/verrazzano/verrazzano/platform-operator/constants"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/helm"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	"github.com/verrazzano/verrazzano/platform-operator/internal/config"
	"path/filepath"
)

type verrazzanoComponent struct {
	helm.HelmComponent
}

func NewComponent() spi.Component {
	return verrazzanoComponent{
		helm.HelmComponent{
			ReleaseName:             ComponentName,
			ChartDir:                filepath.Join(config.GetHelmChartsDir(), ComponentName),
			ChartNamespace:          constants.VerrazzanoSystemNamespace,
			IgnoreNamespaceOverride: true,
			ResolveNamespaceFunc:    ResolveVerrazzanoNamespace,
			PreUpgradeFunc:          VerrazzanoPreUpgrade,
			AppendOverridesFunc:     AppendOverrides,
		},
	}
}

// PostUpgrade Verrazzano-post-upgrade processing
func (c verrazzanoComponent) PostUpgrade(ctx spi.ComponentContext) error {
	ctx.Log().Debugf("Verrazzano component post-upgrade")
	if err := c.HelmComponent.PostUpgrade(ctx); err != nil {
		return err
	}
	return c.updateElasticsearchResources(ctx)
}

// updateElasticsearchResources updates elasticsearch resources
func (c verrazzanoComponent) updateElasticsearchResources(ctx spi.ComponentContext) error {
	if err := fixupElasticSearchReplicaCount(ctx, c.ChartNamespace); err != nil {
		return err
	}
	return nil
}
