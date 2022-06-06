// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package fluentd

import (
	"fmt"
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	vzconst "github.com/verrazzano/verrazzano/platform-operator/constants"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/helm"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	"github.com/verrazzano/verrazzano/platform-operator/internal/config"
	"io/ioutil"
	"path/filepath"
)

const (
	// ComponentName is the name of the component
	ComponentName = "fluentd"

	// ComponentNamespace is the namespace of the component
	ComponentNamespace = vzconst.VerrazzanoSystemNamespace

	// ComponentJSONName is the json name of the verrazzano component in CRD
	ComponentJSONName = "fluentd"

	// HelmChartDir is the name of the helm chart directory
	HelmChartDir = "verrazzano-fluentd"

	// vzImagePullSecretKeyName is the Helm key name for the VZ chart image pull secret
	vzImagePullSecretKeyName = "global.imagePullSecrets[0]"
)

var (
	// For Unit test purposes
	writeFileFunc = ioutil.WriteFile
)

// fluentdComponent represents an Fluentd component
type fluentdComponent struct {
	helm.HelmComponent
}

// Verify that fluentdComponent implements Component
var _ spi.Component = fluentdComponent{}

// NewComponent returns a new Fluentd component
func NewComponent() spi.Component {
	return fluentdComponent{
		helm.HelmComponent{
			ReleaseName:             ComponentName,
			JSONName:                ComponentJSONName,
			ChartDir:                filepath.Join(config.GetHelmChartsDir(), HelmChartDir),
			ChartNamespace:          ComponentNamespace,
			IgnoreNamespaceOverride: true,
			SupportsOperatorInstall: true,
			ImagePullSecretKeyname:  vzImagePullSecretKeyName,
			AppendOverridesFunc:     appendOverrides,
		},
	}
}

// ValidateInstall checks if the specified Verrazzano CR is valid for this component to be installed
func (f fluentdComponent) ValidateInstall(vz *vzapi.Verrazzano) error {
	if err := validateFluentd(vz); err != nil {
		return err
	}
	return f.HelmComponent.ValidateInstall(vz)
}

// ValidateUpdate checks if the specified new Verrazzano CR is valid for this component to be updated
func (f fluentdComponent) ValidateUpdate(old *vzapi.Verrazzano, new *vzapi.Verrazzano) error {
	// Do not allow disabling active components
	if err := f.checkEnabled(old, new); err != nil {
		return err
	}
	if err := validateFluentd(new); err != nil {
		return err
	}
	return f.HelmComponent.ValidateUpdate(old, new)
}

func (f fluentdComponent) checkEnabled(old *vzapi.Verrazzano, new *vzapi.Verrazzano) error {
	// Do not allow disabling of any component post-install for now
	if f.IsEnabled(old) && !f.IsEnabled(new) {
		return fmt.Errorf("disabling component %s is not allowed", ComponentJSONName)
	}
	return nil
}

// PostInstall - post-install, clean up temp files
func (f fluentdComponent) PostInstall(ctx spi.ComponentContext) error {
	cleanTempFiles(ctx)
	// populate the ingress and certificate names before calling PostInstall on Helm component because those will be needed there
	//f.HelmComponent.IngressNames = c.GetIngressNames(ctx)
	//f.HelmComponent.Certificates = c.GetCertificateNames(ctx)
	return f.HelmComponent.PostInstall(ctx)
}

// PostUpgrade Fluentd component post-upgrade processing
func (f fluentdComponent) PostUpgrade(ctx spi.ComponentContext) error {
	ctx.Log().Debugf("Fluentd component post-upgrade")
	cleanTempFiles(ctx)
	/*f.HelmComponent.IngressNames = f.GetIngressNames(ctx)
	f.HelmComponent.Certificates = f.GetCertificateNames(ctx)
	if vzconfig.IsVMOEnabled(ctx.EffectiveCR()) {
		if err := common.ReassociateVMOResources(ctx); err != nil {
			return err
		}
	}*/
	return f.HelmComponent.PostUpgrade(ctx)
}

// PreInstall Fluentd component pre-install processing; create and label required namespaces, copy any
// required secrets
func (f fluentdComponent) PreInstall(ctx spi.ComponentContext) error {
	if err := loggingPreInstall(ctx); err != nil {
		return ctx.Log().ErrorfNewErr("Failed copying logging secrets for Verrazzano: %v", err)
	}
	return nil
}

// Install Fluentd component install processing
func (f fluentdComponent) Install(ctx spi.ComponentContext) error {
	if err := f.HelmComponent.Install(ctx); err != nil {
		return err
	}
	return nil
}

// PreUpgrade Fluentd component pre-upgrade processing
func (f fluentdComponent) PreUpgrade(ctx spi.ComponentContext) error {
	return fluentdPreUpgrade(ctx, ComponentNamespace)
}

// Upgrade Fluentd component upgrade processing
func (f fluentdComponent) Upgrade(ctx spi.ComponentContext) error {
	if err := f.HelmComponent.Upgrade(ctx); err != nil {
		return err
	}
	return nil
}

// IsReady component check
func (f fluentdComponent) IsReady(ctx spi.ComponentContext) bool {
	if f.HelmComponent.IsReady(ctx) {
		return isFluentdReady(ctx)
	}
	return false
}

// IsInstalled component check
func (f fluentdComponent) IsInstalled(ctx spi.ComponentContext) (bool, error) {
	installed, err := f.HelmComponent.IsInstalled(ctx)
	return installed, err
}

// IsEnabled fluentd-specific enabled check for installation
func (f fluentdComponent) IsEnabled(effectiveCR *vzapi.Verrazzano) bool {
	comp := effectiveCR.Spec.Components.Fluentd
	if comp == nil || comp.Enabled == nil {
		return true
	}
	return *comp.Enabled
}
