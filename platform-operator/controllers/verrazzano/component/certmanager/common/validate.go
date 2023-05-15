// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package common

import (
	"context"
	"errors"
	"fmt"
	"net/mail"
	"strings"

	"github.com/verrazzano/verrazzano/pkg/constants"
	"github.com/verrazzano/verrazzano/pkg/k8sutil"
	"github.com/verrazzano/verrazzano/pkg/log/vzlog"
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1beta1"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	"github.com/verrazzano/verrazzano/platform-operator/internal/vzconfig"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	corev1client "k8s.io/client-go/kubernetes/typed/core/v1"
)

const (
	longestSystemURLPrefix = "elasticsearch.vmi.system"
	preOccupiedspace       = len(longestSystemURLPrefix) + 2

	// Valid Let's Encrypt environment values
	letsencryptProduction = "production"
	letsEncryptStaging    = "staging"
)

type GetCoreV1ClientFuncType func(log ...vzlog.VerrazzanoLogger) (corev1client.CoreV1Interface, error)

var GetClientFunc GetCoreV1ClientFuncType = k8sutil.GetCoreV1Client

func ResetCoreV1ClientFunc() {
	GetClientFunc = k8sutil.GetCoreV1Client
}

// IsCA - Check if cert-type is CA, if not it is assumed to be Acme
func IsCA(compContext spi.ComponentContext) (bool, error) {
	componentSpec := compContext.EffectiveCR().Spec.Components
	if componentSpec.CertManager != nil {
		return checkExactlyOneIssuerConfiguration(componentSpec.CertManager.Certificate)
	}
	// If the stanza isn't present assume the Self-signed CA issuer config
	return true, nil
}

// IsACMEConfig - Check if cert-type is ACME
func IsACMEConfig(vz interface{}) (bool, error) {
	vzv1alpha1, err := convertIfNecessary(vz)
	if err != nil {
		return false, err
	}
	if vzv1alpha1.Spec.Components.CertManager != nil {
		isCA, err := checkExactlyOneIssuerConfiguration(vzv1alpha1.Spec.Components.CertManager.Certificate)
		return !isCA, err
	}
	return false, nil
}

// ValidateLongestHostName - validates that the longest possible host name for a system endpoint
// is not greater than 64 characters
func ValidateLongestHostName(effectiveCR runtime.Object) error {
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

// ValidateConfiguration Validates the ClusterIssuer Certificate configuration
// - Verifies that only one of either the CA or ACME fields is set
// - Validates the CA or ACME configurations if necessary
// - returns an error if anything is misconfigured
func ValidateConfiguration(vz interface{}) (err error) {
	vzv1alpha1, err := convertIfNecessary(vz)
	if err != nil {
		return err
	}

	certManager := vzv1alpha1.Spec.Components.CertManager
	if certManager == nil {
		return nil
	}

	// Check if Ca or Acme is empty
	certConfig := certManager.Certificate
	isCAConfig, err := checkExactlyOneIssuerConfiguration(certConfig)
	if err != nil {
		return err
	}

	if isCAConfig { // only validate the CA config if that's what's configured
		if err := validateCAConfiguration(certConfig.CA, constants.CertManagerNamespace); err != nil {
			return err
		}
		return nil
	}
	// Validate the ACME config otherwise
	return validateAcmeConfiguration(certConfig.Acme)
}

func checkExactlyOneIssuerConfiguration(certConfig vzapi.Certificate) (isCAConfig bool, err error) {
	// Check if Ca or Acme is empty
	caNotEmpty := certConfig.CA != vzapi.CA{}
	acmeNotEmpty := certConfig.Acme != vzapi.Acme{}
	if caNotEmpty && acmeNotEmpty {
		return false, errors.New("certificate object Acme and CA cannot be simultaneously populated")
	} else if !caNotEmpty && !acmeNotEmpty {
		return false, errors.New("Either Acme or CA certificate authorities must be configured")
	}
	return caNotEmpty, nil
}

func validateCAConfiguration(ca vzapi.CA, componentNamespace string) error {
	if ca.SecretName == constants.DefaultVerrazzanoCASecretName && ca.ClusterResourceNamespace == componentNamespace {
		// if it's the default self-signed config the secret won't exist until created by CertManager
		return nil
	}
	// Otherwise validate the config exists
	_, err := GetCASecret(ca)
	return err
}

func GetCASecret(ca vzapi.CA) (*corev1.Secret, error) {
	name := ca.SecretName
	namespace := ca.ClusterResourceNamespace
	return GetSecret(namespace, name)
}

func GetSecret(namespace string, name string) (*corev1.Secret, error) {
	v1Client, err := GetClientFunc()
	if err != nil {
		return nil, err
	}
	return v1Client.Secrets(namespace).Get(context.TODO(), name, metav1.GetOptions{})
}

// validateAcmeConfiguration Validate the ACME/LetsEncrypt values
func validateAcmeConfiguration(acme vzapi.Acme) error {
	if !isLetsEncryptProvider(acme) {
		return fmt.Errorf("Invalid ACME certificate provider %v", acme.Provider)
	}
	if len(acme.Environment) > 0 && !isLetsEncryptProductionEnv(acme) && !isLetsEncryptStagingEnv(acme) {
		return fmt.Errorf("Invalid Let's Encrypt environment: %s", acme.Environment)
	}
	if _, err := mail.ParseAddress(acme.EmailAddress); err != nil {
		return err
	}
	return nil
}

func isLetsEncryptProvider(acme vzapi.Acme) bool {
	return strings.ToLower(string(acme.Provider)) == strings.ToLower(string(vzapi.LetsEncrypt))
}

func isLetsEncryptStagingEnv(acme vzapi.Acme) bool {
	return strings.ToLower(acme.Environment) == letsEncryptStaging
}

func isLetsEncryptProductionEnv(acme vzapi.Acme) bool {
	return strings.ToLower(acme.Environment) == letsencryptProduction
}

func IsOCIDNS(vz *vzapi.Verrazzano) bool {
	return vz.Spec.Components.DNS != nil && vz.Spec.Components.DNS.OCI != nil
}
