// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package authproxy

import (
	"github.com/verrazzano/verrazzano-modules/pkg/controller/spi/controllerspi"
	"github.com/verrazzano/verrazzano/pkg/vzcr"
	"github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/common/watch"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/fluentoperator"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/nginx"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
)

// valuesConfig Structure for the translated effective Verrazzano CR values to Module CR Helm values
type valuesConfig struct {
	Ingress                   *v1alpha1.IngressNginxComponent      `json:"ingress,omitempty"`
	DNS                       *v1alpha1.DNSComponent               `json:"dns,omitempty"`
	EnvironmentName           string                               `json:"environmentName,omitempty"`
	Kubernetes                *v1alpha1.AuthProxyKubernetesSection `json:"kubernetes,omitempty"`
	PrometheusOperatorEnabled bool                                 `json:"prometheusOperatorEnabled"`
}

// GetModuleConfigAsHelmValues returns an unstructured JSON valuesConfig representing the portion of the Verrazzano CR that corresponds to the module
func (c authProxyComponent) GetModuleConfigAsHelmValues(effectiveCR *v1alpha1.Verrazzano) (*apiextensionsv1.JSON, error) {
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

	nginx := effectiveCR.Spec.Components.Ingress
	if nginx != nil {
		configSnippet.Ingress = nginx.DeepCopy()
		configSnippet.Ingress.InstallOverrides.ValueOverrides = []v1alpha1.Overrides{}
	}

	authProxy := effectiveCR.Spec.Components.AuthProxy
	if authProxy.Kubernetes != nil {
		configSnippet.Kubernetes = authProxy.Kubernetes.DeepCopy()
	}

	configSnippet.PrometheusOperatorEnabled = vzcr.IsPrometheusOperatorEnabled(effectiveCR)

	if len(effectiveCR.Spec.EnvironmentName) > 0 {
		configSnippet.EnvironmentName = effectiveCR.Spec.EnvironmentName
	}
	return spi.NewModuleConfigHelmValuesWrapper(configSnippet)
}

// GetWatchDescriptors returns the list of WatchDescriptors for objects being watched by the component
func (c authProxyComponent) GetWatchDescriptors() []controllerspi.WatchDescriptor {
	return watch.GetModuleReadyWatches([]string{nginx.ComponentName, fluentoperator.ComponentName})
}
