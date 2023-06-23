// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package clusterapi

import (
	"context"
	"fmt"

	"github.com/verrazzano/verrazzano/pkg/constants"
	"github.com/verrazzano/verrazzano/pkg/k8s/ready"
	"github.com/verrazzano/verrazzano/pkg/vzcr"
	"github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1beta1"
	vpoconstants "github.com/verrazzano/verrazzano/platform-operator/constants"
	cmconstants "github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/certmanager/constants"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/common"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	clusterapi "sigs.k8s.io/cluster-api/cmd/clusterctl/client"
)

// ComponentName is the name of the component
const ComponentName = "cluster-api"

// Namespace for CAPI providers
const ComponentNamespace = constants.VerrazzanoCAPINamespace

// ComponentJSONName is the JSON name of the component in CRD
const ComponentJSONName = "clusterAPI"

const (
	capiCMDeployment                 = "capi-controller-manager"
	capiOcneBootstrapCMDeployment    = "capi-ocne-bootstrap-controller-manager"
	capiOcneControlPlaneCMDeployment = "capi-ocne-control-plane-controller-manager"
	capiociCMDeployment              = "capoci-controller-manager"
	ocneProviderName                 = "ocne"
	ociProviderName                  = "oci"
	clusterAPIProviderName           = "cluster-api"
)

var capiDeployments = []types.NamespacedName{
	{
		Name:      capiCMDeployment,
		Namespace: ComponentNamespace,
	},
	{
		Name:      capiOcneBootstrapCMDeployment,
		Namespace: ComponentNamespace,
	},
	{
		Name:      capiOcneControlPlaneCMDeployment,
		Namespace: ComponentNamespace,
	},
	{
		Name:      capiociCMDeployment,
		Namespace: ComponentNamespace,
	},
}

type CAPIInitFuncType = func(path string, options ...clusterapi.Option) (clusterapi.Client, error)

var capiInitFunc = clusterapi.New

// SetCAPIInitFunc For unit testing, override the CAPI init function
func SetCAPIInitFunc(f CAPIInitFuncType) {
	capiInitFunc = f
}

// ResetCAPIInitFunc For unit testing, reset the CAPI init function to its default
func ResetCAPIInitFunc() {
	capiInitFunc = clusterapi.New
}

type clusterAPIComponent struct {
}

func NewComponent() spi.Component {
	return clusterAPIComponent{}
}

// Name returns the component name.
func (c clusterAPIComponent) Name() string {
	return ComponentName
}

// Namespace returns the component namespace.
func (c clusterAPIComponent) Namespace() string {
	return ComponentNamespace
}

// ShouldInstallBeforeUpgrade returns true if component can be installed before upgrade is done.
func (c clusterAPIComponent) ShouldInstallBeforeUpgrade() bool {
	return false
}

// GetDependencies returns the dependencies of this component.
func (c clusterAPIComponent) GetDependencies() []string {
	return []string{cmconstants.CertManagerComponentName, cmconstants.ClusterIssuerComponentName}
}

// IsReady indicates whether a component is Ready for dependency components.
func (c clusterAPIComponent) IsReady(ctx spi.ComponentContext) bool {
	prefix := fmt.Sprintf("Component %s", ctx.GetComponent())
	return ready.DeploymentsAreReady(ctx.Log(), ctx.Client(), capiDeployments, 1, prefix)
}

// IsAvailable indicates whether a component is Available for end users.
func (c clusterAPIComponent) IsAvailable(ctx spi.ComponentContext) (reason string, available v1alpha1.ComponentAvailability) {
	return (&ready.AvailabilityObjects{DeploymentNames: capiDeployments}).IsAvailable(ctx.Log(), ctx.Client())
}

// IsEnabled returns true if component is enabled for installation.
func (c clusterAPIComponent) IsEnabled(effectiveCR runtime.Object) bool {
	return vzcr.IsClusterAPIEnabled(effectiveCR)
}

// GetMinVerrazzanoVersion returns the minimum Verrazzano version required by the component
func (c clusterAPIComponent) GetMinVerrazzanoVersion() string {
	return vpoconstants.VerrazzanoVersion1_6_0
}

// GetIngressNames returns the list of ingress names associated with the component
func (c clusterAPIComponent) GetIngressNames(_ spi.ComponentContext) []types.NamespacedName {
	return []types.NamespacedName{}
}

// GetCertificateNames returns the list of expected certificates used by this component
func (c clusterAPIComponent) GetCertificateNames(_ spi.ComponentContext) []types.NamespacedName {
	return []types.NamespacedName{}
}

// GetJSONName returns the json name of the verrazzano component in CRD
func (c clusterAPIComponent) GetJSONName() string {
	return ComponentJSONName
}

// GetOverrides returns the Helm override sources for a component
func (c clusterAPIComponent) GetOverrides(object runtime.Object) interface{} {
	if effectiveCR, ok := object.(*v1alpha1.Verrazzano); ok {
		if effectiveCR.Spec.Components.ClusterAPI != nil {
			return effectiveCR.Spec.Components.ClusterAPI.ValueOverrides
		}
		return []v1alpha1.Overrides{}
	} else if effectiveCR, ok := object.(*v1beta1.Verrazzano); ok {
		if effectiveCR.Spec.Components.ClusterAPI != nil {
			return effectiveCR.Spec.Components.ClusterAPI.ValueOverrides
		}
		return []v1beta1.Overrides{}
	}

	return []v1alpha1.Overrides{}
}

// MonitorOverrides indicates whether monitoring of override sources is enabled for a component
func (c clusterAPIComponent) MonitorOverrides(ctx spi.ComponentContext) bool {
	if ctx.EffectiveCR().Spec.Components.ClusterAPI != nil {
		if ctx.EffectiveCR().Spec.Components.ClusterAPI.MonitorChanges != nil {
			return *ctx.EffectiveCR().Spec.Components.ClusterAPI.MonitorChanges
		}
		return true
	}
	return false
}

func (c clusterAPIComponent) IsOperatorInstallSupported() bool {
	return true
}

// IsInstalled checks to see if ClusterAPI is installed
func (c clusterAPIComponent) IsInstalled(ctx spi.ComponentContext) (bool, error) {
	deployment := &appsv1.Deployment{}
	err := ctx.Client().Get(context.TODO(), types.NamespacedName{Namespace: ComponentNamespace, Name: capiCMDeployment}, deployment)
	if errors.IsNotFound(err) {
		return false, nil
	}
	if err != nil {
		ctx.Log().Errorf("Failed to get %s/%s deployment: %v", ComponentNamespace, capiCMDeployment, err)
		return false, err
	}
	return true, nil
}

func (c clusterAPIComponent) PreInstall(ctx spi.ComponentContext) error {
	// If already installed, treat as an upgrade
	installed, err := c.IsInstalled(ctx)
	if err != nil {
		return err
	}
	if installed {
		return preUpgrade(ctx)
	}

	return preInstall(ctx)
}

func (c clusterAPIComponent) Install(ctx spi.ComponentContext) error {
	// If already installed, treat as an upgrade
	installed, err := c.IsInstalled(ctx)
	if err != nil {
		return err
	}
	if installed {
		return c.Upgrade(ctx)
	}

	capiClient, err := capiInitFunc("")
	if err != nil {
		return err
	}

	overrides, err := createOverrides(ctx)
	if err != nil {
		return err
	}

	overridesContext := newOverridesContext(overrides)

	// Set up the init options for the CAPI init.
	initOptions := clusterapi.InitOptions{
		CoreProvider:            fmt.Sprintf("%s:%s", clusterAPIProviderName, overridesContext.GetClusterAPIVersion()),
		BootstrapProviders:      []string{fmt.Sprintf("%s:%s", ocneProviderName, overridesContext.GetOCNEBootstrapVersion())},
		ControlPlaneProviders:   []string{fmt.Sprintf("%s:%s", ocneProviderName, overridesContext.GetOCNEControlPlaneVersion())},
		InfrastructureProviders: []string{fmt.Sprintf("%s:%s", ociProviderName, overridesContext.GetOCIVersion())},
		TargetNamespace:         ComponentNamespace,
	}

	_, err = capiClient.Init(initOptions)
	return err
}

func (c clusterAPIComponent) PostInstall(ctx spi.ComponentContext) error {
	return common.ActivateKontainerDriver(ctx)
}

func (c clusterAPIComponent) IsOperatorUninstallSupported() bool {
	return true
}

func (c clusterAPIComponent) PreUninstall(_ spi.ComponentContext) error {
	return nil
}

func (c clusterAPIComponent) Uninstall(ctx spi.ComponentContext) error {
	capiClient, err := capiInitFunc("")
	if err != nil {
		return err
	}

	overrides, err := createOverrides(ctx)
	if err != nil {
		return err
	}

	overridesContext := newOverridesContext(overrides)

	// Set up the delete options for the CAPI delete operation.
	deleteOptions := clusterapi.DeleteOptions{
		CoreProvider:            fmt.Sprintf("%s:%s", clusterAPIProviderName, overridesContext.GetClusterAPIVersion()),
		BootstrapProviders:      []string{fmt.Sprintf("%s:%s", ocneProviderName, overridesContext.GetOCNEBootstrapVersion())},
		ControlPlaneProviders:   []string{fmt.Sprintf("%s:%s", ocneProviderName, overrides.GetOCNEControlPlaneVersion())},
		InfrastructureProviders: []string{fmt.Sprintf("%s:%s", ociProviderName, overridesContext.GetOCIVersion())},
		IncludeNamespace:        true,
	}
	return capiClient.Delete(deleteOptions)
}

func (c clusterAPIComponent) PostUninstall(_ spi.ComponentContext) error {
	return nil
}

func (c clusterAPIComponent) PreUpgrade(ctx spi.ComponentContext) error {
	return preUpgrade(ctx)
}

func (c clusterAPIComponent) Upgrade(ctx spi.ComponentContext) error {
	capiClient, err := capiInitFunc("")
	if err != nil {
		return err
	}

	overrides, err := createOverrides(ctx)
	if err != nil {
		return err
	}
	overridesContext := newOverridesContext(overrides)

	// Set up the upgrade options for the CAPI apply upgrade.
	const formatString = "%s/%s:%s"
	applyUpgradeOptions := clusterapi.ApplyUpgradeOptions{
		CoreProvider:            fmt.Sprintf(formatString, ComponentNamespace, clusterAPIProviderName, overridesContext.GetClusterAPIVersion()),
		BootstrapProviders:      []string{fmt.Sprintf(formatString, ComponentNamespace, ocneProviderName, overridesContext.GetOCNEBootstrapVersion())},
		ControlPlaneProviders:   []string{fmt.Sprintf(formatString, ComponentNamespace, ocneProviderName, overrides.GetOCNEControlPlaneVersion())},
		InfrastructureProviders: []string{fmt.Sprintf(formatString, ComponentNamespace, ociProviderName, overridesContext.GetOCIVersion())},
	}

	return capiClient.ApplyUpgrade(applyUpgradeOptions)
}

func (c clusterAPIComponent) PostUpgrade(ctx spi.ComponentContext) error {
	return common.ActivateKontainerDriver(ctx)
}

func (c clusterAPIComponent) ValidateInstall(vz *v1alpha1.Verrazzano) error {
	return nil
}

func (c clusterAPIComponent) ValidateUpdate(old *v1alpha1.Verrazzano, new *v1alpha1.Verrazzano) error {
	if c.IsEnabled(old) && !c.IsEnabled(new) {
		return fmt.Errorf("Disabling component %s is not allowed", ComponentJSONName)
	}
	return nil
}

func (c clusterAPIComponent) ValidateInstallV1Beta1(vz *v1beta1.Verrazzano) error {
	return nil
}

func (c clusterAPIComponent) ValidateUpdateV1Beta1(old *v1beta1.Verrazzano, new *v1beta1.Verrazzano) error {
	if c.IsEnabled(old) && !c.IsEnabled(new) {
		return fmt.Errorf("Disabling component %s is not allowed", ComponentJSONName)
	}
	return nil
}

// Reconcile reconciles the ClusterAPI component
func (c clusterAPIComponent) Reconcile(ctx spi.ComponentContext) error {
	return nil
}
