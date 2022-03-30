// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
package spi

import (
	vzconst "github.com/verrazzano/verrazzano/platform-operator/constants"
	"k8s.io/apimachinery/pkg/types"
)

// TODO - Can work around some these, but might be better to do in stages
//   - ComponentContext builder, ComponentContext is polluted with builder semantics
//   - investigate change signature of Reconcile to return an object and err, and let controller update the status
//      - status of comp
//      - ?q
//   - May be able to move component_adapter back to spi package

type ComponentInfoImpl struct {
	ComponentName           string
	MinVersion              string
	Dependencies            []string
	SupportsOperatorInstall bool
	// Ingress names associated with the component
	IngressNames []types.NamespacedName
	// Certificates associated with the component
	Certificates []types.NamespacedName
	// JSONName is the josn name of the verrazzano component in CRD
	JSONName string
}

var _ ComponentInfo = &ComponentInfoImpl{}

// ComponentInstaller interface defines installs operations for components that support it
type ComponentInstaller interface {
	// PreInstall allows components to perform any pre-processing required prior to initial install
	PreInstall(context ComponentContext) error
	// Install performs the initial install of a component
	Install(context ComponentContext) error
	// PostInstall allows components to perform any post-processing required after initial install
	PostInstall(context ComponentContext) error
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

// ComponentInternal - Common internal component lifecycle operations
//
// TODO:
// 	- some or all of these operations/interfaces may go away, and have each component manage itself.  Reconcile() is the
//    main entry point for the controller into components, and whatever happens internally is up to the component
// 	- There may be some benefit to a structured internal lifecycle, but we'll need to evaluate that based on what the truly
//    common aspects are and what are unique to each component
type ComponentInternal interface {
	ComponentInstaller

	//ReconcileSteadyState(ctx ComponentContext) error
}

func (d *ComponentInfoImpl) GetMinVerrazzanoVersion() string {
	if len(d.MinVersion) == 0 {
		return vzconst.VerrazzanoVersion1_0_0
	}
	return d.MinVersion
}

func (d *ComponentInfoImpl) GetIngressNames(_ ComponentContext) []types.NamespacedName {
	return d.IngressNames
}

func (d *ComponentInfoImpl) IsOperatorInstallSupported() bool {
	return d.SupportsOperatorInstall
}

func (d *ComponentInfoImpl) Name() string {
	return d.ComponentName
}

func (d *ComponentInfoImpl) GetDependencies() []string {
	return d.Dependencies
}

func (d *ComponentInfoImpl) GetCertificateNames(context ComponentContext) []types.NamespacedName {
	return d.Certificates
}

func (d *ComponentInfoImpl) GetJSONName() string {
	return d.JSONName
}
