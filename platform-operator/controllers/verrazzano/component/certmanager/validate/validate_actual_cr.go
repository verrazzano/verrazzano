// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package validate

import (
	"errors"
	"fmt"
	"github.com/verrazzano/verrazzano/pkg/constants"
	"github.com/verrazzano/verrazzano/pkg/vzcr"
	"github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1beta1"
)

var (
	emptyCertConfigV1Alpha1 = v1alpha1.Certificate{}

	defaultCertConfigV1Alpha1 = v1alpha1.Certificate{
		CA: v1alpha1.CA{
			SecretName:               constants.DefaultVerrazzanoCASecretName,
			ClusterResourceNamespace: constants.CertManagerNamespace,
		},
	}

	emptyCertConfigV1Beta1 = v1beta1.Certificate{}

	defaultCertConfigV1Beta1 = v1beta1.Certificate{
		CA: v1beta1.CA{
			SecretName:               constants.DefaultVerrazzanoCASecretName,
			ClusterResourceNamespace: constants.CertManagerNamespace,
		},
	}

	bothConfiguredErr = "can not configure both the certManager.certificate field and the clusterIssuer component, " +
		"use of the certManager.certificate field is deprecated "
)

// ValidateActualConfigurationV1Beta1 Validates the unmodified user configuration for the ClusterIssuer and CertManager components
func ValidateActualConfigurationV1Beta1(vz *v1beta1.Verrazzano) []error {
	var errs []error
	if !vzcr.IsCertManagerEnabled(vz) || !vzcr.IsClusterIssuerEnabled(vz) {
		//if one or the other is disabled, there's no conflict
		return errs
	}
	clusterIssuerComponent := vz.Spec.Components.ClusterIssuer
	certManagerComponent := vz.Spec.Components.CertManager
	if certManagerComponent != nil && clusterIssuerComponent != nil {
		if (clusterIssuerComponent.CA != nil || clusterIssuerComponent.LetsEncrypt != nil) &&
			!IsEmptyCertificateConfig(certManagerComponent.Certificate) {
			errs = append(errs, fmt.Errorf(bothConfiguredErr))
		}
	}
	return errs
}

// ValidateActualConfigurationV1Alpha1 - Validates the unmodified user configuration for the ClusterIssuer and CertManager components
func ValidateActualConfigurationV1Alpha1(vz *v1alpha1.Verrazzano) []error {
	var errs []error
	if !vzcr.IsCertManagerEnabled(vz) || !vzcr.IsClusterIssuerEnabled(vz) {
		//if one or the other is disabled, there's no conflict
		return errs
	}
	clusterIssuerComponent := vz.Spec.Components.ClusterIssuer
	certManagerComponent := vz.Spec.Components.CertManager
	if certManagerComponent != nil && clusterIssuerComponent != nil {
		if (clusterIssuerComponent.CA != nil || clusterIssuerComponent.LetsEncrypt != nil) &&
			!IsEmptyCertificateConfig(certManagerComponent.Certificate) {
			errs = append(errs, fmt.Errorf(bothConfiguredErr))
		}
	}
	return errs
}

func IsDefaultCertificateConfig(certificateConfig interface{}) bool {
	if v1alpha1Config, ok := certificateConfig.(v1alpha1.Certificate); ok {
		return v1alpha1Config == defaultCertConfigV1Alpha1
	}
	if v1beta1Config, ok := certificateConfig.(v1beta1.Certificate); ok {
		return v1beta1Config == defaultCertConfigV1Beta1
	}
	return false
}

func IsEmptyCertificateConfig(certificateConfig interface{}) bool {
	if v1alpha1Config, ok := certificateConfig.(v1alpha1.Certificate); ok {
		return v1alpha1Config == emptyCertConfigV1Alpha1
	}
	if v1beta1Config, ok := certificateConfig.(v1beta1.Certificate); ok {
		return v1beta1Config == emptyCertConfigV1Beta1
	}
	return false
}

func IsNotDefaultCANamespace(certManagerConfig interface{}) bool {
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

func IsCAConfig(certConfig interface{}) (isCAConfig bool, err error) {
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
