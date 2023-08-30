// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package operator

import (
	"github.com/verrazzano/verrazzano/pkg/vzcr"
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	corev1 "k8s.io/api/core/v1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
)

// valuesConfig Structure for the translated effective Verrazzano CR values to Module CR Helm values
type valuesConfig struct {
	EnvironmentName string                       `json:"environmentName,omitempty"`
	Ingress         *vzapi.IngressNginxComponent `json:"ingress,omitempty"`
	DNS             *vzapi.DNSComponent          `json:"dns,omitempty"`
	Thanos          *vzapi.ThanosComponent       `json:"thanos,omitempty"`

	PrometheusEnabled          bool `json:"prometheusEnabled"`
	ClusterIssuerEnabled       bool `json:"clusterIssuerEnabled"`
	ApplicationOperatorEnabled bool `json:"applicationOperatorEnabled"`
	ClusterOperatorEnabled     bool `json:"clusterOperatorEnabled"`
	IstioEnabled               bool `json:"istioEnabled"`
	VMOEnabled                 bool `json:"vmoEnabled"`

	DefaultVolumeSource      *corev1.VolumeSource            `json:"defaultVolumeSource,omitempty" patchStrategy:"replace"`
	VolumeClaimSpecTemplates []vzapi.VolumeClaimSpecTemplate `json:"volumeClaimSpecTemplates,omitempty" patchStrategy:"merge,retainKeys" patchMergeKey:"name"`
}

// GetModuleConfigAsHelmValues returns an unstructured JSON valuesConfig representing the portion of the Verrazzano CR that corresponds to the module
func (c prometheusComponent) GetModuleConfigAsHelmValues(effectiveCR *vzapi.Verrazzano) (*apiextensionsv1.JSON, error) {
	if effectiveCR == nil {
		return nil, nil
	}

	configSnippet := valuesConfig{
		DefaultVolumeSource:        effectiveCR.Spec.DefaultVolumeSource,
		VolumeClaimSpecTemplates:   effectiveCR.Spec.VolumeClaimSpecTemplates,
		PrometheusEnabled:          vzcr.IsPrometheusEnabled(effectiveCR),
		ClusterIssuerEnabled:       vzcr.IsClusterIssuerEnabled(effectiveCR),
		ApplicationOperatorEnabled: vzcr.IsApplicationOperatorEnabled(effectiveCR),
		ClusterOperatorEnabled:     vzcr.IsClusterOperatorEnabled(effectiveCR),
		IstioEnabled:               vzcr.IsIstioEnabled(effectiveCR),
		VMOEnabled:                 vzcr.IsVMOEnabled(effectiveCR),
	}

	thanos := effectiveCR.Spec.Components.Thanos
	if thanos != nil {
		// The prometheus operator digs into the Thanos overrides for the Thanos ruler settings, so we need to trigger on that
		configSnippet.Thanos = thanos.DeepCopy()
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

	if len(effectiveCR.Spec.EnvironmentName) > 0 {
		configSnippet.EnvironmentName = effectiveCR.Spec.EnvironmentName
	}

	return spi.NewModuleConfigHelmValuesWrapper(configSnippet)
}
