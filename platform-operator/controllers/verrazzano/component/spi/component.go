// Copyright (c) 2020, 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package spi

import (
	"github.com/verrazzano/verrazzano/pkg/log/vzlog"
	"github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1beta1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	clipkg "sigs.k8s.io/controller-runtime/pkg/client"
)

// ComponentContext Defines the context objects required for Component operations
type ComponentContext interface {
	// Log returns the logger for the context
	Log() vzlog.VerrazzanoLogger
	// GetClient returns the controller client for the context
	Client() clipkg.Client
	// ActualCR returns the actual unmerged v1alpha1.Verrazzano resource
	ActualCR() *v1alpha1.Verrazzano
	// EffectiveCR returns the effective merged v1alpha1.Verrazzano CR
	EffectiveCR() *v1alpha1.Verrazzano
	// ActualCRV1Beta1 returns the actual unmerged v1beta1.Verrazzano resource
	ActualCRV1Beta1() *v1beta1.Verrazzano
	// EffectiveCRV1Beta1 returns the effective merged v1beta1.Verrazzano CR
	EffectiveCRV1Beta1() *v1beta1.Verrazzano
	// IsDryRun indicates the component context is in DryRun mode
	IsDryRun() bool
	// Copy returns a copy of the current context
	Copy() ComponentContext
	// Init returns a copy of the current context with an updated logging component field
	Init(comp string) ComponentContext
	// Operation specifies the logging operation field
	Operation(op string) ComponentContext
	// GetOperation returns the operation object in the context
	GetOperation() string
	// GetComponent returns the component object in the context
	GetComponent() string
}

// ComponentInfo interface defines common information and metadata about components
type ComponentInfo interface {
	// Name returns the name of the Verrazzano component
	Name() string
	// Namespace returns the namespace of the Verrazzano component
	Namespace() string
	// GetDependencies returns the dependencies of this component
	GetDependencies() []string
	// IsReady Indicates whether or not a component is available and ready
	IsReady(context ComponentContext) bool
	// IsEnabled Indicates whether or a component is enabled for installation
	IsEnabled(effectiveCR runtime.Object) bool
	// GetMinVerrazzanoVersion returns the minimum Verrazzano version required by the component
	GetMinVerrazzanoVersion() string
	// GetIngressNames returns a list of names of the ingresses associated with the component
	GetIngressNames(context ComponentContext) []types.NamespacedName
	// GetCertificateNames returns a list of names of the TLS certificates associated with the component
	GetCertificateNames(context ComponentContext) []types.NamespacedName
	// GetJsonName returns the josn name of the verrazzano component in CRD
	GetJSONName() string
	// GetOverrides returns the list of overrides for a component
	GetOverrides(effectiveCR runtime.Object) interface{}
	// MonitorOverrides indicates whether the override sources for a component need to be monitored
	MonitorOverrides(context ComponentContext) bool
}

// ComponentInstaller interface defines installs operations for components that support it
type ComponentInstaller interface {
	// IsOperatorInstallSupported Returns true if the component supports install directly via the platform operator
	// - scaffolding while we move components from the scripts to the operator
	IsOperatorInstallSupported() bool
	// IsInstalled Indicates whether or not the component is installed
	IsInstalled(context ComponentContext) (bool, error)
	// PreInstall allows components to perform any pre-processing required prior to initial install
	PreInstall(context ComponentContext) error
	// Install performs the initial install of a component
	Install(context ComponentContext) error
	// PostInstall allows components to perform any post-processing required after initial install
	PostInstall(context ComponentContext) error
}

// ComponentUninstaller interface defines uninstall operations
type ComponentUninstaller interface {
	// IsOperatorUninstallSupported Returns true if the component supports uninstall directly via the platform operator
	// - scaffolding while we move components from the scripts to the operator
	IsOperatorUninstallSupported() bool
	// PreUninstall allows components to perform any pre-processing required prior to upgrading
	PreUninstall(context ComponentContext) error
	// Uninstall will Uninstall the Verrazzano component specified in the CR.Version field
	Uninstall(context ComponentContext) error
	// PostUninstall allows components to perform any post-processing required after upgrading
	PostUninstall(context ComponentContext) error
}

// ComponentUpgrader interface defines upgrade operations for components that support it
type ComponentUpgrader interface {
	// PreUpgrade allows components to perform any pre-processing required prior to upgrading
	PreUpgrade(context ComponentContext) error
	// Upgrade will upgrade the Verrazzano component specified in the CR.Version field
	Upgrade(context ComponentContext) error
	// PostUpgrade allows components to perform any post-processing required after upgrading
	PostUpgrade(context ComponentContext) error
}

// ComponentValidator interface defines validation operations for components that support it
type ComponentValidator interface {
	// ValidateInstall checks if the specified Verrazzano CR is valid for this component to be installed
	ValidateInstall(vz *v1alpha1.Verrazzano) error
	// ValidateUpdate checks if the specified new Verrazzano CR is valid for this component to be updated
	ValidateUpdate(old *v1alpha1.Verrazzano, new *v1alpha1.Verrazzano) error
	// ValidateInstall checks if the specified Verrazzano CR is valid for this component to be installed
	ValidateInstallV1Beta1(vz *v1beta1.Verrazzano) error
	// ValidateUpdate checks if the specified new Verrazzano CR is valid for this component to be updated
	ValidateUpdateV1Beta1(old *v1beta1.Verrazzano, new *v1beta1.Verrazzano) error
}

// Generate mocs for the spi.Component interface for use in tests.
//go:generate mockgen -destination=../../../../mocks/component_mock.go -package=mocks -copyright_file=../../../../hack/boilerplate.go.txt github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi Component

// Component interface defines the methods implemented by components
type Component interface {
	ComponentInfo
	ComponentInstaller
	ComponentUninstaller
	ComponentUpgrader
	ComponentValidator

	Reconcile(ctx ComponentContext) error
}
