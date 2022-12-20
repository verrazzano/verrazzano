// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

// The Grafana dashboards component is needed to install the Verrazzano Grafana Dashboards
// Helm chart during install and upgrade

package grafanadashboards

import (
	"context"
	"path/filepath"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/verrazzano/verrazzano/pkg/vzcr"
	"github.com/verrazzano/verrazzano/platform-operator/constants"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/helm"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	"github.com/verrazzano/verrazzano/platform-operator/internal/config"
)

const (
	// ComponentName is the name of the component
	ComponentName = "verrazzano-grafana-dashboards"

	// ComponentNamespace is the namespace of the component
	ComponentNamespace = constants.VerrazzanoSystemNamespace

	// ComponentJSONName is the JSON name of the component
	ComponentJSONName = "verrazzanoGrafanaDashboards"

	// legacyDashboardConfigMapName is the name of the configmap used to store dashboards in older Verrazzano releases
	legacyDashboardConfigMapName = "system-dashboards"
)

type grafanaDashboardsComponent struct {
	helm.HelmComponent
}

// NewComponent returns a new grafanaDashboardsComponent
func NewComponent() spi.Component {
	return grafanaDashboardsComponent{
		helm.HelmComponent{
			ReleaseName:               ComponentName,
			JSONName:                  ComponentJSONName,
			ChartDir:                  filepath.Join(config.GetHelmChartsDir(), ComponentName),
			ChartNamespace:            ComponentNamespace,
			MinVerrazzanoVersion:      constants.VerrazzanoVersion1_5_0,
			IgnoreNamespaceOverride:   true,
			SupportsOperatorInstall:   true,
			SupportsOperatorUninstall: true,
			InstallBeforeUpgrade:      true,
		},
	}
}

// IsEnabled returns true when Grafana is enabled, false otherwise (dashboards are enabled when
// Grafana is enabled)
func (g grafanaDashboardsComponent) IsEnabled(effectiveCR runtime.Object) bool {
	return vzcr.IsGrafanaEnabled(effectiveCR)
}

// PostInstall runs post install processing for this component.
func (g grafanaDashboardsComponent) PostInstall(ctx spi.ComponentContext) error {
	return doPostInstallUpgrade(ctx)
}

// PostUpgrade runs post upgrade processing for this component.
func (g grafanaDashboardsComponent) PostUpgrade(ctx spi.ComponentContext) error {
	return doPostInstallUpgrade(ctx)
}

// doPostInstallUpgrade runs common post install and upgrade processing. The Grafana dashboards are
// now in this helm chart so after installing or upgrading this component we delete the legacy
// dashboards configmap.
func doPostInstallUpgrade(ctx spi.ComponentContext) error {
	cm := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      legacyDashboardConfigMapName,
			Namespace: constants.VerrazzanoSystemNamespace,
		},
	}
	err := ctx.Client().Delete(context.TODO(), cm)
	return client.IgnoreNotFound(err)
}
