// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package issuer

import (
	"github.com/verrazzano/verrazzano-modules/pkg/controller/spi/controllerspi"
	"github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	cmconstants "github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/certmanager/constants"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/common/watch"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/nginx"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
)

// issuerValuesConfig Structure for the translated effective Verrazzano CR values to Module CR Helm values
type issuerValuesConfig struct {
	IssuerConfig             v1alpha1.IssuerConfig `json:"issuerConfig"`
	ClusterResourceNamespace string                `json:"clusterResourceNamespace,omitempty"`
}

// GetModuleConfigAsHelmValues returns an unstructured JSON issuerValuesConfig representing the portion of the Verrazzano CR that corresponds to the module
func (c clusterIssuerComponent) GetModuleConfigAsHelmValues(effectiveCR *v1alpha1.Verrazzano) (*apiextensionsv1.JSON, error) {
	if effectiveCR == nil {
		return nil, nil
	}

	clusterIssuer := effectiveCR.Spec.Components.ClusterIssuer

	configSnippet := issuerValuesConfig{
		ClusterResourceNamespace: clusterIssuer.ClusterResourceNamespace,
		IssuerConfig:             clusterIssuer.IssuerConfig,
	}
	return spi.NewModuleConfigHelmValuesWrapper(configSnippet)
}

// GetWatchDescriptors returns the list of WatchDescriptors for objects being watched by the component
func (c clusterIssuerComponent) GetWatchDescriptors() []controllerspi.WatchDescriptor {
	return watch.CombineWatchDescriptors(
		watch.GetModuleInstalledWatches([]string{
			cmconstants.CertManagerComponentName,
			nginx.ComponentName,
		}),
		watch.GetModuleUpdatedWatches([]string{
			nginx.ComponentName,
		}),
	)
}
