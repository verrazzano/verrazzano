// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package grafana

import (
	"fmt"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/networkpolicies"

	"github.com/verrazzano/verrazzano/pkg/k8sutil"
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	installv1beta1 "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1beta1"
	"github.com/verrazzano/verrazzano/platform-operator/constants"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/common"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/vmo"
	"github.com/verrazzano/verrazzano/platform-operator/internal/vzconfig"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
)

const (
	// ComponentName is the name of the component
	ComponentName = "grafana"

	// ComponentNamespace is the namespace of the component
	ComponentNamespace = constants.VerrazzanoSystemNamespace

	// grafanaCertificateName is the name of the TLS certificate used for ingress
	grafanaCertificateName = "system-tls-grafana"
)

// ComponentJSONName is the json name of the component in the Verrazzano CRD
const ComponentJSONName = "grafana"

type grafanaComponent struct{}

// NewComponent creates a new Grafana component
func NewComponent() spi.Component {
	return grafanaComponent{}
}

// Name returns the component name
func (g grafanaComponent) Name() string {
	return ComponentName
}

// Namespace returns the component namespace
func (g grafanaComponent) Namespace() string {
	return ComponentNamespace
}

// ShouldInstallBeforeUpgrade returns true if component can be installed before upgrade is done
func (g grafanaComponent) ShouldInstallBeforeUpgrade() bool {
	return false
}

// GetDependencies returns the dependencies of the Grafana component
func (g grafanaComponent) GetDependencies() []string {
	return []string{networkpolicies.ComponentName, vmo.ComponentName}
}

// GetCertificateNames returns the Grafana certificate names if Nginx is enabled, otherwise returns
// an empty slice
func (g grafanaComponent) GetCertificateNames(ctx spi.ComponentContext) []types.NamespacedName {
	var certificateNames []types.NamespacedName

	if vzconfig.IsNGINXEnabled(ctx.EffectiveCR()) {
		certificateNames = append(certificateNames, types.NamespacedName{
			Namespace: ComponentNamespace,
			Name:      grafanaCertificateName,
		})
	}
	return certificateNames
}

// GetIngressNames returns the Grafana ingress names if Nginx is enabled, otherwise returns
// an empty slice
func (g grafanaComponent) GetIngressNames(ctx spi.ComponentContext) []types.NamespacedName {
	var ingressNames []types.NamespacedName

	if vzconfig.IsNGINXEnabled(ctx.EffectiveCR()) {
		ingressNames = append(ingressNames, types.NamespacedName{
			Namespace: ComponentNamespace,
			Name:      constants.GrafanaIngress,
		})
	}

	return ingressNames
}

// GetJSONName returns the component JSON name
func (g grafanaComponent) GetJSONName() string {
	return ComponentJSONName
}

// GetOverrides returns the Helm overrides for a component
func (g grafanaComponent) GetOverrides(object runtime.Object) interface{} {
	if _, ok := object.(*vzapi.Verrazzano); ok {
		return []vzapi.Overrides{}
	}
	return []installv1beta1.Overrides{}
}

// MonitorOverrides indicates if monitoring of override sources is enabled or not for a component
func (g grafanaComponent) MonitorOverrides(_ spi.ComponentContext) bool {
	return true
}

// GetMinVerrazzanoVersion returns the minimum Verrazzano version required by the Grafana component
func (g grafanaComponent) GetMinVerrazzanoVersion() string {
	return constants.VerrazzanoVersion1_0_0
}

// IsOperatorInstallSupported returns the bool value indicating that operator install is supported
func (g grafanaComponent) IsOperatorInstallSupported() bool {
	return true
}

// IsEnabled returns true if the Grafana component is enabled
func (g grafanaComponent) IsEnabled(effectiveCR runtime.Object) bool {
	return vzconfig.IsGrafanaEnabled(effectiveCR)
}

// IsInstalled returns true if the Grafana component is installed
func (g grafanaComponent) IsInstalled(ctx spi.ComponentContext) (bool, error) {
	return isGrafanaInstalled(ctx), nil
}

// IsReady returns true if the Grafana component is ready
func (g grafanaComponent) IsReady(ctx spi.ComponentContext) bool {
	return isGrafanaReady(ctx)
}

// ValidateInstall checks if the specified Verrazzano CR is valid for this component to be installed
func (g grafanaComponent) ValidateInstall(vz *vzapi.Verrazzano) error {
	return checkExistingCNEGrafana(vz)
}

// ValidateInstall checks if the specified Verrazzano CR is valid for this component to be installed
func (g grafanaComponent) ValidateInstallV1Beta1(vz *installv1beta1.Verrazzano) error {
	return checkExistingCNEGrafana(vz)
}

// PreInstall ensures that preconditions are met before installing the Grafana component
func (g grafanaComponent) PreInstall(ctx spi.ComponentContext) error {
	if err := common.EnsureVMISecret(ctx.Client()); err != nil {
		return err
	}
	if err := common.EnsureBackupSecret(ctx.Client()); err != nil {
		return err
	}
	if err := common.CreateAndLabelVMINamespaces(ctx); err != nil {
		return err
	}
	if err := common.EnsureGrafanaAdminSecret(ctx.Client()); err != nil {
		return err
	}

	return common.EnsureGrafanaDatabaseSecret(ctx)
}

// Install performs Grafana install processing
func (g grafanaComponent) Install(ctx spi.ComponentContext) error {
	if err := createGrafanaConfigMaps(ctx); err != nil {
		return err
	}
	return common.CreateOrUpdateVMI(ctx, updateFunc)
}

// PostInstall checks post install conditions
func (g grafanaComponent) PostInstall(ctx spi.ComponentContext) error {
	return common.CheckIngressesAndCerts(ctx, g)
}

func (g grafanaComponent) IsOperatorUninstallSupported() bool {
	return false
}

func (g grafanaComponent) PreUninstall(context spi.ComponentContext) error {
	return nil
}

func (g grafanaComponent) Uninstall(context spi.ComponentContext) error {
	return nil
}

func (g grafanaComponent) PostUninstall(context spi.ComponentContext) error {
	return nil
}

// PreUpgrade ensures that preconditions are met before upgrading the Grafana component
func (g grafanaComponent) PreUpgrade(ctx spi.ComponentContext) error {
	if err := common.EnsureVMISecret(ctx.Client()); err != nil {
		return err
	}
	if err := common.EnsureGrafanaAdminSecret(ctx.Client()); err != nil {
		return err
	}

	return common.EnsureGrafanaDatabaseSecret(ctx)
}

// Install performs Grafana upgrade processing
func (g grafanaComponent) Upgrade(ctx spi.ComponentContext) error {
	if err := createGrafanaConfigMaps(ctx); err != nil {
		return err
	}
	return common.CreateOrUpdateVMI(ctx, updateFunc)
}

// PostUpgrade checks post upgrade conditions
func (g grafanaComponent) PostUpgrade(ctx spi.ComponentContext) error {
	return common.CheckIngressesAndCerts(ctx, g)
}

// ValidateUpdate checks if the specified new Verrazzano CR is valid for this component to be updated
func (g grafanaComponent) ValidateUpdate(old *vzapi.Verrazzano, new *vzapi.Verrazzano) error {
	// do not allow disabling active components
	if vzconfig.IsGrafanaEnabled(old) && !vzconfig.IsGrafanaEnabled(new) {
		return fmt.Errorf("Disabling component Grafana not allowed")
	}
	return nil
}

// ValidateUpdate checks if the specified new Verrazzano CR is valid for this component to be updated
func (g grafanaComponent) ValidateUpdateV1Beta1(old *installv1beta1.Verrazzano, new *installv1beta1.Verrazzano) error {
	// do not allow disabling active components
	if vzconfig.IsGrafanaEnabled(old) && !vzconfig.IsGrafanaEnabled(new) {
		return fmt.Errorf("Disabling component Grafana not allowed")
	}
	return nil
}

// Reconcile reconciles the Grafana component
func (g grafanaComponent) Reconcile(ctx spi.ComponentContext) error {
	return nil
}

// checkExistingGrafana checks if Grafana is already installed
// OLCNE Istio module may have Grafana installed in istio-system namespace
func checkExistingCNEGrafana(vz runtime.Object) error {
	if !vzconfig.IsGrafanaEnabled(vz) {
		return nil
	}
	if err := k8sutil.ErrorIfDeploymentExists(constants.IstioSystemNamespace, ComponentName); err != nil {
		return err
	}
	if err := k8sutil.ErrorIfServiceExists(constants.IstioSystemNamespace, ComponentName); err != nil {
		return err
	}
	return nil
}
