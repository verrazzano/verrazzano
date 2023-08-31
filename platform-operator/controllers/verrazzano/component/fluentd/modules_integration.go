// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package fluentd

import (
	"github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
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
func (f fluentdComponent) GetModuleConfigAsHelmValues(effectiveCR *v1alpha1.Verrazzano) (*apiextensionsv1.JSON, error) {
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
	return spi.NewModuleConfigHelmValuesWrapper(configSnippet)
}