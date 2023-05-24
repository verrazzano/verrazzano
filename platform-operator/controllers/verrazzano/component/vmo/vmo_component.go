// Copyright (c) 2022, 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package vmo

import (
	"context"
	"path/filepath"

	"k8s.io/apimachinery/pkg/runtime"

	"github.com/verrazzano/verrazzano/pkg/k8s/ready"
	"github.com/verrazzano/verrazzano/pkg/vzcr"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/networkpolicies"

	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"

	"github.com/verrazzano/verrazzano/pkg/k8s/resource"
	vzconst "github.com/verrazzano/verrazzano/platform-operator/constants"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/common"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/helm"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/nginx"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	"github.com/verrazzano/verrazzano/platform-operator/internal/config"
)

// ComponentName is the name of the component
const ComponentName = "verrazzano-monitoring-operator"

// ComponentNamespace is the namespace of the component
const ComponentNamespace = vzconst.VerrazzanoSystemNamespace

// ComponentJSONName is the JSON name of the verrazzano-monitoring-operator component
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
			ReleaseName:               ComponentName,
			JSONName:                  ComponentJSONName,
			ChartDir:                  filepath.Join(config.GetHelmChartsDir(), ComponentName),
			ChartNamespace:            ComponentNamespace,
			IgnoreNamespaceOverride:   true,
			SupportsOperatorInstall:   true,
			SupportsOperatorUninstall: true,
			AppendOverridesFunc:       appendVMOOverrides,
			ImagePullSecretKeyname:    "global.imagePullSecrets[0]",
			Dependencies:              []string{networkpolicies.ComponentName, nginx.ComponentName},
			AvailabilityObjects: &ready.AvailabilityObjects{
				DeploymentNames: []types.NamespacedName{
					{
						Name:      ComponentName,
						Namespace: ComponentNamespace,
					},
				},
			},
		},
	}
}

// IsEnabled VMO enabled check for installation
func (c vmoComponent) IsEnabled(effectiveCR runtime.Object) bool {
	return vzcr.IsVMOEnabled(effectiveCR)
}

// IsReady calls VMO isVmoReady function
func (c vmoComponent) IsReady(context spi.ComponentContext) bool {
	if c.HelmComponent.IsReady(context) {
		return c.isVMOReady(context)
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

// PreInstall applies the VMO CRDs
func (c vmoComponent) PreInstall(ctx spi.ComponentContext) error {
	if err := common.ApplyCRDYaml(ctx, config.GetHelmVMOChartsDir()); err != nil {
		return err
	}
	return c.HelmComponent.PreInstall(ctx)
}

// PreUpgrade VMO pre-upgrade processing
func (c vmoComponent) PreUpgrade(context spi.ComponentContext) error {
	if err := common.ApplyCRDYaml(context, config.GetHelmVMOChartsDir()); err != nil {
		return err
	}
	if err := retainPrometheusPersistentVolume(context); err != nil {
		return err
	}
	return c.HelmComponent.PreUpgrade(context)
}

// Upgrade VMO processing
func (c vmoComponent) Upgrade(context spi.ComponentContext) error {
	return c.HelmComponent.Install(context)
}

// Uninstall VMO processing
func (c vmoComponent) Uninstall(context spi.ComponentContext) error {
	installed, err := c.HelmComponent.IsInstalled(context)
	if err != nil {
		return err
	}

	// If we find that the VMO helm chart is installed, then uninstall
	if installed {
		return c.HelmComponent.Uninstall(context)
	}

	// Attempt to delete the VMO resources if the VMO helm chart is not installed.
	vmoResources := common.GetVMOHelmManagedResources()
	for _, vmoResource := range vmoResources {
		err := resource.Resource{
			Name:      vmoResource.NamespacedName.Name,
			Namespace: vmoResource.NamespacedName.Namespace,
			Client:    context.Client(),
			Object:    vmoResource.Obj,
			Log:       context.Log(),
		}.Delete()
		if err != nil {
			return err
		}
	}

	return nil
}
