// Copyright (c) 2020, 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package v1alpha1

import (
	"context"
	"fmt"
	"github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/common"
	"github.com/verrazzano/verrazzano/platform-operator/internal/config"
	"strings"

	"github.com/verrazzano/verrazzano/pkg/semver"
	"go.uber.org/zap"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type authenticationType string

const (
	// UserPrincipal is default auth type
	userPrincipal authenticationType = "user_principal"
	// InstancePrincipal is used for instance principle auth type
	instancePrincipal authenticationType = "instance_principal"
	// InstancePrincipalDelegationToken is used for instance principle delegation token auth type
	InstancePrincipalDelegationToken authenticationType = "instance_principle_delegation_token"
	// UnknownAuthenticationType is used for none meaningful auth type
	UnknownAuthenticationType authenticationType = "unknown_auth_type"
	// ValidateInProgress error message
	ValidateInProgressError = "Updates to resource not allowed while uninstall or upgrade is in progress"

	ociDNSSecretFileName            = "oci.yaml"
	fluentdOCISecretConfigEntry     = "config"
	fluentdOCISecretPrivateKeyEntry = "key"
)

// OCI DNS Secret Auth
type authData struct {
	Region      string             `json:"region"`
	Tenancy     string             `json:"tenancy"`
	User        string             `json:"user"`
	Key         string             `json:"key"`
	Fingerprint string             `json:"fingerprint"`
	AuthType    authenticationType `json:"authtype"`
}

// OCI DNS Secret Auth Wrapper
type ociAuth struct {
	Auth authData `json:"auth"`
}

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

// ValidateUpgradeRequest Ensures hat an upgrade is requested as part of an update if necessary,
// and that the version of an upgrade request is valid.
func ValidateUpgradeRequest(current *Verrazzano, new *Verrazzano) error {
	if !config.Get().VersionCheckEnabled {
		zap.S().Infof("Version validation disabled")
		return nil
	}

	// Get the current BOM version
	bomVersion, err := common.GetCurrentBomVersion()
	if err != nil {
		return err
	}

	// Make sure the requested version matches what's in the BOM and is not < the current spec version
	newVerString := strings.TrimSpace(new.Spec.Version)
	currStatusVerString := strings.TrimSpace(current.Status.Version)
	currSpecVerString := strings.TrimSpace(current.Spec.Version)
	if len(newVerString) > 0 {
		return common.ValidateNewVersion(currStatusVerString, currSpecVerString, newVerString, bomVersion)
	}

	// No new version set, we haven't done any upgrade before but may need to do one before allowing any edits;
	// this forces the user to opt-in to an upgrade before/with any other update
	if err := common.CheckUpgradeRequired(strings.TrimSpace(current.Status.Version), bomVersion); err != nil {
		return err
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
	return fmt.Errorf(ValidateInProgressError)
}

// validateOCISecrets - Validate that the OCI DNS and Fluentd OCI secrets required by install exists, if configured
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
		if err := common.GetInstallSecret(client, apiSecretName, secret); err != nil {
			return err
		}
		// validate config secret
		if err := common.ValidateFluentdConfigData(secret); err != nil {
			return err
		}
		// Validate key data exists and is a valid pem format
		pemData, err := common.ValidateSecretKey(secret, fluentdOCISecretPrivateKeyEntry, nil)
		if err != nil {
			return err
		}
		if err := common.ValidatePrivateKey(secret.Name, pemData); err != nil {
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
	if err := common.GetInstallSecret(client, ociDNSConfigSecret, secret); err != nil {
		return err
	}
	// Verify that the oci secret has one value
	if len(secret.Data) != 1 {
		return fmt.Errorf("Secret \"%s\" for OCI DNS should have one data key, found %v", ociDNSConfigSecret, len(secret.Data))
	}
	for key := range secret.Data {
		// validate auth_type
		var authProp ociAuth
		if err := common.ValidateSecretContents(secret.Name, secret.Data[key], &authProp); err != nil {
			return err
		}
		if authProp.Auth.AuthType != instancePrincipal && authProp.Auth.AuthType != userPrincipal && authProp.Auth.AuthType != "" {
			return fmt.Errorf("Authtype \"%v\" in OCI secret must be either '%s' or '%s'", authProp.Auth.AuthType, userPrincipal, instancePrincipal)
		}
		if authProp.Auth.AuthType == userPrincipal {
			if err := common.ValidatePrivateKey(secret.Name, []byte(authProp.Auth.Key)); err != nil {
				return err
			}
		}
	}
	return nil
}

//ValidateVersionHigherOrEqual checks that currentVersion matches requestedVersion or is a higher version
func ValidateVersionHigherOrEqual(currentVersion string, requestedVersion string) bool {
	log := zap.S().With("validate", "version")
	log.Info("Validate version")
	if len(requestedVersion) == 0 {
		log.Error("Invalid requestedVersion of length 0.")
		return false
	}

	if len(currentVersion) == 0 {
		log.Error("Invalid currentVersion of length 0.")
		return false
	}

	requestedSemVer, err := semver.NewSemVersion(requestedVersion)
	if err != nil {
		log.Error(fmt.Sprintf("Invalid requestedVersion : %s, error: %v.", requestedVersion, err))
		return false
	}

	currentSemVer, err := semver.NewSemVersion(currentVersion)
	if err != nil {
		log.Error(fmt.Sprintf("Invalid currentVersion : %s, error: %v.", currentVersion, err))
		return false
	}

	return currentSemVer.IsEqualTo(requestedSemVer) || currentSemVer.IsGreatherThan(requestedSemVer)

}

// ValidateInstallOverrides checks that the overrides slice has only one override type per slice item
func ValidateInstallOverrides(Overrides []Overrides) error {
	overridePerItem := 0
	for _, override := range Overrides {
		if override.ConfigMapRef != nil {
			overridePerItem++
		}
		if override.SecretRef != nil {
			overridePerItem++
		}
		if override.Values != nil {
			overridePerItem++
		}
		if overridePerItem > 1 {
			return fmt.Errorf("Invalid install overrides. Cannot specify more than one override type in the same list element")
		}
		if overridePerItem == 0 {
			return fmt.Errorf("Invalid install overrides. No override specified")
		}
		overridePerItem = 0
	}
	return nil
}
