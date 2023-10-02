// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package clusterapi

import (
	"bytes"
	"context"
	"fmt"
	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"os/exec"

	"github.com/verrazzano/verrazzano/pkg/constants"
	"github.com/verrazzano/verrazzano/pkg/k8s/ready"
	"github.com/verrazzano/verrazzano/pkg/k8s/resource"
	"github.com/verrazzano/verrazzano/pkg/log/vzlog"
	"github.com/verrazzano/verrazzano/pkg/vzcr"
	"github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1beta1"
	vpoconstants "github.com/verrazzano/verrazzano/platform-operator/constants"
	cmconstants "github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/certmanager/constants"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
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
	capiVerrazzanoAddonCMDeployment  = "caapv-controller-manager"
	ocneProviderName                 = "ocne"
	ociProviderName                  = "oci"
	clusterAPIProviderName           = "cluster-api"
	verrazzanoAddonProviderName      = "verrazzano"
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
	{
		Name:      capiVerrazzanoAddonCMDeployment,
		Namespace: ComponentNamespace,
	},
}

type capiUpgradeOptions struct {
	CoreProvider            string
	BootstrapProviders      []string
	ControlPlaneProviders   []string
	InfrastructureProviders []string
	AddonProviders          []string
}

// capiRunCmdFunc - required for unit tests
var capiRunCmdFunc func(cmd *exec.Cmd) error

// runCAPICmd - wrapper for executing commands, required for unit testing
func runCAPICmd(cmd *exec.Cmd, log vzlog.VerrazzanoLogger) error {
	if capiRunCmdFunc != nil {
		return capiRunCmdFunc(cmd)
	}
	stdoutBuffer := &bytes.Buffer{}
	stderrBuffer := &bytes.Buffer{}
	cmd.Stdout = stdoutBuffer
	cmd.Stderr = stderrBuffer

	log.Progressf("Component %s is executing the command: %s", ComponentName, cmd.String())
	err := cmd.Run()
	if err != nil {
		log.ErrorfThrottled("command failed with error %s; stdout: %s; stderr: %s", err.Error(), stdoutBuffer.String(), stderrBuffer.String())
	}
	return err
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

// ShouldUseModule returns true if component is implemented using a Module
func (c clusterAPIComponent) ShouldUseModule() bool {
	return true
}

// GetModuleConfigAsHelmValues returns an unstructured JSON snippet representing the portion of the Verrazzano CR that corresponds to the module
func (c clusterAPIComponent) GetModuleConfigAsHelmValues(effectiveCR *v1alpha1.Verrazzano) (*apiextensionsv1.JSON, error) {
	return nil, nil
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

func (c clusterAPIComponent) Exists(context spi.ComponentContext) (bool, error) {
	return c.IsInstalled(context)
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

// IsInstalled checks to see if ClusterAPI providers are installed
func (c clusterAPIComponent) IsInstalled(ctx spi.ComponentContext) (bool, error) {
	found, err := checkClusterAPIDeployment(ctx, capiCMDeployment)
	if !found || err != nil {
		return found, err
	}
	found, err = checkClusterAPIDeployment(ctx, capiociCMDeployment)
	if !found || err != nil {
		return found, err
	}
	found, err = checkClusterAPIDeployment(ctx, capiOcneBootstrapCMDeployment)
	if !found || err != nil {
		return found, err
	}
	found, err = checkClusterAPIDeployment(ctx, capiOcneControlPlaneCMDeployment)
	if !found || err != nil {
		return found, err
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

	overrides, err := createOverrides(ctx)
	if err != nil {
		ctx.Log().ErrorfThrottled("Failed to create overrides for installing cluster-api providers: %v", err)
		return err
	}

	overridesContext := newOverridesContext(overrides)
	coreArgValue := fmt.Sprintf("%s:%s", clusterAPIProviderName, overridesContext.GetClusterAPIVersion())
	controlPlaneArgValue := fmt.Sprintf("%s:%s", ocneProviderName, overridesContext.GetOCNEControlPlaneVersion())
	infrastructureArgValue := fmt.Sprintf("%s:%s", ociProviderName, overridesContext.GetOCIVersion())
	bootstrapArgValue := fmt.Sprintf("%s:%s", ocneProviderName, overridesContext.GetOCNEBootstrapVersion())
	addonArgValue := fmt.Sprintf("%s:%s", verrazzanoAddonProviderName, overridesContext.GetVerrazzanoAddonVersion())
	cmd := exec.Command("clusterctl", "init",
		"--target-namespace", ComponentNamespace,
		"--core", coreArgValue,
		"--control-plane", controlPlaneArgValue,
		"--infrastructure", infrastructureArgValue,
		"--bootstrap", bootstrapArgValue,
		"--addon", addonArgValue)

	return runCAPICmd(cmd, ctx.Log())
}

func (c clusterAPIComponent) PostInstall(ctx spi.ComponentContext) error {
	return nil
}

func (c clusterAPIComponent) IsOperatorUninstallSupported() bool {
	return true
}

func (c clusterAPIComponent) PreUninstall(_ spi.ComponentContext) error {
	return nil
}

func (c clusterAPIComponent) Uninstall(ctx spi.ComponentContext) error {
	cmd := exec.Command("clusterctl", "delete", "--all", "--include-namespace")

	return runCAPICmd(cmd, ctx.Log())
}

func (c clusterAPIComponent) PostUninstall(_ spi.ComponentContext) error {
	return nil
}

func (c clusterAPIComponent) PreUpgrade(ctx spi.ComponentContext) error {
	return preUpgrade(ctx)
}

func (c clusterAPIComponent) Upgrade(ctx spi.ComponentContext) error {
	overrides, err := createOverrides(ctx)
	if err != nil {
		ctx.Log().ErrorfThrottled("Failed to create overrides for upgrading cluster-api providers: %v", err)
		return err
	}

	overridesContext := newOverridesContext(overrides)
	podMatcher := &PodMatcherClusterAPI{}

	// Set up the upgrade options for the CAPI apply upgrade.
	applyUpgradeOptions, err := podMatcher.matchAndPrepareUpgradeOptions(ctx, overridesContext)
	if err != nil {
		ctx.Log().ErrorfThrottled("Failed to setup upgrade options for cluster-api providers: %v", err)
		return err
	}
	if isUpgradeOptionsNotEmpty(applyUpgradeOptions) {
		// get all the resource that will be deleted and recreated
		components, err := getComponentsToUpgrade(ctx.Client(), applyUpgradeOptions)
		if err != nil {
			ctx.Log().ErrorfThrottled("Error generating cluster-api provider components to be upgraded")
			return err
		}

		if err = resource.CleanupResources(ctx, components); err != nil {
			return err
		}
		if err = resource.VerifyResourcesDeleted(ctx, components); err != nil {
			return err
		}

		// Create the variable input list for apply
		args := []string{"upgrade", "apply"}
		if len(applyUpgradeOptions.CoreProvider) > 0 {
			args = append(args, "--core")
			args = append(args, applyUpgradeOptions.CoreProvider)
		}
		if len(applyUpgradeOptions.BootstrapProviders) > 0 {
			args = append(args, "--bootstrap")
			args = append(args, applyUpgradeOptions.BootstrapProviders[0])
		}
		if len(applyUpgradeOptions.ControlPlaneProviders) > 0 {
			args = append(args, "--control-plane")
			args = append(args, applyUpgradeOptions.ControlPlaneProviders[0])
		}
		if len(applyUpgradeOptions.InfrastructureProviders) > 0 {
			args = append(args, "--infrastructure")
			args = append(args, applyUpgradeOptions.InfrastructureProviders[0])
		}
		if len(applyUpgradeOptions.AddonProviders) > 0 {
			args = append(args, "--addon")
			args = append(args, applyUpgradeOptions.AddonProviders[0])
		}

		cmd := exec.Command("clusterctl", args...)
		err = runCAPICmd(cmd, ctx.Log())
		if err != nil {
			return err
		}
	}

	// Initial versions of cluster-api install did not install the Verrazzano cluster-api addon.  If that is the case, we need
	// to install the addon instead of upgrade the addon.
	deployment := appsv1.Deployment{}
	namespacedName := types.NamespacedName{
		Namespace: ComponentNamespace,
		Name:      capiVerrazzanoAddonCMDeployment,
	}
	if err := ctx.Client().Get(context.TODO(), namespacedName, &deployment); err != nil {
		if errors.IsNotFound(err) {
			addonArgValue := fmt.Sprintf("%s:%s", verrazzanoAddonProviderName, overridesContext.GetVerrazzanoAddonVersion())
			cmd := exec.Command("clusterctl", "init",
				"--target-namespace", ComponentNamespace,
				"--addon", addonArgValue)
			return runCAPICmd(cmd, ctx.Log())
		}
		ctx.Log().ErrorfThrottled("Failed to get deployment %v: %v", namespacedName, err)
		return err
	}

	return nil
}

func (c clusterAPIComponent) PostUpgrade(ctx spi.ComponentContext) error {
	return nil
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
