// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package fluentoperator

import (
	"context"
	"fmt"
	"path/filepath"

	"k8s.io/apimachinery/pkg/runtime"

	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"

	"github.com/verrazzano/verrazzano/pkg/k8s/ready"
	"github.com/verrazzano/verrazzano/pkg/vzcr"
	"github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1beta1"

	"github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	vzconst "github.com/verrazzano/verrazzano/platform-operator/constants"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/helm"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	"github.com/verrazzano/verrazzano/platform-operator/internal/config"
)

const (
	// ComponentName is the name of the component
	ComponentName = "fluent-operator"

	// ComponentNamespace is the namespace of the component
	ComponentNamespace = vzconst.VerrazzanoSystemNamespace

	// ComponentJSONName is the JSON name of the verrazzano component in CRD
	ComponentJSONName = "fluentOperator"

	// HelmChartReleaseName is the helm chart release name
	HelmChartReleaseName = "fluent-operator"

	// HelmChartDir is the name of the helm chart directory
	HelmChartDir = "fluent-operator"
)

//var (
//	// For Unit test purposes
//	writeFileFunc = os.WriteFile
//)

// fluentOperatorComponent represents an Fluent-Operator component
type fluentOperatorComponent struct {
	helm.HelmComponent
}

// Verify that fluentOperatorComponent implements Component
var _ spi.Component = fluentOperatorComponent{}

// NewComponent returns a new fluentOperator component
func NewComponent() spi.Component {
	return fluentOperatorComponent{
		helm.HelmComponent{
			ReleaseName:               HelmChartReleaseName,
			JSONName:                  ComponentJSONName,
			ChartDir:                  filepath.Join(config.GetThirdPartyDir(), HelmChartDir),
			ChartNamespace:            ComponentNamespace,
			IgnoreNamespaceOverride:   true,
			SupportsOperatorInstall:   true,
			SupportsOperatorUninstall: true,
			AppendOverridesFunc:       appendOverrides,
			Dependencies:              []string{},
			GetInstallOverridesFunc:   GetOverrides,
			AvailabilityObjects: &ready.AvailabilityObjects{
				DaemonsetNames: []types.NamespacedName{
					fluentOperatorDeployment,
					fluentBitDaemonSet,
				},
			},
		},
	}
}

// ValidateInstall checks if the specified Verrazzano CR is valid for this component to be installed
func (c fluentOperatorComponent) ValidateInstall(vz *v1alpha1.Verrazzano) error {
	vzV1Beta1 := &v1beta1.Verrazzano{}

	if err := vz.ConvertTo(vzV1Beta1); err != nil {
		return err
	}

	return c.ValidateInstallV1Beta1(vzV1Beta1)
}

// ValidateUpdate checks if the specified new Verrazzano CR is valid for this component to be updated
func (c fluentOperatorComponent) ValidateUpdate(old *v1alpha1.Verrazzano, new *v1alpha1.Verrazzano) error {
	oldBeta := &v1beta1.Verrazzano{}
	newBeta := &v1beta1.Verrazzano{}

	if err := old.ConvertTo(oldBeta); err != nil {
		return err
	}

	if err := new.ConvertTo(newBeta); err != nil {
		return err
	}

	return c.ValidateUpdateV1Beta1(oldBeta, newBeta)
}

// ValidateInstall checks if the specified Verrazzano CR is valid for this component to be installed
func (c fluentOperatorComponent) ValidateInstallV1Beta1(vz *v1beta1.Verrazzano) error {
	return c.HelmComponent.ValidateInstallV1Beta1(vz)
}

// ValidateUpdate checks if the specified new Verrazzano CR is valid for this component to be updated
func (c fluentOperatorComponent) ValidateUpdateV1Beta1(old *v1beta1.Verrazzano, new *v1beta1.Verrazzano) error {
	// Do not allow disabling active components
	if err := c.checkEnabled(old, new); err != nil {
		return err
	}
	// Validate install overrides
	if new.Spec.Components.Velero != nil {
		if err := v1alpha1.ValidateInstallOverridesV1Beta1(new.Spec.Components.FluentOperator.ValueOverrides); err != nil {
			return err
		}
	}
	return c.HelmComponent.ValidateUpdateV1Beta1(old, new)
}

func (c fluentOperatorComponent) checkEnabled(old *v1beta1.Verrazzano, new *v1beta1.Verrazzano) error {
	// Do not allow disabling of any component post-install for now
	if c.IsEnabled(old) && !c.IsEnabled(new) {
		return fmt.Errorf("disabling component %s is not allowed", ComponentJSONName)
	}
	return nil
}

// PostInstall - post-install
func (c fluentOperatorComponent) PostInstall(ctx spi.ComponentContext) error {
	return c.HelmComponent.PostInstall(ctx)
}

// PostUpgrade Fluent-Operator component post-upgrade processing
func (c fluentOperatorComponent) PostUpgrade(ctx spi.ComponentContext) error {
	ctx.Log().Debugf("Fluentd-Operator component post-upgrade")
	return c.HelmComponent.PostUpgrade(ctx)
}

// PreInstall Fluent-Operator component pre-install processing; create and label required namespaces, copy any
// required secrets
func (c fluentOperatorComponent) PreInstall(ctx spi.ComponentContext) error {
	return c.HelmComponent.PreInstall(ctx)
}

func (c fluentOperatorComponent) Reconcile(ctx spi.ComponentContext) error {
	installed, err := c.IsInstalled(ctx)
	if err != nil {
		return err
	}
	if installed {
		err = c.Install(ctx)
	}
	return err
}

// Install Fluent-Operator component install processing
func (c fluentOperatorComponent) Install(ctx spi.ComponentContext) error {
	if err := c.HelmComponent.Install(ctx); err != nil {
		return err
	}
	if err := applyClusterOutput(ctx); err != nil {
		return err
	}
	return nil
}

// PreUpgrade Fluentd component pre-upgrade processing
func (c fluentOperatorComponent) PreUpgrade(ctx spi.ComponentContext) error {
	//if err := fluentdPreUpgrade(ctx, ComponentNamespace); err != nil {
	//	return err
	//}
	//if err := checkSecretExists(ctx); err != nil {
	//	return err
	//}
	return c.HelmComponent.PreUpgrade(ctx)
}

// Uninstall Fluentd to handle upgrade case where Fluentd was not its own helm chart.
// In that case, we need to delete the Fluentd resources explicitly
func (c fluentOperatorComponent) Uninstall(context spi.ComponentContext) error {
	installed, err := c.HelmComponent.IsInstalled(context)
	if err != nil {
		return err
	}

	// If the helm chart is installed, then uninstall
	if installed {
		return c.HelmComponent.Uninstall(context)
	}

	// Attempt to delete the Fluent-operator resources
	//rs := getFluentdManagedResources()
	//for _, r := range rs {
	//	err := resource.Resource{
	//		Name:      r.NamespacedName.Name,
	//		Namespace: r.NamespacedName.Namespace,
	//		Client:    context.Client(),
	//		Object:    r.Obj,
	//		Log:       context.Log(),
	//	}.Delete()
	//	if err != nil {
	//		return err
	//	}
	//}

	return nil
}

// Upgrade Fluentd component upgrade processing
func (c fluentOperatorComponent) Upgrade(ctx spi.ComponentContext) error {
	return c.HelmComponent.Install(ctx)
}

// IsReady component check
func (c fluentOperatorComponent) IsReady(ctx spi.ComponentContext) bool {
	if c.HelmComponent.IsReady(ctx) {
		return isFluentOperatorReady(ctx)
	}
	return false
}

// IsInstalled component check
func (c fluentOperatorComponent) IsInstalled(ctx spi.ComponentContext) (bool, error) {
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

// IsEnabled fluentd-specific enabled check for installation
func (c fluentOperatorComponent) IsEnabled(effectiveCR runtime.Object) bool {
	return vzcr.IsFluentOperatorEnabled(effectiveCR)
}

// MonitorOverrides checks whether monitoring of install overrides is enabled or not
func (c fluentOperatorComponent) MonitorOverrides(ctx spi.ComponentContext) bool {
	if ctx.EffectiveCR().Spec.Components.FluentOperator != nil {
		if ctx.EffectiveCR().Spec.Components.FluentOperator.MonitorChanges != nil {
			return *ctx.EffectiveCR().Spec.Components.FluentOperator.MonitorChanges
		}
		return true
	}
	return false
}
