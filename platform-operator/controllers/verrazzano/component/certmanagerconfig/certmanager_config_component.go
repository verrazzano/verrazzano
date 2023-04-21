// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package certmanagerconfig

import (
	"github.com/verrazzano/verrazzano/pkg/log/vzlog"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/certmanager"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/common"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/networkpolicies"
	"k8s.io/apimachinery/pkg/runtime"

	vzconst "github.com/verrazzano/verrazzano/pkg/constants"
	"github.com/verrazzano/verrazzano/platform-operator/constants"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/helm"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
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

var certManagerCRDNames = []string{
	"certificaterequests.cert-manager.io",
	"orders.acme.cert-manager.io",
	"certificates.cert-manager.io",
	"clusterissuers.cert-manager.io",
	"issuers.cert-manager.io",
}

// NewComponent returns a new CertManager component
func NewComponent() spi.Component {
	return certManagerConfigComponent{
		helm.HelmComponent{
			ReleaseName: ComponentName,
			JSONName:    ComponentJSONName,
			//ChartDir:                  filepath.Join(config.GetThirdPartyDir(), "cert-manager-config"),
			ChartNamespace:            ComponentNamespace,
			IgnoreNamespaceOverride:   true,
			SupportsOperatorInstall:   true,
			SupportsOperatorUninstall: true,
			//ImagePullSecretKeyname:    "global.imagePullSecrets[0].name",
			MinVerrazzanoVersion: constants.VerrazzanoVersion1_0_0,
			Dependencies:         []string{networkpolicies.ComponentName, certmanager.ComponentName},
		},
	}
}

// IsEnabled returns true if the cert-manager-config is enabled, which is the default
func (c certManagerConfigComponent) IsEnabled(_ runtime.Object) bool {
	exists, err := c.checkExistingCertManagerResources()
	if err != nil {
		vzlog.DefaultLogger().ErrorfThrottled("CertManager config: unexpected error checking for existing Cert-Manager: %v", err)
	}
	return exists
}

// IsReady component check
func (c certManagerConfigComponent) IsReady(ctx spi.ComponentContext) bool {
	return c.verrazzanoCertManagerResourcesReady(ctx)
}

func (c certManagerConfigComponent) IsInstalled(ctx spi.ComponentContext) (bool, error) {
	return c.verrazzanoCertManagerResourcesReady(ctx), nil
}

//// ValidateInstall checks if the specified new Verrazzano CR is valid for this component to be installed
//func (c certManagerConfigComponent) ValidateInstall(vz *v1alpha1.Verrazzano) error {
//	vzV1Beta1 := &v1beta1.Verrazzano{}
//	if err := vz.ConvertTo(vzV1Beta1); err != nil {
//		return err
//	}
//	return c.ValidateInstallV1Beta1(vzV1Beta1)
//}
//
//// ValidateInstallV1Beta1 checks if the specified new Verrazzano CR is valid for this component to be installed
//func (c certManagerConfigComponent) ValidateInstallV1Beta1(vz *v1beta1.Verrazzano) error {
//	cmCRDsExists, err := c.checkExistingCertManagerResources()
//	if err != nil {
//		return err
//	}
//	if !cmCRDsExists && !vzcr.IsCertManagerEnabled(vz) {
//		return fmt.Errorf("no Cert-Manager installation detected, required for Verrazzano installation")
//	}
//	return nil
//}

// PreInstall runs before cert-manager-config component is executed
func (c certManagerConfigComponent) PreInstall(compContext spi.ComponentContext) error {
	// If it is a dry-run, do nothing
	if compContext.IsDryRun() {
		compContext.Log().Debug("cert-manager-config PostInstall dry run")
		return nil
	}
	if err := c.certManagerExistsInCluster(compContext); err != nil {
		return err
	}
	if err := common.ProcessAdditionalCertificates(compContext.Log(), compContext.Client(), compContext.EffectiveCR()); err != nil {
		return err
	}
	return nil
}

func (c certManagerConfigComponent) Install(compContext spi.ComponentContext) error {
	// Set up cluster issuer, eventually perhaps move resource config to a chart or a different controller
	return c.createOrUpdateClusterIssuer(compContext)
}

// PostInstall applies necessary cert-manager-config resources after the install has occurred
// In the case of an Acme cert, we install Acme resources
// In the case of a CA cert, we install CA resources
//func (c certManagerConfigComponent) PostInstall(compContext spi.ComponentContext) error {
//	// If it is a dry-run, do nothing
//	if compContext.IsDryRun() {
//		compContext.Log().Debug("cert-manager-config PostInstall dry run")
//		return nil
//	}
//}

func (c certManagerConfigComponent) Upgrade(compContext spi.ComponentContext) error {
	// Update cluster issuer and certs if necessary, eventually perhaps move resource config to a chart or a different controller
	return c.Install(compContext)
}

func (c certManagerConfigComponent) PreUpgrade(compContext spi.ComponentContext) error {
	return c.certManagerExistsInCluster(compContext)
}

// PostUpgrade applies necessary cert-manager-config resources after upgrade has occurred
// In the case of an Acme cert, we install/update Acme resources
// In the case of a CA cert, we install/update CA resources
//func (c certManagerConfigComponent) PostUpgrade(compContext spi.ComponentContext) error {
//	// If it is a dry-run, do nothing
//	if compContext.IsDryRun() {
//		compContext.Log().Debug("cert-manager-config PostInstall dry run")
//		return nil
//	}
//	return c.createOrUpdateClusterIssuer(compContext)
//}

// Uninstall removes cert-manager-config objects that are created outside of Helm
func (c certManagerConfigComponent) Uninstall(compContext spi.ComponentContext) error {
	if compContext.IsDryRun() {
		compContext.Log().Debug("cert-manager-configPostUninstall dry run")
		return nil
	}
	if err := c.certManagerExistsInCluster(compContext); err != nil {
		return err
	}
	return uninstallVerrazzanoCertManagerResources(compContext)
}

//func (c certManagerConfigComponent) PostUninstall(context spi.ComponentContext) error {
//	// Nothing to install, eventually perhaps move resource config to a chart or a different controller
//	return nil
//}
