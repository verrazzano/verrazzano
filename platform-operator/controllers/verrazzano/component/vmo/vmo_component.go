// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package vmo

import (
	"context"
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	vzconst "github.com/verrazzano/verrazzano/platform-operator/constants"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/common"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/helm"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/nginx"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	"github.com/verrazzano/verrazzano/platform-operator/internal/config"
	"github.com/verrazzano/verrazzano/platform-operator/internal/vzconfig"
	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"path/filepath"
)

// ComponentName is the name of the component
const ComponentName = "verrazzano-monitoring-operator"

// ComponentNamespace is the namespace of the component
const ComponentNamespace = vzconst.VerrazzanoSystemNamespace

// ComponentJSONName is the json name of the verrazzano-monitoring-operator component
const ComponentJSONName = "verrazzano-monitoring-operator"

// vmoComponent represents a VMO component
type vmoComponent struct {
	helm.HelmComponent
}

// Verify that vmoComponent implements Component
var _ spi.Component = vmoComponent{}

// NewComponent returns a new VMO component
func NewComponent() spi.Component {
	return vmoComponent{
		helm.HelmComponent{
			ReleaseName:             ComponentName,
			JSONName:                ComponentJSONName,
			ChartDir:                filepath.Join(config.GetHelmChartsDir(), ComponentName),
			ChartNamespace:          ComponentNamespace,
			IgnoreNamespaceOverride: true,
			SupportsOperatorInstall: true,
			AppendOverridesFunc:     appendVMOOverrides,
			ImagePullSecretKeyname:  "global.imagePullSecrets[0]",
			Dependencies:            []string{nginx.ComponentName},
		},
	}
}

// IsEnabled VMO enabled check for installation
func (c vmoComponent) IsEnabled(effectiveCR *vzapi.Verrazzano) bool {
	return vzconfig.IsVMOEnabled(effectiveCR)
}

// IsReady calls VMO isVmoReady function
func (c vmoComponent) IsReady(context spi.ComponentContext) bool {
	if c.HelmComponent.IsReady(context) {
		return isVMOReady(context)
	}
	return false
}

// IsInstalled checks if VMO is installed
func (c vmoComponent) IsInstalled(ctx spi.ComponentContext) (bool, error) {
	deployment := &appsv1.Deployment{}
	err := ctx.Client().Get(context.TODO(), types.NamespacedName{Namespace: ComponentNamespace, Name: ComponentName}, deployment)
	if errors.IsNotFound(err) {
		return false, nil
	}
	if err != nil {
		ctx.Log().Errorf("Failed to get %s/%s deployment: %v", ComponentNamespace, ComponentName, err)
		return false, err
	}
	return true, nil
}

// PreUpgrade VMO pre-upgrade processing
func (c vmoComponent) PreUpgrade(context spi.ComponentContext) error {
	return common.ApplyCRDYaml(context, config.GetHelmVMOChartsDir())
}

// Upgrade VMO processing
func (c vmoComponent) Upgrade(context spi.ComponentContext) error {
	return c.HelmComponent.Install(context)
}
