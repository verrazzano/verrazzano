// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package issuer

import (
	"github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
)

// issuerValuesConfig Structure for the translated effective Verrazzano CR values to Module CR Helm values
type issuerValuesConfig struct {
	DNS                      *v1alpha1.DNSComponent `json:"dns,omitempty"`
	IssuerConfig             v1alpha1.IssuerConfig  `json:"issuerConfig"`
	ClusterResourceNamespace string                 `json:"clusterResourceNamespace,omitempty"`
}

// GetModuleConfigAsHelmValues returns an unstructured JSON issuerValuesConfig representing the portion of the Verrazzano CR that corresponds to the module
func (c clusterIssuerComponent) GetModuleConfigAsHelmValues(effectiveCR *v1alpha1.Verrazzano) (*apiextensionsv1.JSON, error) {
	if effectiveCR == nil {
		return nil, nil
	}

	clusterIssuer := effectiveCR.Spec.Components.ClusterIssuer
	dns := effectiveCR.Spec.Components.DNS

	var dnsCopy *v1alpha1.DNSComponent
	if dns != nil {
		dnsCopy = &v1alpha1.DNSComponent{
			External:         dns.External,
			InstallOverrides: v1alpha1.InstallOverrides{}, // always ignore the overrides here, those are handled separately
			OCI:              dns.OCI,
			Wildcard:         dns.Wildcard,
		}

	}
	configSnippet := issuerValuesConfig{
		DNS:                      dnsCopy,
		ClusterResourceNamespace: clusterIssuer.ClusterResourceNamespace,
		IssuerConfig:             clusterIssuer.IssuerConfig,
	}
	return spi.NewModuleConfigHelmValuesWrapper(configSnippet)
}
