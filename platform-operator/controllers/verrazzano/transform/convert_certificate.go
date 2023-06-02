// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package transform

import (
	"fmt"
	"github.com/verrazzano/verrazzano/pkg/constants"
	"github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1beta1"
	cmvalidate "github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/certmanager/validate"
)

// convertCertificateToClusterIssuerV1Beta1 Ensures backwards compatibility between the newer ClusterIssuerComponent
// and the legacy CertManagerComponent.Certificate object.
//
// The general rule is if the ClusterIssuerComponent is not set or is defaulted, then use the CertManagerComponent.Certificate
// configuration if appropriate.
func convertCertificateToClusterIssuerV1Beta1(effectiveCR *v1beta1.Verrazzano) error {
	// Edge case, initialize CertManager with defaults if necessary
	if effectiveCR.Spec.Components.CertManager == nil {
		effectiveCR.Spec.Components.CertManager = &v1beta1.CertManagerComponent{}
	}
	certManagerConfig := effectiveCR.Spec.Components.CertManager
	if cmvalidate.IsEmptyCertificateConfig(certManagerConfig.Certificate) {
		certManagerConfig.Certificate = v1beta1.Certificate{
			CA: v1beta1.CA{
				SecretName:               constants.DefaultVerrazzanoCASecretName,
				ClusterResourceNamespace: constants.CertManagerNamespace,
			},
		}
	}

	// Edge case, initialize ClusterIssuer with defaults if necessary
	if effectiveCR.Spec.Components.ClusterIssuer == nil {
		effectiveCR.Spec.Components.ClusterIssuer = v1beta1.NewDefaultClusterIssuer()
	}
	clusterIssuerConfig := effectiveCR.Spec.Components.ClusterIssuer
	if clusterIssuerConfig.IssuerConfig.CA == nil && clusterIssuerConfig.IssuerConfig.LetsEncrypt == nil {
		clusterIssuerConfig.IssuerConfig.CA = v1beta1.NewDefaultClusterIssuer().CA
	}

	isDefaultIssuer, err := clusterIssuerConfig.IsDefaultIssuer()
	if err != nil {
		return err
	}

	// Issuer is the default configuration, but the CM Certificate config is configured explicitly,
	// align the ClusterIssuer configuration internally on the CM configuration
	if err := convertCertificateConfiguration(certManagerConfig.Certificate, clusterIssuerConfig, isDefaultIssuer); err != nil {
		return err
	}

	return nil
}

// convertCertificateToClusterIssuerV1Beta1 Ensures backwards compatibility between the newer ClusterIssuerComponent
// and the legacy CertManagerComponent.Certificate object.
//
// The general rule is if the ClusterIssuerComponent is not set or is defaulted, then use the CertManagerComponent.Certificate
// configuration if appropriate.
func convertCertificateToClusterIssuerV1Alpha1(effectiveCR *v1alpha1.Verrazzano) error {
	// Edge case, initialize CertManager with defaults if necessary
	if effectiveCR.Spec.Components.CertManager == nil {
		effectiveCR.Spec.Components.CertManager = &v1alpha1.CertManagerComponent{}
	}
	certManagerConfig := effectiveCR.Spec.Components.CertManager
	if cmvalidate.IsEmptyCertificateConfig(certManagerConfig.Certificate) {
		certManagerConfig.Certificate = v1alpha1.Certificate{
			CA: v1alpha1.CA{
				SecretName:               constants.DefaultVerrazzanoCASecretName,
				ClusterResourceNamespace: constants.CertManagerNamespace,
			},
		}
	}

	// Edge case, initialize ClusterIssuer with defaults if necessary
	if effectiveCR.Spec.Components.ClusterIssuer == nil {
		effectiveCR.Spec.Components.ClusterIssuer = v1alpha1.NewDefaultClusterIssuer()
	}
	clusterIssuerConfig := effectiveCR.Spec.Components.ClusterIssuer
	if clusterIssuerConfig.IssuerConfig.CA == nil && clusterIssuerConfig.IssuerConfig.LetsEncrypt == nil {
		clusterIssuerConfig.IssuerConfig.CA = v1alpha1.NewDefaultClusterIssuer().CA
	}

	isDefaultIssuer, err := clusterIssuerConfig.IsDefaultIssuer()
	if err != nil {
		return err
	}

	// Issuer is the default configuration, but the CM Certificate config is configured explicitly,
	// align the ClusterIssuer configuration internally on the CM configuration
	if err := convertCertificateConfiguration(certManagerConfig.Certificate, clusterIssuerConfig, isDefaultIssuer); err != nil {
		return err
	}

	return nil
}

func convertCertificateConfiguration(cmCertificateConfig interface{}, clusterIssuerConfig interface{}, isDefaultIssuer bool) error {
	isDefaultCertificateConfig := cmvalidate.IsDefaultCertificateConfig(cmCertificateConfig)
	if !isDefaultCertificateConfig && !isDefaultIssuer {
		return fmt.Errorf("Illegal state, both CertManager Certificate and ClusterIssuer components are configured")
	}

	if !isDefaultIssuer {
		return nil
	}

	// Check if it's a CA issuer config
	isCAConfig, err := cmvalidate.IsCAConfig(cmCertificateConfig)
	if err != nil {
		// The certificate object is invalid, do nothing; validators should catch it and returning an error
		// here will throw off the validators.
		return nil
	}

	if cmvalidate.IsEmptyCertificateConfig(cmCertificateConfig) {
		return nil
	}

	return doConversion(cmCertificateConfig, clusterIssuerConfig, isCAConfig)
}

func doConversion(cmCertificateConfig interface{}, clusterIssuerConfig interface{}, isCAConfig bool) error {
	if issuerV1Alpha1, ok := clusterIssuerConfig.(*v1alpha1.ClusterIssuerComponent); ok {
		certV1Alpha1, _ := cmCertificateConfig.(v1alpha1.Certificate)
		if isCAConfig {
			// align on the CA config
			issuerV1Alpha1.CA = &v1alpha1.CAIssuer{SecretName: certV1Alpha1.CA.SecretName}
			issuerV1Alpha1.LetsEncrypt = nil
		} else {
			// align on the LetsEncrypt config
			issuerV1Alpha1.CA = nil
			issuerV1Alpha1.LetsEncrypt = &v1alpha1.LetsEncryptACMEIssuer{
				EmailAddress: certV1Alpha1.Acme.EmailAddress,
				Environment:  certV1Alpha1.Acme.Environment,
			}
		}
		// Update the clusterResourceNamespace if it is not the default
		if cmvalidate.IsNotDefaultCANamespace(certV1Alpha1) {
			issuerV1Alpha1.ClusterResourceNamespace = certV1Alpha1.CA.ClusterResourceNamespace
		}
		return nil
	}
	if issuerV1Beta1, ok := clusterIssuerConfig.(*v1beta1.ClusterIssuerComponent); ok {
		certV1Beta1, _ := cmCertificateConfig.(v1beta1.Certificate)
		if isCAConfig {
			// align on the CA config
			issuerV1Beta1.CA = &v1beta1.CAIssuer{SecretName: certV1Beta1.CA.SecretName}
			issuerV1Beta1.LetsEncrypt = nil
		} else {
			// align on the LetsEncrypt config
			issuerV1Beta1.CA = nil
			issuerV1Beta1.LetsEncrypt = &v1beta1.LetsEncryptACMEIssuer{
				EmailAddress: certV1Beta1.Acme.EmailAddress,
				Environment:  certV1Beta1.Acme.Environment,
			}
		}
		// Update the clusterResourceNamespace if it is not the default
		if cmvalidate.IsNotDefaultCANamespace(certV1Beta1) {
			issuerV1Beta1.ClusterResourceNamespace = certV1Beta1.CA.ClusterResourceNamespace
		}
		return nil
	}
	return fmt.Errorf("Unknown issuer type passed in: %T", clusterIssuerConfig)
}
