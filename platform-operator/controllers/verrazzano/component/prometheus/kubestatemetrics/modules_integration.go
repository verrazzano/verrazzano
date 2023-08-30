// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package kubestatemetrics

import (
	"github.com/verrazzano/verrazzano/pkg/vzcr"
	"github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
)

type valuesConfig struct {
	PrometheusOperatorEnabled bool `json:"prometheusOperatorEnabled"`
}

// GetModuleConfigAsHelmValues returns an unstructured JSON snippet representing the portion of the Verrazzano CR that corresponds to the module
func (c kubeStateMetricsComponent) GetModuleConfigAsHelmValues(effectiveCR *v1alpha1.Verrazzano) (*apiextensionsv1.JSON, error) {
	if effectiveCR == nil {
		return nil, nil
	}

	configSnippet := valuesConfig{
		PrometheusOperatorEnabled: vzcr.IsPrometheusOperatorEnabled(effectiveCR),
	}

	return spi.NewModuleConfigHelmValuesWrapper(configSnippet)
}
