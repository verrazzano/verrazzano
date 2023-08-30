// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package operator

import (
	"github.com/verrazzano/verrazzano/pkg/vzcr"
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1beta1"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	corev1 "k8s.io/api/core/v1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
)

// valuesConfig Structure for the translated effective Verrazzano CR values to Module CR Helm values
type valuesConfig struct {
	Ingress         *vzapi.IngressNginxComponent `json:"ingress,omitempty"`
	DNS             *vzapi.DNSComponent          `json:"dns,omitempty"`
	EnvironmentName string                       `json:"environmentName,omitempty"`

	KeycloakEnabled   bool `json:"keycloakEnabled"`
	PrometheusEnabled bool `json:"prometheusEnabled"`

	OpenSearch *v1beta1.OpenSearchComponent `json:"opensearch,omitempty"`

	DefaultVolumeSource      *corev1.VolumeSource            `json:"defaultVolumeSource,omitempty" patchStrategy:"replace"`
	VolumeClaimSpecTemplates []vzapi.VolumeClaimSpecTemplate `json:"volumeClaimSpecTemplates,omitempty" patchStrategy:"merge,retainKeys" patchMergeKey:"name"`
}

// GetModuleConfigAsHelmValues returns an unstructured JSON valuesConfig representing the portion of the Verrazzano CR that corresponds to the module
func (c jaegerOperatorComponent) GetModuleConfigAsHelmValues(effectiveCR *vzapi.Verrazzano) (*apiextensionsv1.JSON, error) {
	if effectiveCR == nil {
		return nil, nil
	}

	configSnippet := valuesConfig{
		DefaultVolumeSource:      effectiveCR.Spec.DefaultVolumeSource,
		VolumeClaimSpecTemplates: effectiveCR.Spec.VolumeClaimSpecTemplates,
		KeycloakEnabled:          vzcr.IsKeycloakEnabled(effectiveCR),
		PrometheusEnabled:        vzcr.IsPrometheusEnabled(effectiveCR),
	}

	opensearchV1Alpha1 := effectiveCR.Spec.Components.Elasticsearch
	if opensearchV1Alpha1 != nil {
		vzv1beta1 := &v1beta1.Verrazzano{}
		effectiveCR.ConvertTo(vzv1beta1)
		configSnippet.OpenSearch = vzv1beta1.Spec.Components.OpenSearch.DeepCopy()
	}

	dns := effectiveCR.Spec.Components.DNS
	if dns != nil {
		configSnippet.DNS = &vzapi.DNSComponent{
			External:         dns.External,
			InstallOverrides: vzapi.InstallOverrides{},
			OCI:              dns.OCI,
			Wildcard:         dns.Wildcard,
		}
	}

	nginx := effectiveCR.Spec.Components.Ingress
	if nginx != nil {
		configSnippet.Ingress = nginx.DeepCopy()
		configSnippet.Ingress.InstallOverrides.ValueOverrides = []vzapi.Overrides{}
	}

	if len(effectiveCR.Spec.EnvironmentName) > 0 {
		configSnippet.EnvironmentName = effectiveCR.Spec.EnvironmentName
	}

	return spi.NewModuleConfigHelmValuesWrapper(configSnippet)
}
