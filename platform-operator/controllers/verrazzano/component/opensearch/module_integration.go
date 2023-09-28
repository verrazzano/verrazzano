// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package opensearch

import (
	"github.com/verrazzano/verrazzano-modules/pkg/controller/spi/controllerspi"
	vmov1 "github.com/verrazzano/verrazzano-monitoring-operator/pkg/apis/vmcontroller/v1"
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/common/watch"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/fluentoperator"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/vmo"
	corev1 "k8s.io/api/core/v1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"reflect"
)

// valuesConfig Structure for the translated effective Verrazzano CR values to Module CR Helm values
type valuesConfig struct {
	Nodes                []vzapi.OpenSearchNode        `json:"nodes,omitempty" patchStrategy:"merge,retainKeys" patchMergeKey:"name"`
	Policies             []vmov1.IndexManagementPolicy `json:"policies,omitempty"`
	Plugins              vmov1.OpenSearchPlugins       `json:"plugins,omitempty"`
	DisableDefaultPolicy bool                          `json:"disableDefaultPolicy,omitempty"`
	ESInstallArgs        []vzapi.InstallArgs           `json:"installArgs,omitempty" patchStrategy:"merge,retainKeys" patchMergeKey:"name"`

	DefaultVolumeSource      *corev1.VolumeSource            `json:"defaultVolumeSource,omitempty" patchStrategy:"replace"`
	VolumeClaimSpecTemplates []vzapi.VolumeClaimSpecTemplate `json:"volumeClaimSpecTemplates,omitempty" patchStrategy:"merge,retainKeys" patchMergeKey:"name"`
}

var emptyConfig = valuesConfig{}

// GetModuleConfigAsHelmValues returns an unstructured JSON valuesConfig representing the portion of the Verrazzano CR that corresponds to the module
func (o opensearchComponent) GetModuleConfigAsHelmValues(effectiveCR *vzapi.Verrazzano) (*apiextensionsv1.JSON, error) {
	if effectiveCR == nil {
		return nil, nil
	}

	configSnippet := valuesConfig{
		DefaultVolumeSource:      effectiveCR.Spec.DefaultVolumeSource,
		VolumeClaimSpecTemplates: effectiveCR.Spec.VolumeClaimSpecTemplates,
	}

	opensearch := effectiveCR.Spec.Components.Elasticsearch
	if opensearch != nil {
		configSnippet.Nodes = opensearch.Nodes
		configSnippet.Policies = opensearch.Policies
		configSnippet.Plugins = opensearch.Plugins
		configSnippet.DisableDefaultPolicy = opensearch.DisableDefaultPolicy
		configSnippet.ESInstallArgs = opensearch.ESInstallArgs
	}

	if reflect.DeepEqual(emptyConfig, configSnippet) {
		return nil, nil
	}

	return spi.NewModuleConfigHelmValuesWrapper(configSnippet)
}

// ShouldUseModule returns true if component is implemented using a Module
func (o opensearchComponent) ShouldUseModule() bool {
	return true
}

// GetWatchDescriptors returns the list of WatchDescriptors for objects being watched by the component
func (o opensearchComponent) GetWatchDescriptors() []controllerspi.WatchDescriptor {
	return watch.CombineWatchDescriptors(
		watch.GetModuleInstalledWatches([]string{vmo.ComponentName, fluentoperator.ComponentName}),
		watch.GetModuleUpdatedWatches([]string{vmo.ComponentName, fluentoperator.ComponentName}),
	)
}
