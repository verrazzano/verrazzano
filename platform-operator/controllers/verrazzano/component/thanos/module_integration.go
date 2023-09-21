// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package thanos

import (
	"github.com/verrazzano/verrazzano-modules/pkg/controller/spi/controllerspi"
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/common/watch"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/fluentoperator"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/istio"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/nginx"
	promoperator "github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/prometheus/operator"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
)

//type valuesConfig struct {
//	IstioInjectionEnabled bool `json:"istioInjectionEnabled"`
//}

// GetModuleConfigAsHelmValues returns an unstructured JSON snippet representing the portion of the Verrazzano CR that corresponds to the module
func (c ThanosComponent) GetModuleConfigAsHelmValues(effectiveCR *vzapi.Verrazzano) (*apiextensionsv1.JSON, error) {
	return nil, nil
	//if effectiveCR == nil {
	//	return nil, nil
	//}
	//
	//configSnippet := valuesConfig{
	//	IstioInjectionEnabled: vzcr.IsIstioInjectionEnabled(effectiveCR),
	//}
	//
	//return spi.NewModuleConfigHelmValuesWrapper(configSnippet)
}

// GetWatchDescriptors returns the list of WatchDescriptors for objects being watched by the component
func (c ThanosComponent) GetWatchDescriptors() []controllerspi.WatchDescriptor {
	return watch.CombineWatchDescriptors(
		watch.GetModuleInstalledWatches([]string{istio.ComponentName, nginx.ComponentName, promoperator.ComponentName, fluentoperator.ComponentName}),
		watch.GetModuleUpdatedWatches([]string{nginx.ComponentName, istio.ComponentName, promoperator.ComponentName}),
	)
}
