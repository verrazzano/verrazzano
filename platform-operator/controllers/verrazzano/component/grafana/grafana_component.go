// Copyright (c) 2022, 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package grafana

import (
	"fmt"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"

	"github.com/verrazzano/verrazzano/pkg/k8s/ready"
	"github.com/verrazzano/verrazzano/pkg/k8sutil"
	"github.com/verrazzano/verrazzano/pkg/vzcr"
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	installv1beta1 "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1beta1"
	"github.com/verrazzano/verrazzano/platform-operator/constants"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/common"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/fluentoperator"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/grafanadashboards"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/networkpolicies"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/vmo"
)

const (
	// ComponentName is the name of the component
	ComponentName = "grafana"

	// ComponentNamespace is the namespace of the component
	ComponentNamespace = constants.VerrazzanoSystemNamespace

	// grafanaCertificateName is the name of the TLS certificate used for ingress
	grafanaCertificateName = "system-tls-grafana"

	// fluentbitFilterAndParserTemplate is the template name that consists Fluentbit Filter and Parser resource for Grafana.
	fluentbitFilterAndParserTemplate = "grafana-filter-parser.yaml"
)

// ComponentJSONName is the JSON name of the component in the Verrazzano CRD
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
	return []string{networkpolicies.ComponentName, vmo.ComponentName, grafanadashboards.ComponentName, fluentoperator.ComponentName}
}

// GetCertificateNames returns the Grafana certificate names if Nginx is enabled, otherwise returns
// an empty slice
func (g grafanaComponent) GetCertificateNames(ctx spi.ComponentContext) []types.NamespacedName {
	var certificateNames []types.NamespacedName

	if vzcr.IsNGINXEnabled(ctx.EffectiveCR()) {
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

	if vzcr.IsNGINXEnabled(ctx.EffectiveCR()) {
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
	return vzcr.IsGrafanaEnabled(effectiveCR)
}

// IsInstalled returns true if the Grafana component is installed
func (g grafanaComponent) IsInstalled(ctx spi.ComponentContext) (bool, error) {
	return isGrafanaInstalled(ctx), nil
}

func (g grafanaComponent) IsAvailable(ctx spi.ComponentContext) (reason string, available vzapi.ComponentAvailability) {
	return (&ready.AvailabilityObjects{DeploymentNames: newDeployments()}).IsAvailable(ctx.Log(), ctx.Client())
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
	if err := applyDatasourcesConfigmap(ctx); err != nil {
		return err
	}
	return common.CreateOrUpdateVMI(ctx, updateFunc)
}

// PostInstall checks post install conditions
func (g grafanaComponent) PostInstall(ctx spi.ComponentContext) error {
	if err := common.CheckIngressesAndCerts(ctx, g); err != nil {
		return err
	}
	if err := common.CreateOrDeleteFluentbitFilterAndParser(ctx, fluentbitFilterAndParserTemplate, ComponentNamespace, false); err != nil {
		return err
	}
	return restartGrafanaPod(ctx)
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
	if err := common.CreateOrDeleteFluentbitFilterAndParser(context, fluentbitFilterAndParserTemplate, ComponentNamespace, true); err != nil {
		return err
	}
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
	if err := applyDatasourcesConfigmap(ctx); err != nil {
		return err
	}
	return common.CreateOrUpdateVMI(ctx, updateFunc)
}

// PostUpgrade checks post upgrade conditions and restarts the Grafana pod to ensure that any changes
// to the datasources configmap are picked up
func (g grafanaComponent) PostUpgrade(ctx spi.ComponentContext) error {
	if err := common.CheckIngressesAndCerts(ctx, g); err != nil {
		return err
	}
	return restartGrafanaPod(ctx)
}

// ValidateUpdate checks if the specified new Verrazzano CR is valid for this component to be updated
func (g grafanaComponent) ValidateUpdate(old *vzapi.Verrazzano, new *vzapi.Verrazzano) error {
	// do not allow disabling active components
	if vzcr.IsGrafanaEnabled(old) && !vzcr.IsGrafanaEnabled(new) {
		return fmt.Errorf("Disabling component %s not allowed", ComponentJSONName)
	}
	return nil
}

// ValidateUpdate checks if the specified new Verrazzano CR is valid for this component to be updated
func (g grafanaComponent) ValidateUpdateV1Beta1(old *installv1beta1.Verrazzano, new *installv1beta1.Verrazzano) error {
	// do not allow disabling active components
	if vzcr.IsGrafanaEnabled(old) && !vzcr.IsGrafanaEnabled(new) {
		return fmt.Errorf("Disabling component %s not allowed", ComponentJSONName)
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
	if !vzcr.IsGrafanaEnabled(vz) {
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
