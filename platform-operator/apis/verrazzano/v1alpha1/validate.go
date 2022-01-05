// Copyright (c) 2020, 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package v1alpha1

import (
	"context"
	"errors"
	"fmt"
	"reflect"

	"sigs.k8s.io/yaml"

	"github.com/verrazzano/verrazzano/pkg/bom"
	"github.com/verrazzano/verrazzano/platform-operator/internal/config"

	"github.com/verrazzano/verrazzano/pkg/semver"
	"github.com/verrazzano/verrazzano/platform-operator/constants"

	"go.uber.org/zap"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
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
	ociSecretFileName                            = "oci.yaml"
)

// OCI Secret Auth
type authData struct {
	Region      string             `yaml:"region"`
	Tenancy     string             `yaml:"tenancy"`
	User        string             `yaml:"user"`
	Key         string             `yaml:"key"`
	Fingerprint string             `yaml:"fingerprint"`
	AuthType    authenticationType `yaml:"authtype"`
}
type ociAuth struct {
	auth authData `yaml:"auth"`
}

// GetCurrentBomVersion Get the version string from the bom and return it as a semver object
func GetCurrentBomVersion() (*semver.SemVersion, error) {
	bom, err := bom.NewBom(config.GetDefaultBOMFilePath())
	if err != nil {
		return nil, err
	}
	v := bom.GetVersion()
	if v == constants.BomVerrazzanoVersion {
		// This will only happen during development testing, the value doesn't matter
		v = "1.0.1"
	}
	return semver.NewSemVersion(fmt.Sprintf("v%s", v))
}

// ValidateVersion check that requestedVersion matches BOM requestedVersion
func ValidateVersion(requestedVersion string) error {
	if !config.Get().VersionCheckEnabled {
		zap.S().Infof("Version validation disabled")
		return nil
	}
	if len(requestedVersion) == 0 {
		return nil
	}
	requestedSemVer, err := semver.NewSemVersion(requestedVersion)
	if err != nil {
		return err
	}
	bomSemVer, err := GetCurrentBomVersion()
	if err != nil {
		return err
	}
	if !requestedSemVer.IsEqualTo(bomSemVer) {
		return fmt.Errorf("Requested version %s does not match BOM version %s", requestedSemVer.ToString(), bomSemVer.ToString())
	}
	return nil
}

// ValidateProfile check that requestedProfile is valid
func ValidateProfile(requestedProfile ProfileType) error {
	if len(requestedProfile) != 0 {
		switch requestedProfile {
		case Prod, Dev, ManagedCluster:
			return nil
		default:
			return fmt.Errorf("Requested profile %s is invalid.  Valid options are dev, prod, or managed-cluster",
				requestedProfile)
		}
	}
	return nil
}

// ValidateUpgradeRequest Ensures that for the upgrade case only the version field has changed
func ValidateUpgradeRequest(currentSpec *VerrazzanoSpec, newSpec *VerrazzanoSpec) error {
	if !config.Get().VersionCheckEnabled {
		zap.S().Infof("Version validation disabled")
		return nil
	}
	// Short-circuit if the version strings are the same
	if currentSpec.Version == newSpec.Version {
		return nil
	}
	if len(newSpec.Version) == 0 {
		// if we get here, the current version is not empty, but the new version is
		return fmt.Errorf("Requested version is not specified")
	}
	if err := ValidateVersion(newSpec.Version); err != nil {
		// new version is not nil, but we couldn't parse it
		return err
	}

	requestedSemVer, err := semver.NewSemVersion(newSpec.Version)
	if err != nil {
		// parse error on new version string
		return err
	}

	// Verify that the new version request is > than the currently version
	if len(currentSpec.Version) > 0 {
		currentSemVer, err := semver.NewSemVersion(currentSpec.Version)
		if err != nil {
			// Unable to parse the current spec version; this should never happen
			return err
		}
		if requestedSemVer.IsLessThan(currentSemVer) {
			return fmt.Errorf("Requested version %s is not newer than current version %s", requestedSemVer.ToString(), currentSemVer.ToString())
		}
	}

	// If any other field has changed from the stored spec return false
	if newSpec.Profile != currentSpec.Profile ||
		newSpec.EnvironmentName != currentSpec.EnvironmentName ||
		!reflect.DeepEqual(newSpec.Components, currentSpec.Components) {
		return errors.New("Configuration updates not allowed during upgrade between Verrazzano versions")
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
func ValidateInProgress(old *Verrazzano, new *Verrazzano) error {
	if old.Status.State == "" || old.Status.State == Ready || old.Status.State == Failed {
		return nil
	}
	// Allow enable component during install
	if old.Status.State == Installing {
		if isCoherenceEnabled(new.Spec.Components.CoherenceOperator) && !isCoherenceEnabled(old.Spec.Components.CoherenceOperator) {
			return nil
		}
		if isWebLogicEnabled(new.Spec.Components.WebLogicOperator) && !isWebLogicEnabled(old.Spec.Components.WebLogicOperator) {
			return nil
		}
	}
	return fmt.Errorf("Updates to resource not allowed while install, uninstall or upgrade is in progress")
}

// ValidateOciDNSSecret makes sure that the OCI DNS secret required by install exists
func ValidateOciDNSSecret(client client.Client, spec *VerrazzanoSpec) error {
	if spec.Components.DNS != nil && spec.Components.DNS.OCI != nil {
		secret := &corev1.Secret{}
		err := client.Get(context.TODO(), types.NamespacedName{Name: spec.Components.DNS.OCI.OCIConfigSecret, Namespace: constants.VerrazzanoInstallNamespace}, secret)
		if err != nil {
			if k8serrors.IsNotFound(err) {
				return fmt.Errorf("The secret \"%s\" must be created in the %s namespace before installing Verrrazzano for OCI DNS", spec.Components.DNS.OCI.OCIConfigSecret, constants.VerrazzanoInstallNamespace)
			}
			return err
		}

		// validate auth_type
		var authProp ociAuth
		err = yaml.Unmarshal(secret.Data[ociSecretFileName], &authProp)
		if err != nil {
			zap.S().Errorf("yaml unmarshalling failed due to %v", err)
			return err
		}
		if authProp.auth.AuthType != instancePrincipal && authProp.auth.AuthType != userPrincipal && authProp.auth.AuthType != "" {
			return fmt.Errorf("The authtype \"%v\" in OCI secret must be either '%s' or '%s'", authProp.auth.AuthType, userPrincipal, instancePrincipal)
		}
	}

	return nil
}

// isCoherenceEnabled returns true if the component is enabled, which is the default
func isCoherenceEnabled(comp *CoherenceOperatorComponent) bool {
	if comp == nil || comp.Enabled == nil {
		return true
	}
	return *comp.Enabled
}

// isWebLogicEnabled returns true if the component is enabled, which is the default
func isWebLogicEnabled(comp *WebLogicOperatorComponent) bool {
	if comp == nil || comp.Enabled == nil {
		return true
	}
	return *comp.Enabled
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
