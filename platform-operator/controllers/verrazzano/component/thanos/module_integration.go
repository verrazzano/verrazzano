// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package thanos

import (
	"github.com/verrazzano/verrazzano-modules/pkg/controller/spi/controllerspi"
	"github.com/verrazzano/verrazzano/pkg/vzcr"
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/common/watch"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/fluentoperator"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/istio"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/nginx"
	promoperator "github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/prometheus/operator"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
)

type valuesConfig struct {
	IstioInjectionEnabled bool                         `json:"istioInjectionEnabled"`
	Ingress               *vzapi.IngressNginxComponent `json:"ingress,omitempty"`
	DNS                   *vzapi.DNSComponent          `json:"dns,omitempty"`
	EnvironmentName       string                       `json:"environmentName,omitempty"`
}

// GetModuleConfigAsHelmValues returns an unstructured JSON snippet representing the portion of the Verrazzano CR that corresponds to the module
func (c ThanosComponent) GetModuleConfigAsHelmValues(effectiveCR *vzapi.Verrazzano) (*apiextensionsv1.JSON, error) {
	if effectiveCR == nil {
		return nil, nil
	}

	configSnippet := valuesConfig{
		IstioInjectionEnabled: vzcr.IsIstioInjectionEnabled(effectiveCR),
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

// GetWatchDescriptors returns the list of WatchDescriptors for objects being watched by the component
func (c ThanosComponent) GetWatchDescriptors() []controllerspi.WatchDescriptor {
	return watch.GetModuleInstalledWatches([]string{istio.ComponentName, nginx.ComponentName, promoperator.ComponentName, fluentoperator.ComponentName})
}
