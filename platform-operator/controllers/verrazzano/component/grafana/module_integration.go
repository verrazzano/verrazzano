// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package grafana

import (
	"github.com/verrazzano/verrazzano-modules/pkg/controller/spi/controllerspi"
	vmov1 "github.com/verrazzano/verrazzano-monitoring-operator/pkg/apis/vmcontroller/v1"
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/common/watch"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/fluentoperator"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/grafanadashboards"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/nginx"
	prometheusOperator "github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/prometheus/operator"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/thanos"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/vmo"
	"github.com/verrazzano/verrazzano/platform-operator/internal/config"
	corev1 "k8s.io/api/core/v1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"reflect"
)

// valuesConfig Structure for the translated effective Verrazzano CR values to Module CR Helm values
type valuesConfig struct {
	Database *vzapi.DatabaseInfo `json:"database,omitempty"`
	Replicas *int32              `json:"replicas,omitempty"`
	SMTP     *vmov1.SMTPInfo     `json:"smtp,omitempty"`

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

	emptyConfig := valuesConfig{}
	if reflect.DeepEqual(configSnippet, emptyConfig) {
		return nil, nil
	}

	return spi.NewModuleConfigHelmValuesWrapper(configSnippet)
}

// ShouldUseModule returns true if component is implemented using a Module
func (g grafanaComponent) ShouldUseModule() bool {
	return config.Get().ModuleIntegration
}

// GetWatchDescriptors returns the list of WatchDescriptors for objects being watched by the component
func (g grafanaComponent) GetWatchDescriptors() []controllerspi.WatchDescriptor {
	return watch.CombineWatchDescriptors(
		watch.GetModuleInstalledWatches([]string{vmo.ComponentName, fluentoperator.ComponentName, nginx.ComponentJSONName, thanos.ComponentName, prometheusOperator.ComponentName}),
		watch.GetModuleUpdatedWatches([]string{vmo.ComponentName, fluentoperator.ComponentName, nginx.ComponentName, grafanadashboards.ComponentName}),
	)
}
