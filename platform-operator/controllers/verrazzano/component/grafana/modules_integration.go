// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package grafana

import (
	"github.com/verrazzano/verrazzano-modules/pkg/controller/spi/controllerspi"
	vmov1 "github.com/verrazzano/verrazzano-monitoring-operator/pkg/apis/vmcontroller/v1"
	"github.com/verrazzano/verrazzano/pkg/vzcr"
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	"github.com/verrazzano/verrazzano/platform-operator/internal/config"
	corev1 "k8s.io/api/core/v1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
)

// valuesConfig Structure for the translated effective Verrazzano CR values to Module CR Helm values
type valuesConfig struct {
	Database *vzapi.DatabaseInfo `json:"database,omitempty"`
	Replicas *int32              `json:"replicas,omitempty"`
	SMTP     *vmov1.SMTPInfo     `json:"smtp,omitempty"`

	Ingress         *vzapi.IngressNginxComponent `json:"ingress,omitempty"`
	DNS             *vzapi.DNSComponent          `json:"dns,omitempty"`
	EnvironmentName string                       `json:"environmentName,omitempty"`

	ThanosEnabled             bool `json:"thanosEnabled"`
	PrometheusEnabled         bool `json:"prometheusEnabled"`
	PrometheusOperatorEnabled bool `json:"prometheusOperatorEnabled"`

	DefaultVolumeSource      *corev1.VolumeSource            `json:"defaultVolumeSource,omitempty" patchStrategy:"replace"`
	VolumeClaimSpecTemplates []vzapi.VolumeClaimSpecTemplate `json:"volumeClaimSpecTemplates,omitempty" patchStrategy:"merge,retainKeys" patchMergeKey:"name"`
}

// GetModuleConfigAsHelmValues returns an unstructured JSON valuesConfig representing the portion of the Verrazzano CR that corresponds to the module
func (g grafanaComponent) GetModuleConfigAsHelmValues(effectiveCR *vzapi.Verrazzano) (*apiextensionsv1.JSON, error) {
	if effectiveCR == nil {
		return nil, nil
	}

	configSnippet := valuesConfig{
		DefaultVolumeSource:      effectiveCR.Spec.DefaultVolumeSource,
		VolumeClaimSpecTemplates: effectiveCR.Spec.VolumeClaimSpecTemplates,
	}

	grafana := effectiveCR.Spec.Components.Grafana
	if grafana != nil {
		configSnippet.Database = grafana.Database
		configSnippet.SMTP = grafana.SMTP
		configSnippet.Replicas = grafana.Replicas
	}

	dns := effectiveCR.Spec.Components.DNS
	if dns != nil {
		configSnippet.DNS = &vzapi.DNSComponent{
			External:         dns.External,
			InstallOverrides: vzapi.InstallOverrides{}, // always ignore the overrides here, those are handled separately
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

	configSnippet.PrometheusEnabled = vzcr.IsPrometheusEnabled(effectiveCR)
	configSnippet.ThanosEnabled = vzcr.IsThanosEnabled(effectiveCR)
	configSnippet.PrometheusOperatorEnabled = vzcr.IsPrometheusOperatorEnabled(effectiveCR)

	return spi.NewModuleConfigHelmValuesWrapper(configSnippet)
}

// ShouldUseModule returns true if component is implemented using a Module
func (g grafanaComponent) ShouldUseModule() bool {
	return config.Get().ModuleIntegration
}

// GetWatchDescriptors returns the list of WatchDescriptors for objects being watched by the component
func (g grafanaComponent) GetWatchDescriptors() []controllerspi.WatchDescriptor {
	return nil
}
