// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package opensearchdashboards

import (
	"fmt"
	installv1beta1 "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1beta1"
	"k8s.io/apimachinery/pkg/runtime"

	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/constants"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/common"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/vmo"
	"github.com/verrazzano/verrazzano/platform-operator/internal/vzconfig"
	"k8s.io/apimachinery/pkg/types"
)

const (
	// ComponentName is the name of the component
	ComponentName = "opensearch-dashboards"

	// ComponentNamespace is the namespace of the component
	ComponentNamespace = constants.VerrazzanoSystemNamespace

	// Certificate names
	osdCertificateName = "system-tls-opensearchdashboards"
)

// ComponentJSONName is the json name of the OpenSearch-Dashboards component in CRD
const ComponentJSONName = "opensearch-dashboards"

type opensearchDashboardsComponent struct{}

// Namespace returns the component namespace
func (d opensearchDashboardsComponent) Namespace() string {
	return ComponentNamespace
}

// ShouldInstallBeforeUpgrade returns true if component can be installed before upgrade is done
func (d opensearchDashboardsComponent) ShouldInstallBeforeUpgrade() bool {
	return false
}

// GetDependencies returns the dependencies of the OpenSearch-Dashbaords component
func (d opensearchDashboardsComponent) GetDependencies() []string {
	return []string{vmo.ComponentName}
}

// GetMinVerrazzanoVersion returns the minimum Verrazzano version required by the OpenSearch-Dashboards component
func (d opensearchDashboardsComponent) GetMinVerrazzanoVersion() string {
	return constants.VerrazzanoVersion1_0_0
}

// GetJSONName returns the json name of the OpenSearch-Dashboards component in CRD
func (d opensearchDashboardsComponent) GetJSONName() string {
	return ComponentJSONName
}

// GetOverrides returns the Helm override sources for a component
func (d opensearchDashboardsComponent) GetOverrides(object runtime.Object) interface{} {
	if _, ok := object.(*vzapi.Verrazzano); ok {
		return []vzapi.Overrides{}
	} else if _, ok := object.(*installv1beta1.Verrazzano); ok {
		return []installv1beta1.Overrides{}
	}
	return []vzapi.Overrides{}
}

// MonitorOverrides indicates whether monitoring of Helm override sources is enabled for a component
func (d opensearchDashboardsComponent) MonitorOverrides(_ spi.ComponentContext) bool {
	return true
}

// IsOperatorInstallSupported OpenSearch-Dashboards component function
func (d opensearchDashboardsComponent) IsOperatorInstallSupported() bool {
	return true
}

// IsInstalled OpenSearch-Dashboards component function
func (d opensearchDashboardsComponent) IsInstalled(ctx spi.ComponentContext) (bool, error) {
	return doesOSDExist(ctx), nil
}

// Reconcile OpenSearch-Dashboards component function
func (d opensearchDashboardsComponent) Reconcile(ctx spi.ComponentContext) error {
	return nil
}

// NewComponent OpenSearch-Dashboards component function
func NewComponent() spi.Component {
	return opensearchDashboardsComponent{}
}

// PreInstall OpenSearch-Dashboards component pre-install processing; create and label required namespaces, copy any
// required secrets
func (d opensearchDashboardsComponent) PreInstall(ctx spi.ComponentContext) error {
	// create or update  VMI secret
	if err := common.EnsureVMISecret(ctx.Client()); err != nil {
		return err
	}
	// create or update backup VMI secret
	if err := common.EnsureBackupSecret(ctx.Client()); err != nil {
		return err
	}
	ctx.Log().Debug("OpenSearch-Dashboards pre-install")
	if err := common.CreateAndLabelVMINamespaces(ctx); err != nil {
		return ctx.Log().ErrorfNewErr("Failed creating/labeling namespace %s for OpenSearch-Dashboards : %v", ComponentNamespace, err)
	}
	return nil
}

// Install OpenSearch-Dashboards component install processing
func (d opensearchDashboardsComponent) Install(ctx spi.ComponentContext) error {
	return common.CreateOrUpdateVMI(ctx, updateFunc)
}

func (d opensearchDashboardsComponent) IsOperatorUninstallSupported() bool {
	return false
}

func (d opensearchDashboardsComponent) PreUninstall(context spi.ComponentContext) error {
	return nil
}

func (d opensearchDashboardsComponent) Uninstall(context spi.ComponentContext) error {
	return nil
}

func (d opensearchDashboardsComponent) PostUninstall(context spi.ComponentContext) error {
	return nil
}

// PreUpgrade OpenSearch-Dashboards component pre-upgrade processing
func (d opensearchDashboardsComponent) PreUpgrade(ctx spi.ComponentContext) error {
	// create or update  VMI secret
	return common.EnsureVMISecret(ctx.Client())
}

// Upgrade OpenSearch-Dashboards component upgrade processing
func (d opensearchDashboardsComponent) Upgrade(ctx spi.ComponentContext) error {
	return common.CreateOrUpdateVMI(ctx, updateFunc)
}

func (d opensearchDashboardsComponent) IsAvailable(context spi.ComponentContext) (reason string, available bool) {
	available = d.IsReady(context)
	if available {
		return fmt.Sprintf("%s is available", d.Name()), true
	}
	return fmt.Sprintf("%s is unavailable: failed readiness checks", d.Name()), false
}

// IsReady component check
func (d opensearchDashboardsComponent) IsReady(ctx spi.ComponentContext) bool {
	return isOSDReady(ctx)
}

// PostInstall OpenSearch-Dashboards post-install processing
func (d opensearchDashboardsComponent) PostInstall(ctx spi.ComponentContext) error {
	ctx.Log().Debugf("OpenSearch-Dashboards component post-upgrade")
	return common.CheckIngressesAndCerts(ctx, d)

}

// PostUpgrade OpenSearch-Dashboards post-upgrade processing
func (d opensearchDashboardsComponent) PostUpgrade(ctx spi.ComponentContext) error {
	ctx.Log().Debugf("OpenSearch-Dashboards component post-upgrade")
	return common.CheckIngressesAndCerts(ctx, d)

}

// IsEnabled OpenSearch-Dashboards specific enabled check for installation
func (d opensearchDashboardsComponent) IsEnabled(effectiveCR runtime.Object) bool {
	return vzconfig.IsOpenSearchDashboardsEnabled(effectiveCR)
}

// ValidateUpdate checks if the specified new Verrazzano CR is valid for this component to be updated
func (d opensearchDashboardsComponent) ValidateUpdate(old *vzapi.Verrazzano, new *vzapi.Verrazzano) error {
	// Do not allow disabling active components
	if err := d.isOpenSearchDashboardEnabled(old, new); err != nil {
		return err
	}
	// Do not allow any updates to storage settings via the volumeClaimSpecTemplates/defaultVolumeSource
	if err := common.CompareStorageOverrides(old, new, ComponentJSONName); err != nil {
		return err
	}
	return nil
}

// ValidateInstall checks if the specified Verrazzano CR is valid for this component to be installed
func (d opensearchDashboardsComponent) ValidateInstall(_ *vzapi.Verrazzano) error {
	return nil
}

// ValidateInstall checks if the specified Verrazzano CR is valid for this component to be installed
func (d opensearchDashboardsComponent) ValidateInstallV1Beta1(vz *installv1beta1.Verrazzano) error {
	return nil
}

// ValidateUpdate checks if the specified new Verrazzano CR is valid for this component to be updated
func (d opensearchDashboardsComponent) ValidateUpdateV1Beta1(old *installv1beta1.Verrazzano, new *installv1beta1.Verrazzano) error {
	// Do not allow disabling active components
	if err := d.isOpenSearchDashboardEnabled(old, new); err != nil {
		return err
	}
	// Do not allow any updates to storage settings via the volumeClaimSpecTemplates/defaultVolumeSource
	if err := common.CompareStorageOverridesV1Beta1(old, new, ComponentJSONName); err != nil {
		return err
	}
	return nil
}

// Name returns the component name
func (d opensearchDashboardsComponent) Name() string {
	return ComponentName
}

func (d opensearchDashboardsComponent) isOpenSearchDashboardEnabled(old runtime.Object, new runtime.Object) error {
	// Do not allow disabling of any component post-install for now
	if vzconfig.IsOpenSearchDashboardsEnabled(old) && !vzconfig.IsOpenSearchDashboardsEnabled(new) {
		return fmt.Errorf("Disabling component OpenSearch-Dashboards not allowed")
	}
	return nil
}

// GetIngressNames - gets the names of the ingresses associated with this component
func (d opensearchDashboardsComponent) GetIngressNames(ctx spi.ComponentContext) []types.NamespacedName {
	var ingressNames []types.NamespacedName

	if vzconfig.IsNGINXEnabled(ctx.EffectiveCR()) {
		ingressNames = append(ingressNames, types.NamespacedName{
			Namespace: ComponentNamespace,
			Name:      constants.OpensearchDashboardsIngress,
		})
	}

	return ingressNames
}

// GetCertificateNames - gets the names of the certificates associated with this component
func (d opensearchDashboardsComponent) GetCertificateNames(_ spi.ComponentContext) []types.NamespacedName {
	return []types.NamespacedName{
		{
			Namespace: ComponentNamespace,
			Name:      osdCertificateName,
		},
	}
}
