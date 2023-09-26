// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package fluentd

import (
	"github.com/verrazzano/verrazzano-modules/pkg/controller/spi/controllerspi"
	"github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	vzconst "github.com/verrazzano/verrazzano/platform-operator/constants"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/common/watch"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
)

// valuesConfig Structure for the translated effective Verrazzano CR values to Module CR Helm values
type valuesConfig struct {
	ElasticsearchSecret string                            `json:"elasticsearchSecret,omitempty"`
	ElasticsearchURL    string                            `json:"elasticsearchURL,omitempty"`
	ExtraVolumeMounts   []v1alpha1.VolumeMount            `json:"extraVolumeMounts,omitempty" patchStrategy:"merge,retainKeys" patchMergeKey:"source"`
	OCI                 *v1alpha1.OciLoggingConfiguration `json:"oci,omitempty"`
}

// GetModuleConfigAsHelmValues returns an unstructured JSON valuesConfig representing the portion of the Verrazzano CR that corresponds to the module
func (c fluentdComponent) GetModuleConfigAsHelmValues(effectiveCR *v1alpha1.Verrazzano) (*apiextensionsv1.JSON, error) {
	if effectiveCR == nil {
		return nil, nil
	}
	var configSnippet *valuesConfig
	fluentd := effectiveCR.Spec.Components.Fluentd
	if fluentd != nil {
		configSnippet = &valuesConfig{
			ElasticsearchSecret: fluentd.ElasticsearchSecret,
			ElasticsearchURL:    fluentd.ElasticsearchURL,
			OCI:                 fluentd.OCI.DeepCopy(),
			ExtraVolumeMounts:   fluentd.ExtraVolumeMounts,
		}
	}

	if configSnippet == nil {
		return nil, nil
	}

	return spi.NewModuleConfigHelmValuesWrapper(configSnippet)
}

// GetWatchDescriptors returns the list of WatchDescriptors for objects being watched by the component
func (c fluentdComponent) GetWatchDescriptors() []controllerspi.WatchDescriptor {
	return watch.CombineWatchDescriptors(
		watch.GetCreateSecretWatch(vzconst.MCRegistrationSecret, vzconst.VerrazzanoSystemNamespace),
		watch.GetUpdateSecretWatch(vzconst.MCRegistrationSecret, vzconst.VerrazzanoSystemNamespace),
		watch.GetDeleteSecretWatch(vzconst.MCRegistrationSecret, vzconst.VerrazzanoSystemNamespace),
	)
}
