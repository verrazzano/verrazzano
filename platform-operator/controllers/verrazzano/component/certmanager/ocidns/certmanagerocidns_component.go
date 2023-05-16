// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package ocidns

import (
	"github.com/verrazzano/verrazzano/pkg/constants"
	"github.com/verrazzano/verrazzano/pkg/log/vzlog"
	"github.com/verrazzano/verrazzano/pkg/vzcr"
	cmcommon "github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/certmanager/common"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/certmanager/controller"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/common"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/helm"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/networkpolicies"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	"github.com/verrazzano/verrazzano/platform-operator/internal/config"
	"k8s.io/apimachinery/pkg/runtime"
	"path/filepath"
)

// ComponentName is the name of the component
const ComponentName = cmcommon.CertManagerOCIDNSComponentName

// ComponentNamespace is the namespace of the component
const ComponentNamespace = constants.VerrazzanoSystemNamespace

const componentChartName = "verrazzano-cert-manager-ocidns-webhook"

// certManagerOciDnsComponent represents an CertManager component
type certManagerOciDNSComponent struct {
	helm.HelmComponent
}

// Verify that certManagerComponent implements Component
var _ spi.Component = certManagerOciDNSComponent{}

// NewComponent returns a new CertManager component
func NewComponent() spi.Component {
	return certManagerOciDNSComponent{
		helm.HelmComponent{
			ReleaseName:               ComponentName,
			ChartDir:                  filepath.Join(config.GetHelmChartsDir(), componentChartName),
			ChartNamespace:            ComponentNamespace,
			IgnoreNamespaceOverride:   true,
			SupportsOperatorInstall:   true,
			SupportsOperatorUninstall: true,
			InstallBeforeUpgrade:      true,
			AppendOverridesFunc:       appendOCIDNSOverrides,
			ImagePullSecretKeyname:    "global.imagePullSecrets[0].name",
			Dependencies:              []string{networkpolicies.ComponentName, controller.ComponentName},
		},
	}
}

func (c certManagerOciDNSComponent) PreInstall(ctx spi.ComponentContext) error {
	if err := common.CopyOCIDNSSecret(ctx, constants.CertManagerNamespace); err != nil {
		return err
	}
	return nil
}

// IsEnabled returns true if the cert-manager is enabled, which is the default
func (c certManagerOciDNSComponent) IsEnabled(effectiveCR runtime.Object) bool {
	logger := vzlog.DefaultLogger()
	err := cmcommon.CertManagerExistsInCluster(logger)
	if err != nil {
		logger.ErrorfThrottled("Unexpected error checking for CertManager in cluster: %v", err)
		return false
	}
	isACMEConfig, err := cmcommon.IsACMEConfig(effectiveCR)
	if err != nil {
		logger.ErrorfThrottled("Unexpected error checking certificate configuration: %v", err.Error())
		return false
	}
	return isACMEConfig && vzcr.IsOCIDNSEnabled(effectiveCR)
}

func (c certManagerOciDNSComponent) PostUninstall(ctx spi.ComponentContext) error {
	return c.postUninstall(ctx)
}

// IsReady component check
func (c certManagerOciDNSComponent) IsReady(ctx spi.ComponentContext) bool {
	if ctx.IsDryRun() {
		ctx.Log().Debug("cert-manager-config PostInstall dry run")
		return true
	}
	if c.HelmComponent.IsReady(ctx) {
		return isCertManagerOciDNSReady(ctx)
	}
	return false
}
