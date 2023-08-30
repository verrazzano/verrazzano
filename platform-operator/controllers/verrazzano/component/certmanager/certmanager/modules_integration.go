// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package certmanager

import (
	"github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
)

// certManagerModuleConfig Internal component configuration used to communicate Verrazzano CR config for CertManager to
// this component through the Module interface as Helm values
type certManagerModuleConfig struct {
	ClusterResourceNamespace string `json:"clusterResourceNamespace,omitempty"`
}

// GetModuleConfigAsHelmValues returns an unstructured JSON snippet representing the portion of the Verrazzano CR that corresponds to the module
func (c certManagerComponent) GetModuleConfigAsHelmValues(effectiveCR *v1alpha1.Verrazzano) (*apiextensionsv1.JSON, error) {
	// Convert the CertManager Verrazzano CR config to internal well-known Helm values
	compConfig := effectiveCR.Spec.Components.CertManager
	if compConfig == nil {
		return nil, nil
	}
	// TODO: Review this, the CM component only uses the ClusterIssuerComponent.ClusterResourceNamespace for configuration
	//  beyond the basic enable/disable and overrides capability.  Because we handle the InstallOverrides separately
	//  this may be all we need to trigger reconciles of the CM install
	clusterResourceNamespace := compConfig.Certificate.CA.ClusterResourceNamespace
	issuerConfig := effectiveCR.Spec.Components.ClusterIssuer
	if issuerConfig != nil {
		clusterResourceNamespace = issuerConfig.ClusterResourceNamespace
	}
	return spi.NewModuleConfigHelmValuesWrapper(
		certManagerModuleConfig{
			ClusterResourceNamespace: clusterResourceNamespace,
		},
	)
}
