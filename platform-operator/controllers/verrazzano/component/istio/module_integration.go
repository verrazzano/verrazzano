// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package istio

import (
	"github.com/verrazzano/verrazzano-modules/pkg/controller/spi/controllerspi"
	"github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/common/watch"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/fluentoperator"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
)

// issuerValuesConfig Structure for the translated effective Verrazzano CR values to Module CR Helm values
type valuesConfig struct {
	Istio *v1alpha1.IstioComponent `json:"istio,omitempty"`
}

// GetModuleConfigAsHelmValues returns an unstructured JSON issuerValuesConfig representing the portion of the Verrazzano CR that corresponds to the module
func (i istioComponent) GetModuleConfigAsHelmValues(effectiveCR *v1alpha1.Verrazzano) (*apiextensionsv1.JSON, error) {
	if effectiveCR == nil {
		return nil, nil
	}

	istio := effectiveCR.Spec.Components.Istio
	if istio == nil {
		return nil, nil
	}
	configSnippet := valuesConfig{
		Istio: istio.DeepCopy(),
	}
	configSnippet.Istio.InstallOverrides = v1alpha1.InstallOverrides{}

	return spi.NewModuleConfigHelmValuesWrapper(configSnippet)
}

// ShouldUseModule returns true if component is implemented using a Module
func (i istioComponent) ShouldUseModule() bool {
	return true
}

// GetWatchDescriptors returns the list of WatchDescriptors for objects being watched by the component
func (i istioComponent) GetWatchDescriptors() []controllerspi.WatchDescriptor {
	return watch.GetModuleInstalledWatches([]string{fluentoperator.ComponentName})
}
