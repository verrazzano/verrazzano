// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package config

import (
	"errors"
	"fmt"
	"github.com/verrazzano/verrazzano/pkg/constants"
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1beta1"
	cmcommon "github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/certmanager/common"
	"github.com/verrazzano/verrazzano/platform-operator/internal/vzconfig"
	"k8s.io/apimachinery/pkg/runtime"
	"net/mail"
)

const (
	longestSystemURLPrefix = "elasticsearch.vmi.system"
	preOccupiedspace       = len(longestSystemURLPrefix) + 2
)

// validateLongestHostName - validates that the longest possible host name for a system endpoint
// is not greater than 64 characters
func validateLongestHostName(effectiveCR runtime.Object) error {
	envName := getEnvironmentName(effectiveCR)
	dnsSuffix, wildcard := getDNSSuffix(effectiveCR)
	spaceOccupied := preOccupiedspace
	longestHostName := fmt.Sprintf("%s.%s.%s", longestSystemURLPrefix, envName, dnsSuffix)
	if len(longestHostName) > 64 {
		if wildcard {
			spaceOccupied = spaceOccupied + len(dnsSuffix)
			return fmt.Errorf("Failed: spec.environmentName %s is too long. For the given configuration it must have at most %v characters", envName, 64-spaceOccupied)
		}

		return fmt.Errorf("Failed: spec.environmentName %s and DNS suffix %s are too long. For the given configuration they must have at most %v characters in combination", envName, dnsSuffix, 64-spaceOccupied)
	}
	return nil
}

func getEnvironmentName(effectiveCR runtime.Object) string {
	if cr, ok := effectiveCR.(*vzapi.Verrazzano); ok {
		return cr.Spec.EnvironmentName
	}
	cr := effectiveCR.(*v1beta1.Verrazzano)
	return cr.Spec.EnvironmentName
}

func getDNSSuffix(effectiveCR runtime.Object) (string, bool) {
	dnsSuffix, wildcard := "0.0.0.0", true
	if cr, ok := effectiveCR.(*vzapi.Verrazzano); ok {
		if cr.Spec.Components.DNS == nil || cr.Spec.Components.DNS.Wildcard != nil {
			return fmt.Sprintf("%s.%s", dnsSuffix, vzconfig.GetWildcardDomain(cr.Spec.Components.DNS)), wildcard
		} else if cr.Spec.Components.DNS.OCI != nil {
			wildcard = false
			dnsSuffix = cr.Spec.Components.DNS.OCI.DNSZoneName
		} else if cr.Spec.Components.DNS.External != nil {
			wildcard = false
			dnsSuffix = cr.Spec.Components.DNS.External.Suffix
		}
		return dnsSuffix, wildcard
	}

	cr := effectiveCR.(*v1beta1.Verrazzano)
	if cr.Spec.Components.DNS == nil || cr.Spec.Components.DNS.Wildcard != nil {
		return fmt.Sprintf("%s.%s", dnsSuffix, vzconfig.GetWildcardDomain(cr.Spec.Components.DNS)), wildcard
	} else if cr.Spec.Components.DNS.OCI != nil {
		wildcard = false
		dnsSuffix = cr.Spec.Components.DNS.OCI.DNSZoneName
	} else if cr.Spec.Components.DNS.External != nil {
		wildcard = false
		dnsSuffix = cr.Spec.Components.DNS.External.Suffix
	}
	return dnsSuffix, wildcard
}

// validateConfiguration Validates the ClusterIssuer Certificate configuration
// - Verifies that only one of either the CA or LetsEncrypt fields is set
// - Validates the CA or LetsEncrypt configurations if necessary
// - returns an error if anything is misconfigured
func validateConfiguration(vz *v1beta1.Verrazzano) (err error) {
	if err := validateComponentConfigurationV1Beta1(vz); err != nil {
		return err
	}
	return validateIssuerConfig(err, vz.Spec.Components.ClusterIssuer)
}

func validateIssuerConfig(err error, issuerComponent *v1beta1.ClusterIssuerComponent) error {
	if issuerComponent == nil {
		return nil
	}

	// Check if Ca or Acme is empty
	isCAConfig, err := issuerComponent.IsCAIssuer()
	if err != nil {
		return err
	}

	if isCAConfig { // only validate the CA config if that's what's configured
		if err := validateCAConfiguration(issuerComponent.CA, issuerComponent.ClusterResourceNamespace); err != nil {
			return err
		}
		return nil
	}
	// Validate the LetsEncrypt config otherwise
	return validateAcmeConfiguration(*issuerComponent.LetsEncrypt)
}

func validateCAConfiguration(ca *v1beta1.CAIssuer, clusterResourceNamespace string) error {
	if ca.SecretName == constants.DefaultVerrazzanoCASecretName {
		// if it's the default self-signed config the secret won't exist until created by CertManager
		return nil
	}
	// Otherwise validate the config exists
	_, err := cmcommon.GetSecret(clusterResourceNamespace, ca.SecretName)
	return err
}

// validateAcmeConfiguration Validate the LetsEncrypt/LetsEncrypt values
func validateAcmeConfiguration(acme v1beta1.LetsEncryptACMEIssuer) error {
	if len(acme.Environment) > 0 && !cmcommon.IsLetsEncryptProductionEnv(acme) && !cmcommon.IsLetsEncryptStagingEnv(acme) {
		return fmt.Errorf("Invalid Let's Encrypt environment: %s", acme.Environment)
	}
	if _, err := mail.ParseAddress(acme.EmailAddress); err != nil {
		return err
	}
	return nil
}

// validateComponentConfigurationV1Beta1 validates that only one of either the ClusterIssuerComponent or the
// CertManager.Certificate field can be configured with non-defaults at the same time.
func validateComponentConfigurationV1Beta1(vz *v1beta1.Verrazzano) error {

	certManagerComp := vz.Spec.Components.CertManager
	clusterIssuerComp := vz.Spec.Components.ClusterIssuer

	if certManagerComp == nil && clusterIssuerComp == nil {
		return nil
	}

	// NOTE: Disable this for now, since we can't distinguish between a user edit and the merged profile settings;
	// we may have to do this validation in the ComponentValidatorImpl before the EffectiveCR is generated
	//
	// We only allow configuring either the deprecated CertManager Certificate field, or the ClusterIssuerComponent
	//if certManagerComp != nil && clusterIssuerComp != nil {
	//	isDefaultIssuer, err := clusterIssuerComp.IsDefaultIssuer()
	//	if err != nil {
	//		return err
	//	}
	//	if !isDefaultIssuer && !isDefaultCertificateConfiguration(certManagerComp.Certificate) {
	//		return fmt.Errorf("Can not simultaneously configure both the CertManager and ClusterIssuer components")
	//	}
	//}

	return validateCertificate(certManagerComp)
}

func validateCertificate(comp *v1beta1.CertManagerComponent) error {
	if comp == nil {
		return nil
	}
	// Check if Ca or Acme is empty
	caNotEmpty := comp.Certificate.CA != v1beta1.CA{}
	acmeNotEmpty := comp.Certificate.Acme != v1beta1.Acme{}
	if caNotEmpty && acmeNotEmpty {
		return errors.New("Certificate object Acme and CA cannot be simultaneously populated")
	}
	if caNotEmpty {
		if err := validateCertificateCAConfiguration(comp.Certificate.CA); err != nil {
			return err
		}
		return nil
	} else if acmeNotEmpty {
		if err := validateCertificateAcmeConfiguration(comp.Certificate.Acme); err != nil {
			return err
		}
		return nil
	}
	return nil
}

func validateCertificateCAConfiguration(ca v1beta1.CA) error {
	if ca.SecretName == constants.DefaultVerrazzanoCASecretName && ca.ClusterResourceNamespace == ComponentNamespace {
		// if it's the default self-signed config the secret won't exist until created by CertManager
		return nil
	}
	// Otherwise validate the config exists
	_, err := cmcommon.GetSecret(ca.ClusterResourceNamespace, ca.SecretName)
	return err
}

// validateAcmeConfiguration Validate the ACME/LetsEncrypt values
func validateCertificateAcmeConfiguration(acme v1beta1.Acme) error {
	if !cmcommon.IsLetsEncryptProvider(acme) {
		return fmt.Errorf("Invalid ACME certificate provider %v", acme.Provider)
	}
	if len(acme.Environment) > 0 && !cmcommon.IsLetsEncryptProductionEnv(acme) && !cmcommon.IsLetsEncryptStagingEnv(acme) {
		return fmt.Errorf("Invalid Let's Encrypt environment: %s", acme.Environment)
	}
	if _, err := mail.ParseAddress(acme.EmailAddress); err != nil {
		return err
	}
	return nil
}

//func isDefaultCertificateConfiguration(cmCert v1beta1.Certificate) bool {
//	var emptyCertConfig = v1beta1.Certificate{}
//	defaultCertConfig := v1beta1.Certificate{
//		CA: v1beta1.CA{
//			SecretName:               constants.DefaultVerrazzanoCASecretName,
//			ClusterResourceNamespace: constants.CertManagerNamespace,
//		},
//	}
//	return cmCert == emptyCertConfig || cmCert == defaultCertConfig
//}
