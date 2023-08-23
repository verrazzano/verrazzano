// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package externaldns

import (
	"github.com/verrazzano/verrazzano/pkg/vzcr"
	"github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
)

// valuesConfig Structure for the translated effective Verrazzano CR values to Module CR Helm values
type valuesConfig struct {
	DNS          *v1alpha1.DNSComponent `json:"dns,omitempty"`
	IstioEnabled bool                   `json:"istioEnabled"`
}

// GetModuleConfigAsHelmValues returns an unstructured JSON valuesConfig representing the portion of the Verrazzano CR that corresponds to the module
func (c externalDNSComponent) GetModuleConfigAsHelmValues(effectiveCR *v1alpha1.Verrazzano) (*apiextensionsv1.JSON, error) {
	if effectiveCR == nil {
		return nil, nil
	}

	configSnippet := valuesConfig{}

	dns := effectiveCR.Spec.Components.DNS
	if dns != nil {
		configSnippet.DNS = &v1alpha1.DNSComponent{
			External:         dns.External,
			InstallOverrides: v1alpha1.InstallOverrides{}, // always ignore the overrides here, those are handled separately
			OCI:              dns.OCI,
			Wildcard:         dns.Wildcard,
		}
	}

	configSnippet.IstioEnabled = vzcr.IsIstioEnabled(effectiveCR)

	return spi.NewModuleConfigHelmValuesWrapper(configSnippet)
}