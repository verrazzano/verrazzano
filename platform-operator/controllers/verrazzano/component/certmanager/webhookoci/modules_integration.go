// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package webhookoci

import (
	"github.com/verrazzano/verrazzano/pkg/constants"
	"github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
)

// webhookOCIValuesConfig Structure for the translated effective Verrazzano CR values to Module CR Helm values
type webhookOCIValuesConfig struct {
	OCIConfigSecret          string `json:"ociConfigSecret,omitempty"`
	ClusterResourceNamespace string `json:"clusterResourceNamespace,omitempty"`
}

// GetModuleConfigAsHelmValues returns an unstructured JSON webhookOCIValuesConfig representing the portion of the Verrazzano CR that corresponds to the module
func (c certManagerWebhookOCIComponent) GetModuleConfigAsHelmValues(effectiveCR *v1alpha1.Verrazzano) (*apiextensionsv1.JSON, error) {
	if effectiveCR == nil {
		return nil, nil
	}

	clusterIssuer := effectiveCR.Spec.Components.ClusterIssuer
	dns := effectiveCR.Spec.Components.DNS

	var ociConfigSecret string
	if dns != nil && dns.OCI != nil {
		ociConfigSecret = dns.OCI.OCIConfigSecret
	}

	clusterResourceNamespace := constants.CertManagerNamespace
	if clusterIssuer != nil && len(clusterIssuer.ClusterResourceNamespace) > 0 {
		clusterResourceNamespace = clusterIssuer.ClusterResourceNamespace
	}

	configSnippet := webhookOCIValuesConfig{
		OCIConfigSecret:          ociConfigSecret,
		ClusterResourceNamespace: clusterResourceNamespace,
	}

	return spi.NewModuleConfigHelmValuesWrapper(configSnippet)
}
