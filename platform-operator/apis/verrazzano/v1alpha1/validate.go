// Copyright (c) 2020, 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package v1alpha1

import (
	"context"
	"fmt"
	"github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1beta1"
	"github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/validators"
	corev1 "k8s.io/api/core/v1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// ValidateProfile check that requestedProfile is valid
func ValidateProfile(requestedProfile ProfileType) error {
	if len(requestedProfile) != 0 {
		switch requestedProfile {
		case Prod, Dev, ManagedCluster:
			return nil
		default:
			return fmt.Errorf("Requested profile %s is invalid, valid options are dev, prod, or managed-cluster",
				requestedProfile)
		}
	}
	return nil
}

// ValidateActiveInstall enforces that only one install of Verrazzano is allowed.
func ValidateActiveInstall(client client.Client) error {
	vzList := &VerrazzanoList{}

	err := client.List(context.Background(), vzList)
	if err != nil {
		return err
	}

	if len(vzList.Items) != 0 {
		return fmt.Errorf("Only one install of Verrazzano is allowed")
	}

	return nil
}

// ValidateInProgress makes sure there is not an install, uninstall or upgrade in progress
func ValidateInProgress(old *Verrazzano) error {
	if old.Status.State == "" || old.Status.State == VzStateReady || old.Status.State == VzStateFailed || old.Status.State == VzStatePaused || old.Status.State == VzStateReconciling {
		return nil
	}
	return fmt.Errorf(validators.ValidateInProgressError)
}

func validateOCISecrets(client client.Client, spec *VerrazzanoSpec) error {
	if err := validateOCIDNSSecret(client, spec); err != nil {
		return err
	}
	if err := validateFluentdOCIAuthSecret(client, spec); err != nil {
		return err
	}
	return nil
}

func validateFluentdOCIAuthSecret(client client.Client, spec *VerrazzanoSpec) error {
	if spec.Components.Fluentd == nil || spec.Components.Fluentd.OCI == nil {
		return nil
	}
	apiSecretName := spec.Components.Fluentd.OCI.APISecret
	if len(apiSecretName) > 0 {
		secret := &corev1.Secret{}
		if err := validators.GetInstallSecret(client, apiSecretName, secret); err != nil {
			return err
		}
		// validate config secret
		if err := validators.ValidateFluentdConfigData(secret); err != nil {
			return err
		}
		// Validate key data exists and is a valid pem format
		pemData, err := validators.ValidateSecretKey(secret, validators.FluentdOCISecretPrivateKeyEntry, nil)
		if err != nil {
			return err
		}
		if err := validators.ValidatePrivateKey(secret.Name, pemData); err != nil {
			return err
		}
	}
	return nil
}

func validateOCIDNSSecret(client client.Client, spec *VerrazzanoSpec) error {
	if spec.Components.DNS == nil || spec.Components.DNS.OCI == nil {
		return nil
	}
	secret := &corev1.Secret{}
	ociDNSConfigSecret := spec.Components.DNS.OCI.OCIConfigSecret
	if err := validators.GetInstallSecret(client, ociDNSConfigSecret, secret); err != nil {
		return err
	}
	// Verify that the oci secret has one value
	if len(secret.Data) != 1 {
		return fmt.Errorf("Secret \"%s\" for OCI DNS should have one data key, found %v", ociDNSConfigSecret, len(secret.Data))
	}
	for key := range secret.Data {
		// validate auth_type
		var authProp validators.OciAuth
		if err := validators.ValidateSecretContents(secret.Name, secret.Data[key], &authProp); err != nil {
			return err
		}
		if authProp.Auth.AuthType != validators.InstancePrincipal && authProp.Auth.AuthType != validators.UserPrincipal && authProp.Auth.AuthType != "" {
			return fmt.Errorf("Authtype \"%v\" in OCI secret must be either '%s' or '%s'", authProp.Auth.AuthType, validators.UserPrincipal, validators.InstancePrincipal)
		}
		if authProp.Auth.AuthType == validators.UserPrincipal {
			if err := validators.ValidatePrivateKey(secret.Name, []byte(authProp.Auth.Key)); err != nil {
				return err
			}
		}
	}
	return nil
}

//ValidateInstallOverridesV1Beta1 checks that the overrides slice has only one override type per slice item for v1beta1
func ValidateInstallOverridesV1Beta1(overrides []v1beta1.Overrides) error {
	for _, override := range overrides {
		if err := isValidOverrideItems(override.ConfigMapRef, override.SecretRef, override.Values); err != nil {
			return err
		}
	}
	return nil
}

// ValidateInstallOverrides checks that the overrides slice has only one override type per slice item
func ValidateInstallOverrides(overrides []Overrides) error {
	for _, override := range overrides {
		if err := isValidOverrideItems(override.ConfigMapRef, override.SecretRef, override.Values); err != nil {
			return err
		}
	}
	return nil
}

func isValidOverrideItems(cm *corev1.ConfigMapKeySelector, s *corev1.SecretKeySelector, v *apiextensionsv1.JSON) error {
	items := 0
	if cm != nil {
		items++
	}
	if s != nil {
		items++
	}
	if v != nil {
		items++
	}
	if items > 1 {
		return fmt.Errorf("Invalid install overrides. Cannot specify more than one override type in the same list element")
	}
	if items == 0 {
		return fmt.Errorf("Invalid install overrides. No override specified")
	}
	return nil
}
