// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package rancher

import (
	"github.com/verrazzano/verrazzano-modules/pkg/controller/spi/controllerspi"
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	cmconstants "github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/certmanager/constants"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/common/watch"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/fluentoperator"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/nginx"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"reflect"
)

// valuesConfig Structure for the translated effective Verrazzano CR values to Module CR Helm values
type valuesConfig struct {
	KeycloakAuthEnabled *bool `json:"keycloakAuthEnabled,omitempty"`
}

var emptyConfig = valuesConfig{}

// GetModuleConfigAsHelmValues returns an unstructured JSON valuesConfig representing the portion of the Verrazzano CR that corresponds to the module
func (r rancherComponent) GetModuleConfigAsHelmValues(effectiveCR *vzapi.Verrazzano) (*apiextensionsv1.JSON, error) {
	if effectiveCR == nil {
		return nil, nil
	}

	configSnippet := valuesConfig{}

	rancher := effectiveCR.Spec.Components.Rancher
	if rancher != nil {
		// The prometheus operator digs into the Thanos overrides for the Thanos ruler settings, so we need to trigger on that
		configSnippet.KeycloakAuthEnabled = rancher.KeycloakAuthEnabled
	}

	if reflect.DeepEqual(emptyConfig, configSnippet) {
		return nil, nil
	}

	return spi.NewModuleConfigHelmValuesWrapper(configSnippet)
}

// GetWatchDescriptors returns the list of WatchDescriptors for objects being watched by the component
func (r rancherComponent) GetWatchDescriptors() []controllerspi.WatchDescriptor {
	return watch.CombineWatchDescriptors(
		watch.GetModuleInstalledWatches([]string{nginx.ComponentName, cmconstants.CertManagerComponentName, cmconstants.ClusterIssuerComponentName, fluentoperator.ComponentName}),
		watch.GetModuleUpdatedWatches([]string{nginx.ComponentName, cmconstants.CertManagerComponentName, cmconstants.ClusterIssuerComponentName, fluentoperator.ComponentName}),
		watch.GetCreateNamespaceWatch(CattleGlobalDataNamespace),
		watch.GetUpdateNamespaceWatch(CattleGlobalDataNamespace),
	)
}
