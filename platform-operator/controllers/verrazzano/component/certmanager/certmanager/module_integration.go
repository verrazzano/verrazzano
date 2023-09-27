// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package certmanager

import (
	"github.com/verrazzano/verrazzano-modules/pkg/controller/spi/controllerspi"
	"github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/common"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/common/watch"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/fluentoperator"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"reflect"
)

// valuesConfig Internal component configuration used to communicate Verrazzano CR config for CertManager to
// this component through the Module interface as Helm values
type valuesConfig struct {
	ClusterResourceNamespace string `json:"clusterResourceNamespace,omitempty"`
}

var emptyConfig = valuesConfig{}

// GetModuleConfigAsHelmValues returns an unstructured JSON snippet representing the portion of the Verrazzano CR that corresponds to the module
func (c certManagerComponent) GetModuleConfigAsHelmValues(effectiveCR *v1alpha1.Verrazzano) (*apiextensionsv1.JSON, error) {
	// Convert the CertManager Verrazzano CR config to internal well-known Helm values
	configSnippet := effectiveCR.Spec.Components.CertManager
	if configSnippet == nil {
		return nil, nil
	}
	clusterResourceNamespace := configSnippet.Certificate.CA.ClusterResourceNamespace
	issuerConfig := effectiveCR.Spec.Components.ClusterIssuer
	if issuerConfig != nil {
		clusterResourceNamespace = issuerConfig.ClusterResourceNamespace
	}

	if reflect.DeepEqual(emptyConfig, configSnippet) {
		return nil, nil
	}

	return spi.NewModuleConfigHelmValuesWrapper(
		valuesConfig{
			ClusterResourceNamespace: clusterResourceNamespace,
		},
	)
}

// GetWatchDescriptors returns the list of WatchDescriptors for objects being watched by the component
func (c certManagerComponent) GetWatchDescriptors() []controllerspi.WatchDescriptor {
	return watch.GetModuleInstalledWatches([]string{fluentoperator.ComponentName, common.PrometheusOperatorComponentName})
}
