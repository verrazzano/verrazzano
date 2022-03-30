// Copyright (c) 2020, 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package spi

import (
	"github.com/verrazzano/verrazzano/pkg/log/vzlog"
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"k8s.io/apimachinery/pkg/types"
	clipkg "sigs.k8s.io/controller-runtime/pkg/client"
)

// ComponentContext Defines the context objects required for Component operations
type ComponentContext interface {
	// Log returns the logger for the context
	Log() vzlog.VerrazzanoLogger
	// GetClient returns the controller client for the context
	Client() clipkg.Client
	// ActualCR returns the actual unmerged Verrazzano resource
	ActualCR() *vzapi.Verrazzano
	// EffectiveCR returns the effective merged Verrazzano CR
	EffectiveCR() *vzapi.Verrazzano
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
	// GetComponentRegistry returns the current registry of components for this context
	GetComponentRegistry() ComponentRegistry
}

// ComponentInfo interface defines common information and metadata about components
type ComponentInfo interface {
	// Name returns the name of the Verrazzano component
	Name() string
	// GetDependencies returns the dependencies of this component
	GetDependencies() []string
	// GetMinVerrazzanoVersion returns the minimum Verrazzano version required by the component
	GetMinVerrazzanoVersion() string
	// GetIngressNames returns a list of names of the ingresses associated with the component
	GetIngressNames(context ComponentContext) []types.NamespacedName
	// GetCertificateNames returns a list of names of the TLS certificates associated with the component
	GetCertificateNames(context ComponentContext) []types.NamespacedName
	// GetJsonName returns the josn name of the verrazzano component in CRD
	GetJSONName() string
}

// ComponentValidator interface defines validation operations for components that support it
type ComponentValidator interface {
	// ValidateInstall checks if the specified Verrazzano CR is valid for this component to be installed
	ValidateInstall(vz *vzapi.Verrazzano) error
	// ValidateUpdate checks if the specified new Verrazzano CR is valid for this component to be updated
	ValidateUpdate(old *vzapi.Verrazzano, new *vzapi.Verrazzano) error
}

// Generate mocs for the spi.Component interface for use in tests.
//go:generate mockgen -destination=../../../../mocks/component_mock.go -package=mocks -copyright_file=../../../../hack/boilerplate.go.txt github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi Component

// Component interface defines the methods implemented by components
type Component interface {
	ComponentInfo
	ComponentUpgrader // This should move to ComponentInternal once Upgrade moves to the Reconcile method/op
	ComponentValidator

	// IsOperatorInstallSupported Returns true if the component supports install directly via the platform operator
	// - scaffolding while we move components from the scripts to the operator
	IsOperatorInstallSupported() bool

	// IsInstalled Indicates whether or not the component is installed
	IsInstalled(context ComponentContext) (bool, error)

	// IsReady Indicates whether or not a component is available and ready
	IsReady(context ComponentContext) bool

	// IsEnabled Indicates whether or a component is enabled for installation
	IsEnabled(effectiveCR *vzapi.Verrazzano) bool

	Reconcile(ctx ComponentContext) error
}

type ComponentRegistry interface {
	GetComponents() []Component
	FindComponent(releaseName string) (bool, Component)
}
