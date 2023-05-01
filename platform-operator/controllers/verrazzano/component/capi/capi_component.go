// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package capi

import (
	"context"
	"fmt"
	"github.com/verrazzano/verrazzano/pkg/k8s/ready"
	"github.com/verrazzano/verrazzano/pkg/vzcr"
	"github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1beta1"
	"github.com/verrazzano/verrazzano/platform-operator/constants"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/certmanager"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	clusterapi "sigs.k8s.io/cluster-api/cmd/clusterctl/client"
)

// ComponentName is the name of the component
const ComponentName = "capi"

// Namespace for CAPI providers
const VerrazzanoCAPINamespace = "verrazzano-capi"

// ComponentJSONName is the JSON name of the component in CRD
const ComponentJSONName = "capi"

const (
	capiCMDeployment                 = "capi-controller-manager"
	capiOcneBootstrapCMDeployment    = "capi-ocne-bootstrap-controller-manager"
	capiOcneControlPlaneCMDeployment = "capi-ocne-control-plane-controller-manager"
	capiociCMDeployment              = "capoci-controller-manager"
)

var capiDeployments = []types.NamespacedName{
	{
		Name:      capiCMDeployment,
		Namespace: VerrazzanoCAPINamespace,
	},
	{
		Name:      capiOcneBootstrapCMDeployment,
		Namespace: VerrazzanoCAPINamespace,
	},
	{
		Name:      capiOcneControlPlaneCMDeployment,
		Namespace: VerrazzanoCAPINamespace,
	},
	{
		Name:      capiociCMDeployment,
		Namespace: VerrazzanoCAPINamespace,
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
	return VerrazzanoCAPINamespace
}

// ShouldInstallBeforeUpgrade returns true if component can be installed before upgrade is done.
func (c capiComponent) ShouldInstallBeforeUpgrade() bool {
	return false
}

// GetDependencies returns the dependencies of this component.
func (c capiComponent) GetDependencies() []string {
	return []string{certmanager.ComponentName}
}

// IsReady indicates whether a component is Ready for dependency components.
func (c capiComponent) IsReady(ctx spi.ComponentContext) bool {
	prefix := fmt.Sprintf("Component %s", ctx.GetComponent())
	return ready.DeploymentsAreReady(ctx.Log(), ctx.Client(), capiDeployments, 1, prefix)
}

// IsAvailable indicates whether a component is Available for end users.
func (c capiComponent) IsAvailable(ctx spi.ComponentContext) (reason string, available v1alpha1.ComponentAvailability) {
	return (&ready.AvailabilityObjects{DeploymentNames: capiDeployments}).IsAvailable(ctx.Log(), ctx.Client())
}

// IsEnabled returns true if component is enabled for installation.
func (c capiComponent) IsEnabled(effectiveCR runtime.Object) bool {
	return vzcr.IsCAPIEnabled(effectiveCR)
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
	return nil
}

// MonitorOverrides indicates whether monitoring of override sources is enabled for a component
func (c capiComponent) MonitorOverrides(ctx spi.ComponentContext) bool {
	return false
}

func (c capiComponent) IsOperatorInstallSupported() bool {
	return true
}

// IsInstalled checks to see if CAPI is installed
func (c capiComponent) IsInstalled(ctx spi.ComponentContext) (bool, error) {
	daemonSet := &appsv1.Deployment{}
	err := ctx.Client().Get(context.TODO(), types.NamespacedName{Namespace: VerrazzanoCAPINamespace, Name: capiCMDeployment}, daemonSet)
	if errors.IsNotFound(err) {
		return false, nil
	}
	if err != nil {
		ctx.Log().Errorf("Failed to get %s/%s deployment: %v", VerrazzanoCAPINamespace, capiCMDeployment, err)
		return false, err
	}
	return true, nil
}

func (c capiComponent) PreInstall(ctx spi.ComponentContext) error {
	return preInstall(ctx)
}

func (c capiComponent) Install(_ spi.ComponentContext) error {
	capiClient, err := capiInitFunc("")
	if err != nil {
		return err
	}

	// TODO: version of providers should come from the BOM. Is kubeadm optional?
	// Set up the init options for the CAPI init.
	initOptions := clusterapi.InitOptions{
		CoreProvider:            "cluster-api:v1.3.3",
		BootstrapProviders:      []string{"ocne:v0.1.0"},
		ControlPlaneProviders:   []string{"ocne:v0.1.0"},
		InfrastructureProviders: []string{"oci:v0.8.1"},
		TargetNamespace:         VerrazzanoCAPINamespace,
	}

	_, err = capiClient.Init(initOptions)
	return err
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
	capiClient, err := capiInitFunc("")
	if err != nil {
		return err
	}

	// Set up the init options for the CAPI init.
	deleteOptions := clusterapi.DeleteOptions{
		DeleteAll:        true,
		IncludeNamespace: true,
	}
	return capiClient.Delete(deleteOptions)
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
