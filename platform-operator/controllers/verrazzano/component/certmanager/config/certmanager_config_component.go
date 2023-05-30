// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package config

import (
	vzconst "github.com/verrazzano/verrazzano/pkg/constants"
	"github.com/verrazzano/verrazzano/pkg/vzcr"
	"github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1beta1"
	"github.com/verrazzano/verrazzano/platform-operator/constants"
	cmcommon "github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/certmanager/common"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/certmanager/controller"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/certmanager/ocidns"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/common"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/networkpolicies"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
)

// ComponentName is the name of the component
const ComponentName = cmcommon.ClusterIssuerConfigComponentName

// ComponentNamespace is the namespace of the component
const ComponentNamespace = vzconst.CertManagerNamespace

// ComponentJSONName - this is not a real component but declare it for compatibility
const ComponentJSONName = "clusterIssuer"

// clusterIssuerComponent represents an CertManager component
type clusterIssuerComponent struct{}

// Verify that clusterIssuerComponent implements Component
var _ spi.Component = clusterIssuerComponent{}

// NewComponent returns a new CertManager component
func NewComponent() spi.Component {
	return clusterIssuerComponent{}
}

// IsEnabled returns true if the cert-manager-config is enabled, which is the default
func (c clusterIssuerComponent) IsEnabled(effectiveCR runtime.Object) bool {
	return vzcr.IsClusterIssuerEnabled(effectiveCR) || vzcr.IsCertManagerEnabled(effectiveCR)
}

// IsReady component check
func (c clusterIssuerComponent) IsReady(ctx spi.ComponentContext) bool {
	if ctx.IsDryRun() {
		ctx.Log().Debug("cert-manager-config IsReady dry run")
		return true
	}
	return c.verrazzanoCertManagerResourcesReady(ctx)
}

func (c clusterIssuerComponent) IsInstalled(ctx spi.ComponentContext) (bool, error) {
	if ctx.IsDryRun() {
		ctx.Log().Debug("cert-manager-config IsInstalled dry run")
		return true, nil
	}
	return c.verrazzanoCertManagerResourcesReady(ctx), nil
}

// PreInstall runs before cert-manager-config component is executed
func (c clusterIssuerComponent) PreInstall(compContext spi.ComponentContext) error {
	// If it is a dry-run, do nothing
	if compContext.IsDryRun() {
		compContext.Log().Debug("cert-manager-config PreInstall dry run")
		return nil
	}
	if err := common.ProcessAdditionalCertificates(compContext.Log(), compContext.Client(), compContext.EffectiveCR()); err != nil {
		return err
	}
	return nil
}

func (c clusterIssuerComponent) Install(compContext spi.ComponentContext) error {
	if compContext.IsDryRun() {
		compContext.Log().Debug("cert-manager-config Install dry run")
		return nil
	}
	// Set up cluster issuer, eventually perhaps move resource config to a chart or a different controller
	return c.createOrUpdateClusterIssuer(compContext)
}

func (c clusterIssuerComponent) Upgrade(compContext spi.ComponentContext) error {
	if compContext.IsDryRun() {
		compContext.Log().Debug("cert-manager-config Upgrade dry run")
		return nil
	}
	// Update cluster issuer and certs if necessary, eventually perhaps move resource config to a chart or a different controller
	return c.Install(compContext)
}

func (c clusterIssuerComponent) PreUpgrade(compContext spi.ComponentContext) error {
	return nil
}

// Uninstall removes cert-manager-config objects that are created outside of Helm
func (c clusterIssuerComponent) Uninstall(compContext spi.ComponentContext) error {
	if compContext.IsDryRun() {
		compContext.Log().Debug("cert-manager-config Uninstall dry run")
		return nil
	}
	return c.uninstallVerrazzanoCertManagerResources(compContext)
}

// ValidateInstall checks if the specified new Verrazzano CR is valid for this component to be installed
func (c clusterIssuerComponent) ValidateInstall(vz *v1alpha1.Verrazzano) error {
	vzV1Beta1 := &v1beta1.Verrazzano{}
	if err := vz.ConvertTo(vzV1Beta1); err != nil {
		return err
	}
	return c.ValidateInstallV1Beta1(vzV1Beta1)
}

// ValidateInstallV1Beta1 checks if the specified new Verrazzano CR is valid for this component to be installed
func (c clusterIssuerComponent) ValidateInstallV1Beta1(vz *v1beta1.Verrazzano) error {
	if err := c.validateConfiguration(vz); err != nil {
		return err
	}
	return nil
}

// ValidateUpdate checks if the specified new Verrazzano CR is valid for this component to be updated
func (c clusterIssuerComponent) ValidateUpdate(old *v1alpha1.Verrazzano, new *v1alpha1.Verrazzano) error {
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

// ValidateUpdateV1Beta1 checks if the specified new Verrazzano CR is valid for this component to be updated
func (c clusterIssuerComponent) ValidateUpdateV1Beta1(old *v1beta1.Verrazzano, new *v1beta1.Verrazzano) error {
	if err := c.validateConfiguration(new); err != nil {
		return err
	}
	return nil
}

func (c clusterIssuerComponent) validateConfiguration(new *v1beta1.Verrazzano) error {
	if err := validateLongestHostName(new); err != nil {
		return err
	}

	if !c.IsEnabled(new) && !vzcr.IsCertManagerEnabled(new) {
		return nil
	}

	if err := validateConfiguration(new); err != nil {
		return err
	}
	return nil
}

func (c clusterIssuerComponent) Name() string {
	return ComponentName
}

func (c clusterIssuerComponent) Namespace() string {
	return ComponentNamespace
}

func (c clusterIssuerComponent) GetJSONName() string {
	return ComponentJSONName
}

func (c clusterIssuerComponent) ShouldInstallBeforeUpgrade() bool {
	return false
}

func (c clusterIssuerComponent) GetDependencies() []string {
	return []string{networkpolicies.ComponentName, controller.ComponentName, ocidns.ComponentName}
}

func (c clusterIssuerComponent) IsAvailable(context spi.ComponentContext) (string, v1alpha1.ComponentAvailability) {
	if c.IsReady(context) {
		return "", v1alpha1.ComponentAvailable
	}
	return "Waiting for ClusterIssuer to be ready", v1alpha1.ComponentUnavailable
}

func (c clusterIssuerComponent) GetMinVerrazzanoVersion() string {
	return constants.VerrazzanoVersion1_0_0
}

func (c clusterIssuerComponent) GetIngressNames(context spi.ComponentContext) []types.NamespacedName {
	return []types.NamespacedName{}
}

func (c clusterIssuerComponent) GetCertificateNames(context spi.ComponentContext) []types.NamespacedName {
	return []types.NamespacedName{}
}

func (c clusterIssuerComponent) GetOverrides(effectiveCR runtime.Object) interface{} {
	return []v1alpha1.Overrides{}
}

func (c clusterIssuerComponent) MonitorOverrides(context spi.ComponentContext) bool {
	return true
}

func (c clusterIssuerComponent) IsOperatorInstallSupported() bool {
	return true
}

func (c clusterIssuerComponent) PostInstall(context spi.ComponentContext) error {
	return nil
}

func (c clusterIssuerComponent) IsOperatorUninstallSupported() bool {
	return true
}

func (c clusterIssuerComponent) PreUninstall(context spi.ComponentContext) error {
	return nil
}

func (c clusterIssuerComponent) PostUninstall(context spi.ComponentContext) error {
	return nil
}

func (c clusterIssuerComponent) PostUpgrade(context spi.ComponentContext) error {
	return nil
}

func (c clusterIssuerComponent) Reconcile(ctx spi.ComponentContext) error {
	return nil
}
