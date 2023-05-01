// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package common

import (
	"fmt"
	"github.com/verrazzano/verrazzano/pkg/constants"
	"github.com/verrazzano/verrazzano/pkg/vzcr"
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	installv1beta1 "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1beta1"
)

// CertManagerConfiguration contains information about the Cert-Manager instance being used to manage
// the Verrazzano ClusterIssuer and related resources
type CertManagerConfiguration struct {
	Enabled                  bool
	External                 bool
	Namespace                string
	ClusterResourceNamespace string
	ServiceAccountName       string
	Certificate              vzapi.Certificate
}

// ValidateCertManagerConfiguration checks that either CertManagerComponent or ExternalCertManager component are
// enabled but not both
func ValidateCertManagerConfiguration(vz interface{}) error {

	cr, err := convertIfNecessary(vz)
	if err != nil {
		return err
	}

	if vzcr.IsCertManagerEnabled(cr) && vzcr.IsExternalCertManagerEnabled(cr) {
		return fmt.Errorf("can not simultaneously enable both %s and %s", CertManagerComponentJSONName, ExternalDNSComponentJSONName)
	}
	return nil
}

// GetCertManagerConfiguration Returns the Cert-Manager settings based on the Verrazzano CR,
// either the default Verrazzano-managed instance (via the CertManagerComponent) or a customer-managed instance
// (via the ExternalCertManagerComponent)
func GetCertManagerConfiguration(vz interface{}) (CertManagerConfiguration, error) {

	cr, err := convertIfNecessary(vz)
	if err != nil {
		return CertManagerConfiguration{}, err
	}

	externalCertManagerEnabled, err := isExternalCertManagerEnabled(cr)
	if err != nil {
		return CertManagerConfiguration{}, fmt.Errorf("Error checking if External CertManager is enabled: %s", err.Error())
	}

	cmNamespace, err := deriveCertManagerNamespace(cr, externalCertManagerEnabled)
	if err != nil {
		return CertManagerConfiguration{}, err
	}

	clusterResourceNamespace, err := deriveCertManagerClusterResourceNamespace(cr, cmNamespace, externalCertManagerEnabled)
	if err != nil {
		return CertManagerConfiguration{}, err
	}

	certConfig, err := deriveCertificateConfiguration(cr, clusterResourceNamespace)
	if err != nil {
		return CertManagerConfiguration{}, err
	}

	saName, err := deriveCertManagerServiceAccountName(cr)
	if err != nil {
		return CertManagerConfiguration{}, err
	}

	return CertManagerConfiguration{
		Enabled:                  vzcr.IsAnyCertManagerEnabled(cr),
		External:                 externalCertManagerEnabled,
		Namespace:                cmNamespace,
		ClusterResourceNamespace: clusterResourceNamespace,
		ServiceAccountName:       saName,
		Certificate:              certConfig,
	}, nil
}

// IsCAConfigured - Check if cert-type for the Verrazzano CR is a self-signed CA configuration, if not it is assumed
// to be ACME
func IsCAConfigured(vz *vzapi.Verrazzano) (bool, error) {
	if vz == nil {
		return false, fmt.Errorf("No Verrazzano resource defined")
	}
	if vz.Spec.Components.CertManager != nil {
		return IsCAConfig(vz.Spec.Components.CertManager.Certificate)
	}
	// If no CM field is set, default is true
	return true, nil
}

// IsCAConfig - Check the Ccertificate cert-type is CA, if not it is assumed to be ACME
//func IsCAConfig(certConfig vzapi.Certificate) (bool, error) {
//	return checkExactlyOneIssuerConfiguration(certConfig)
//}

func IsOCIDNS(vz *vzapi.Verrazzano) bool {
	return vz.Spec.Components.DNS != nil && vz.Spec.Components.DNS.OCI != nil
}

func convertIfNecessary(vz interface{}) (*vzapi.Verrazzano, error) {
	if vz == nil {
		return nil, fmt.Errorf("Unable to convert, nil Verrazzano reference")
	}
	if vzv1beta1, ok := vz.(*installv1beta1.Verrazzano); ok {
		cr := &vzapi.Verrazzano{}
		if err := cr.ConvertFrom(vzv1beta1); err != nil {
			return nil, err
		}
		return cr, nil
	}
	cr, ok := vz.(*vzapi.Verrazzano)
	if !ok {
		return nil, fmt.Errorf("Unable to convert, not a Verrazzano v1alpha1 reference")
	}
	return cr, nil
}

func deriveCertificateConfiguration(cr *vzapi.Verrazzano, defaultClusterResourceNamespace string) (vzapi.Certificate, error) {
	externalCertManagerEnabled, err := isExternalCertManagerEnabled(cr)
	if err != nil {
		return vzapi.Certificate{}, err
	}
	if externalCertManagerEnabled {
		var emptyCertConfig vzapi.Certificate
		extCMCert := cr.Spec.Components.ExternalCertManager.Certificate
		if extCMCert == emptyCertConfig {
			extCMCert.CA.ClusterResourceNamespace = defaultClusterResourceNamespace
			extCMCert.CA.SecretName = DefaultCACertificateSecretName
		}
		return extCMCert, nil
	}
	if cr.Spec.Components.CertManager != nil {
		return cr.Spec.Components.CertManager.Certificate, nil
	}
	return vzapi.Certificate{}, nil
}

func deriveCertManagerNamespace(cr *vzapi.Verrazzano, externalCertManagerEnabled bool) (string, error) {
	cmNamespace := constants.CertManagerNamespace
	if externalCertManagerEnabled && len(cr.Spec.Components.ExternalCertManager.Namespace) > 0 {
		cmNamespace = cr.Spec.Components.ExternalCertManager.Namespace
	}
	return cmNamespace, nil
}

func deriveCertManagerClusterResourceNamespace(cr *vzapi.Verrazzano, defaultValue string, externalCertManagerEnabled bool) (string, error) {
	clusterResourceNamespace := defaultValue
	if externalCertManagerEnabled && len(cr.Spec.Components.ExternalCertManager.ClusterResourceNamespace) > 0 {
		clusterResourceNamespace = cr.Spec.Components.ExternalCertManager.ClusterResourceNamespace
	}
	return clusterResourceNamespace, nil
}

func deriveCertManagerServiceAccountName(cr *vzapi.Verrazzano) (string, error) {
	serviceAccountName := CertManagerComponentName
	externalCertManagerEnabled, err := isExternalCertManagerEnabled(cr)
	if err != nil {
		return "", fmt.Errorf("Error checking if External CertManager is enabled: %s", err.Error())
	}
	if externalCertManagerEnabled && len(cr.Spec.Components.ExternalCertManager.ServiceAccountName) > 0 {
		serviceAccountName = cr.Spec.Components.ExternalCertManager.ServiceAccountName
	}
	return serviceAccountName, nil
}

func isExternalCertManagerEnabled(cr *vzapi.Verrazzano) (bool, error) {
	if err := ValidateCertManagerConfiguration(cr); err != nil {
		return false, err
	}
	return vzcr.IsExternalCertManagerEnabled(cr), nil
}
