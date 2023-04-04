// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package capi

import (
	"github.com/verrazzano/verrazzano/pkg/k8s/ready"
	"github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1beta1"
	"github.com/verrazzano/verrazzano/platform-operator/constants"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
)

// ComponentName is the name of the component
const ComponentName = "verrazzano-capi"

// ComponentNamespace is the namespace of the component
const ComponentNamespace = constants.VerrazzanoSystemNamespace

// ComponentJSONName is the JSON name of the verrazzano component in CRD
const ComponentJSONName = "verrazzano-capi"

var capiDeployments = []types.NamespacedName{}

type capiComponent struct {
}

func NewComponent() spi.Component {
	return capiComponent{}
}

// Name returns the component name.
func (c capiComponent) Name() string {
	return ComponentName
}

// Namespace returns the component namespace.
func (c capiComponent) Namespace() string {
	return ComponentNamespace
}

// ShouldInstallBeforeUpgrade returns true if component can be installed before upgrade is done.
func (c capiComponent) ShouldInstallBeforeUpgrade() bool {
	return false
}

// GetDependencies returns the dependencies of this component.
func (c capiComponent) GetDependencies() []string {
	return []string{}
}

// IsReady indicates whether a component is Ready for dependency components.
func (c capiComponent) IsReady(context spi.ComponentContext) bool {
	return true
}

// IsAvailable indicates whether a component is Available for end users.
func (c capiComponent) IsAvailable(ctx spi.ComponentContext) (reason string, available v1alpha1.ComponentAvailability) {
	return (&ready.AvailabilityObjects{DeploymentNames: capiDeployments}).IsAvailable(ctx.Log(), ctx.Client())
}

// IsEnabled returns true if component is enabled for installation.
func (c capiComponent) IsEnabled(effectiveCR runtime.Object) bool {
	return true
	// TODO: uncomment when component is added to verrazzano API
	// return vzcr.IsCapiEnabled(effectiveCR)
}

// GetMinVerrazzanoVersion returns the minimum Verrazzano version required by the component
func (c capiComponent) GetMinVerrazzanoVersion() string {
	return constants.VerrazzanoVersion1_6_0
}

// GetIngressNames returns the list of ingress names associated with the component
func (c capiComponent) GetIngressNames(_ spi.ComponentContext) []types.NamespacedName {
	return []types.NamespacedName{}
}

// GetCertificateNames returns the list of expected certificates used by this component
func (c capiComponent) GetCertificateNames(_ spi.ComponentContext) []types.NamespacedName {
	return []types.NamespacedName{}
}

// GetJSONName returns the json name of the verrazzano component in CRD
func (c capiComponent) GetJSONName() string {
	return ComponentJSONName
}

// GetOverrides returns the Helm override sources for a component
func (c capiComponent) GetOverrides(object runtime.Object) interface{} {
	// TODO: update when capi component is added to Verrazzano API
	if _, ok := object.(*v1alpha1.Verrazzano); ok {
		//		if effectiveCR.Spec.Components.Capi != nil {
		//			return effectiveCR.Spec.Components.Capi.ValueOverrides
		//		}
		return []v1alpha1.Overrides{}
	}
	//effectiveCR := object.(*v1beta1.Verrazzano)
	//	if effectiveCR.Spec.Components.Capi != nil {
	//		return effectiveCR.Spec.Components.Capi.ValueOverrides
	//	}
	return []v1beta1.Overrides{}
}

// MonitorOverrides indicates whether monitoring of override sources is enabled for a component
func (c capiComponent) MonitorOverrides(_ spi.ComponentContext) bool {
	// TODO: update when capi component is added to Verrazzano API
	//	if ctx.EffectiveCR().Spec.Components.Capi == nil {
	//		return false
	//	}
	//	if ctx.EffectiveCR().Spec.Components.Capi.MonitorChanges != nil {
	//		return *ctx.EffectiveCR().Spec.Components.Istio.MonitorChanges
	//	}
	return true
}

func (c capiComponent) IsOperatorInstallSupported() bool {
	return true
}

func (c capiComponent) IsInstalled(_ spi.ComponentContext) (bool, error) {
	// TODO: how do we check if capi is installed
	return false, nil
}

func (c capiComponent) PreInstall(_ spi.ComponentContext) error {
	return nil
}

func (c capiComponent) Install(_ spi.ComponentContext) error {
	return nil
}

func (c capiComponent) PostInstall(_ spi.ComponentContext) error {
	return nil
}

func (c capiComponent) IsOperatorUninstallSupported() bool {
	return true
}

func (c capiComponent) PreUninstall(_ spi.ComponentContext) error {
	return nil
}

func (c capiComponent) Uninstall(_ spi.ComponentContext) error {
	return nil
}

func (c capiComponent) PostUninstall(_ spi.ComponentContext) error {
	return nil
}

func (c capiComponent) PreUpgrade(_ spi.ComponentContext) error {
	return nil
}

func (c capiComponent) Upgrade(_ spi.ComponentContext) error {
	return nil
}

func (c capiComponent) PostUpgrade(_ spi.ComponentContext) error {
	return nil
}

func (c capiComponent) ValidateInstall(vz *v1alpha1.Verrazzano) error {
	return nil
}

func (c capiComponent) ValidateUpdate(old *v1alpha1.Verrazzano, new *v1alpha1.Verrazzano) error {
	return nil
}

func (c capiComponent) ValidateInstallV1Beta1(vz *v1beta1.Verrazzano) error {
	return nil
}

func (c capiComponent) ValidateUpdateV1Beta1(old *v1beta1.Verrazzano, new *v1beta1.Verrazzano) error {
	return nil
}

// Reconcile reconciles the CAPI component
func (c capiComponent) Reconcile(ctx spi.ComponentContext) error {
	return nil
}
