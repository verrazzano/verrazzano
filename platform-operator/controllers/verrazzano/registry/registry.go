// Copyright (c) 2021, 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package registry

import (
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/appoper"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/certmanager"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/coherence"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/externaldns"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/istio"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/keycloak"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/kiali"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/mysql"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/nginx"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/oam"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/rancher"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/verrazzano"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/weblogic"
)

type VzComponentRegistry struct{}

var _ spi.ComponentRegistry = VzComponentRegistry{}

type GetCompoentsFnType func() []spi.Component

var getComponentsFn = getComponents

var componentsRegistry []spi.Component

// OverrideGetComponentsFn Allows overriding the set of registry components for testing purposes
func OverrideGetComponentsFn(fnType GetCompoentsFnType) {
	getComponentsFn = fnType
}

// ResetGetComponentsFn Restores the GetComponents implementation to the default if it's been overridden for testing
func ResetGetComponentsFn() {
	getComponentsFn = getComponents
}

// GetComponents returns the list of components that are installable and upgradeable.
// The components will be processed in the order items in the array
// The components will be processed in the order items in the array
func (r VzComponentRegistry) GetComponents() []spi.Component {
	return getComponentsFn()
}

// getComponents is the internal impl function for GetComponents, to allow overriding it for testing purposes
func getComponents() []spi.Component {
	if len(componentsRegistry) == 0 {
		componentsRegistry = []spi.Component{
			istio.NewComponent(),
			nginx.NewComponent(),
			certmanager.NewComponent(),
			externaldns.NewComponent(),
			rancher.NewComponent(),
			verrazzano.NewComponent(),
			coherence.NewComponent(),
			weblogic.NewComponent(),
			oam.NewComponent(),
			appoper.NewComponent(),
			mysql.NewComponent(),
			keycloak.NewComponent(),
			kiali.NewComponent(),
		}
	}
	return componentsRegistry
}

func (r VzComponentRegistry) FindComponent(releaseName string) (bool, spi.Component) {
	for _, comp := range r.GetComponents() {
		if comp.Name() == releaseName {
			return true, comp
		}
	}
	return false, nil
}
