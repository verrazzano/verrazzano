// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package transform

import (
	"errors"
	"fmt"

	"github.com/verrazzano/verrazzano/pkg/constants"
	"github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1beta1"
)

var (
	defaultCertConfigV1Alpha1 = v1alpha1.Certificate{
		CA: v1alpha1.CA{
			SecretName:               constants.DefaultVerrazzanoCASecretName,
			ClusterResourceNamespace: constants.CertManagerNamespace,
		},
	}

	defaultCertConfigV1Beta1 = v1beta1.Certificate{
		CA: v1beta1.CA{
			SecretName:               constants.DefaultVerrazzanoCASecretName,
			ClusterResourceNamespace: constants.CertManagerNamespace,
		},
	}
)

// convertCertificateToClusterIssuerV1Beta1 aligns the ClusterIssuer configurations between CertManager and the newer ClusterIssuer
// configurations.  The webhook validators will ensure only one is set to a non-defaulted value, so if one is
// configured align it with the other.
func convertCertificateToClusterIssuerV1Beta1(effectiveCR *v1beta1.Verrazzano) error {
	// Internally components will derive Issuer configuration from the ClusterIssuerComponent, so
	// we need to determine if the legacy CertManager component is configured and convert that to the ClusterIssuer
	// configuration.
	certManagerConfig := effectiveCR.Spec.Components.CertManager
	clusterIssuerConfig := effectiveCR.Spec.Components.ClusterIssuer

	isDefaultIssuer, err := clusterIssuerConfig.IsDefaultIssuer()
	if err != nil {
		return err
	}

	cmCertificateConfig := certManagerConfig.Certificate

	// Check if it's a CA issuer config
	isCAConfig, err := isCAConfig(cmCertificateConfig)
	if err != nil {
		// The certificate object is invalid
		return err
	}

	isDefaultCertificateConfig := isDefaultCertificateConfig(cmCertificateConfig)
	if !isDefaultCertificateConfig && !isDefaultIssuer {
		return fmt.Errorf("Illegal state, both CertManager Certificate and ClusterIssuer components are configured")
	}

	if isDefaultIssuer && !isDefaultCertificateConfig {
		// Issuer is the default configuration, but the CM Certificate config is configured explicitly,
		// align the ClusterIssuer configuration internally on the CM configuration
		if err := convertCertificateConfiguration(cmCertificateConfig, clusterIssuerConfig, isCAConfig); err != nil {
			return err
		}
	}
	return nil
}

// convertCertificateToClusterIssuerV1Alpha1 converts a legacy CertManagerComponent.Certificate object to a
// ClusterIssuerComponent configuration.  The conversion is done only in the case where the ClusterIssuerComponent is
// not explicitly configured with a non-default issuer configuration.
//
// The webhook validators will ensure only one is set to a non-defaulted value.
func convertCertificateToClusterIssuerV1Alpha1(effectiveCR *v1alpha1.Verrazzano) error {
	// Internally components will derive Issuer configuration from the ClusterIssuerComponent, so
	// we need to determine if the legacy CertManager component is configured and convert that to the ClusterIssuer
	// configuration.
	certManagerConfig := effectiveCR.Spec.Components.CertManager
	clusterIssuerConfig := effectiveCR.Spec.Components.ClusterIssuer

	isDefaultIssuer, err := clusterIssuerConfig.IsDefaultIssuer()
	if err != nil {
		return err
	}

	cmCertificateConfig := certManagerConfig.Certificate

	// Check if it's a CA issuer config
	isCAConfig, err := isCAConfig(cmCertificateConfig)
	if err != nil {
		// The certificate object is invalid
		return err
	}

	isDefaultCertificateConfig := isDefaultCertificateConfig(cmCertificateConfig)
	if !isDefaultCertificateConfig && !isDefaultIssuer {
		return fmt.Errorf("Illegal state, both CertManager Certificate and ClusterIssuer components are configured")
	}

	if isDefaultIssuer && !isDefaultCertificateConfig {
		// Issuer is the default configuration, but the CM Certificate config is configured explicitly,
		// align the ClusterIssuer configuration internally on the CM configuration
		if err := convertCertificateConfiguration(cmCertificateConfig, clusterIssuerConfig, isCAConfig); err != nil {
			return err
		}
	}
	return nil
}

func isDefaultCertificateConfig(certificateConfig interface{}) bool {
	if v1alpha1Config, ok := certificateConfig.(v1alpha1.Certificate); ok {
		return v1alpha1Config == defaultCertConfigV1Alpha1
	}
	if v1beta1Config, ok := certificateConfig.(v1beta1.Certificate); ok {
		return v1beta1Config == defaultCertConfigV1Beta1
	}
	return false
}

func isNotDefaultCANamespace(certManagerConfig interface{}) bool {
	if v1alpha1Config, ok := certManagerConfig.(v1alpha1.Certificate); ok {
		return len(v1alpha1Config.CA.ClusterResourceNamespace) > 0 &&
			v1alpha1Config.CA.ClusterResourceNamespace != constants.CertManagerNamespace
	}
	if v1beta11Config, ok := certManagerConfig.(v1beta1.Certificate); ok {
		return len(v1beta11Config.CA.ClusterResourceNamespace) > 0 &&
			v1beta11Config.CA.ClusterResourceNamespace != constants.CertManagerNamespace
	}
	return false
}

func isCAConfig(certConfig interface{}) (isCAConfig bool, err error) {
	var caNotEmpty bool
	if certv1alpha1, ok := certConfig.(v1alpha1.Certificate); ok {
		// Check if Ca or Acme is empty
		caNotEmpty = certv1alpha1.CA != v1alpha1.CA{}
		acmeNotEmpty := certv1alpha1.Acme != v1alpha1.Acme{}
		if caNotEmpty && acmeNotEmpty {
			return false, errors.New("certificate object Acme and CA cannot be simultaneously populated")
		} else if !caNotEmpty && !acmeNotEmpty {
			return false, errors.New("Either Acme or CA certificate authorities must be configured")
		}
	}
	if certv1beta1, ok := certConfig.(v1beta1.Certificate); ok {
		// Check if Ca or Acme is empty
		caNotEmpty = certv1beta1.CA != v1beta1.CA{}
		acmeNotEmpty := certv1beta1.Acme != v1beta1.Acme{}
		if caNotEmpty && acmeNotEmpty {
			return false, errors.New("certificate object Acme and CA cannot be simultaneously populated")
		} else if !caNotEmpty && !acmeNotEmpty {
			return false, errors.New("Either Acme or CA certificate authorities must be configured")
		}
	}
	return caNotEmpty, nil
}

func convertCertificateConfiguration(cmCertificateConfig interface{}, clusterIssuerConfig interface{}, isCAConfig bool) error {
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
		if isNotDefaultCANamespace(certV1Alpha1) {
			issuerV1Alpha1.ClusterResourceNamespace = certV1Alpha1.CA.ClusterResourceNamespace
		}
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
		if isNotDefaultCANamespace(certV1Beta1) {
			issuerV1Beta1.ClusterResourceNamespace = certV1Beta1.CA.ClusterResourceNamespace
		}
	}

	return nil
}
