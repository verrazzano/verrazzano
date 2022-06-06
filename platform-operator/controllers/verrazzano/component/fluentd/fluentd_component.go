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

	tmpFilePrefix        = "verrazzano-fluentd-overrides-"
	tmpSuffix            = "yaml"
	tmpFileCreatePattern = tmpFilePrefix + "*." + tmpSuffix
	tmpFileCleanPattern  = tmpFilePrefix + ".*\\." + tmpSuffix
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
	if f.IsEnabled(old) && !f.IsEnabled(new) {
		return fmt.Errorf("disabling component %s is not allowed", ComponentJSONName)
	}
	if err := validateFluentd(new); err != nil {
		return err
	}
	return f.HelmComponent.ValidateUpdate(old, new)
}

// PostInstall - post-install, clean up temp files
func (f fluentdComponent) PostInstall(ctx spi.ComponentContext) error {
	cleanTempFiles(ctx)
	// populate the ingress and certificate names before calling PostInstall on Helm component because those will be needed there
	//f.HelmComponent.IngressNames = c.GetIngressNames(ctx)
	//f.HelmComponent.Certificates = c.GetCertificateNames(ctx)
	return f.HelmComponent.PostInstall(ctx)
}

// PostUpgrade Verrazzano-Fluentd-post-upgrade processing
func (f fluentdComponent) PostUpgrade(ctx spi.ComponentContext) error {
	ctx.Log().Debugf("Verrazzano Fluentd component post-upgrade")
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

// PreInstall Verrazzano component pre-install processing; create and label required namespaces, copy any
// required secrets
func (f fluentdComponent) PreInstall(ctx spi.ComponentContext) error {
	if err := loggingPreInstall(ctx); err != nil {
		return ctx.Log().ErrorfNewErr("Failed copying logging secrets for Verrazzano: %v", err)
	}
	return nil
}
