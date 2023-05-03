// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package certmanagerconfig

import (
	vzconst "github.com/verrazzano/verrazzano/pkg/constants"
	"github.com/verrazzano/verrazzano/pkg/log/vzlog"
	"github.com/verrazzano/verrazzano/platform-operator/constants"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/certmanager"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/certmanagerocidns"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/common"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/helm"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/networkpolicies"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	"k8s.io/apimachinery/pkg/runtime"
)

// ComponentName is the name of the component
const ComponentName = "cert-manager-config"

// ComponentNamespace is the namespace of the component
const ComponentNamespace = vzconst.CertManagerNamespace

// ComponentJSONName - this is not a real component but declare it for compatibility
const ComponentJSONName = "certManagerConfig"

// certManagerConfigComponent represents an CertManager component
type certManagerConfigComponent struct {
	helm.HelmComponent
}

// Verify that certManagerConfigComponent implements Component
var _ spi.Component = certManagerConfigComponent{}

// NewComponent returns a new CertManager component
func NewComponent() spi.Component {
	return certManagerConfigComponent{
		helm.HelmComponent{
			ReleaseName:               ComponentName,
			JSONName:                  ComponentJSONName,
			ChartNamespace:            ComponentNamespace,
			IgnoreNamespaceOverride:   true,
			SupportsOperatorInstall:   true,
			SupportsOperatorUninstall: true,
			MinVerrazzanoVersion:      constants.VerrazzanoVersion1_0_0,
			Dependencies:              []string{networkpolicies.ComponentName, certmanager.ComponentName, certmanagerocidns.ComponentName},
		},
	}
}

// IsEnabled returns true if the cert-manager-config is enabled, which is the default
func (c certManagerConfigComponent) IsEnabled(_ runtime.Object) bool {
	exists, err := common.CertManagerCrdsExist(nil)
	if err != nil {
		vzlog.DefaultLogger().ErrorfThrottled("CertManager config: unexpected error checking for existing Cert-Manager: %v", err)
	}
	return exists
}

// IsReady component check
func (c certManagerConfigComponent) IsReady(ctx spi.ComponentContext) bool {
	if ctx.IsDryRun() {
		ctx.Log().Debug("cert-manager-config IsReady dry run")
		return true
	}
	if !c.cmCRDsExist(ctx.Log(), ctx.Client()) {
		return false
	}
	return c.verrazzanoCertManagerResourcesReady(ctx)
}

func (c certManagerConfigComponent) IsInstalled(ctx spi.ComponentContext) (bool, error) {
	if ctx.IsDryRun() {
		ctx.Log().Debug("cert-manager-config IsInstalled dry run")
		return true, nil
	}
	return c.verrazzanoCertManagerResourcesReady(ctx), nil
}

// PreInstall runs before cert-manager-config component is executed
func (c certManagerConfigComponent) PreInstall(compContext spi.ComponentContext) error {
	// If it is a dry-run, do nothing
	if compContext.IsDryRun() {
		compContext.Log().Debug("cert-manager-config PreInstall dry run")
		return nil
	}
	if err := common.CertManagerExistsInCluster(compContext.Log(), compContext.Client()); err != nil {
		return err
	}
	if err := common.ProcessAdditionalCertificates(compContext.Log(), compContext.Client(), compContext.EffectiveCR()); err != nil {
		return err
	}
	return nil
}

func (c certManagerConfigComponent) Install(compContext spi.ComponentContext) error {
	if compContext.IsDryRun() {
		compContext.Log().Debug("cert-manager-config Install dry run")
		return nil
	}
	// Set up cluster issuer, eventually perhaps move resource config to a chart or a different controller
	return c.createOrUpdateClusterIssuer(compContext)
}

func (c certManagerConfigComponent) Upgrade(compContext spi.ComponentContext) error {
	if compContext.IsDryRun() {
		compContext.Log().Debug("cert-manager-config Upgrade dry run")
		return nil
	}
	// Update cluster issuer and certs if necessary, eventually perhaps move resource config to a chart or a different controller
	return c.Install(compContext)
}

func (c certManagerConfigComponent) PreUpgrade(compContext spi.ComponentContext) error {
	if compContext.IsDryRun() {
		compContext.Log().Debug("cert-manager-config PreUpgrade dry run")
		return nil
	}
	return common.CertManagerExistsInCluster(compContext.Log(), compContext.Client())
}

// Uninstall removes cert-manager-config objects that are created outside of Helm
func (c certManagerConfigComponent) Uninstall(compContext spi.ComponentContext) error {
	if compContext.IsDryRun() {
		compContext.Log().Debug("cert-manager-config Uninstall dry run")
		return nil
	}
	return c.uninstallVerrazzanoCertManagerResources(compContext)
}
