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
	IngressNames            []types.NamespacedName
	SupportsOperatorInstall bool
}

var _ ComponentInfo = &ComponentInfoImpl{}

type ComponentInternal interface {
	ReconcileSteadyState(ctx ComponentContext) error
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
