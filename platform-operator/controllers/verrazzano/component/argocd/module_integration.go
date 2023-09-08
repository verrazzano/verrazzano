// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package argocd

import (
	"github.com/verrazzano/verrazzano-modules/pkg/controller/spi/controllerspi"
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	cmconstants "github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/certmanager/constants"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/common/watch"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/istio"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/keycloak"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/nginx"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
)

// valuesConfig Structure for the translated effective Verrazzano CR values to Module CR Helm values
type valuesConfig struct {
	Ingress         *vzapi.IngressNginxComponent `json:"ingress,omitempty"`
	DNS             *vzapi.DNSComponent          `json:"dns,omitempty"`
	EnvironmentName string                       `json:"environmentName,omitempty"`

	ClusterIssuer *vzapi.ClusterIssuerComponent `json:"clusterIssuer,omitempty"`
}

// GetModuleConfigAsHelmValues returns an unstructured JSON valuesConfig representing the portion of the Verrazzano CR that corresponds to the module
func (c argoCDComponent) GetModuleConfigAsHelmValues(effectiveCR *vzapi.Verrazzano) (*apiextensionsv1.JSON, error) {
	if effectiveCR == nil {
		return nil, nil
	}

	configSnippet := valuesConfig{
		EnvironmentName: effectiveCR.Spec.EnvironmentName,
	}

	issuer := effectiveCR.Spec.Components.ClusterIssuer
	if issuer != nil {
		configSnippet.ClusterIssuer = issuer.DeepCopy()
	}

	dns := effectiveCR.Spec.Components.DNS
	if dns != nil {
		configSnippet.DNS = dns.DeepCopy()
		configSnippet.DNS.InstallOverrides = vzapi.InstallOverrides{}
	}

	nginx := effectiveCR.Spec.Components.Ingress
	if nginx != nil {
		configSnippet.Ingress = nginx.DeepCopy()
		configSnippet.Ingress.InstallOverrides.ValueOverrides = []vzapi.Overrides{}
	}

	return spi.NewModuleConfigHelmValuesWrapper(configSnippet)
}

// GetWatchDescriptors returns the list of WatchDescriptors for objects being watched by the component
func (c argoCDComponent) GetWatchDescriptors() []controllerspi.WatchDescriptor {
	return watch.GetModuleInstalledWatches([]string{istio.ComponentName, nginx.ComponentName, cmconstants.CertManagerComponentName, keycloak.ComponentName})
}
